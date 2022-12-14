package glacier

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/go-utils/array"
)

type serviceEntry struct {
	service infra.Service
	name    string
}

func newServiceEntry(srv infra.Service) *serviceEntry {
	return &serviceEntry{service: srv, name: resolveNameable(srv)}
}

func (s serviceEntry) Name() string {
	return s.name
}

// Service add a service
func (impl *framework) Service(services ...infra.Service) {
	for _, p := range services {
		validateShouldLoadMethod(reflect.TypeOf(p))
	}

	impl.services = append(impl.services, array.Map(services, func(srv infra.Service, _ int) *serviceEntry {
		return newServiceEntry(srv)
	})...)
}

// registerServices 注册所有的 Services
func (impl *framework) registerServices() error {
	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode
	if infra.DEBUG && len(impl.services) > 0 {
		parentGraphNode = impl.pushGraphvizNode("register services", false)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}

	impl.services = impl.servicesFilter()
	for _, s := range impl.services {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("register service %s", s.Name()), false, parentGraphNode))
		}
		if reflect.ValueOf(s).Kind() == reflect.Ptr {
			if err := impl.cc.AutoWire(s); err != nil {
				return fmt.Errorf("[glacier] service %s autowired failed: %v", reflect.TypeOf(s).String(), err)
			}
		}
	}

	if infra.DEBUG && len(impl.services) > 0 {
		impl.pushGraphvizNode("register services done", false, childGraphNodes...)
	}

	return nil
}

func (impl *framework) initServices() error {
	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode
	if infra.DEBUG && len(impl.services) > 0 {
		parentGraphNode = impl.pushGraphvizNode("init services", false)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}
	// initialize all services
	var initializedServicesCount int
	for _, s := range impl.services {
		if srv, ok := s.service.(infra.Initializer); ok {
			if infra.DEBUG {
				childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("init service %s", s.Name()), false, parentGraphNode))
				log.Debugf("[glacier] initialize service %s", s.Name())
			}

			initializedServicesCount++
			if err := srv.Init(impl.cc); err != nil {
				return fmt.Errorf("[glacier] service %s initialize failed: %v", s.Name(), err)
			}
		}
	}

	if infra.DEBUG && initializedServicesCount > 0 {
		impl.pushGraphvizNode("all services has been initialized", false, childGraphNodes...)
		log.Debugf("[glacier] all services has been initialized, total %d", initializedServicesCount)
	}

	return nil
}

func (impl *framework) startServices(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(len(impl.services))

	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode
	if infra.DEBUG && len(impl.services) > 0 {
		parentGraphNode = impl.pushGraphvizNode("start services", false)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}

	var startedServicesCount int
	for _, s := range impl.services {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("start service %s", s.Name()), true, parentGraphNode))
			log.Debugf("[glacier] service %s starting ...", s.Name())
		}

		go func(s *serviceEntry) {
			defer wg.Done()

			impl.cc.MustResolve(func(gf infra.Graceful) {
				if srv, ok := s.service.(infra.Stoppable); ok {
					gf.AddShutdownHandler(srv.Stop)
				}

				if srv, ok := s.service.(infra.Reloadable); ok {
					gf.AddReloadHandler(srv.Reload)
				}

				startedServicesCount++
				if err := s.service.Start(); err != nil {
					log.Errorf("[glacier] service %s stopped with error: %v", s.Name(), err)
					return
				}

				if infra.DEBUG {
					log.Debugf("[glacier] service %s stopped", s.Name())
				}
			})
		}(s)
	}

	if infra.DEBUG && startedServicesCount > 0 {
		impl.pushGraphvizNode("all services has been started", false, childGraphNodes...)
		log.Debugf("[glacier] all services has been started, total %d", startedServicesCount)
	}

	return nil
}

// servicesFilter 预处理 services，排除不需要加载的 services
func (impl *framework) servicesFilter() []*serviceEntry {
	services := make([]*serviceEntry, 0)
	for _, s := range impl.services {
		if !impl.shouldLoadModule(reflect.ValueOf(s.service)) {
			continue
		}

		services = append(services, s)
	}

	uniqAggregates := make(map[reflect.Type]int)
	for _, s := range services {
		st := reflect.TypeOf(s.service)
		v, ok := uniqAggregates[st]
		if ok && infra.WARN {
			log.Warningf("[glacier] service %s are loaded more than once: %d", st.Name(), v+1)
		}

		uniqAggregates[st] = v + 1
	}

	sort.Sort(Services(services))
	return services
}

type Services []*serviceEntry

func (p Services) Len() int {
	return len(p)
}

func (p Services) Less(i, j int) bool {
	vi, vj := 1000, 1000

	if pi, ok := p[i].service.(infra.Priority); ok {
		vi = pi.Priority()
	}
	if pj, ok := p[j].service.(infra.Priority); ok {
		vj = pj.Priority()
	}

	if vi == vj {
		return i < j
	}

	return vi < vj
}

func (p Services) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
