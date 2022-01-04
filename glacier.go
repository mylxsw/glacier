package glacier

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/graceful"
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

// glacierImpl is the server
type glacierImpl struct {
	version   string
	container container.Container
	logger    log.Logger

	delayTasks      []DelayTask
	delayTaskClosed bool
	lock            sync.RWMutex

	handler func(cliCtx infra.FlagContext) error

	providers []infra.Provider
	services  []infra.Service

	beforeInitialize    func(c infra.FlagContext) error
	beforeServerStart   func(cc container.Container) error
	afterServerStart    func(cc infra.Resolver) error
	beforeServerStop    func(cc infra.Resolver) error
	afterProviderBooted interface{}
	mainFunc            interface{}

	gracefulBuilder func() graceful.Graceful

	singletons []interface{}
	prototypes []interface{}

	status Status
}

// CreateGlacier a new glacierImpl server
func CreateGlacier(version string) infra.Glacier {
	glacier := &glacierImpl{}
	glacier.version = version
	glacier.singletons = make([]interface{}, 0)
	glacier.prototypes = make([]interface{}, 0)
	glacier.providers = make([]infra.Provider, 0)
	glacier.services = make([]infra.Service, 0)
	glacier.delayTasks = make([]DelayTask, 0)
	glacier.handler = glacier.createServer()
	glacier.status = Unknown

	return glacier
}

// Graceful 设置优雅停机实现
func (glacier *glacierImpl) Graceful(builder func() graceful.Graceful) infra.Glacier {
	glacier.gracefulBuilder = builder
	return glacier
}

func (glacier *glacierImpl) Handler() func(cliContext infra.FlagContext) error {
	return glacier.handler
}

// BeforeInitialize set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (glacier *glacierImpl) BeforeInitialize(f func(c infra.FlagContext) error) infra.Glacier {
	glacier.beforeInitialize = f
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

// Logger set a log implements
func (glacier *glacierImpl) Logger(logger log.Logger) infra.Glacier {
	glacier.logger = logger
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

// Main execute main business logic
func (glacier *glacierImpl) Main(f interface{}) infra.Glacier {
	glacier.mainFunc = f
	return glacier
}

func (glacier *glacierImpl) createServer() func(c infra.FlagContext) error {
	startupTs := time.Now()
	return func(cliCtx infra.FlagContext) error {
		if glacier.beforeInitialize != nil {
			if err := glacier.beforeInitialize(cliCtx); err != nil {
				return err
			}
		}

		if glacier.logger == nil {
			glacier.logger = log.Module("glacier")
		}

		defer func() {
			if err := recover(); err != nil {
				glacier.logger.Criticalf("application initialize failed with a panic, Err: %s, Stack: \n%s", err, debug.Stack())
			}
		}()

		// 创建容器
		ctx, cancel := context.WithCancel(context.Background())
		cc := container.NewWithContext(ctx)
		glacier.container = cc

		// 运行信息
		cc.MustBindValue(infra.VersionKey, glacier.version)
		cc.MustBindValue(infra.StartupTimeKey, startupTs)
		cc.MustSingleton(func() infra.FlagContext { return cliCtx })
		cc.MustSingletonOverride(func() infra.Resolver { return cc })
		cc.MustSingletonOverride(func() infra.Binder { return cc })
		cc.MustSingletonOverride(func() infra.Hook { return glacier })

		err := glacier.initialize(cc, cliCtx)
		cc.MustResolve(func(gf graceful.Graceful) {
			gf.AddShutdownHandler(cancel)
		})
		if err != nil {
			return err
		}

		// 服务启动前回调
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
			go func(s infra.Service) {
				defer wg.Done()

				cc.MustResolve(func(gf graceful.Graceful) {
					gf.AddShutdownHandler(s.Stop)
					gf.AddReloadHandler(s.Reload)
					if err := s.Start(); err != nil {
						glacier.logger.Errorf("service %s has stopped: %v", s.Name(), err)
					}
				})
			}(s)
		}

		defer cc.MustResolve(func(conf *Config) {
			if err := recover(); err != nil {
				glacier.logger.Criticalf("application startup failed, Err: %v, Stack: %s", err, debug.Stack())
			}

			if conf.ShutdownTimeout > 0 {
				ok := make(chan interface{}, 0)
				go func() {
					wg.Wait()
					ok <- struct{}{}
				}()
				select {
				case <-ok:
					if glacier.logger.DebugEnabled() {
						glacier.logger.Debugf("all services has been stopped")
					}
				case <-time.After(conf.ShutdownTimeout):
					glacier.logger.Errorf("shutdown timeout, exit directly")
				}
			} else {
				wg.Wait()
				if glacier.logger.DebugEnabled() {
					glacier.logger.Debugf("all services has been stopped")
				}
			}
		})

		return cc.ResolveWithError(glacier.startServer(cc, startupTs))
	}
}

// initialize 初始化 Glacier
func (glacier *glacierImpl) initialize(cc container.Container, cliCtx infra.FlagContext) error {
	// 基本配置加载
	cc.MustSingletonOverride(ConfigLoader)
	cc.MustSingletonOverride(func() log.Logger { return glacier.logger })

	// 优雅停机
	cc.MustSingletonOverride(func(conf *Config) graceful.Graceful {
		if glacier.gracefulBuilder != nil {
			return glacier.gracefulBuilder()
		}
		return graceful.NewWithDefault(conf.ShutdownTimeout)
	})

	// 注册其它对象
	for _, i := range glacier.singletons {
		cc.MustSingletonOverride(i)
	}

	for _, i := range glacier.prototypes {
		cc.MustPrototypeOverride(i)
	}

	glacier.providers = glacier.providersFilter(cliCtx)
	glacier.services = glacier.servicesFilter(cliCtx)

	for _, p := range glacier.providers {
		p.Register(cc)
	}

	for _, s := range glacier.services {
		if reflect.ValueOf(s).Kind() == reflect.Ptr {
			if err := cc.AutoWire(s); err != nil {
				return fmt.Errorf("service %s autowired failed: %v", reflect.TypeOf(s).String(), err)
			}
		}

		if err := s.Init(cc); err != nil {
			return fmt.Errorf("service %s initialize failed: %v", reflect.TypeOf(s).String(), err)
		}
	}

	glacier.status = Initialized
	return nil
}

// servicesFilter 预处理 services，排除不需要加载的 services
func (glacier *glacierImpl) servicesFilter(cliCtx infra.FlagContext) []infra.Service {
	services := make([]infra.Service, 0)
	for _, s := range glacier.services {
		if po, ok := s.(infra.ModuleLoadPolicy); ok && !po.ShouldLoad(cliCtx) {
			continue
		}

		services = append(services, s)
	}

	uniqAggregates := make(map[reflect.Type]int)
	for _, s := range services {
		st := reflect.TypeOf(s)
		v, ok := uniqAggregates[st]
		if ok {
			glacier.logger.WithFields(log.Fields{"count": v + 1}).
				Warningf("service %s are loaded more than once", st.Name())
		}

		uniqAggregates[st] = v + 1
	}

	return services
}

// providersFilter 预处理 providers，排除掉不需要加载的 providers
func (glacier *glacierImpl) providersFilter(cliCtx infra.FlagContext) []infra.Provider {
	aggregates := make([]infra.Provider, 0)
	for _, p := range glacier.providers {
		if po, ok := p.(infra.ModuleLoadPolicy); ok && !po.ShouldLoad(cliCtx) {
			continue
		}

		aggregates = append(append(aggregates, resolveProviderAggregate(p)...), p)
	}

	uniqAggregates := make(map[reflect.Type]int)
	for _, p := range aggregates {
		pt := reflect.TypeOf(p)
		v, ok := uniqAggregates[pt]
		if ok {
			glacier.logger.WithFields(log.Fields{"count": v + 1}).
				Warningf("provider %s %s are loaded more than once", pt.PkgPath(), pt.String())
		}

		uniqAggregates[pt] = v + 1
	}

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
func (glacier *glacierImpl) startServer(cc container.Container, startupTs time.Time) func(gf graceful.Graceful) error {
	return func(gf graceful.Graceful) error {
		// 服务都启动之后的回调
		if glacier.afterServerStart != nil {
			if err := glacier.afterServerStart(cc); err != nil {
				return err
			}
		}

		if glacier.beforeServerStop != nil {
			gf.AddShutdownHandler(func() {
				_ = glacier.beforeServerStop(cc)
			})
		}

		if glacier.logger.DebugEnabled() {
			glacier.logger.Debugf("started glacier application in %v", time.Since(startupTs))
		}

		go func() {
			glacier.lock.RLock()
			defer glacier.lock.RUnlock()

			if glacier.mainFunc != nil {
				cc.MustResolve(glacier.mainFunc)
			}

			for _, t := range glacier.delayTasks {
				cc.MustResolve(t.Func)
			}

			glacier.delayTaskClosed = true
			glacier.delayTasks = nil
		}()

		glacier.status = Started
		return gf.Start()
	}
}
