package web

import (
	"context"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"time"

	"github.com/mylxsw/glacier/log"

	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/infra"
)

type Option func(cc infra.Resolver, conf *Config)

type Server interface {
	Start(listener net.Listener) error
	Options(cc infra.Resolver, options ...Option)
}

// serverImpl is the web app
type serverImpl struct {
	cc     infra.Container
	conf   *Config
	status ServerStatus
}

type ServerStatus int

const (
	serverStatusInit    = 0
	serverStatusStarted = 1
)

// NewServer create a new serverImpl
func NewServer(cc infra.Container, options ...Option) Server {
	server := &serverImpl{
		cc:     cc,
		conf:   DefaultConfig(),
		status: serverStatusInit,
	}
	server.Options(cc, options...)

	return server
}

func (app *serverImpl) Options(cc infra.Resolver, options ...Option) {
	if app.status > serverStatusInit {
		panic("can not change options after server started")
	}

	for _, opt := range options {
		opt(cc, app.conf)
	}
}

// Start create the http server
func (app *serverImpl) Start(listener net.Listener) error {
	if app.conf.initHandler != nil {
		if err := app.conf.initHandler(app.cc, app, app.conf); err != nil {
			return err
		}
	}

	app.status = serverStatusStarted
	return app.cc.Resolve(func(gf infra.Graceful) error {
		srv := &http.Server{
			Handler:           app.router(app.cc),
			WriteTimeout:      app.conf.HttpWriteTimeout,
			ReadTimeout:       app.conf.HttpReadTimeout,
			IdleTimeout:       app.conf.HttpIdleTimeout,
			ReadHeaderTimeout: app.conf.HttpReadHeaderTimeout,
		}

		if app.conf.serverConfigHandler != nil {
			app.conf.serverConfigHandler(srv, listener)
		}

		gf.AddShutdownHandler(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if infra.DEBUG {
				log.Debugf("[glacier] prepare to shutdown http server...")
			}

			if err := srv.Shutdown(ctx); err != nil {
				log.Errorf("[glacier] shutdown http server failed: %s", err)
			}

			if infra.DEBUG {
				log.Debug("[glacier] http server has been shutdown")
			}
		})

		if infra.DEBUG {
			log.Debugf("[glacier] http server started, listening on %s", listener.Addr())
		}

		if err := srv.Serve(listener); err != nil {
			if infra.DEBUG {
				log.Debugf("[glacier] http server stopped: %s", err)
			}

			if !errors.Is(err, http.ErrServerClosed) {
				gf.Shutdown()
			}
		}

		return nil
	})
}

func (app *serverImpl) router(cc infra.Container) http.Handler {
	router := NewRouterWithContainer(cc, app.conf)
	mw := NewRequestMiddleware()

	if app.conf.routeHandler != nil {
		app.conf.routeHandler(cc, router, mw)
	}

	return router.Perform(app.conf.exceptionHandler, func(muxRouter *mux.Router) {
		if app.conf.muxRouteHandler == nil {
			return
		}

		app.conf.muxRouteHandler(cc, muxRouter)
	})
}
