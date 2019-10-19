package controller

import (
	"fmt"
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

func (d *DemoController) Get(ctx *hades.WebContext, router *mux.Router) hades.HTTPResponse {
	var routes = make([]string, 0)
	if err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		pathRegexp, err := route.GetPathRegexp()
		if err != nil {
			return err
		}
		methods, err := route.GetMethods()
		if err != nil {
			return err
		}

		routes = append(routes, fmt.Sprintf("%s -> %s | %s | %s", route.GetName(), strings.Join(methods, "/"), pathTemplate, pathRegexp))

		return nil
	}); err != nil {
		return ctx.Error(err.Error(), http.StatusInternalServerError)
	}

	rr, _ := router.Get("demo:create").GetPathRegexp()

	return ctx.JSON(hades.M{
		"routes":                routes,
		"regex_for_demo:create": rr,
	})
}

func (d *DemoController) Create(ctx *hades.WebContext) hades.HTTPResponse {
	name := ctx.InputWithDefault("name", "Tom")
	if len(name) < 2 {
		return ctx.JSONError("invalid name", http.StatusUnprocessableEntity)
	}

	return ctx.JSON(hades.M{
		"name": strings.ToUpper(name),
	})
}
