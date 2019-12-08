package glacier

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/graceful"
)

type InitRouterHandler func(router *web.Router, mw web.RequestMiddleware)
type InitMuxRouterHandler func(router *mux.Router)
type InitServerHandler func(server *http.Server, listener net.Listener)

// WithHttpServer with http server support
func (glacier *glacierImpl) WithHttpServer(listenAddr string) Glacier {
	if listenAddr == "" {
		listenAddr = ":19950"
	}

	glacier.httpListenAddr = listenAddr

	return glacier
}


// WebAppInit set a hook func for app init
func (glacier *glacierImpl) WebAppInit(initFunc interface{}) Glacier {
	glacier.webAppInitFunc = initFunc
	return glacier
}

// WebAppServerInit is a function for initialize http server
func (glacier *glacierImpl) WebAppServerInit(handler InitServerHandler) Glacier {
	glacier.webAppServerFunc = handler
	return glacier
}

// WebAppRouter add routes for http server
func (glacier *glacierImpl) WebAppRouter(handler InitRouterHandler) Glacier {
	glacier.webAppRouterFunc = handler
	return glacier
}

// WebAppMuxRouter add mux routes for http server
func (glacier *glacierImpl) WebAppMuxRouter(handler InitMuxRouterHandler) Glacier {
	glacier.webAppMuxRouterFunc = handler
	return glacier
}

// WebAppExceptionHandler set exception handler for web app
func (glacier *glacierImpl) WebAppExceptionHandler(handler web.ExceptionHandler) Glacier {
	glacier.webAppExceptionHandler = handler
	return glacier
}


// WebApp is the web app
type WebApp struct {
	cc                 container.Container
	initRouter         InitRouterHandler
	initServerListener InitServerHandler
	muxRouter          InitMuxRouterHandler
	exceptionHandler   web.ExceptionHandler

	conf *web.Config
}

// NewWebApp create a new WebApp
func NewWebApp(cc container.Container, initRouter InitRouterHandler, initServerListener InitServerHandler) *WebApp {
	return &WebApp{
		cc:                 cc,
		initRouter:         initRouter,
		initServerListener: initServerListener,
		conf:               web.DefaultConfig(),
	}
}

// UpdateConfig update WebAPP configurations
func (app *WebApp) UpdateConfig(cb func(conf *web.Config)) {
	cb(app.conf)
}

// ExceptionHandler set exception handler
func (app *WebApp) ExceptionHandler(handler web.ExceptionHandler) {
	app.exceptionHandler = handler
}

func (app *WebApp) MuxRouter(f InitMuxRouterHandler) {
	app.muxRouter = f
}

func (app *WebApp) Init(initFunc interface{}) error {
	if initFunc == nil {
		return nil
	}

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

		if app.initServerListener != nil {
			app.initServerListener(srv, listener)
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
	router := web.NewRouterWithContainer(app.cc, app.conf)
	mw := web.NewRequestMiddleware()

	app.initRouter(router, mw)

	muxRouter := router.Perform(app.exceptionHandler)
	if app.muxRouter != nil {
		app.muxRouter(muxRouter)
	}

	app.cc.MustSingleton(func() *mux.Router {
		return muxRouter
	})

	return muxRouter
}
