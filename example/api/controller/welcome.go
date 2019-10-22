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
		router.Get("/", w.Hello).Name("welcome:hello")
		router.Get("/{name}/", w.Hello2).Name("welcome:hello2")
	})
}

func (w *WelcomeController) Hello(ctx hades.Context) hades.M {
	return hades.M{
		"message": fmt.Sprintf("Hello, %s", ctx.Input("name")),
	}
}

func (w *WelcomeController) Hello2(req hades.Request) string {
	return fmt.Sprintf("Hello, %s", req.PathVar("name"))
}
