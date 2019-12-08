package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/mylxsw/container"
)

// Router 定制的路由
type Router struct {
	router     *mux.Router
	container  container.Container
	routes     []*RouteRule
	decorators []HandlerDecorator
	prefix     string
}

// RouteRule 路由规则
type RouteRule struct {
	method     string
	path       string
	webHandler WebHandler
	decorators []HandlerDecorator
	custom     func(rou *mux.Route)

	name    string
	host    string
	queries []string
	schemes []string
	headers []string
}

// Custom add more control to underlying mux.Route
func (rr *RouteRule) Custom(custom func(rou *mux.Route)) *RouteRule {
	rr.custom = custom
	return rr
}

// Name sets the name for the route, used to build URLs.
// It is an error to call Name more than once on a route.
func (rr *RouteRule) Name(name string) *RouteRule {
	rr.name = name
	return rr
}

// Headers adds a matcher for request header values.
// It accepts a sequence of key/value pairs to be matched. For example:
//
//     r.Headers("Content-Type", "application/json",
//               "X-Requested-With", "XMLHttpRequest")
//
// The above route will only match if both request header values match.
// If the value is an empty string, it will match any value if the key is set.
func (rr *RouteRule) Headers(pairs ...string) *RouteRule {
	rr.headers = pairs
	return rr
}

// Queries adds a matcher for URL query values.
// It accepts a sequence of key/value pairs. Values may define variables.
// For example:
//
//     r.Queries("foo", "bar", "id", "{id:[0-9]+}")
//
// The above route will only match if the URL contains the defined queries
// values, e.g.: ?foo=bar&id=42.
//
// If the value is an empty string, it will match any value if the key is set.
//
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
func (rr *RouteRule) Queries(pairs ...string) *RouteRule {
	rr.queries = pairs
	return rr
}

// Schemes adds a matcher for URL schemes.
// It accepts a sequence of schemes to be matched, e.g.: "http", "https".
func (rr *RouteRule) Schemes(schemes ...string) *RouteRule {
	rr.schemes = schemes
	return rr
}

// Host adds a matcher for the URL host.
// It accepts a template with zero or more URL variables enclosed by {}.
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next dot.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     r.Host("www.example.com")
//     r.Host("{subdomain}.domain.com")
//     r.Host("{subdomain:[a-z]+}.domain.com")
//
// Variable names must be unique in a given route. They can be retrieved
// calling mux.Vars(request).
func (rr *RouteRule) Host(tpl string) *RouteRule {
	rr.host = tpl
	return rr
}

// NewRouter 创建一个路由器
func NewRouter(conf *Config, decorators ...HandlerDecorator) *Router {
	return NewRouterWithContainer(container.New(), conf, decorators...)
}

// NewRouterWithContainer 创建一个路由器，带有依赖注入容器支持
func NewRouterWithContainer(c container.Container, conf *Config, decorators ...HandlerDecorator) *Router {
	cc := container.Extend(c)
	cc.MustSingleton(func() *schema.Decoder {
		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)
		return decoder
	})

	cc.MustBindValue("config", conf)
	cc.MustSingleton(func() *Config { return conf })

	return create(cc, mux.NewRouter(), decorators...)
}

// create 创建定制路由器
func create(c container.Container, router *mux.Router, decorators ...HandlerDecorator) *Router {
	return &Router{
		router:     router,
		routes:     []*RouteRule{},
		decorators: decorators,
		prefix:     "",
		container:  c,
	}
}

// Group 创建路由组
func (router *Router) Group(prefix string, f func(rou *Router), decors ...HandlerDecorator) {
	r := create(router.container, router.router, decors...)
	r.prefix = prefix

	f(r)
	r.parse()

	for _, route := range r.GetRoutes() {
		rr := router.addWebHandler(route.method, route.path, route.webHandler, route.decorators...)

		rr.name = route.name
		rr.headers = route.headers
		rr.schemes = route.schemes
		rr.queries = route.queries
		rr.host = route.host

		rr.Custom(route.custom)
	}
}

// Perform 将路由规则添加到路由器
func (router *Router) Perform(exceptionHandler ExceptionHandler) *mux.Router {
	for _, r := range router.routes {
		var handler http.Handler

		// cors support and exception handler
		corsHandler := func(rt *RouteRule) WebHandler {
			return func(ctx Context) (resp Response) {
				defer func() {
					if err := recover(); err != nil {
						if exceptionHandler != nil {
							resp = exceptionHandler(ctx, err)
						}

						if resp == nil {
							_resp, err := ErrorToResponse(ctx, err)
							if err != nil {
								resp = ctx.Error(fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
							} else {
								resp = _resp
							}
						}
					}
				}()

				if ctx.Method() == http.MethodOptions {
					return ctx.NewHTMLResponse("")
				}

				return rt.webHandler(ctx)
			}
		}

		handler = newWebHandler(router.container, corsHandler(r), r.decorators...)
		route := router.router.Handle(r.path, handler)
		if r.method != "" {
			route.Methods(r.method, http.MethodOptions)
		}

		if r.host != "" {
			route.Host(r.host)
		}

		if r.queries != nil {
			route.Queries(r.queries...)
		}

		if r.schemes != nil {
			route.Schemes(r.schemes...)
		}

		if r.name != "" {
			route.Name(r.name)
		}

		if r.headers != nil {
			route.Headers(r.headers...)
		}

		if r.custom != nil {
			r.custom(route)
		}
	}

	return router.router
}

// GetRoutes 获取所有路由规则
func (router *Router) GetRoutes() []*RouteRule {
	return router.routes
}

func (router *Router) addWebHandler(method string, path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	rou := &RouteRule{
		method:     method,
		path:       path,
		webHandler: handler,
		decorators: middlewares,
	}
	router.routes = append(router.routes, rou)

	return rou
}

// Parse 解析路由规则，将中间件信息同步到路由规则
func (router *Router) parse() {
	for i := range router.routes {
		router.routes[i].path = fmt.Sprintf("%s/%s", strings.TrimRight(router.prefix, "/"), strings.TrimLeft(router.routes[i].path, "/"))
		router.routes[i].decorators = append(router.routes[i].decorators, router.decorators...)
	}
}

func (router *Router) addHandler(method string, path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler(method, path, func(ctx Context) Response {
		return ctx.Resolve(handler)
	}, middlewares...)
}

// Any 指定所有请求方式的路由规则
func (router *Router) Any(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("", path, handler, middlewares...)
}

// Get 指定所有GET方式的路由规则
func (router *Router) Get(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("GET", path, handler, middlewares...)
}

// Post 指定所有Post方式的路由规则
func (router *Router) Post(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("POST", path, handler, middlewares...)
}

// Delete 指定所有DELETE方式的路由规则
func (router *Router) Delete(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("DELETE", path, handler, middlewares...)
}

// Put 指定所有Put方式的路由规则
func (router *Router) Put(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("PUT", path, handler, middlewares...)
}

// Patch 指定所有Patch方式的路由规则
func (router *Router) Patch(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("PATCH", path, handler, middlewares...)
}

// Head 指定所有Head方式的路由规则
func (router *Router) Head(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("HEAD", path, handler, middlewares...)
}

// Options 指定所有OPTIONS方式的路由规则
func (router *Router) Options(path string, handler interface{}, middlewares ...HandlerDecorator) *RouteRule {
	return router.addHandler("OPTIONS", path, handler, middlewares...)
}

// WebAny 指定所有请求方式的路由规则，WebHandler方式
func (router *Router) WebAny(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("", path, handler, middlewares...)
}

// WebGet 指定GET请求方式的路由规则，WebHandler方式
func (router *Router) WebGet(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("GET", path, handler, middlewares...)
}

// WebPost 指定POST请求方式的路由规则，WebHandler方式
func (router *Router) WebPost(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("POST", path, handler, middlewares...)
}

// WebPut 指定所有Put方式的路由规则，WebHandler方式
func (router *Router) WebPut(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("PUT", path, handler, middlewares...)
}

// WebDelete 指定所有DELETE方式的路由规则，WebHandler方式
func (router *Router) WebDelete(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("DELETE", path, handler, middlewares...)
}

// WebPatch 指定所有PATCH方式的路由规则，WebHandler方式
func (router *Router) WebPatch(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("PATCH", path, handler, middlewares...)
}

// WebHead 指定所有HEAD方式的路由规则，WebHandler方式
func (router *Router) WebHead(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("HEAD", path, handler, middlewares...)
}

// WebOptions 指定所有OPTIONS方式的路由规则，WebHandler方式
func (router *Router) WebOptions(path string, handler WebHandler, middlewares ...HandlerDecorator) *RouteRule {
	return router.addWebHandler("OPTIONS", path, handler, middlewares...)
}
