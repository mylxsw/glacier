package controller

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mylxsw/hades"
)

type DemoController struct {
}

func NewDemoController() hades.Controller {
	return &DemoController{}
}

func (d *DemoController) Register(router *hades.Router) {
	router.Group("/demo", func(router *hades.Router) {
		router.Post("/", d.Create).Custom(func(rou *mux.Route) {
			rou.Name("demo:create")
		})
		router.Get("/", d.Get).Custom(func(rou *mux.Route) {
			rou.Name("demo:routes")
		})
	})
}

func (d *DemoController) Get(ctx hades.Context, router *mux.Router) hades.Response {
	rr, _ := router.Get("demo:create").GetPathRegexp()
	routes := hades.GetAllRoutes(router)

	return ctx.JSON(hades.M{
		"routes":                routes,
		"regex_for_demo:create": rr,
	})
}

func (d *DemoController) Create(ctx hades.Context) hades.Response {
	name := ctx.InputWithDefault("name", "Tom")
	if len(name) < 2 {
		return ctx.JSONError("invalid name", http.StatusUnprocessableEntity)
	}

	return ctx.JSON(hades.M{
		"name": strings.ToUpper(name),
	})
}
