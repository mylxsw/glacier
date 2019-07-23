package glacier

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/asteria/misc"
	"github.com/mylxsw/go-toolkit/container"
	"github.com/mylxsw/go-toolkit/graceful"
	"github.com/mylxsw/go-toolkit/web"
)

type InitRouterHandler func(router *web.Router, mw web.RequestMiddleware)

// WebApp is the web app
type WebApp struct {
	cc         *container.Container
	initRouter InitRouterHandler
}

// NewWebApp create a new WebApp
func NewWebApp(cc *container.Container, initRouter InitRouterHandler) *WebApp {
	return &WebApp{
		cc:         cc,
		initRouter: initRouter,
	}
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
			fmt.Println(misc.CallGraph(1))
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
	router := web.NewRouterWithContainer(app.cc)
	mw := web.NewRequestMiddleware()

	app.initRouter(router, mw)

	muxRouter := router.Perform()
	return muxRouter
}
