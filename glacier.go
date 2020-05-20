package glacier

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/cron"
	"github.com/mylxsw/glacier/event"
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

	handler func(cliCtx FlagContext) error

	providers []ServiceProvider
	services  []Service

	beforeInitialize  func(c FlagContext) error
	beforeServerStart func(cc container.Container) error
	afterServerStart  func(cc container.Container) error
	beforeServerStop  func(cc container.Container) error
	mainFunc          interface{}

	webAppInitFunc         interface{}
	webAppRouterFunc       InitRouterHandler
	webAppMuxRouterFunc    InitMuxRouterHandler
	webAppServerFunc       InitServerHandler
	webAppExceptionHandler web.ExceptionHandler

	cronTaskFunc      CronTaskFunc
	eventListenerFunc EventListenerFunc

	httpListenAddr string

	singletons []interface{}
	prototypes []interface{}
}

func (glacier *glacierImpl) HttpListenAddr() string {
	return glacier.httpListenAddr
}

type CronTaskFunc func(cr cron.Manager, cc container.Container) error
type EventListenerFunc func(listener event.Manager, cc container.Container)

// CreateGlacier a new glacierImpl server
func CreateGlacier(version string) Glacier {
	glacier := &glacierImpl{}
	glacier.version = version
	glacier.webAppInitFunc = func() error { return nil }
	glacier.webAppRouterFunc = func(router *web.Router, mw web.RequestMiddleware) {}
	glacier.singletons = make([]interface{}, 0)
	glacier.prototypes = make([]interface{}, 0)
	glacier.providers = make([]ServiceProvider, 0)
	glacier.services = make([]Service, 0)
	glacier.handler = glacier.createServer()

	return glacier
}

func (glacier *glacierImpl) Handler() func(cliContext FlagContext) error {
	return glacier.handler
}

// BeforeInitialize set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (glacier *glacierImpl) BeforeInitialize(f func(c FlagContext) error) Glacier {
	glacier.beforeInitialize = f
	return glacier
}

// BeforeServerStart set a hook func executed before server start
func (glacier *glacierImpl) BeforeServerStart(f func(cc container.Container) error) Glacier {
	glacier.beforeServerStart = f
	return glacier
}

// AfterServerStart set a hook func executed after server started
func (glacier *glacierImpl) AfterServerStart(f func(cc container.Container) error) Glacier {
	glacier.afterServerStart = f
	return glacier
}

// BeforeServerStop set a hook func executed before server stop
func (glacier *glacierImpl) BeforeServerStop(f func(cc container.Container) error) Glacier {
	glacier.beforeServerStop = f
	return glacier
}

// Cron add cron tasks
func (glacier *glacierImpl) Cron(f CronTaskFunc) Glacier {
	glacier.cronTaskFunc = f
	return glacier
}

// Logger set a log implements
func (glacier *glacierImpl) Logger(logger log.Logger) Glacier {
	glacier.logger = logger
	return glacier
}

// EventListener add event listeners
func (glacier *glacierImpl) EventListener(f EventListenerFunc) Glacier {
	glacier.eventListenerFunc = f
	return glacier
}

// Singleton add a singleton instance to container
func (glacier *glacierImpl) Singleton(ins interface{}) Glacier {
	glacier.singletons = append(glacier.singletons, ins)
	return glacier
}

// Prototype add a prototype to container
func (glacier *glacierImpl) Prototype(ins interface{}) Glacier {
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
func (glacier *glacierImpl) Main(f interface{}) Glacier {
	glacier.mainFunc = f
	return glacier
}

func (glacier *glacierImpl) createServer() func(c FlagContext) error {
	startupTs := time.Now()
	return func(cliCtx FlagContext) error {

		if glacier.logger == nil {
			glacier.logger = log.Module("glacier")
		}

		defer func() {
			if err := recover(); err != nil {
				glacier.logger.Criticalf("application initialize failed with a panic, Err: %s, Stack: \n%s", err, debug.Stack())
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cc := container.NewWithContext(ctx)
		glacier.container = cc
		cc.MustSingleton(func() FlagContext { return cliCtx })

		cc.MustBindValue("version", glacier.version)
		cc.MustBindValue("startup_time", startupTs)
		cc.MustSingleton(ConfigLoader)
		cc.MustSingleton(func() log.Logger { return glacier.logger })

		cc.MustSingleton(func(conf *Config) *graceful.Graceful {
			return graceful.NewWithDefault(conf.ShutdownTimeout)
		})
		cc.MustResolve(func(gf *graceful.Graceful) {
			gf.AddShutdownHandler(cancel)
		})

		cc.MustSingleton(func() event.Store { return event.NewMemoryEventStore(false) })
		cc.MustSingleton(event.NewEventManager)

		cc.MustSingleton(func(conf *Config) (net.Listener, error) {
			return net.Listen("tcp", conf.HttpListen)
		})

		cc.MustSingleton(func() *WebApp {
			return NewWebApp(cc, glacier.webAppRouterFunc, glacier.webAppServerFunc)
		})

		cc.MustSingleton(func() *cronV3.Cron {
			return cronV3.New(cronV3.WithSeconds(), cronV3.WithLogger(cronLogger{logger: glacier.logger}))
		})
		cc.MustSingleton(cron.NewManager)

		for _, i := range glacier.singletons {
			cc.MustSingleton(i)
		}

		for _, i := range glacier.prototypes {
			cc.MustPrototype(i)
		}

		for _, p := range glacier.providers {
			p.Register(cc)
		}

		glacier.logger.WithFields(log.Fields{
			"count":   len(glacier.providers),
			"version": glacier.version,
		}).Debugf("service providers has been registered, starting ...")

		if glacier.beforeInitialize != nil {
			if err := glacier.beforeInitialize(cliCtx); err != nil {
				return err
			}
		}

		if glacier.beforeServerStart != nil {
			if err := glacier.beforeServerStart(cc); err != nil {
				return err
			}
		}

		if glacier.eventListenerFunc != nil {
			if err := cc.Resolve(glacier.eventListenerFunc); err != nil {
				return err
			}
		}

		var wg sync.WaitGroup
		var daemonServiceProviderCount int
		for _, p := range glacier.providers {
			p.Boot(glacier)
			if pp, ok := p.(DaemonServiceProvider); ok {
				wg.Add(1)
				daemonServiceProviderCount++
				go func(pp DaemonServiceProvider) {
					defer wg.Done()
					pp.Daemon(ctx, glacier)
				}(pp)
			}
		}

		glacier.logger.WithFields(log.Fields{
			"boot_count":   len(glacier.providers),
			"daemon_count": daemonServiceProviderCount,
		}).Debugf("service providers has been started")

		// start services
		for i, s := range glacier.services {
			if err := s.Init(cc); err != nil {
				return fmt.Errorf("service %d initialize failed: %v", i, err)
			}

			wg.Add(1)
			go func(s Service) {
				defer wg.Done()

				cc.MustResolve(func(gf *graceful.Graceful) {
					gf.AddShutdownHandler(s.Stop)
					gf.AddReloadHandler(s.Reload)
					if err := s.Start(); err != nil {
						glacier.logger.Errorf("service %s has stopped: %v", s.Name(), err)
					}
				})
			}(s)
		}

		glacier.logger.WithFields(log.Fields{
			"count": len(glacier.services),
		}).Debugf("services has been started")

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
					glacier.logger.Debugf("all services has been stopped")
				case <-time.After(conf.ShutdownTimeout):
					glacier.logger.Errorf("shutdown timeout, exit directly")
				}
			} else {
				wg.Wait()
				glacier.logger.Debugf("all services has been stopped")
			}
		})

		if glacier.httpListenAddr != "" {
			if err := cc.ResolveWithError(func(webApp *WebApp) error {
				webApp.UpdateConfig(func(conf *web.Config) {
					conf.ViewTemplatePathPrefix = cliCtx.String("web_template_prefix")
					conf.MultipartFormMaxMemory = cliCtx.Int64("web_multipart_form_max_memory")
				})
				webApp.MuxRouter(glacier.webAppMuxRouterFunc)
				webApp.ExceptionHandler(glacier.webAppExceptionHandler)
				if err := webApp.Init(glacier.webAppInitFunc); err != nil {
					return err
				}

				if err := webApp.Start(); err != nil {
					return err
				}

				return nil
			}); err != nil {
				return err
			}
		}

		err := cc.ResolveWithError(func(cr cron.Manager, gf *graceful.Graceful) error {
			if glacier.cronTaskFunc != nil {
				if err := glacier.cronTaskFunc(cr, cc); err != nil {
					return err
				}
			}

			// start cron task server
			gf.AddShutdownHandler(cr.Stop)
			cr.Start()

			glacier.logger.Debugf("cron task server has been started")

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

			glacier.logger.Debugf("started glacier application in %v", time.Now().Sub(startupTs))

			return gf.Start()
		})

		if err != nil {
			return err
		}

		return nil
	}
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
