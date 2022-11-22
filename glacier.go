package glacier

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/glacier/graceful"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/go-utils/array"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
)

// Status 当前 Glacier 的状态
type Status int

func (s Status) String() string {
	switch s {
	case Initialized:
		return "Initialized"
	case Started:
		return "Started"
	}

	return "Unknown"
}

const (
	Unknown     Status = 0
	Initialized Status = 1
	Started     Status = 2
)

type namedFunc struct {
	name string
	fn   interface{}
}

func newNamedFunc(fn interface{}) namedFunc {
	return namedFunc{fn: fn, name: resolveNameable(fn)}
}

// framework is the Glacier framework
type framework struct {
	version string

	cc     container.Container
	logger infra.Logger

	lock sync.RWMutex

	handler func(cliCtx infra.FlagContext) error

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

	status     Status
	graphNodes infra.GraphNodes
}

// CreateGlacier a new framework server
func CreateGlacier(version string, asyncJobRunnerCount int) infra.Glacier {
	impl := &framework{}
	impl.version = version
	impl.singletons = make([]interface{}, 0)
	impl.prototypes = make([]interface{}, 0)
	impl.providers = make([]*providerEntry, 0)
	impl.services = make([]*serviceEntry, 0)
	impl.asyncJobs = make([]asyncJob, 0)
	impl.asyncRunnerCount = asyncJobRunnerCount
	impl.handler = impl.createServer()
	impl.status = Unknown
	impl.flagContextInit = func(flagCtx infra.FlagContext) infra.FlagContext { return flagCtx }

	if infra.DEBUG {
		impl.graphNodes = make(infra.GraphNodes, 0)
		impl.graphNodes = append(impl.graphNodes, &infra.GraphNode{Name: "start"})
	}

	return impl
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

func (impl *framework) Start(cliCtx infra.FlagContext) error {
	return impl.handler(cliCtx)
}

// SetLogger set default logger for glacier
func (impl *framework) SetLogger(logger infra.Logger) infra.Glacier {
	impl.logger = logger
	return impl
}

// Init set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (impl *framework) Init(f func(c infra.FlagContext) error) infra.Glacier {
	impl.init = f
	return impl
}

// OnServerReady call a function on server ready
func (impl *framework) OnServerReady(ffs ...interface{}) {
	impl.lock.Lock()
	defer impl.lock.Unlock()

	if impl.status == Started {
		panic(fmt.Errorf("[glacier] can not call OnServerReady since server has been started"))
	}

	for _, f := range ffs {
		fn := newNamedFunc(f)
		if reflect.TypeOf(f).Kind() != reflect.Func {
			panic(fmt.Errorf("[glacier] argument for OnServerReady [%s] must be a callable function", fn.name))
		}

		impl.onServerReadyHooks = append(impl.onServerReadyHooks, fn)
	}
}

// BeforeServerStop set a hook func executed before server stop
func (impl *framework) BeforeServerStop(f func(cc infra.Resolver) error) infra.Glacier {
	impl.beforeServerStop = f
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

// PreBind 设置预绑定实例，这里会确保在容器中第一次进行对象实例化之前完成实例绑定
func (impl *framework) PreBind(fn func(binder infra.Binder)) infra.Glacier {
	impl.preBinder = fn
	return impl
}

// ResolveWithError is a proxy to container's ResolveWithError function
func (impl *framework) ResolveWithError(resolver interface{}) error {
	return impl.cc.ResolveWithError(resolver)
}

// MustResolve is a proxy to container's MustResolve function
func (impl *framework) MustResolve(resolver interface{}) {
	impl.cc.MustResolve(resolver)
}

// Container return container instance
func (impl *framework) Container() container.Container {
	return impl.cc
}

func (impl *framework) buildFlagContext(cliCtx infra.FlagContext) func() (infra.FlagContext, error) {
	return func() (infra.FlagContext, error) {
		res, err := impl.cc.CallWithProvider(impl.flagContextInit, impl.cc.Provider(func() infra.FlagContext {
			return cliCtx
		}))

		if err != nil {
			return nil, err
		}

		return res[0].(infra.FlagContext), nil
	}
}

func (impl *framework) createServer() func(fc infra.FlagContext) error {
	startupTs := time.Now()
	return func(cliCtx infra.FlagContext) error {
		// 初始化日志实现
		if impl.logger != nil {
			if infra.DEBUG {
				impl.createGraphNode("init logger", false)
			}
			log.SetDefaultLogger(impl.logger)
		}

		// 执行初始化钩子，用于在框架运行前执行一系列的前置操作
		if impl.init != nil {
			if infra.DEBUG {
				impl.createGraphNode("invoke init hook", false).Color = infra.GraphNodeColorBlue
				log.Debug("[glacier] call beforeInitialize hook")
			}

			if err := impl.init(cliCtx); err != nil {
				return err
			}
		}

		// 全局异常处理
		defer func() {
			if err := recover(); err != nil {
				if infra.DEBUG {
					impl.createGraphNode("global panic recover", false).Color = infra.GraphNodeColorRed
				}
				log.Criticalf("[glacier] application initialize failed with a panic, Err: %s, Stack: \n%s", err, debug.Stack())
			}
		}()

		// 创建容器
		if infra.DEBUG {
			impl.createGraphNode("create container", false)
		}

		ctx, cancel := context.WithCancel(context.Background())
		impl.cc = container.NewWithContext(ctx)

		impl.cc.MustBindValue(infra.VersionKey, impl.version)
		impl.cc.MustBindValue(infra.StartupTimeKey, startupTs)
		impl.cc.MustSingleton(impl.buildFlagContext(cliCtx))
		impl.cc.MustSingletonOverride(func() infra.Resolver { return impl.cc })
		impl.cc.MustSingletonOverride(func() infra.Binder { return impl.cc })
		impl.cc.MustSingletonOverride(func() infra.Hook { return impl })

		// 基本配置加载
		impl.cc.MustSingletonOverride(ConfigLoader)
		impl.cc.MustSingletonOverride(log.Default)

		// 优雅停机
		impl.cc.MustSingletonOverride(func(conf *Config) infra.Graceful {
			if impl.gracefulBuilder != nil {
				return impl.gracefulBuilder()
			}
			return graceful.NewWithDefault(conf.ShutdownTimeout)
		})

		impl.cc.MustResolve(func(gf infra.Graceful) {
			gf.AddShutdownHandler(func() {
				if infra.DEBUG {
					impl.createGraphNode("cancel framework context", false)
				}
				cancel()
			})
		})

		// 注册全局对象
		if infra.DEBUG {
			impl.createGraphNode("add singletons to container", false)
		}
		for _, i := range impl.singletons {
			impl.cc.MustSingletonOverride(i)
		}

		if infra.DEBUG {
			impl.createGraphNode("add prototypes to container", false)
		}
		for _, i := range impl.prototypes {
			impl.cc.MustPrototypeOverride(i)
		}

		impl.updateServerStatus(Initialized)

		// 完成预绑定对象的绑定
		if impl.preBinder != nil {
			if infra.DEBUG {
				impl.createGraphNode("invoke preBind hook", false).Color = infra.GraphNodeColorBlue
				log.Debugf("[glacier] invoke pre-bind hook")
			}
			impl.preBinder(impl.cc)
		}

		// 注册 Providers & Services
		if err := impl.registerProviders(); err != nil {
			return err
		}

		if err := impl.registerServices(); err != nil {
			return err
		}

		var wg sync.WaitGroup

		// 启动 asyncRunners
		stop := impl.startAsyncRunners()
		impl.consumeAsyncJobs()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-stop
		}()

		// 初始化 Services
		if err := impl.initServices(); err != nil {
			return err
		}

		// 启动 Providers
		if err := impl.bootProviders(); err != nil {
			return err
		}

		// 启动 Daemon Providers
		if err := impl.startDaemonProviders(ctx, &wg); err != nil {
			return err
		}

		// 启动 Services
		if err := impl.startServices(ctx, &wg); err != nil {
			return err
		}

		defer impl.cc.MustResolve(func(conf *Config) {
			if err := recover(); err != nil {
				log.Criticalf("[glacier] application startup failed, err: %v, stack: %s", err, debug.Stack())
			}

			if infra.DEBUG {
				impl.createGraphNode("shutdown", false)
			}

			if conf.ShutdownTimeout > 0 {
				ok := make(chan interface{})
				go func() {
					wg.Wait()
					ok <- struct{}{}
				}()
				select {
				case <-ok:
					if infra.DEBUG {
						log.Debugf("[glacier] all modules has been stopped, application will exit safely")
					}
				case <-time.After(conf.ShutdownTimeout):
					log.Errorf("[glacier] shutdown timeout, exit directly")
				}
			} else {
				wg.Wait()
				if infra.DEBUG {
					log.Debugf("[glacier] all modules has been stopped")
				}
			}

			if infra.DEBUG && infra.PrintGraph {
				fmt.Println(impl.graphNodes.Draw())
			}
		})

		return impl.cc.ResolveWithError(impl.startServer(impl.cc, startupTs))
	}
}

func (impl *framework) startDaemonProviders(ctx context.Context, wg *sync.WaitGroup) error {
	daemonServiceProviderCount := len(array.Filter(impl.providers, func(p *providerEntry) bool {
		_, ok := p.provider.(infra.DaemonProvider)
		return ok
	}))

	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode
	if infra.DEBUG && daemonServiceProviderCount > 0 {
		parentGraphNode = impl.createGraphNode("start daemon providers", false)
	}

	// 如果是 DaemonProvider，需要在单独的 Goroutine 执行，一般都是阻塞执行的
	for _, p := range impl.providers {
		if pp, ok := p.provider.(infra.DaemonProvider); ok {
			wg.Add(1)

			if infra.DEBUG {
				childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("start daemon provider: %s", p.name), true, parentGraphNode))
				log.Debugf("[glacier] daemon provider %s starting ...", p.Name())
			}

			go func(pp infra.DaemonProvider) {
				defer wg.Done()
				pp.Daemon(ctx, impl.cc)

				if infra.DEBUG {
					log.Debugf("[glacier] daemon provider %s has been stopped", p.Name())
				}
			}(pp)
		}
	}

	if infra.DEBUG && daemonServiceProviderCount > 0 {
		impl.createGraphNode("all daemon providers started", false, childGraphNodes...)
		log.Debugf("[glacier] all daemon providers has been started, total %d", daemonServiceProviderCount)
	}

	return nil
}

func (impl *framework) bootProviders() error {
	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode
	if infra.DEBUG {
		parentGraphNode = impl.createGraphNode("booting providers", false)
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
				childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("booting provider: %s", p.name), false, parentGraphNode))
				log.Debugf("[glacier] booting provider %s", p.Name())
			}
			bootedProviderCount++
			providerBoot.Boot(impl.cc)
		}
	}

	if infra.DEBUG && bootedProviderCount > 0 {
		impl.createGraphNode("all providers booted", false, childGraphNodes...)
		log.Debugf("[glacier] all providers has been booted, total %d", bootedProviderCount)
	}

	return nil
}

func (impl *framework) initServices() error {
	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode
	if infra.DEBUG && len(impl.services) > 0 {
		parentGraphNode = impl.createGraphNode("init services", false)
	}
	// initialize all services
	var initializedServicesCount int
	for _, s := range impl.services {
		if srv, ok := s.service.(infra.Initializer); ok {
			if infra.DEBUG {
				childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("init service %s", s.Name()), false, parentGraphNode))
				log.Debugf("[glacier] initialize service %s", s.Name())
			}

			initializedServicesCount++
			if err := srv.Init(impl.cc); err != nil {
				return fmt.Errorf("[glacier] service %s initialize failed: %v", s.Name(), err)
			}
		}
	}

	if infra.DEBUG && initializedServicesCount > 0 {
		impl.createGraphNode("all services has been initialized", false, childGraphNodes...)
		log.Debugf("[glacier] all services has been initialized, total %d", initializedServicesCount)
	}

	return nil
}

func (impl *framework) startServices(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(len(impl.services))

	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode
	if infra.DEBUG && len(impl.services) > 0 {
		parentGraphNode = impl.createGraphNode("start services", false)
	}

	var startedServicesCount int
	for _, s := range impl.services {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("start service %s", s.Name()), true, parentGraphNode))
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
		impl.createGraphNode("all services has been started", false, childGraphNodes...)
		log.Debugf("[glacier] all services has been started, total %d", startedServicesCount)
	}

	return nil
}

func (impl *framework) startAsyncRunners() <-chan interface{} {
	stop := make(chan interface{})

	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode

	if infra.DEBUG {
		parentGraphNode = impl.createGraphNode("start async runners", true)
	}

	impl.asyncJobChannel = make(chan asyncJob)
	impl.cc.MustResolve(func(gf infra.Graceful) {
		gf.AddShutdownHandler(func() {
			close(impl.asyncJobChannel)
		})
	})

	var wg sync.WaitGroup
	wg.Add(impl.asyncRunnerCount)

	for i := 0; i < impl.asyncRunnerCount; i++ {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("start async runner %d", i), false, parentGraphNode))
			log.Debugf("[glacier] async runner %d starting ...", i)
		}

		go func(i int) {
			defer wg.Done()

			for job := range impl.asyncJobChannel {
				if err := job.Call(impl.cc); err != nil {
					log.Errorf("[glacier] async runner [async-runner-%d] failed: %v", i, err)
				}
			}

			if infra.DEBUG {
				log.Debugf("[glacier] async runner [async-runner-%d] stopping...", i)
			}
		}(i)
	}

	if infra.DEBUG {
		impl.createGraphNode("all async runners started", false, childGraphNodes...)
	}

	go func() {
		wg.Wait()

		if infra.DEBUG {
			impl.createGraphNode("all async runners stopped", false)
			log.Debug("[glacier] all async runners stopped")
		}

		close(stop)
	}()

	return stop
}

func (impl *framework) consumeAsyncJobs() {
	impl.lock.Lock()
	defer impl.lock.Unlock()

	for _, job := range impl.asyncJobs {
		impl.asyncJobChannel <- job
	}
	impl.asyncJobs = nil
}

// registerProviders 注册所有的 Providers
func (impl *framework) registerProviders() error {
	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode
	if infra.DEBUG && len(impl.providers) > 0 {
		parentGraphNode = impl.createGraphNode("register providers", false)
	}

	impl.providers = impl.providersFilter()
	for _, p := range impl.providers {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("register provider %s", p.Name()), false, parentGraphNode))
			log.Debugf("[glacier] register provider %s", p.Name())
		}
		p.provider.Register(impl.cc)
	}

	if infra.DEBUG && len(impl.providers) > 0 {
		impl.createGraphNode("register providers done", false, childGraphNodes...)
		log.Debugf("[glacier] all providers registered, total %d", len(impl.providers))
	}

	return nil
}

// registerServices 注册所有的 Services
func (impl *framework) registerServices() error {
	var parentGraphNode *infra.GraphNode
	var childGraphNodes []*infra.GraphNode
	if infra.DEBUG && len(impl.services) > 0 {
		parentGraphNode = impl.createGraphNode("register services", false)
	}

	impl.services = impl.servicesFilter()
	for _, s := range impl.services {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.createGraphNode(fmt.Sprintf("register service %s", s.Name()), false, parentGraphNode))
		}
		if reflect.ValueOf(s).Kind() == reflect.Ptr {
			if err := impl.cc.AutoWire(s); err != nil {
				return fmt.Errorf("[glacier] service %s autowired failed: %v", reflect.TypeOf(s).String(), err)
			}
		}
	}

	if infra.DEBUG && len(impl.services) > 0 {
		impl.createGraphNode("register services done", false, childGraphNodes...)
	}

	return nil
}

// startServer 启动 Glacier
func (impl *framework) startServer(resolver infra.Resolver, startupTs time.Time) func(gf infra.Graceful) error {
	return func(gf infra.Graceful) error {
		// 设置服务关闭钩子
		if impl.beforeServerStop != nil {
			gf.AddShutdownHandler(func() {
				if infra.DEBUG {
					impl.createGraphNode("invoke beforeServerStop hook", false).Color = infra.GraphNodeColorBlue
					log.Debugf("[glacier] invoke beforeServerStop hook")
				}
				_ = impl.beforeServerStop(resolver)
			})
		}

		impl.updateServerStatus(Started)

		// 执行 onServerReady Hooks
		var childGraphNodes []*infra.GraphNode
		if len(impl.onServerReadyHooks) > 0 {
			var wg sync.WaitGroup
			wg.Add(len(impl.onServerReadyHooks))

			var parentGraphNode *infra.GraphNode
			if infra.DEBUG {
				parentGraphNode = impl.createGraphNode("invoke onServerReady hooks", true)
				parentGraphNode.Color = infra.GraphNodeColorBlue
			}

			for _, hook := range impl.onServerReadyHooks {
				if infra.DEBUG {
					childGraphNodes = append(childGraphNodes, impl.createGraphNode("invoke onServerReady hook: "+hook.name, true, parentGraphNode))
					log.Debugf("[glacier] invoke onServerReady hook [%s]", hook.name)
				}

				go func(hook namedFunc) {
					defer wg.Done()
					if err := resolver.ResolveWithError(hook.fn); err != nil {
						log.Errorf("[glacier] onServerReady hook [%s] failed: %v", hook.name, err)
					}
				}(hook)
			}

			gf.AddShutdownHandler(wg.Wait)
		}

		if infra.DEBUG {
			impl.createGraphNode("launched", false, childGraphNodes...)
			log.Debugf("[glacier] application launched successfully, took %s", time.Since(startupTs))
		}

		return gf.Start()
	}
}

func (impl *framework) updateServerStatus(status Status) {
	if infra.DEBUG {
		impl.createGraphNode(fmt.Sprintf("update framework status to %s", status.String()), false)
	}

	impl.lock.Lock()
	defer impl.lock.Unlock()

	impl.status = status
}
func (impl *framework) createGraphNode(name string, async bool, parent ...*infra.GraphNode) *infra.GraphNode {
	if parent == nil {
		parent = []*infra.GraphNode{impl.graphNodes[len(impl.graphNodes)-1]}
	}

	node := infra.GraphNode{Name: name, ParentNode: parent, Async: async}
	impl.graphNodes = append(impl.graphNodes, &node)
	return &node
}
