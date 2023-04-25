package api

import (
	"runtime/debug"

	"github.com/mylxsw/glacier/log"

	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/example/api/controller"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
	"github.com/mylxsw/glacier/web"
)

type ServiceProvider struct{}

func (s ServiceProvider) Priority() int {
	return 100
}

func (s ServiceProvider) Aggregates() []infra.Provider {
	return []infra.Provider{
		web.Provider(
			listener.FlagContext("listen"),
			web.SetRouteHandlerOption(s.router),
			web.SetOptions(func(cc infra.Resolver) []web.Option {
				return []web.Option{
					web.SetIgnoreLastSlashOption(true),
					web.SetExceptionHandlerOption(s.exceptionHandler),
					web.SetMuxRouteHandlerOption(s.muxRouteHandler),
				}
			}),
		),
	}
}

func (s ServiceProvider) router(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
	auth := mw.AuthHandler(func(ctx web.Context, typ, credential string) error {
		ctx.Provide(func() *controller.User { return &controller.User{ID: 1, Name: "mylxsw"} })
		return nil
	})
	router.WithMiddleware(mw.AccessLog(log.Default()), auth).
		Controllers(
			"/api",
			controller.NewWelcomeController(cc),
			controller.NewDemoController(),
		)
}

func (s ServiceProvider) muxRouteHandler(cc infra.Resolver, router *mux.Router) {
	for _, r := range web.GetAllRoutes(router) {
		log.Debugf("[example] route: %s -> %s | %s | %s", r.Name, r.Methods, r.PathTemplate, r.PathRegexp)
	}
}

func (s ServiceProvider) exceptionHandler(ctx web.Context, err interface{}) web.Response {
	log.Errorf("[example] stack: %s", debug.Stack())
	return nil
}

func (s ServiceProvider) Register(app infra.Binder) {
	log.Debug("[example] provider api.ServiceProvider loaded")
}

func (s ServiceProvider) Boot(app infra.Resolver) {}
