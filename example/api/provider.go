package api

import (
	"runtime/debug"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/example/api/controller"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
	"github.com/mylxsw/glacier/web"
)

type ServiceProvider struct{}

func (s ServiceProvider) Aggregates() []infra.Provider {
	return []infra.Provider{
		web.Provider(listener.FlagContext("listen")),
	}
}

func (s ServiceProvider) Register(app container.Container) {

}

func (s ServiceProvider) Boot(app infra.Glacier) {
	app.MustResolve(func(server web.Server) {
		server.Options(
			web.SetIgnoreLastSlashOption(true),
			web.SetExceptionHandlerOption(func(ctx web.Context, err interface{}) web.Response {
				log.Errorf("stack: %s", debug.Stack())
				return nil
			}),
			web.SetRouteHandlerOption(func(router *web.Router, mw web.RequestMiddleware) {
				router.WithMiddleware(mw.AccessLog(log.Module("request"))).
					Controllers(
						"/api",
						controller.NewWelcomeController(app.Container()),
						controller.NewDemoController(),
					)
			}),
			web.SetMuxRouteHandlerOption(func(router *mux.Router) {
				for _, r := range web.GetAllRoutes(router) {
					log.Debugf("route: %s -> %s | %s | %s", r.Name, r.Methods, r.PathTemplate, r.PathRegexp)
				}
			}),
		)
	})
}
