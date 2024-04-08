package controller

import (
	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/web"
	"net/http"
	"strings"
)

type User struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type DemoController struct {
}

func NewDemoController() web.Controller {
	return &DemoController{}
}

func (d *DemoController) Register(router web.Router) {
	router.Group("/demo", func(router web.Router) {
		router.Post("/", d.Create).Custom(func(rou *mux.Route) {
			rou.Name("demo:create")
		})
		router.Get("/raw", d.RawRes)
	})
}

func (d *DemoController) RawRes(ctx web.Context) web.Response {
	return ctx.NewRawResponse(func(w http.ResponseWriter) {
		w.Header().Set("X-Ray", "OOPS")
		_, _ = w.Write([]byte("Hello, world"))
	})
}

func (d *DemoController) Create(ctx web.Context) web.Response {
	name := ctx.InputWithDefault("name", "Tom")
	if len(name) < 2 {
		return ctx.JSONError("invalid name", http.StatusUnprocessableEntity)
	}

	return ctx.JSON(web.M{
		"name": strings.ToUpper(name),
	})
}
