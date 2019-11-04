package web

// Controllers add controllers to router
func (router *Router) Controllers(prefix string, controllers ...Controller) {
	router.Group(prefix, func(router *Router) {
		for _, controller := range controllers {
			controller.Register(router)
		}
	})
}

// WithMiddleware create a MiddlewareRouter
func (router *Router) WithMiddleware(decors ...HandlerDecorator) *MiddlewareRouter {
	return &MiddlewareRouter{
		router: router,
		decors: decors,
	}
}

type MiddlewareRouter struct {
	router *Router
	decors []HandlerDecorator
}

func (mr *MiddlewareRouter) Controllers(prefix string, controllers ...Controller) {
	mr.router.Group(prefix, func(router *Router) {
		for _, controller := range controllers {
			controller.Register(router)
		}
	}, mr.decors...)
}

func (mr *MiddlewareRouter) Group(prefix string, f func(rou *Router)) {
	mr.router.Group(prefix, f, mr.decors...)
}
