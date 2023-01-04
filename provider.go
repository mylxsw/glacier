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

// Provider add a service provider
func (impl *framework) Provider(providers ...infra.Provider) {
	for _, p := range providers {
		validateShouldLoadMethod(reflect.TypeOf(p))
	}

	impl.providers = append(impl.providers, array.Map(providers, func(p infra.Provider, _ int) *providerEntry {
		return newProviderEntry(p)
	})...)
}

// registerProviders 注册所有的 Providers
func (impl *framework) registerProviders() error {
	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode
	if infra.DEBUG && len(impl.providers) > 0 {
		parentGraphNode = impl.pushGraphvizNode("register providers", false)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}

	impl.providers = impl.providersFilter()
	for _, p := range impl.providers {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("register provider %s", p.Name()), false, parentGraphNode))
			log.Debugf("[glacier] register provider %s", p.Name())
		}
		p.provider.Register(impl.cc)
	}

	if infra.DEBUG && len(impl.providers) > 0 {
		impl.pushGraphvizNode("register providers done", false, childGraphNodes...)
		log.Debugf("[glacier] all providers registered, total %d", len(impl.providers))
	}

	return nil
}

func (impl *framework) bootProviders() error {
	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode
	if infra.DEBUG {
		parentGraphNode = impl.pushGraphvizNode("booting providers", false)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}

	var bootedProviderCount int
	for _, p := range impl.providers {
		if reflect.ValueOf(p.provider).Kind() == reflect.Ptr {
			if err := impl.cc.AutoWire(p); err != nil {
				return fmt.Errorf("[glacier] can not autowire provider: %v", err)
			}
		}

		if providerBoot, ok := p.provider.(infra.ProviderBoot); ok {
			if infra.DEBUG {
				childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("booting provider: %s", p.name), false, parentGraphNode))
				log.Debugf("[glacier] booting provider %s", p.Name())
			}
			bootedProviderCount++
			providerBoot.Boot(impl.cc)
		}
	}

	if infra.DEBUG && bootedProviderCount > 0 {
		impl.pushGraphvizNode("all providers booted", false, childGraphNodes...)
		log.Debugf("[glacier] all providers has been booted, total %d", bootedProviderCount)
	}

	return nil
}

func (impl *framework) startDaemonProviders(ctx context.Context, wg *sync.WaitGroup) error {
	daemonServiceProviderCount := len(array.Filter(impl.providers, func(p *providerEntry, _ int) bool {
		_, ok := p.provider.(infra.DaemonProvider)
		return ok
	}))

	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode
	if infra.DEBUG && daemonServiceProviderCount > 0 {
		parentGraphNode = impl.pushGraphvizNode("start daemon providers", false)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}

	// 如果是 DaemonProvider，需要在单独的 Goroutine 执行，一般都是阻塞执行的
	for _, p := range impl.providers {
		if pp, ok := p.provider.(infra.DaemonProvider); ok {
			wg.Add(1)

			if infra.DEBUG {
				childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("start daemon provider: %s", p.name), true, parentGraphNode))
				log.Debugf("[glacier] daemon provider %s starting ...", p.Name())
			}

			go func(pp infra.DaemonProvider, p *providerEntry) {
				defer wg.Done()
				pp.Daemon(ctx, impl.cc)

				if infra.DEBUG {
					log.Debugf("[glacier] daemon provider %s has been stopped", p.Name())
				}
			}(pp, p)
		}
	}

	if infra.DEBUG && daemonServiceProviderCount > 0 {
		impl.pushGraphvizNode("all daemon providers started", false, childGraphNodes...)
		log.Debugf("[glacier] all daemon providers has been started, total %d", daemonServiceProviderCount)
	}

	return nil
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
