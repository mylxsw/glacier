package controller

import (
	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/glacier/web"
	"net/http"
	"os"
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
		router.Post("/upload", d.Upload)
	})
}

func (d *DemoController) Upload(ctx web.Context) web.Response {
	up, err := ctx.File("file")
	if err != nil {
		return ctx.JSONError(err.Error(), http.StatusBadRequest)
	}

	if err := up.Store("/tmp/" + up.Name()); err != nil {
		return ctx.JSONError(err.Error(), http.StatusInternalServerError)
	}

	data, err := os.ReadFile("/tmp/" + up.Name())
	if err != nil {
		return ctx.JSONError(err.Error(), http.StatusInternalServerError)
	}

	return ctx.JSON(web.M{
		"filename": up.Name(),
		"size":     up.Size(),
		"content":  string(data),
	})
}

func (d *DemoController) RawRes(ctx web.Context) web.Response {
	return ctx.NewRawResponse(func(w http.ResponseWriter) {
		w.Header().Set("X-Ray", "OOPS")
		_, _ = w.Write([]byte("Hello, world"))
	})
}

func (d *DemoController) Create(ctx web.Context) web.Response {
	log.Debug("request body is ", string(ctx.Body()))

	name := ctx.InputWithDefault("name", "Tom")
	if len(name) < 2 {
		return ctx.JSONError("invalid name", http.StatusUnprocessableEntity)
	}

	return ctx.JSON(web.M{
		"name": strings.ToUpper(name),
		"body": string(ctx.Body()),
	})
}
