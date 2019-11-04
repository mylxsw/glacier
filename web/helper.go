package web

import (
	"github.com/gorilla/mux"
)

// Route is a route represent for query all routes
type Route struct {
	Name             string   `json:"name"`
	Methods          []string `json:"methods"`
	PathTemplate     string   `json:"path_template"`
	PathRegexp       string   `json:"path_regexp"`
	QueriesRegexp    []string `json:"queries_regexp"`
	QueriesTemplates []string `json:"queries_templates"`
	HostTemplate     string   `json:"host_template"`
}

// GetAllRoutes return all routes
func GetAllRoutes(router *mux.Router) []Route {
	routes := make([]Route, 0)

	_ = router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		pathRegexp, _ := route.GetPathRegexp()
		methods, _ := route.GetMethods()
		routeName := route.GetName()
		hostTemplate, _ := route.GetHostTemplate()
		queriesTemplates, _ := route.GetQueriesTemplates()
		queriesRegexp, _ := route.GetQueriesRegexp()

		routes = append(routes, Route{
			Name:             routeName,
			Methods:          methods,
			PathTemplate:     pathTemplate,
			PathRegexp:       pathRegexp,
			QueriesRegexp:    queriesRegexp,
			QueriesTemplates: queriesTemplates,
			HostTemplate:     hostTemplate,
		})

		return nil
	})

	return routes
}
