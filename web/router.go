package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/mylxsw/container"
)

// routerImpl 定制的路由
type routerImpl struct {
	router          *mux.Router
	container       container.Container
	routes          []RouteRule
	decorators      []HandlerDecorator
	prefix          string
	ignoreLastSlash bool
}

// routeRuleImpl 路由规则
type routeRuleImpl struct {
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

func (rr *routeRuleImpl) GetName() string {
	return rr.name
}

func (rr *routeRuleImpl) GetHost() string {
	return rr.host
}

func (rr *routeRuleImpl) GetQueries() []string {
	return rr.queries
}

func (rr *routeRuleImpl) GetSchemas() []string {
	return rr.schemes
}

func (rr *routeRuleImpl) GetHeaders() []string {
	return rr.headers
}

func (rr *routeRuleImpl) GetCustom() func(rou *mux.Route) {
	return rr.custom
}

func (rr *routeRuleImpl) GetDecorators() []HandlerDecorator {
	return rr.decorators
}

func (rr *routeRuleImpl) GetWebHandler() WebHandler {
	return rr.webHandler
}

func (rr *routeRuleImpl) GetPath() string {
	return rr.path
}

func (rr *routeRuleImpl) GetMethod() string {
	return rr.method
}

func (rr *routeRuleImpl) Decorators(dec ...HandlerDecorator) RouteRule {
	rr.decorators = dec
	return rr
}

func (rr *routeRuleImpl) Path(path string) RouteRule {
	rr.path = path
	return rr
}

// Custom add more control to underlying mux.Route
func (rr *routeRuleImpl) Custom(custom func(rou *mux.Route)) RouteRule {
	rr.custom = custom
	return rr
}

// Name sets the name for the route, used to build URLs.
// It is an error to call Name more than once on a route.
func (rr *routeRuleImpl) Name(name string) RouteRule {
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
func (rr *routeRuleImpl) Headers(pairs ...string) RouteRule {
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
func (rr *routeRuleImpl) Queries(pairs ...string) RouteRule {
	rr.queries = pairs
	return rr
}

// Schemes adds a matcher for URL schemes.
// It accepts a sequence of schemes to be matched, e.g.: "http", "https".
func (rr *routeRuleImpl) Schemes(schemes ...string) RouteRule {
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
func (rr *routeRuleImpl) Host(tpl string) RouteRule {
	rr.host = tpl
	return rr
}

// NewRouter 创建一个路由器
func NewRouter(conf *Config, decorators ...HandlerDecorator) Router {
	return NewRouterWithContainer(container.New(), conf, decorators...)
}

// NewRouterWithContainer 创建一个路由器，带有依赖注入容器支持
func NewRouterWithContainer(c container.Container, conf *Config, decorators ...HandlerDecorator) Router {
	cc := container.Extend(c)
	cc.MustSingleton(func() *schema.Decoder {
		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)
		return decoder
	})

	cc.MustBindValue("config", conf)
	cc.MustSingleton(func() *Config { return conf })

	return create(cc, conf.IgnoreLastSlash, mux.NewRouter(), decorators...)
}

// create 创建定制路由器
func create(c container.Container, ignoreLastSlash bool, router *mux.Router, decorators ...HandlerDecorator) *routerImpl {
	return &routerImpl{
		router:          router,
		routes:          make([]RouteRule, 0),
		decorators:      decorators,
		prefix:          "",
		container:       c,
		ignoreLastSlash: ignoreLastSlash,
	}
}

// Group 创建路由组
func (router *routerImpl) Group(prefix string, f func(rou Router), decors ...HandlerDecorator) {
	r := create(router.container, router.ignoreLastSlash, router.router, decors...)
	r.prefix = prefix

	f(r)
	r.parse()

	for _, route := range r.GetRoutes() {
		rr := router.addWebHandler(route.GetMethod(), route.GetPath(), route.GetWebHandler(), route.GetDecorators()...)

		rr.Name(route.GetName())
		rr.Headers(route.GetHeaders()...)
		rr.Schemes(route.GetSchemas()...)
		rr.Queries(route.GetQueries()...)
		rr.Host(route.GetHost())
		rr.Custom(route.GetCustom())
	}
}

type requestModifyMiddleware struct {
	handler http.Handler
	router  *routerImpl
}

func (m requestModifyMiddleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if m.router.ignoreLastSlash {
		request.URL.Path = strings.TrimRight(request.URL.Path, "/")
	}

	m.handler.ServeHTTP(writer, request)
}

// Perform 将路由规则添加到路由器
func (router *routerImpl) Perform(exceptionHandler ExceptionHandler, cb func(*mux.Router)) http.Handler {
	// cors support and exception handler
	corsHandler := func(rt RouteRule) WebHandler {
		return func(ctx Context) (resp Response) {
			defer func() {
				if err := recover(); err != nil {
					if exceptionHandler != nil {
						resp = exceptionHandler(ctx, err)
					}

					if resp == nil {
						_resp, err := ErrorToResponse(ctx, err)
						if err != nil {
							resp = ctx.Error(fmt.Sprintf("Internal serverImpl Error: %v", err), http.StatusInternalServerError)
						} else {
							resp = _resp
						}
					}
				}
			}()

			if ctx.Method() == http.MethodOptions {
				return ctx.NewHTMLResponse("")
			}

			return rt.GetWebHandler()(ctx)
		}
	}

	for _, r := range router.routes {
		var handler http.Handler = newWebHandler(router.container, corsHandler(r), r.GetDecorators()...)
		route := router.router.Handle(r.GetPath(), handler)
		if r.GetMethod() != "" {
			route.Methods(r.GetMethod(), http.MethodOptions)
		}

		if r.GetHost() != "" {
			route.Host(r.GetHost())
		}

		if r.GetQueries() != nil {
			route.Queries(r.GetQueries()...)
		}

		if r.GetSchemas() != nil {
			route.Schemes(r.GetSchemas()...)
		}

		if r.GetName() != "" {
			route.Name(r.GetName())
		}

		if r.GetHeaders() != nil {
			route.Headers(r.GetHeaders()...)
		}

		if r.GetCustom() != nil {
			r.GetCustom()(route)
		}
	}

	cb(router.router)
	return requestModifyMiddleware{
		handler: router.router,
		router:  router,
	}
}

// GetRoutes 获取所有路由规则
func (router *routerImpl) GetRoutes() []RouteRule {
	return router.routes
}

func (router *routerImpl) addWebHandler(method string, path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	if router.ignoreLastSlash {
		path = strings.TrimRight(path, "/")
	}

	rou := &routeRuleImpl{
		method:     method,
		path:       path,
		webHandler: handler,
		decorators: middlewares,
	}
	router.routes = append(router.routes, rou)

	return rou
}

// Parse 解析路由规则，将中间件信息同步到路由规则
func (router *routerImpl) parse() {
	for i := range router.routes {
		router.routes[i].Path(fmt.Sprintf("%s/%s", strings.TrimRight(router.prefix, "/"), strings.TrimLeft(router.routes[i].GetPath(), "/")))
		router.routes[i].Decorators(append(router.routes[i].GetDecorators(), router.decorators...)...)
	}
}

func (router *routerImpl) addHandler(method string, path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler(method, path, func(ctx Context) Response {
		return ctx.Resolve(handler)
	}, middlewares...)
}

// Any 指定所有请求方式的路由规则
func (router *routerImpl) Any(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("", path, handler, middlewares...)
}

// Get 指定所有GET方式的路由规则
func (router *routerImpl) Get(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("GET", path, handler, middlewares...)
}

// Post 指定所有Post方式的路由规则
func (router *routerImpl) Post(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("POST", path, handler, middlewares...)
}

// Delete 指定所有DELETE方式的路由规则
func (router *routerImpl) Delete(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("DELETE", path, handler, middlewares...)
}

// Put 指定所有Put方式的路由规则
func (router *routerImpl) Put(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("PUT", path, handler, middlewares...)
}

// Patch 指定所有Patch方式的路由规则
func (router *routerImpl) Patch(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("PATCH", path, handler, middlewares...)
}

// Head 指定所有Head方式的路由规则
func (router *routerImpl) Head(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("HEAD", path, handler, middlewares...)
}

// Options 指定所有OPTIONS方式的路由规则
func (router *routerImpl) Options(path string, handler interface{}, middlewares ...HandlerDecorator) RouteRule {
	return router.addHandler("OPTIONS", path, handler, middlewares...)
}

// WebAny 指定所有请求方式的路由规则，WebHandler方式
func (router *routerImpl) WebAny(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("", path, handler, middlewares...)
}

// WebGet 指定GET请求方式的路由规则，WebHandler方式
func (router *routerImpl) WebGet(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("GET", path, handler, middlewares...)
}

// WebPost 指定POST请求方式的路由规则，WebHandler方式
func (router *routerImpl) WebPost(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("POST", path, handler, middlewares...)
}

// WebPut 指定所有Put方式的路由规则，WebHandler方式
func (router *routerImpl) WebPut(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("PUT", path, handler, middlewares...)
}

// WebDelete 指定所有DELETE方式的路由规则，WebHandler方式
func (router *routerImpl) WebDelete(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("DELETE", path, handler, middlewares...)
}

// WebPatch 指定所有PATCH方式的路由规则，WebHandler方式
func (router *routerImpl) WebPatch(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("PATCH", path, handler, middlewares...)
}

// WebHead 指定所有HEAD方式的路由规则，WebHandler方式
func (router *routerImpl) WebHead(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("HEAD", path, handler, middlewares...)
}

// WebOptions 指定所有OPTIONS方式的路由规则，WebHandler方式
func (router *routerImpl) WebOptions(path string, handler WebHandler, middlewares ...HandlerDecorator) RouteRule {
	return router.addWebHandler("OPTIONS", path, handler, middlewares...)
}
