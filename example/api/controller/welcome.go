package controller

import (
	"errors"
	"fmt"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/go-utils/ternary"
)

type WelcomeController struct {
	cc infra.Resolver
}

func NewWelcomeController(cc infra.Resolver) web.Controller {
	return &WelcomeController{cc: cc}
}

func (w *WelcomeController) Register(router web.Router) {
	router.Group("/welcome", func(router web.Router) {
		router.Get("/", w.Hello).Name("welcome:hello")
		router.Get("/hello/{name}/", w.Hello2).Name("welcome:hello2")
		router.Get("/hello2/", w.Hello2).Name("welcome:hello2.1")
		router.Get("/panic/", w.Hello3).Name("welcome:panic")
	})
}

func (w *WelcomeController) Hello(ctx web.Context) web.M {
	panicURL, _ := ctx.RouteURL("welcome:panic")
	return web.M{
		"message":   fmt.Sprintf("Hello, %s", ctx.Input("name")),
		"path":      ctx.Request().Raw().RequestURI,
		"url":       ctx.Request().Raw().URL,
		"name":      ctx.CurrentRoute().GetName(),
		"panic_url": panicURL.String(),
	}
}

func (w *WelcomeController) Hello2(req web.Request, user *User) string {
	return fmt.Sprintf("Hello, %s", ternary.IfLazy(req.PathVar("name") == "", func() string { return user.Name }, func() string { return req.PathVar("name") }))
}

func (w *WelcomeController) Hello3(req web.Request) {
	panic(errors.New("hello"))
}
