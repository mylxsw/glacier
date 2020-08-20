package glacier

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/graceful"
)

// WithHttpServer with http server support
func (glacier *glacierImpl) WithHttpServer(builder infra.ListenerBuilder) infra.Glacier {
	glacier.enableHTTPServer = true
	glacier.tcpListenerBuilder = builder
	return glacier
}

// WebAppInit set a hook func for app init
func (glacier *glacierImpl) WebAppInit(initFunc infra.InitWebAppHandler) infra.Glacier {
	glacier.webAppInitFunc = initFunc
	return glacier
}

// WebAppServerInit is a function for initialize http server
func (glacier *glacierImpl) WebAppServerInit(handler infra.InitServerHandler) infra.Glacier {
	glacier.webAppServerFunc = handler
	return glacier
}

// WebAppRouter add routes for http server
func (glacier *glacierImpl) WebAppRouter(handler infra.InitRouterHandler) infra.Glacier {
	glacier.webAppRouterFunc = handler
	return glacier
}

// WebAppMuxRouter add mux routes for http server
func (glacier *glacierImpl) WebAppMuxRouter(handler infra.InitMuxRouterHandler) infra.Glacier {
	glacier.webAppMuxRouterFunc = handler
	return glacier
}

// WebAppExceptionHandler set exception handler for web app
func (glacier *glacierImpl) WebAppExceptionHandler(handler web.ExceptionHandler) infra.Glacier {
	glacier.webAppExceptionHandler = handler
	return glacier
}

// WebServer is the web app
type WebServer struct {
	cc                 container.Container
	initRouter         infra.InitRouterHandler
	initServerListener infra.InitServerHandler
	muxRouter          infra.InitMuxRouterHandler
	exceptionHandler   web.ExceptionHandler

	conf *web.Config
}

// NewWebApp create a new WebServer
func NewWebApp(cc container.Container, initRouter infra.InitRouterHandler, initServerListener infra.InitServerHandler) infra.Web {
	return &WebServer{
		cc:                 cc,
		initRouter:         initRouter,
		initServerListener: initServerListener,
		conf:               web.DefaultConfig(),
	}
}

// UpdateConfig update WebAPP configurations
func (app *WebServer) UpdateConfig(cb func(conf *web.Config)) {
	cb(app.conf)
}

// ExceptionHandler set exception handler
func (app *WebServer) ExceptionHandler(handler web.ExceptionHandler) {
	app.exceptionHandler = handler
}

func (app *WebServer) MuxRouter(f infra.InitMuxRouterHandler) {
	app.muxRouter = f
}

func (app *WebServer) Init(initFunc infra.InitWebAppHandler) error {
	if initFunc == nil {
		return nil
	}

	return initFunc(app.cc, app, app.conf)
}

// Start create the http server
func (app *WebServer) Start() error {
	return app.cc.ResolveWithError(func(conf *Config, listener net.Listener, gf graceful.Graceful, logger log.Logger) error {
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

func (app *WebServer) router() http.Handler {
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
