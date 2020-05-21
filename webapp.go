package glacier

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/graceful"
)

type InitRouterHandler func(router *web.Router, mw web.RequestMiddleware)
type InitMuxRouterHandler func(router *mux.Router)
type InitServerHandler func(server *http.Server, listener net.Listener)
type InitWebAppHandler func(cc container.Container, webApp *WebApp, conf *web.Config) error

func (glacier *glacierImpl) TCPListenerAddr(addr string) Glacier {
	if addr == "" {
		addr = ":8080"
	}

	glacier.httpListenAddr = addr
	return glacier
}

func (glacier *glacierImpl) TCPListener(ln net.Listener) Glacier {
	glacier.listener = ln
	glacier.httpListenAddr = ln.Addr().String()
	return glacier
}

// WithHttpServer with http server support
func (glacier *glacierImpl) WithHttpServer() Glacier {
	glacier.enableHTTPServer = true
	return glacier
}

// WebAppInit set a hook func for app init
func (glacier *glacierImpl) WebAppInit(initFunc InitWebAppHandler) Glacier {
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

func (app *WebApp) Init(initFunc InitWebAppHandler) error {
	if initFunc == nil {
		return nil
	}

	return initFunc(app.cc, app, app.conf)
}

// Start create the http server
func (app *WebApp) Start() error {
	return app.cc.ResolveWithError(func(conf *Config, listener net.Listener, gf *graceful.Graceful, logger log.Logger) error {
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

			logger.Debugf("prepare to shutdown http server...")
			if err := srv.Shutdown(ctx); err != nil {
				logger.Errorf("shutdown http server failed: %s", err)
			}

			logger.Warning("http server has been shutdown")
		})

		go func() {
			logger.Debugf("http server started, listening on %s", listener.Addr())
			if err := srv.Serve(listener); err != nil {
				logger.Debugf("http server stopped: %s", err)
				if err != http.ErrServerClosed {
					gf.Shutdown()
				}
			}
		}()

		return nil
	})
}

func (app *WebApp) router() http.Handler {
	router := web.NewRouterWithContainer(app.cc, app.conf)
	mw := web.NewRequestMiddleware()

	app.initRouter(router, mw)

	return router.Perform(app.exceptionHandler, func(muxRouter *mux.Router) {
		if app.muxRouter != nil {
			app.muxRouter(muxRouter)
		}

		app.cc.MustSingleton(func() *mux.Router { return muxRouter })
		app.cc.MustSingleton(func() http.Handler { return muxRouter })
	})
}
