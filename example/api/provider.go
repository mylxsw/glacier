package api

import (
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/example/api/controller"
	"github.com/mylxsw/hades"
)

type ServiceProvider struct{}

func (s ServiceProvider) Register(app *container.Container) {

}

func (s ServiceProvider) Boot(app *glacier.Glacier) {
	app.WebAppRouter(func(router *hades.Router, mw hades.RequestMiddleware) {
		router.WithMiddleware(mw.AccessLog()).Controllers("/api",
			controller.NewWelcomeController(app.Container()),
			controller.NewDemoController(),
		)
	})
}
