package glacier

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/ternary"

	"github.com/mylxsw/glacier/infra"
)

var errorKind = reflect.TypeOf((*error)(nil)).Elem()

func resolveNameable(item interface{}) string {
	var name string
	if p, ok := item.(infra.Nameable); ok {
		name = p.Name()
	}

	if name == "" {
		pf := reflect.TypeOf(item)
		name = fmt.Sprintf("%s:%s", ternary.If(pf.PkgPath() == "", ".", pf.PkgPath()), pf.String())
	}
	return name
}

type providerEntry struct {
	provider infra.Provider
	name     string
}

func newProviderEntry(provider infra.Provider) *providerEntry {
	return &providerEntry{provider: provider, name: resolveNameable(provider)}
}

func (p providerEntry) Name() string {
	return p.name
}

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

// Provider add a service provider
func (impl *framework) Provider(providers ...infra.Provider) {
	for _, p := range providers {
		validateShouldLoadMethod(reflect.TypeOf(p))
	}

	impl.providers = append(impl.providers, array.Map(providers, func(p infra.Provider) *providerEntry {
		return newProviderEntry(p)
	})...)
}

func validateShouldLoadMethod(pType reflect.Type) {
	if method, ok := pType.MethodByName("ShouldLoad"); ok {
		returnValueCount := method.Type.NumOut()
		if method.Type.Out(0).Kind() != reflect.Bool {
			panic(fmt.Errorf("invalid provider %s: the first return value for ShouldLoad method must a bool", pType.String()))
		}
		if returnValueCount == 0 || returnValueCount > 2 {
			panic(fmt.Errorf("invalid provider %s: ShouldLoad method must be func(...) bool or func(...) (bool, error)", pType.String()))
		} else if returnValueCount == 2 {
			if !method.Type.Out(1).Implements(errorKind) {
				panic(fmt.Errorf("invalid provider %s: the second return value for ShouldLoad method must be an error", pType.String()))
			}
		}
	}
}

// Service add a service
func (impl *framework) Service(services ...infra.Service) {
	for _, p := range services {
		validateShouldLoadMethod(reflect.TypeOf(p))
	}

	impl.services = append(impl.services, array.Map(services, func(srv infra.Service) *serviceEntry {
		return newServiceEntry(srv)
	})...)
}

type asyncJob struct {
	fn interface{}
}

func (aj asyncJob) Call(resolver infra.Resolver) error {
	return resolver.ResolveWithError(aj.fn)
}

// Async 添加一个异步执行函数
func (impl *framework) Async(fns ...interface{}) {
	for i, fn := range fns {
		if reflect.TypeOf(fn).Kind() != reflect.Func {
			panic(fmt.Errorf("invalid argument: fn at %d must be a func", i))
		}

		impl.lock.Lock()
		if impl.status == Started {
			impl.asyncJobChannel <- asyncJob{fn: fn}
		} else {
			impl.asyncJobs = append(impl.asyncJobs, asyncJob{fn: fn})
		}
		impl.lock.Unlock()
	}
}

type Providers []*providerEntry

func (p Providers) Len() int {
	return len(p)
}

func (p Providers) Less(i, j int) bool {
	vi, vj := 1000, 1000

	if pi, ok := p[i].provider.(infra.Priority); ok {
		vi = pi.Priority()
	}
	if pj, ok := p[j].provider.(infra.Priority); ok {
		vj = pj.Priority()
	}

	if vi == vj {
		return i < j
	}

	return vi < vj
}

func (p Providers) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
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

// providersFilter 预处理 providers，排除掉不需要加载的 providers
func (impl *framework) providersFilter() []*providerEntry {
	aggregates := make([]*providerEntry, 0)
	for _, p := range impl.providers {
		if !impl.shouldLoadModule(reflect.ValueOf(p.provider)) {
			if infra.DEBUG {
				log.Debugf("[glacier] provider %s is ignored because ShouldLoad()=false", p.Name())
			}
			continue
		}

		aggregates = append(append(aggregates, resolveProviderAggregate(p)...), p)
	}

	uniqAggregates := make(map[reflect.Type]int)
	for _, p := range aggregates {
		pt := reflect.TypeOf(p.provider)
		v, ok := uniqAggregates[pt]
		if ok && infra.WARN {
			log.Warningf("[glacier] provider %s %s are loaded more than once: %d", pt.PkgPath(), pt.String(), v+1)
		}

		uniqAggregates[pt] = v + 1
	}

	sort.Sort(Providers(aggregates))
	return aggregates
}

func resolveProviderAggregate(provider *providerEntry) []*providerEntry {
	providers := make([]*providerEntry, 0)
	if ex, ok := provider.provider.(infra.ProviderAggregate); ok {
		for _, exp := range ex.Aggregates() {
			pr := newProviderEntry(exp)
			providers = append(append(providers, resolveProviderAggregate(pr)...), pr)
		}
	}

	return providers
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

func (impl *framework) shouldLoadModule(pValue reflect.Value) bool {
	shouldLoadMethod := pValue.MethodByName("ShouldLoad")
	if shouldLoadMethod.IsValid() && !shouldLoadMethod.IsZero() {
		res, err := impl.cc.Call(shouldLoadMethod)
		if err != nil {
			panic(fmt.Errorf("[glacier] call %s.ShouldLoad method failed: %v", pValue.Kind().String(), err))
		}

		if len(res) > 1 {
			if err, ok := res[1].(error); ok && err != nil {
				panic(fmt.Errorf("[glacier] call %s.Should method return an error value: %v", pValue.Kind().String(), err))
			}
		}

		return res[0].(bool)
	}

	return true
}
