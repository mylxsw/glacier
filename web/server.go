package web

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/graceful"
)

type Option func(conf *Config)

type Server interface {
	Start(listener net.Listener) error
	Options(options ...Option)
}

// serverImpl is the web app
type serverImpl struct {
	cc     container.Container
	conf   *Config
	status ServerStatus
}

type ServerStatus int

const (
	serverStatusInit    = 0
	serverStatusStarted = 1
)

// NewServer create a new serverImpl
func NewServer(cc container.Container, options ...Option) Server {
	server := &serverImpl{
		cc:     cc,
		conf:   DefaultConfig(),
		status: serverStatusInit,
	}
	server.Options(options...)

	return server
}

func (app *serverImpl) Options(options ...Option) {
	if app.status > serverStatusInit {
		panic("can not change options after server started")
	}

	for _, opt := range options {
		opt(app.conf)
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
	return app.cc.ResolveWithError(func(gf graceful.Graceful, logger log.Logger) error {
		srv := &http.Server{
			Handler:      app.router(),
			WriteTimeout: app.conf.HttpWriteTimeout,
			ReadTimeout:  app.conf.HttpReadTimeout,
			IdleTimeout:  app.conf.HttpIdleTimeout,
		}

		if app.conf.listenerHandler != nil {
			app.conf.listenerHandler(srv, listener)
		}

		gf.AddShutdownHandler(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if logger.DebugEnabled() {
				logger.Debugf("prepare to shutdown http server...")
			}

			if err := srv.Shutdown(ctx); err != nil {
				logger.Errorf("shutdown http server failed: %s", err)
			}

			if logger.WarningEnabled() {
				logger.Warning("http server has been shutdown")
			}
		})

		if logger.DebugEnabled() {
			logger.Debugf("http server started, listening on %s", listener.Addr())
		}

		if err := srv.Serve(listener); err != nil {
			if logger.DebugEnabled() {
				logger.Debugf("http server stopped: %s", err)
			}

			if err != http.ErrServerClosed {
				gf.Shutdown()
			}
		}

		return nil
	})
}

func (app *serverImpl) router() http.Handler {
	router := NewRouterWithContainer(app.cc, app.conf)
	mw := NewRequestMiddleware()

	if app.conf.routeHandler != nil {
		app.conf.routeHandler(router, mw)
	}

	return router.Perform(app.conf.exceptionHandler, func(muxRouter *mux.Router) {
		if app.conf.muxRouteHandler == nil {
			return
		}

		app.conf.muxRouteHandler(muxRouter)
	})
}
