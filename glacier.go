package glacier

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/cron"
	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/graceful"
	cronV3 "github.com/robfig/cron/v3"
)

// glacierImpl is the server
type glacierImpl struct {
	appName   string
	version   string
	container container.Container
	logger    log.Logger

	handler func(cliCtx infra.FlagContext) error

	providers []infra.ServiceProvider
	services  []infra.Service

	beforeInitialize  func(c infra.FlagContext) error
	beforeServerStart func(cc container.Container) error
	afterServerStart  func(cc container.Container) error
	beforeServerStop  func(cc container.Container) error
	mainFunc          interface{}

	webAppInitFunc         infra.InitWebAppHandler
	webAppRouterFunc       infra.InitRouterHandler
	webAppMuxRouterFunc    infra.InitMuxRouterHandler
	webAppServerFunc       infra.InitServerHandler
	webAppExceptionHandler web.ExceptionHandler
	webAppOptions          []infra.WebServerOption

	cronTaskFuncs      []infra.CronTaskFunc
	eventListenerFuncs []infra.EventListenerFunc

	httpListenAddr   string
	enableHTTPServer bool

	tcpListenerBuilder infra.ListenerBuilder
	gracefulBuilder    func() graceful.Graceful

	singletons []interface{}
	prototypes []interface{}
}

func (glacier *glacierImpl) HttpListenAddr() string {
	return glacier.httpListenAddr
}

// CreateGlacier a new glacierImpl server
func CreateGlacier(version string) infra.Glacier {
	glacier := &glacierImpl{}
	glacier.version = version
	glacier.enableHTTPServer = false
	glacier.webAppInitFunc = func(cc container.Container, webApp infra.Web, conf *web.Config) error { return nil }
	glacier.webAppRouterFunc = func(router *web.Router, mw web.RequestMiddleware) {}
	glacier.webAppOptions = make([]infra.WebServerOption, 0)
	glacier.singletons = make([]interface{}, 0)
	glacier.prototypes = make([]interface{}, 0)
	glacier.providers = make([]infra.ServiceProvider, 0)
	glacier.services = make([]infra.Service, 0)
	glacier.handler = glacier.createServer()
	glacier.eventListenerFuncs = make([]infra.EventListenerFunc, 0)
	glacier.cronTaskFuncs = make([]infra.CronTaskFunc, 0)

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

// BeforeServerStart set a hook func executed before server start
func (glacier *glacierImpl) BeforeServerStart(f func(cc container.Container) error) infra.Glacier {
	glacier.beforeServerStart = f
	return glacier
}

// AfterServerStart set a hook func executed after server started
func (glacier *glacierImpl) AfterServerStart(f func(cc container.Container) error) infra.Glacier {
	glacier.afterServerStart = f
	return glacier
}

// BeforeServerStop set a hook func executed before server stop
func (glacier *glacierImpl) BeforeServerStop(f func(cc container.Container) error) infra.Glacier {
	glacier.beforeServerStop = f
	return glacier
}

// Cron add cron tasks
func (glacier *glacierImpl) Cron(f infra.CronTaskFunc) infra.Glacier {
	glacier.cronTaskFuncs = append(glacier.cronTaskFuncs, f)
	return glacier
}

// Logger set a log implements
func (glacier *glacierImpl) Logger(logger log.Logger) infra.Glacier {
	glacier.logger = logger
	return glacier
}

// EventListener add event listeners
func (glacier *glacierImpl) EventListener(f infra.EventListenerFunc) infra.Glacier {
	glacier.eventListenerFuncs = append(glacier.eventListenerFuncs, f)
	return glacier
}

// Singleton add a singleton instance to container
func (glacier *glacierImpl) Singleton(ins interface{}) infra.Glacier {
	glacier.singletons = append(glacier.singletons, ins)
	return glacier
}

// Prototype add a prototype to container
func (glacier *glacierImpl) Prototype(ins interface{}) infra.Glacier {
	glacier.prototypes = append(glacier.prototypes, ins)
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

		err := glacier.initialize(cc)
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

		// 启动事件监听，注册事件监听函数
		if glacier.eventListenerFuncs != nil {
			for _, el := range glacier.eventListenerFuncs {
				if err := cc.Resolve(el); err != nil {
					return err
				}
			}
		}

		// 初始化 ServiceProvider
		var wg sync.WaitGroup
		var daemonServiceProviderCount int
		for _, p := range glacier.providers {
			if reflect.ValueOf(p).Kind() == reflect.Ptr {
				if err := cc.AutoWire(p); err != nil {
					return fmt.Errorf("can not autowire provider: %v", err)
				}
			}

			p.Boot(glacier)
			// 如果是 DaemonServiceProvider，需要在单独的 Goroutine 执行，一般都是阻塞执行的
			if pp, ok := p.(infra.DaemonServiceProvider); ok {
				wg.Add(1)
				daemonServiceProviderCount++
				go func(pp infra.DaemonServiceProvider) {
					defer wg.Done()
					pp.Daemon(ctx, glacier)
				}(pp)
			}
		}

		if glacier.logger.DebugEnabled() {
			glacier.logger.WithFields(log.Fields{
				"boot_count":   len(glacier.providers),
				"daemon_count": daemonServiceProviderCount,
			}).Debugf("service providers has been started")
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

		if glacier.logger.DebugEnabled() {
			glacier.logger.WithFields(log.Fields{
				"count": len(glacier.services),
			}).Debugf("services has been started")
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
func (glacier *glacierImpl) initialize(cc container.Container) error {
	// 基本配置加载
	cc.MustSingleton(ConfigLoader)
	cc.MustSingletonOverride(func() log.Logger { return glacier.logger })

	// 优雅停机
	cc.MustSingletonOverride(func(conf *Config) graceful.Graceful {
		if glacier.gracefulBuilder != nil {
			return glacier.gracefulBuilder()
		}
		return graceful.NewWithDefault(conf.ShutdownTimeout)
	})

	// 事件管理器
	cc.MustSingletonOverride(func() event.Store { return event.NewMemoryEventStore(false) })
	cc.MustSingletonOverride(event.NewEventManager)

	// Listener 对象
	cc.MustSingletonOverride(func() (net.Listener, error) {
		ln, err := glacier.buildTCPListener()
		if err == nil {
			glacier.httpListenAddr = ln.Addr().String()
		}
		return ln, err
	})

	// WebAPP 对象
	cc.MustSingletonOverride(func(cliCtx infra.FlagContext) (infra.Web, error) {
		webApp := NewWebApp(cc, glacier.webAppRouterFunc, glacier.webAppServerFunc)
		webApp.UpdateConfig(func(conf *web.Config) {
			for _, opt := range glacier.webAppOptions {
				opt(conf)
			}
		})

		webApp.MuxRouter(glacier.webAppMuxRouterFunc)
		webApp.ExceptionHandler(glacier.webAppExceptionHandler)
		if err := webApp.Init(glacier.webAppInitFunc); err != nil {
			return nil, err
		}

		return webApp, nil
	})

	// 定时任务对象
	cc.MustSingletonOverride(func() *cronV3.Cron {
		return cronV3.New(cronV3.WithSeconds(), cronV3.WithLogger(cronLogger{logger: glacier.logger}))
	})
	cc.MustSingletonOverride(cron.NewManager)

	// 注册其它对象
	for _, i := range glacier.singletons {
		cc.MustSingleton(i)
	}

	for _, i := range glacier.prototypes {
		cc.MustPrototype(i)
	}

	// 注册服务提供者对象（模块）
	for _, p := range glacier.providers {
		p.Register(cc)
	}

	if glacier.logger.DebugEnabled() {
		glacier.logger.WithFields(log.Fields{
			"count":   len(glacier.providers),
			"version": glacier.version,
		}).Debugf("service providers has been registered, starting ...")
	}

	// 初始化 Services
	for i, s := range glacier.services {
		if reflect.ValueOf(s).Kind() == reflect.Ptr {
			if err := cc.AutoWire(s); err != nil {
				return fmt.Errorf("can not autowire service: %v", err)
			}
		}

		if err := s.Init(cc); err != nil {
			return fmt.Errorf("service %d initialize failed: %v", i, err)
		}
	}

	return nil
}

// buildTCPListener 创建 tcpListenerBuilder 对象
func (glacier *glacierImpl) buildTCPListener() (net.Listener, error) {
	if glacier.tcpListenerBuilder == nil {
		glacier.tcpListenerBuilder = listener.Default("127.0.0.1:8080")
	}

	listener, err := glacier.tcpListenerBuilder.Build(glacier.container)
	if err != nil {
		return nil, err
	}

	glacier.httpListenAddr = listener.Addr().String()
	return listener, nil
}

// startServer 启动 Glacier
func (glacier *glacierImpl) startServer(cc container.Container, startupTs time.Time) func(cr cron.Manager, gf graceful.Graceful) error {
	return func(cr cron.Manager, gf graceful.Graceful) error {
		if err := glacier.startCronTaskServer(cr, gf, cc); err != nil {
			return err
		}

		if glacier.enableHTTPServer {
			if err := cc.ResolveWithError(func(webApp infra.Web) error {
				return webApp.Start()
			}); err != nil {
				return err
			}
		}

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

		if glacier.mainFunc != nil {
			go cc.MustResolve(glacier.mainFunc)
		}

		if glacier.logger.DebugEnabled() {
			glacier.logger.Debugf("started glacier application in %v", time.Now().Sub(startupTs))
		}

		return gf.Start()
	}
}

// startCronTaskServer 启动定时任务
func (glacier *glacierImpl) startCronTaskServer(cr cron.Manager, gf graceful.Graceful, cc container.Container) error {
	// 设置定时任务
	if glacier.cronTaskFuncs != nil {
		for _, ct := range glacier.cronTaskFuncs {
			if err := ct(cr, cc); err != nil {
				return err
			}
		}
	}

	// 启动定时任务管理器
	gf.AddShutdownHandler(cr.Stop)
	cr.Start()

	if glacier.logger.DebugEnabled() {
		glacier.logger.Debugf("cron task server has been started")
	}

	return nil
}

type cronLogger struct {
	logger log.Logger
}

func (l cronLogger) Info(msg string, keysAndValues ...interface{}) {
	// Just drop it, we don't care
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.logger.WithFields(log.Fields{
		"arguments": keysAndValues,
	}).Errorf("%s: %v", msg, err)
}
