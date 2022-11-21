package glacier

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/mylxsw/glacier/graceful"
	"github.com/mylxsw/glacier/log"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
)

// Status 当前 Glacier 的状态
type Status int

const (
	Unknown     Status = 0
	Initialized Status = 1
	Started     Status = 2
)

type DelayTask struct {
	Func interface{}
}

type serviceRegister struct {
	service infra.Service
	name    string
}

func (s serviceRegister) Name() string {
	return s.name
}

// glacierImpl is the server
type glacierImpl struct {
	version string
	// asyncJobRunnerCount 异步任务执行器数量
	asyncJobRunnerCount int
	container           container.Container
	logger              infra.Logger

	delayTasks      []DelayTask
	delayTaskClosed bool
	lock            sync.RWMutex

	handler   func(cliCtx infra.FlagContext) error
	preBinder func(binder infra.Binder)

	providers       []infra.Provider
	services        []serviceRegister
	asyncJobs       []asyncJob
	asyncJobChannel chan asyncJob

	beforeInitialize    func(fc infra.FlagContext) error
	afterInitialized    func(resolver infra.Resolver) error
	afterProviderBooted interface{}

	beforeServerStart func(cc container.Container) error
	afterServerStart  func(resolver infra.Resolver) error
	beforeServerStop  func(resolver infra.Resolver) error

	gracefulBuilder func() infra.Graceful

	flagContextInit interface{}
	singletons      []interface{}
	prototypes      []interface{}

	status Status
}

// CreateGlacier a new glacierImpl server
func CreateGlacier(version string, asyncJobRunnerCount int) infra.Glacier {
	glacier := &glacierImpl{}
	glacier.version = version
	glacier.singletons = make([]interface{}, 0)
	glacier.prototypes = make([]interface{}, 0)
	glacier.providers = make([]infra.Provider, 0)
	glacier.services = make([]serviceRegister, 0)
	glacier.asyncJobs = make([]asyncJob, 0)
	glacier.delayTasks = make([]DelayTask, 0)
	glacier.asyncJobChannel = make(chan asyncJob)
	glacier.asyncJobRunnerCount = asyncJobRunnerCount
	glacier.handler = glacier.createServer()
	glacier.status = Unknown
	glacier.flagContextInit = func(flagCtx infra.FlagContext) infra.FlagContext { return flagCtx }

	return glacier
}

func (glacier *glacierImpl) WithFlagContext(fn interface{}) infra.Glacier {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func || fnType.NumOut() != 1 || fnType.Out(0) != reflect.TypeOf(infra.FlagContext(nil)) {
		panic("invalid argument for WithFlagContext: must be a function like `func(...) infra.FlagContext`")
	}

	glacier.flagContextInit = fn

	return glacier
}

// Graceful 设置优雅停机实现
func (glacier *glacierImpl) Graceful(builder func() infra.Graceful) infra.Glacier {
	glacier.gracefulBuilder = builder
	return glacier
}

func (glacier *glacierImpl) Main(cliCtx infra.FlagContext) error {
	return glacier.handler(cliCtx)
}

// SetLogger set default logger for glacier
func (glacier *glacierImpl) SetLogger(logger infra.Logger) infra.Glacier {
	glacier.logger = logger
	return glacier
}

// BeforeInitialize set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (glacier *glacierImpl) BeforeInitialize(f func(c infra.FlagContext) error) infra.Glacier {
	glacier.beforeInitialize = f
	return glacier
}

// AfterInitialized set a hook func executed after server initialized
// Usually, we use this method to initialize the log configuration
func (glacier *glacierImpl) AfterInitialized(f func(resolver infra.Resolver) error) infra.Glacier {
	glacier.afterInitialized = f
	return glacier
}

// OnServerReady call a function on server ready
func (glacier *glacierImpl) OnServerReady(f interface{}) {
	if reflect.TypeOf(f).Kind() != reflect.Func {
		panic(errors.New("argument for OnServerReady must be a callable function"))
	}

	glacier.lock.Lock()
	defer glacier.lock.Unlock()

	if glacier.delayTaskClosed {
		panic(errors.New("can not call this function since server has been started"))
	}

	glacier.delayTasks = append(glacier.delayTasks, DelayTask{Func: f})
}

// BeforeServerStart set a hook func executed before server start
func (glacier *glacierImpl) BeforeServerStart(f func(cc container.Container) error) infra.Glacier {
	glacier.beforeServerStart = f
	return glacier
}

// AfterServerStart set a hook func executed after server started
func (glacier *glacierImpl) AfterServerStart(f func(cc infra.Resolver) error) infra.Glacier {
	glacier.afterServerStart = f
	return glacier
}

// BeforeServerStop set a hook func executed before server stop
func (glacier *glacierImpl) BeforeServerStop(f func(cc infra.Resolver) error) infra.Glacier {
	glacier.beforeServerStop = f
	return glacier
}

// AfterProviderBooted set a hook func executed after all providers has been booted
func (glacier *glacierImpl) AfterProviderBooted(f interface{}) infra.Glacier {
	glacier.afterProviderBooted = f
	return glacier
}

// Singleton add a singleton instance to container
func (glacier *glacierImpl) Singleton(ins ...interface{}) infra.Glacier {
	if glacier.status >= Initialized {
		panic("can not invoke this method after Glacier has been initialize")
	}

	glacier.singletons = append(glacier.singletons, ins...)
	return glacier
}

// Prototype add a prototype to container
func (glacier *glacierImpl) Prototype(ins ...interface{}) infra.Glacier {
	if glacier.status >= Initialized {
		panic("can not invoke this method after Glacier has been initialize")
	}

	glacier.prototypes = append(glacier.prototypes, ins...)
	return glacier
}

// PreBind 设置预绑定实例，这里会确保在容器中第一次进行对象实例化之前完成实例绑定
func (glacier *glacierImpl) PreBind(fn func(binder infra.Binder)) infra.Glacier {
	glacier.preBinder = fn
	return glacier
}

// ResolveWithError is a proxy to container's ResolveWithError function
func (glacier *glacierImpl) ResolveWithError(resolver interface{}) error {
	return glacier.container.ResolveWithError(resolver)
}

// MustResolve is a proxy to container's MustResolve function
func (glacier *glacierImpl) MustResolve(resolver interface{}) {
	glacier.container.MustResolve(resolver)
}

// Container return container instance
func (glacier *glacierImpl) Container() container.Container {
	return glacier.container
}

func (glacier *glacierImpl) createServer() func(fc infra.FlagContext) error {
	startupTs := time.Now()
	return func(cliCtx infra.FlagContext) error {
		if glacier.logger != nil {
			log.SetDefaultLogger(glacier.logger)
		}

		if glacier.beforeInitialize != nil {
			if err := glacier.beforeInitialize(cliCtx); err != nil {
				return err
			}
		}

		defer func() {
			if err := recover(); err != nil {
				log.Criticalf("application initialize failed with a panic, Err: %s, Stack: \n%s", err, debug.Stack())
			}
		}()

		// 创建容器
		ctx, cancel := context.WithCancel(context.Background())
		cc := container.NewWithContext(ctx)
		glacier.container = cc

		// 运行信息
		cc.MustBindValue(infra.VersionKey, glacier.version)
		cc.MustBindValue(infra.StartupTimeKey, startupTs)
		cc.MustSingleton(func() (infra.FlagContext, error) {
			res, err := cc.CallWithProvider(glacier.flagContextInit, cc.Provider(func() infra.FlagContext {
				return cliCtx
			}))
			if err != nil {
				return nil, err
			}

			return res[0].(infra.FlagContext), nil
		})
		cc.MustSingletonOverride(func() infra.Resolver { return cc })
		cc.MustSingletonOverride(func() infra.Binder { return cc })
		cc.MustSingletonOverride(func() infra.Hook { return glacier })

		// 基本配置加载
		cc.MustSingletonOverride(ConfigLoader)
		cc.MustSingletonOverride(log.Default)

		// 优雅停机
		cc.MustSingletonOverride(func(conf *Config) infra.Graceful {
			if glacier.gracefulBuilder != nil {
				return glacier.gracefulBuilder()
			}
			return graceful.NewWithDefault(conf.ShutdownTimeout)
		})

		cc.MustResolve(func(gf infra.Graceful) {
			gf.AddShutdownHandler(cancel)
			gf.AddShutdownHandler(func() {
				close(glacier.asyncJobChannel)
			})
		})

		err := glacier.initialize(cc)
		if err != nil {
			return err
		}

		// 服务启动前回调
		if glacier.afterInitialized != nil {
			if err := glacier.afterInitialized(cc); err != nil {
				return err
			}
		}
		if glacier.beforeServerStart != nil {
			if err := glacier.beforeServerStart(cc); err != nil {
				return err
			}
		}

		// 初始化 Provider
		var wg sync.WaitGroup
		var daemonServiceProviderCount int
		for _, p := range glacier.providers {
			if reflect.ValueOf(p).Kind() == reflect.Ptr {
				if err := cc.AutoWire(p); err != nil {
					return fmt.Errorf("can not autowire provider: %v", err)
				}
			}

			if providerBoot, ok := p.(infra.ProviderBoot); ok {
				providerBoot.Boot(cc)
			}
		}

		if glacier.afterProviderBooted != nil {
			if err := cc.ResolveWithError(glacier.afterProviderBooted); err != nil {
				return err
			}
		}

		// initialize all services
		for _, s := range glacier.services {
			if srv, ok := s.service.(infra.Initializer); ok {
				if err := srv.Init(cc); err != nil {
					return fmt.Errorf("service %s initialize failed: %v", s.Name(), err)
				}
			}
		}

		// 如果是 DaemonProvider，需要在单独的 Goroutine 执行，一般都是阻塞执行的
		for _, p := range glacier.providers {
			if pp, ok := p.(infra.DaemonProvider); ok {
				wg.Add(1)
				daemonServiceProviderCount++
				go func(pp infra.DaemonProvider) {
					defer wg.Done()
					pp.Daemon(ctx, cc)
				}(pp)
			}
		}

		// start services
		for _, s := range glacier.services {
			wg.Add(1)
			go func(s serviceRegister) {
				defer wg.Done()

				cc.MustResolve(func(gf infra.Graceful) {
					if srv, ok := s.service.(infra.Stoppable); ok {
						gf.AddShutdownHandler(srv.Stop)
					}

					if srv, ok := s.service.(infra.Reloadable); ok {
						gf.AddReloadHandler(srv.Reload)
					}

					if err := s.service.Start(); err != nil {
						log.Errorf("service %s has stopped: %v", s.Name(), err)
					}
				})
			}(s)
		}

		// add async job processor
		glacier.delayTasks = append(glacier.delayTasks, DelayTask{Func: glacier.asyncJobRunner})

		defer cc.MustResolve(func(conf *Config) {
			if err := recover(); err != nil {
				log.Criticalf("application startup failed, err: %v, stack: %s", err, debug.Stack())
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
						log.Debugf("all services has been stopped")
					}
				case <-time.After(conf.ShutdownTimeout):
					log.Errorf("shutdown timeout, exit directly")
				}
			} else {
				wg.Wait()
				if infra.DEBUG {
					log.Debugf("all services has been stopped")
				}
			}
		})

		return cc.ResolveWithError(glacier.startServer(cc, startupTs))
	}
}

func (glacier *glacierImpl) asyncJobRunner(resolver infra.Resolver, gf infra.Graceful) {
	var wg sync.WaitGroup
	wg.Add(glacier.asyncJobRunnerCount)

	for i := 0; i < glacier.asyncJobRunnerCount; i++ {
		go func(i int) {
			defer wg.Done()

			for job := range glacier.asyncJobChannel {
				if err := job.Call(resolver); err != nil {
					log.Errorf("[async-runner-%d] async job failed: %v", i, err)
				}
			}

			if infra.DEBUG {
				log.Debugf("[async-runner-%d] async job runner stopping...", i)
			}
		}(i)
	}

	glacier.consumeAsyncJobs()
	wg.Wait()

	if infra.DEBUG {
		log.Debug("all async job runners stopped")
	}
}

func (glacier *glacierImpl) consumeAsyncJobs() {
	glacier.lock.Lock()
	defer glacier.lock.Unlock()

	for _, job := range glacier.asyncJobs {
		glacier.asyncJobChannel <- job
	}
	glacier.asyncJobs = nil
}

// initialize 初始化 Glacier
func (glacier *glacierImpl) initialize(cc container.Container) error {
	// 注册其它对象
	for _, i := range glacier.singletons {
		cc.MustSingletonOverride(i)
	}

	for _, i := range glacier.prototypes {
		cc.MustPrototypeOverride(i)
	}

	// 完成预绑定对象的绑定
	if glacier.preBinder != nil {
		glacier.preBinder(glacier.container)
	}

	glacier.providers = glacier.providersFilter()
	glacier.services = glacier.servicesFilter()

	for _, p := range glacier.providers {
		p.Register(cc)
	}

	for _, s := range glacier.services {
		if reflect.ValueOf(s).Kind() == reflect.Ptr {
			if err := cc.AutoWire(s); err != nil {
				return fmt.Errorf("service %s autowired failed: %v", reflect.TypeOf(s).String(), err)
			}
		}
	}

	glacier.status = Initialized
	return nil
}

// servicesFilter 预处理 services，排除不需要加载的 services
func (glacier *glacierImpl) servicesFilter() []serviceRegister {
	services := make([]serviceRegister, 0)
	for _, s := range glacier.services {
		if !glacier.shouldLoadModule(reflect.ValueOf(s.service)) {
			continue
		}

		services = append(services, s)
	}

	uniqAggregates := make(map[reflect.Type]int)
	for _, s := range services {
		st := reflect.TypeOf(s.service)
		v, ok := uniqAggregates[st]
		if ok {
			log.Warningf("service %s are loaded more than once: %d", st.Name(), v+1)
		}

		uniqAggregates[st] = v + 1
	}

	sort.Sort(Services(services))
	return services
}

func (glacier *glacierImpl) shouldLoadModule(pValue reflect.Value) bool {
	shouldLoadMethod := pValue.MethodByName("ShouldLoad")
	if shouldLoadMethod.IsValid() && !shouldLoadMethod.IsZero() {
		res, err := glacier.container.Call(shouldLoadMethod)
		if err != nil {
			panic(fmt.Errorf("call %s.ShouldLoad method failed: %v", pValue.Kind().String(), err))
		}

		if len(res) > 1 {
			if err, ok := res[1].(error); ok && err != nil {
				panic(fmt.Errorf("call %s.Should method return an error value: %v", pValue.Kind().String(), err))
			}
		}

		return res[0].(bool)
	}

	return true
}

// providersFilter 预处理 providers，排除掉不需要加载的 providers
func (glacier *glacierImpl) providersFilter() []infra.Provider {
	aggregates := make([]infra.Provider, 0)
	for _, p := range glacier.providers {
		if !glacier.shouldLoadModule(reflect.ValueOf(p)) {
			continue
		}

		aggregates = append(append(aggregates, resolveProviderAggregate(p)...), p)
	}

	uniqAggregates := make(map[reflect.Type]int)
	for _, p := range aggregates {
		pt := reflect.TypeOf(p)
		v, ok := uniqAggregates[pt]
		if ok {
			log.Warningf("provider %s %s are loaded more than once: %d", pt.PkgPath(), pt.String(), v+1)
		}

		uniqAggregates[pt] = v + 1
	}

	sort.Sort(Providers(aggregates))
	return aggregates
}

func resolveProviderAggregate(provider infra.Provider) []infra.Provider {
	providers := make([]infra.Provider, 0)
	if ex, ok := provider.(infra.ProviderAggregate); ok {
		for _, exp := range ex.Aggregates() {
			providers = append(append(providers, resolveProviderAggregate(exp)...), exp)
		}
	}

	return providers
}

// startServer 启动 Glacier
func (glacier *glacierImpl) startServer(resolver infra.Resolver, startupTs time.Time) func(gf infra.Graceful) error {
	return func(gf infra.Graceful) error {
		// 服务都启动之后的回调
		if glacier.afterServerStart != nil {
			if err := glacier.afterServerStart(resolver); err != nil {
				return err
			}
		}

		if glacier.beforeServerStop != nil {
			gf.AddShutdownHandler(func() {
				_ = glacier.beforeServerStop(resolver)
			})
		}

		if infra.DEBUG {
			log.Debugf("glacier launched successfully, took %s", time.Since(startupTs))
		}
		glacier.status = Started

		delayTasks := make([]DelayTask, 0)

		glacier.lock.Lock()
		delayTasks = append(delayTasks, glacier.delayTasks...)
		glacier.delayTaskClosed = true
		glacier.delayTasks = nil
		glacier.lock.Unlock()

		var wg sync.WaitGroup
		wg.Add(len(delayTasks))
		if infra.DEBUG {
			log.Debug("add delay tasks, count: ", len(delayTasks))
		}
		for i, t := range delayTasks {
			go func(i int, t DelayTask) {
				defer wg.Done()

				resolver.MustResolve(t.Func)
				if infra.DEBUG {
					log.Debugf("delay task %d stopped", i)
				}
			}(i, t)
		}

		gf.AddShutdownHandler(func() {
			wg.Wait()
			if infra.DEBUG {
				log.Debugf("all delay tasks stopped")
			}
		})

		return gf.Start()
	}
}

type Providers []infra.Provider

func (p Providers) Len() int {
	return len(p)
}

func (p Providers) Less(i, j int) bool {
	vi, vj := 1000, 1000

	if pi, ok := p[i].(infra.Priority); ok {
		vi = pi.Priority()
	}
	if pj, ok := p[j].(infra.Priority); ok {
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

type Services []serviceRegister

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
