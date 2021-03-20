package api

import (
	"runtime/debug"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/example/api/controller"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
	"github.com/mylxsw/glacier/web"
)

type ServiceProvider struct{}

func (s ServiceProvider) Aggregates() []infra.Provider {
	return []infra.Provider{
		web.Provider(
			listener.FlagContext("listen"),
			web.SetIgnoreLastSlashOption(true),
			web.SetExceptionHandlerOption(s.exceptionHandler),
			web.SetRouteHandlerOption(s.router),
			web.SetMuxRouteHandlerOption(s.muxRouteHandler),
		),
	}
}

func (s ServiceProvider) router(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
	router.WithMiddleware(mw.AccessLog(log.Module("request"))).
		Controllers(
			"/api",
			controller.NewWelcomeController(cc),
			controller.NewDemoController(),
		)
}

func (s ServiceProvider) muxRouteHandler(router *mux.Router) {
	for _, r := range web.GetAllRoutes(router) {
		log.Debugf("route: %s -> %s | %s | %s", r.Name, r.Methods, r.PathTemplate, r.PathRegexp)
	}
}

func (s ServiceProvider) exceptionHandler(ctx web.Context, err interface{}) web.Response {
	log.Errorf("stack: %s", debug.Stack())
	return nil
}

func (s ServiceProvider) Register(app infra.Binder) {}

func (s ServiceProvider) Boot(app infra.Resolver) {}
