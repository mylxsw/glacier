package controller

import (
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/web"
)

type WelcomeController struct {
	cc container.Container
}

func NewWelcomeController(cc container.Container) web.Controller {
	return &WelcomeController{cc: cc}
}

func (w *WelcomeController) Register(router *web.Router) {
	router.Any("/", w.Home).Name("welcome:home")
}

type WelcomeMessage struct {
	Version string `json:"version"`
}

func (w *WelcomeController) Home(ctx web.Context, req web.Request) WelcomeMessage {
	return WelcomeMessage{Version: w.cc.MustGet("version").(string)}
}
