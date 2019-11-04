package controller

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/web"
)

type DemoController struct {
}

func NewDemoController() web.Controller {
	return &DemoController{}
}

func (d *DemoController) Register(router *web.Router) {
	router.Group("/demo", func(router *web.Router) {
		router.Post("/", d.Create).Custom(func(rou *mux.Route) {
			rou.Name("demo:create")
		})
		router.Get("/", d.Get).Custom(func(rou *mux.Route) {
			rou.Name("demo:routes")
		})
	})
}

func (d *DemoController) Get(ctx web.Context, router *mux.Router) web.Response {
	rr, _ := router.Get("demo:create").GetPathRegexp()
	routes := web.GetAllRoutes(router)

	return ctx.JSON(web.M{
		"routes":                routes,
		"regex_for_demo:create": rr,
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
