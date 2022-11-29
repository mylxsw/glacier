package glacier

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-ioc"
)

// framework is the Glacier framework
type framework struct {
	version   string
	startTime time.Time

	cc     ioc.Container
	logger infra.Logger

	lock sync.RWMutex

	providers []*providerEntry
	services  []*serviceEntry

	// asyncRunnerCount 异步任务执行器数量
	asyncRunnerCount int
	asyncJobs        []asyncJob
	asyncJobChannel  chan asyncJob

	init               func(fc infra.FlagContext) error
	preBinder          func(binder infra.Binder)
	beforeServerStop   func(resolver infra.Resolver) error
	onServerReadyHooks []namedFunc

	gracefulBuilder func() infra.Graceful

	flagContextInit interface{}
	singletons      []interface{}
	prototypes      []interface{}

	status   Status
	nodes    infra.GraphvizNodes
	nodeLock sync.Mutex
}

// New a new framework server
func New(version string, asyncJobRunnerCount int) infra.Glacier {
	impl := &framework{startTime: time.Now()}
	impl.version = version
	impl.singletons = make([]interface{}, 0)
	impl.prototypes = make([]interface{}, 0)
	impl.providers = make([]*providerEntry, 0)
	impl.services = make([]*serviceEntry, 0)
	impl.asyncJobs = make([]asyncJob, 0)
	impl.asyncRunnerCount = asyncJobRunnerCount
	impl.status = Unknown
	impl.flagContextInit = func(flagCtx infra.FlagContext) infra.FlagContext { return flagCtx }

	if infra.DEBUG {
		impl.nodes = make(infra.GraphvizNodes, 0)
		impl.nodes = append(impl.nodes, &infra.GraphvizNode{Name: "start"})
	}

	return impl
}

func (impl *framework) pushGraphvizNode(name string, async bool, parent ...*infra.GraphvizNode) *infra.GraphvizNode {
	if infra.DEBUG {
		return nil
	}

	impl.nodeLock.Lock()
	defer impl.nodeLock.Unlock()

	if len(parent) == 0 {
		parentNode := impl.nodes[len(impl.nodes)-1]
		if parentNode.Type != infra.GraphvizNodeTypeNode {
			if len(parentNode.ParentNode) > 0 {
				parentNode = parentNode.ParentNode[0]
			}
		}

		parent = []*infra.GraphvizNode{parentNode}
	}

	node := infra.GraphvizNode{Name: name, ParentNode: parent, Async: async, Type: infra.GraphvizNodeTypeNode}
	impl.nodes = append(impl.nodes, &node)
	return &node
}

func (impl *framework) updateGlacierStatus(status Status) {
	if infra.DEBUG {
		impl.pushGraphvizNode(fmt.Sprintf("update framework status to %s", status.String()), false)
	}

	impl.lock.Lock()
	defer impl.lock.Unlock()

	impl.status = status
}

func (impl *framework) WithFlagContext(fn interface{}) infra.Glacier {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func || fnType.NumOut() != 1 || fnType.Out(0) != reflect.TypeOf(infra.FlagContext(nil)) {
		panic("[glacier] invalid argument for WithFlagContext: must be a function like `func(...) infra.FlagContext`")
	}

	impl.flagContextInit = fn

	return impl
}

// Graceful 设置优雅停机实现
func (impl *framework) Graceful(builder func() infra.Graceful) infra.Glacier {
	impl.gracefulBuilder = builder
	return impl
}

// SetLogger set default logger for glacier
func (impl *framework) SetLogger(logger infra.Logger) infra.Glacier {
	impl.logger = logger
	return impl
}

// Singleton add a singleton instance to container
func (impl *framework) Singleton(ins ...interface{}) infra.Glacier {
	if impl.status >= Initialized {
		panic("[glacier] can not invoke this method after Glacier has been initialize")
	}

	impl.singletons = append(impl.singletons, ins...)
	return impl
}

// Prototype add a prototype to container
func (impl *framework) Prototype(ins ...interface{}) infra.Glacier {
	if impl.status >= Initialized {
		panic("[glacier] can not invoke this method after Glacier has been initialize")
	}

	impl.prototypes = append(impl.prototypes, ins...)
	return impl
}

// Resolve is a proxy to container's Resolve function
func (impl *framework) Resolve(resolver interface{}) error {
	return impl.cc.Resolve(resolver)
}

// MustResolve is a proxy to container's MustResolve function
func (impl *framework) MustResolve(resolver interface{}) {
	impl.cc.MustResolve(resolver)
}

// Container return container instance
func (impl *framework) Container() infra.Container {
	return impl.cc
}

// Resolver return container instance
func (impl *framework) Resolver() infra.Resolver {
	return impl.cc
}

// Binder return container instance
func (impl *framework) Binder() infra.Binder {
	return impl.cc
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
