package controller

import (
	"fmt"

	"github.com/mylxsw/container"
	"github.com/mylxsw/hades"
)

type WelcomeController struct {
	cc *container.Container
}

func NewWelcomeController(cc *container.Container) hades.Controller {
	return &WelcomeController{cc: cc}
}

func (w *WelcomeController) Register(router *hades.Router) {
	router.Group("/welcome", func(router *hades.Router) {
		router.Get("/", w.Hello)
		router.Get("/{name}/", w.Hello2)
	})
}

func (w *WelcomeController) Hello(ctx *hades.WebContext) hades.HTTPResponse {
	return ctx.JSON(hades.M{
		"message": fmt.Sprintf("Hello, %s", ctx.Input("name")),
	})
}

func (w *WelcomeController) Hello2(req *hades.HttpRequest) string {
	return fmt.Sprintf("Hello, %s", req.PathVar("name"))
}
