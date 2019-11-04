package controller

import (
	"fmt"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/web"
)

type WelcomeController struct {
	cc *container.Container
}

func NewWelcomeController(cc *container.Container) web.Controller {
	return &WelcomeController{cc: cc}
}

func (w *WelcomeController) Register(router *web.Router) {
	router.Group("/welcome", func(router *web.Router) {
		router.Get("/", w.Hello).Name("welcome:hello")
		router.Get("/{name}/", w.Hello2).Name("welcome:hello2")
	})
}

func (w *WelcomeController) Hello(ctx web.Context) web.M {
	return web.M{
		"message": fmt.Sprintf("Hello, %s", ctx.Input("name")),
	}
}

func (w *WelcomeController) Hello2(req web.Request) string {
	return fmt.Sprintf("Hello, %s", req.PathVar("name"))
}
