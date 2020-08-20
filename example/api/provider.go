package api

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/example/api/controller"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/web"
)

type ServiceProvider struct{}

func (s ServiceProvider) Register(app container.Container) {

}

func (s ServiceProvider) Boot(app infra.Glacier) {
	app.WebAppRouter(func(router *web.Router, mw web.RequestMiddleware) {
		router.WithMiddleware(mw.AccessLog(log.Module("request"))).Controllers("/api",
			controller.NewWelcomeController(app.Container()),
			controller.NewDemoController(),
		)
	})
}
