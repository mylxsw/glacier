package glacier

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
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

// framework is the Glacier framework
type framework struct {
	version string

	container container.Container
	logger    infra.Logger

	delayTasks      []DelayTask
	delayTaskClosed bool
	lock            sync.RWMutex

	handler   func(cliCtx infra.FlagContext) error
	preBinder func(binder infra.Binder)

	providers []*providerEntry
	services  []*serviceEntry

	// asyncRunnerCount 异步任务执行器数量
	asyncRunnerCount int
	asyncJobs        []asyncJob
	asyncJobChannel  chan asyncJob

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

// CreateGlacier a new framework server
func CreateGlacier(version string, asyncJobRunnerCount int) infra.Glacier {
	glacier := &framework{}
	glacier.version = version
	glacier.singletons = make([]interface{}, 0)
	glacier.prototypes = make([]interface{}, 0)
	glacier.providers = make([]*providerEntry, 0)
	glacier.services = make([]*serviceEntry, 0)
	glacier.asyncJobs = make([]asyncJob, 0)
	glacier.delayTasks = make([]DelayTask, 0)
	glacier.asyncJobChannel = make(chan asyncJob)
	glacier.asyncRunnerCount = asyncJobRunnerCount
	glacier.handler = glacier.createServer()
	glacier.status = Unknown
	glacier.flagContextInit = func(flagCtx infra.FlagContext) infra.FlagContext { return flagCtx }

	return glacier
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

func (impl *framework) Main(cliCtx infra.FlagContext) error {
	return impl.handler(cliCtx)
}

// SetLogger set default logger for glacier
func (impl *framework) SetLogger(logger infra.Logger) infra.Glacier {
	impl.logger = logger
	return impl
}

// BeforeInitialize set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (impl *framework) BeforeInitialize(f func(c infra.FlagContext) error) infra.Glacier {
	impl.beforeInitialize = f
	return impl
}

// AfterInitialized set a hook func executed after server initialized
// Usually, we use this method to initialize the log configuration
func (impl *framework) AfterInitialized(f func(resolver infra.Resolver) error) infra.Glacier {
	impl.afterInitialized = f
	return impl
}

// OnServerReady call a function on server ready
func (impl *framework) OnServerReady(f interface{}) {
	if reflect.TypeOf(f).Kind() != reflect.Func {
		panic(errors.New("[glacier] argument for OnServerReady must be a callable function"))
	}

	impl.lock.Lock()
	defer impl.lock.Unlock()

	if impl.delayTaskClosed {
		panic(errors.New("[glacier] can not call this function since server has been started"))
	}

	impl.delayTasks = append(impl.delayTasks, DelayTask{Func: f})
}

// BeforeServerStart set a hook func executed before server start
func (impl *framework) BeforeServerStart(f func(cc container.Container) error) infra.Glacier {
	impl.beforeServerStart = f
	return impl
}

// AfterServerStart set a hook func executed after server started
func (impl *framework) AfterServerStart(f func(cc infra.Resolver) error) infra.Glacier {
	impl.afterServerStart = f
	return impl
}

// BeforeServerStop set a hook func executed before server stop
func (impl *framework) BeforeServerStop(f func(cc infra.Resolver) error) infra.Glacier {
	impl.beforeServerStop = f
	return impl
}

// AfterProviderBooted set a hook func executed after all providers has been booted
func (impl *framework) AfterProviderBooted(f interface{}) infra.Glacier {
	impl.afterProviderBooted = f
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
	return impl.container.ResolveWithError(resolver)
}

// MustResolve is a proxy to container's MustResolve function
func (impl *framework) MustResolve(resolver interface{}) {
	impl.container.MustResolve(resolver)
}

// Container return container instance
func (impl *framework) Container() container.Container {
	return impl.container
}

func (impl *framework) createServer() func(fc infra.FlagContext) error {
	startupTs := time.Now()
	return func(cliCtx infra.FlagContext) error {
		if impl.logger != nil {
			log.SetDefaultLogger(impl.logger)
		}

		if impl.beforeInitialize != nil {
			if infra.DEBUG {
				log.Debug("[glacier] call beforeInitialize hook")
			}

			if err := impl.beforeInitialize(cliCtx); err != nil {
				return err
			}
		}

		defer func() {
			if err := recover(); err != nil {
				log.Criticalf("[glacier] application initialize failed with a panic, Err: %s, Stack: \n%s", err, debug.Stack())
			}
		}()

		// 创建容器
		ctx, cancel := context.WithCancel(context.Background())
		cc := container.NewWithContext(ctx)
		impl.container = cc

		// 运行信息
		cc.MustBindValue(infra.VersionKey, impl.version)
		cc.MustBindValue(infra.StartupTimeKey, startupTs)
		cc.MustSingleton(func() (infra.FlagContext, error) {
			res, err := cc.CallWithProvider(impl.flagContextInit, cc.Provider(func() infra.FlagContext {
				return cliCtx
			}))
			if err != nil {
				return nil, err
			}

			return res[0].(infra.FlagContext), nil
		})
		cc.MustSingletonOverride(func() infra.Resolver { return cc })
		cc.MustSingletonOverride(func() infra.Binder { return cc })
		cc.MustSingletonOverride(func() infra.Hook { return impl })

		// 基本配置加载
		cc.MustSingletonOverride(ConfigLoader)
		cc.MustSingletonOverride(log.Default)

		// 优雅停机
		cc.MustSingletonOverride(func(conf *Config) infra.Graceful {
			if impl.gracefulBuilder != nil {
				return impl.gracefulBuilder()
			}
			return graceful.NewWithDefault(conf.ShutdownTimeout)
		})

		cc.MustResolve(func(gf infra.Graceful) {
			gf.AddShutdownHandler(cancel)
			gf.AddShutdownHandler(func() {
				close(impl.asyncJobChannel)
			})
		})

		err := impl.initialize(cc)
		if err != nil {
			return err
		}

		// 服务启动前回调
		if impl.afterInitialized != nil {
			if infra.DEBUG {
				log.Debugf("[glacier] call afterInitialized hook")
			}
			if err := impl.afterInitialized(cc); err != nil {
				return err
			}
		}
		if impl.beforeServerStart != nil {
			if infra.DEBUG {
				log.Debugf("[glacier] call beforeServerStart hook")
			}
			if err := impl.beforeServerStart(cc); err != nil {
				return err
			}
		}

		// 初始化 Provider
		var wg sync.WaitGroup
		var bootedProviderCount int
		for _, p := range impl.providers {
			if reflect.ValueOf(p.provider).Kind() == reflect.Ptr {
				if err := cc.AutoWire(p); err != nil {
					return fmt.Errorf("[glacier] can not autowire provider: %v", err)
				}
			}

			if providerBoot, ok := p.provider.(infra.ProviderBoot); ok {
				if infra.DEBUG {
					log.Debugf("[glacier] booting provider %s", p.Name())
				}
				bootedProviderCount++
				providerBoot.Boot(cc)
			}
		}

		if infra.DEBUG && bootedProviderCount > 0 {
			log.Debugf("[glacier] all providers has been booted, total %d", bootedProviderCount)
		}

		if impl.afterProviderBooted != nil {
			if infra.DEBUG {
				log.Debugf("[glacier] invoke afterProviderBooted hook")
			}

			if err := cc.ResolveWithError(impl.afterProviderBooted); err != nil {
				return err
			}
		}

		// initialize all services
		var initializedServicesCount int
		for _, s := range impl.services {
			if srv, ok := s.service.(infra.Initializer); ok {
				if infra.DEBUG {
					log.Debugf("[glacier] initialize service %s", s.Name())
				}

				initializedServicesCount++
				if err := srv.Init(cc); err != nil {
					return fmt.Errorf("[glacier] service %s initialize failed: %v", s.Name(), err)
				}
			}
		}

		if infra.DEBUG && initializedServicesCount > 0 {
			log.Debugf("[glacier] all services has been initialized, total %d", initializedServicesCount)
		}

		// 如果是 DaemonProvider，需要在单独的 Goroutine 执行，一般都是阻塞执行的
		var daemonServiceProviderCount int
		for _, p := range impl.providers {
			if pp, ok := p.provider.(infra.DaemonProvider); ok {
				wg.Add(1)
				daemonServiceProviderCount++

				if infra.DEBUG {
					log.Debugf("[glacier] run daemon provider %s", p.Name())
				}

				go func(pp infra.DaemonProvider) {
					defer wg.Done()
					pp.Daemon(ctx, cc)

					if infra.DEBUG {
						log.Debugf("[glacier] daemon provider %s has been stopped", p.Name())
					}
				}(pp)
			}
		}

		if infra.DEBUG && daemonServiceProviderCount > 0 {
			log.Debugf("[glacier] all daemon providers has been started, total %d", daemonServiceProviderCount)
		}

		// start services
		var startedServicesCount int
		for _, s := range impl.services {
			wg.Add(1)
			go func(s *serviceEntry) {
				defer wg.Done()

				cc.MustResolve(func(gf infra.Graceful) {
					if srv, ok := s.service.(infra.Stoppable); ok {
						gf.AddShutdownHandler(srv.Stop)
					}

					if srv, ok := s.service.(infra.Reloadable); ok {
						gf.AddReloadHandler(srv.Reload)
					}

					if infra.DEBUG {
						log.Debugf("[glacier] service %s starting ...", s.Name())
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
			log.Debugf("[glacier] all services has been started, total %d", startedServicesCount)
		}

		// add async job processor
		impl.delayTasks = append(impl.delayTasks, DelayTask{Func: impl.startAsyncRunners})

		defer cc.MustResolve(func(conf *Config) {
			if err := recover(); err != nil {
				log.Criticalf("[glacier] application startup failed, err: %v, stack: %s", err, debug.Stack())
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
		})

		return cc.ResolveWithError(impl.startServer(cc, startupTs))
	}
}

func (impl *framework) startAsyncRunners(resolver infra.Resolver, gf infra.Graceful) {
	var wg sync.WaitGroup
	wg.Add(impl.asyncRunnerCount)

	for i := 0; i < impl.asyncRunnerCount; i++ {
		go func(i int) {
			defer wg.Done()

			for job := range impl.asyncJobChannel {
				if err := job.Call(resolver); err != nil {
					log.Errorf("[glacier] async runner [async-runner-%d] failed: %v", i, err)
				}
			}

			if infra.DEBUG {
				log.Debugf("[glacier] async runner [async-runner-%d] stopping...", i)
			}
		}(i)
	}

	impl.consumeAsyncJobs()
	wg.Wait()

	if infra.DEBUG {
		log.Debug("[glacier] all async runners stopped")
	}
}

func (impl *framework) consumeAsyncJobs() {
	impl.lock.Lock()
	defer impl.lock.Unlock()

	for _, job := range impl.asyncJobs {
		impl.asyncJobChannel <- job
	}
	impl.asyncJobs = nil
}

// initialize 初始化 Glacier
func (impl *framework) initialize(cc container.Container) error {
	// 注册其它对象
	for _, i := range impl.singletons {
		cc.MustSingletonOverride(i)
	}

	for _, i := range impl.prototypes {
		cc.MustPrototypeOverride(i)
	}

	// 完成预绑定对象的绑定
	if impl.preBinder != nil {
		if infra.DEBUG {
			log.Debugf("[glacier] invoke pre-bind hook")
		}

		impl.preBinder(impl.container)
	}

	impl.providers = impl.providersFilter()
	impl.services = impl.servicesFilter()

	for _, p := range impl.providers {
		if infra.DEBUG {
			log.Debugf("[glacier] register provider %s", p.Name())
		}
		p.provider.Register(cc)
	}

	if infra.DEBUG && len(impl.providers) > 0 {
		log.Debugf("[glacier] all providers registered, total %d", len(impl.providers))
	}

	for _, s := range impl.services {
		if reflect.ValueOf(s).Kind() == reflect.Ptr {
			if err := cc.AutoWire(s); err != nil {
				return fmt.Errorf("[glacier] service %s autowired failed: %v", reflect.TypeOf(s).String(), err)
			}
		}
	}

	impl.status = Initialized
	return nil
}

// startServer 启动 Glacier
func (impl *framework) startServer(resolver infra.Resolver, startupTs time.Time) func(gf infra.Graceful) error {
	return func(gf infra.Graceful) error {
		// 服务都启动之后的回调
		if impl.afterServerStart != nil {
			if infra.DEBUG {
				log.Debugf("[glacier] invoke afterServerStart hook")
			}
			if err := impl.afterServerStart(resolver); err != nil {
				return err
			}
		}

		if impl.beforeServerStop != nil {
			gf.AddShutdownHandler(func() {
				if infra.DEBUG {
					log.Debugf("[glacier] invoke beforeServerStop hook")
				}
				_ = impl.beforeServerStop(resolver)
			})
		}

		delayTasks := make([]DelayTask, 0)
		impl.lock.Lock()
		delayTasks = append(delayTasks, impl.delayTasks...)
		impl.delayTaskClosed = true
		impl.delayTasks = nil
		impl.lock.Unlock()

		var wg sync.WaitGroup
		wg.Add(len(delayTasks))
		if infra.DEBUG && len(delayTasks) > 0 {
			log.Debug("[glacier] add delay tasks, total ", len(delayTasks))
		}
		for i, t := range delayTasks {
			go func(i int, t DelayTask) {
				defer wg.Done()

				resolver.MustResolve(t.Func)
				if infra.DEBUG {
					log.Debugf("[glacier] delay task %d stopped", i)
				}
			}(i, t)
		}

		gf.AddShutdownHandler(func() {
			wg.Wait()
			if infra.DEBUG {
				log.Debugf("[glacier] all delay tasks stopped")
			}
		})

		impl.status = Started
		if infra.DEBUG {
			log.Debugf("[glacier] application launched successfully, took %s", time.Since(startupTs))
		}

		return gf.Start()
	}
}
