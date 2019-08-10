package glacier

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/graceful"
	"github.com/mylxsw/hades"
)

type InitRouterHandler func(router *hades.Router, mw hades.RequestMiddleware)
type InitMuxRouterHandler func(router *mux.Router)

// WebApp is the web app
type WebApp struct {
	cc         *container.Container
	initRouter InitRouterHandler
	muxRouter  InitMuxRouterHandler
}

// NewWebApp create a new WebApp
func NewWebApp(cc *container.Container, initRouter InitRouterHandler) *WebApp {
	return &WebApp{
		cc:         cc,
		initRouter: initRouter,
	}
}

func (app *WebApp) MuxRouter(f InitMuxRouterHandler) {
	app.muxRouter = f
}

func (app *WebApp) Init(initFunc interface{}) error {
	return app.cc.ResolveWithError(initFunc)
}

// Start create the http server
func (app *WebApp) Start() error {
	return app.cc.ResolveWithError(func(conf *Config, gf *graceful.Graceful) error {
		listener, err := net.Listen("tcp", conf.HttpListen)
		if err != nil {
			return err
		}

		srv := &http.Server{
			Handler:      app.router(),
			WriteTimeout: conf.HttpWriteTimeout,
			ReadTimeout:  conf.HttpReadTimeout,
			IdleTimeout:  conf.HttpIdleTimeout,
		}

		gf.AddShutdownHandler(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			log.Debugf("prepare to shutdown http server...")
			if err := srv.Shutdown(ctx); err != nil {
				log.Errorf("shutdown http server failed: %s", err)
			}

			log.Warning("http server has been shutdown")
		})

		go func() {
			log.Debugf("http server started, listening on %s", conf.HttpListen)
			if err := srv.Serve(listener); err != nil {
				log.Debugf("http server stopped: %s", err)
				if err != http.ErrServerClosed {
					gf.Shutdown()
				}
			}
		}()

		return nil
	})
}

func (app *WebApp) router() *mux.Router {
	router := hades.NewRouterWithContainer(app.cc)
	mw := hades.NewRequestMiddleware()

	app.initRouter(router, mw)

	muxRouter := router.Perform()
	if app.muxRouter != nil {
		app.muxRouter(muxRouter)
	}

	return muxRouter
}
