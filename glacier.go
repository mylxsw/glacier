package glacier

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mylxsw/asteria/formatter"
	"github.com/mylxsw/asteria/level"
	logger "github.com/mylxsw/asteria/log"
	"github.com/mylxsw/asteria/writer"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/cron"
	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/go-toolkit/period_job"
	"github.com/mylxsw/graceful"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"

	cronV3 "github.com/robfig/cron/v3"
)

var log = logger.Module("glacier")

type ServiceProvider interface {
	// Register add some dependency for current module
	// this method is called one by one synchronous
	Register(app *container.Container)
	// Boot start the module
	// this method is called one by one synchronous after all register methods called
	Boot(app *Glacier)
}

type DaemonServiceProvider interface {
	ServiceProvider
	// Daemon is a async method called after boot
	// this method is called asynchronous and concurrent
	Daemon(ctx context.Context, app *Glacier)
}

// Glacier is the server
type Glacier struct {
	appName   string
	version   string
	app       *cli.App
	container *container.Container

	providers []ServiceProvider
	services  []Service

	beforeInitialize  func(c *cli.Context) error
	beforeServerStart func(cc *container.Container) error
	afterServerStart  func(cc *container.Container) error
	beforeServerStop  func(cc *container.Container) error
	mainFunc          interface{}

	useStackLogger      func(cc *container.Container, stackWriter *writer.StackWriter)
	defaultLogFormatter formatter.Formatter

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

type CronTaskFunc func(cr cron.Manager, cc *container.Container) error
type EventListenerFunc func(listener event.Manager, cc *container.Container)

var glacierIns *Glacier

// App return Glacier instance you created
func App() *Glacier {
	if glacierIns == nil {
		panic("you should create a Glacier application by call Create function first!")
	}

	return glacierIns
}

// Container return container instance for glacier
func Container() *container.Container {
	return App().Container()
}

// Create a new Glacier server
func Create(version string, flags ...cli.Flag) *Glacier {
	if glacierIns != nil {
		panic("a glacier instance has been created")
	}

	serverFlags := []cli.Flag{
		cli.StringFlag{
			Name:  "conf",
			Value: "",
			Usage: "configuration file path",
		},
		altsrc.NewStringFlag(cli.StringFlag{
			Name:  "log_level",
			Value: "DEBUG",
			Usage: "set default log level",
		}),
		altsrc.NewBoolTFlag(cli.BoolTFlag{
			Name:  "log_color",
			Usage: "log with colorful support",
		}),
		altsrc.NewDurationFlag(cli.DurationFlag{
			Name:   "shutdown_timeout",
			Usage:  "set a shutdown timeout for each service",
			EnvVar: "GLACIER_SHUTDOWN_TIMOUT",
			Value:  5 * time.Second,
		}),
	}

	serverFlags = append(serverFlags, flags...)

	app := cli.NewApp()
	app.Version = version
	app.Before = func(c *cli.Context) error {
		conf := c.String("conf")
		if conf == "" {
			return nil
		}

		inputSource, err := altsrc.NewYamlSourceFromFile(conf)
		if err != nil {
			return err
		}

		return altsrc.ApplyInputSourceValues(c, inputSource, c.App.Flags)
	}
	app.Flags = serverFlags

	glacierIns = &Glacier{}
	glacierIns.app = app
	glacierIns.version = version
	glacierIns.webAppInitFunc = func() error { return nil }
	glacierIns.webAppRouterFunc = func(router *web.Router, mw web.RequestMiddleware) {}
	glacierIns.singletons = make([]interface{}, 0)
	glacierIns.prototypes = make([]interface{}, 0)
	glacierIns.providers = make([]ServiceProvider, 0)
	glacierIns.services = make([]Service, 0)

	app.Action = createServer(glacierIns)

	return glacierIns
}

// Provider add a service provider
func (glacier *Glacier) Provider(provider ServiceProvider) {
	glacier.providers = append(glacier.providers, provider)
}

// Service add a service
func (glacier *Glacier) Service(service Service) {
	glacier.services = append(glacier.services, service)
}

// WithHttpServer with http server support
func (glacier *Glacier) WithHttpServer(listenAddr string) *Glacier {
	if listenAddr == "" {
		listenAddr = ":19950"
	}

	glacier.httpListenAddr = listenAddr

	return glacier
}

// AddFlags add flags to app
func (glacier *Glacier) AddFlags(flags ...cli.Flag) *Glacier {
	glacier.app.Flags = append(glacier.app.Flags, flags...)
	return glacier
}

// DefaultLogFormatter set default log formatter
// if not set, will use default formatter: formatter.DefaultFormatter
func (glacier *Glacier) DefaultLogFormatter(f formatter.Formatter) *Glacier {
	glacier.defaultLogFormatter = f
	return glacier
}

// UseStackLogger set cronLogger to use stack log writer
func (glacier *Glacier) UseStackLogger(f func(cc *container.Container, stackWriter *writer.StackWriter)) *Glacier {
	glacier.useStackLogger = f
	return glacier
}

// UseDefaultStackLogger use default stack cronLogger as cronLogger
// all logs will be sent to stdout
func (glacier *Glacier) UseDefaultStackLogger() *Glacier {
	return glacier.UseStackLogger(func(cc *container.Container, stackWriter *writer.StackWriter) {
		stackWriter.PushWithLevels(writer.NewStdoutWriter())
	})
}

// BeforeInitialize set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (glacier *Glacier) BeforeInitialize(f func(c *cli.Context) error) *Glacier {
	glacier.beforeInitialize = f
	return glacier
}

// BeforeServerStart set a hook func executed before server start
func (glacier *Glacier) BeforeServerStart(f func(cc *container.Container) error) *Glacier {
	glacier.beforeServerStart = f
	return glacier
}

// AfterServerStart set a hook func executed after server started
func (glacier *Glacier) AfterServerStart(f func(cc *container.Container) error) *Glacier {
	glacier.afterServerStart = f
	return glacier
}

// BeforeServerStop set a hook func executed before server stop
func (glacier *Glacier) BeforeServerStop(f func(cc *container.Container) error) *Glacier {
	glacier.beforeServerStop = f
	return glacier
}

// WebAppInit set a hook func for app init
func (glacier *Glacier) WebAppInit(initFunc interface{}) *Glacier {
	glacier.webAppInitFunc = initFunc
	return glacier
}

// WebAppServerInit is a function for initialize http server
func (glacier *Glacier) WebAppServerInit(handler InitServerHandler) *Glacier {
	glacier.webAppServerFunc = handler
	return glacier
}

// WebAppRouter add routes for http server
func (glacier *Glacier) WebAppRouter(handler InitRouterHandler) *Glacier {
	glacier.webAppRouterFunc = handler
	return glacier
}

// WebAppMuxRouter add mux routes for http server
func (glacier *Glacier) WebAppMuxRouter(handler InitMuxRouterHandler) *Glacier {
	glacier.webAppMuxRouterFunc = handler
	return glacier
}

// WebAppExceptionHandler set exception handler for web app
func (glacier *Glacier) WebAppExceptionHandler(handler web.ExceptionHandler) *Glacier {
	glacier.webAppExceptionHandler = handler
	return glacier
}

// Cron add cron tasks
func (glacier *Glacier) Cron(f CronTaskFunc) *Glacier {
	glacier.cronTaskFunc = f
	return glacier
}

// EventListener add event listeners
func (glacier *Glacier) EventListener(f EventListenerFunc) *Glacier {
	glacier.eventListenerFunc = f
	return glacier
}

// Singleton add a singleton instance to container
func (glacier *Glacier) Singleton(ins interface{}) *Glacier {
	glacier.singletons = append(glacier.singletons, ins)
	return glacier
}

// Prototype add a prototype to container
func (glacier *Glacier) Prototype(ins interface{}) *Glacier {
	glacier.prototypes = append(glacier.prototypes, ins)
	return glacier
}

// ResolveWithError is a proxy to container's ResolveWithError function
func (glacier *Glacier) ResolveWithError(resolver interface{}) error {
	return glacier.container.ResolveWithError(resolver)
}

// MustResolve is a proxy to container's MustResolve function
func (glacier *Glacier) MustResolve(resolver interface{}) {
	glacier.container.MustResolve(resolver)
}

// Container return container instance
func (glacier *Glacier) Container() *container.Container {
	return glacier.container
}

// Main execute main business logic
func (glacier *Glacier) Main(f interface{}) *Glacier {
	glacier.mainFunc = f
	return glacier
}

// Run start Glacier server
func (glacier *Glacier) Run(args []string) error {
	if glacier.httpListenAddr != "" {
		glacier.app.Flags = append(
			glacier.app.Flags,
			altsrc.NewStringFlag(cli.StringFlag{
				Name:  "listen",
				Value: glacier.httpListenAddr,
				Usage: "http server listen address",
			}),
			altsrc.NewStringFlag(cli.StringFlag{
				Name:  "web_template_prefix",
				Usage: "web template path prefix",
				Value: "",
			}),
			altsrc.NewInt64Flag(cli.Int64Flag{
				Name:  "web_multipart_form_max_memory",
				Usage: "multipart form max memory size in bytes",
				Value: int64(10 << 20),
			}))
	}

	return glacier.app.Run(args)
}

func createServer(glacier *Glacier) func(c *cli.Context) error {
	startupTs := time.Now()
	return func(cliCtx *cli.Context) error {
		defer func() {
			if err := recover(); err != nil {
				log.Criticalf("application initialize failed with a panic: %s", err)
			}
		}()

		logger.DefaultDynamicModuleName(true)
		logger.DefaultLogLevel(level.GetLevelByName(cliCtx.String("log_level")))
		if glacier.defaultLogFormatter == nil {
			glacier.defaultLogFormatter = formatter.NewDefaultFormatter(cliCtx.Bool("log_color"))
		}
		logger.DefaultLogFormatter(glacier.defaultLogFormatter)

		ctx, cancel := context.WithCancel(context.Background())
		cc := container.NewWithContext(ctx)
		glacier.container = cc
		cc.MustSingleton(func() *cli.Context { return cliCtx })

		cc.MustBindValue("version", glacier.version)
		cc.MustBindValue("startup_time", startupTs)
		cc.MustSingleton(ConfigLoader)

		cc.MustSingleton(func(conf *Config) *graceful.Graceful {
			return graceful.NewWithDefault(conf.ShutdownTimeout)
		})
		cc.MustResolve(func(gf *graceful.Graceful) {
			gf.AddShutdownHandler(cancel)
		})

		cc.MustSingleton(func() event.Store { return event.NewMemoryEventStore(false) })
		cc.MustSingleton(event.NewEventManager)

		cc.MustSingleton(func() *WebApp {
			return NewWebApp(cc, glacier.webAppRouterFunc, glacier.webAppServerFunc)
		})

		cc.MustSingleton(func() *cronV3.Cron {
			return cronV3.New(cronV3.WithSeconds(), cronV3.WithLogger(cronLogger{}))
		})
		cc.MustSingleton(cron.NewManager)
		cc.MustSingleton(period_job.NewManager)

		for _, i := range glacier.singletons {
			cc.MustSingleton(i)
		}

		for _, i := range glacier.prototypes {
			cc.MustPrototype(i)
		}

		for _, p := range glacier.providers {
			p.Register(cc)
		}

		if glacier.useStackLogger != nil {
			stackWriter := writer.NewStackWriter()
			glacier.useStackLogger(cc, stackWriter)
			logger.All().LogWriter(stackWriter)
		}

		log.WithFields(logger.Fields{
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

		log.WithFields(logger.Fields{
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
						log.Errorf("service %s has stopped: %v", s.Name(), err)
					}
				})
			}(s)
		}

		log.WithFields(logger.Fields{
			"count": len(glacier.services),
		}).Debugf("services has been started")

		defer cc.MustResolve(func(conf *Config) {
			if err := recover(); err != nil {
				log.Criticalf("application startup failed: %v", err)
			}

			if conf.ShutdownTimeout > 0 {
				ok := make(chan interface{}, 0)
				go func() {
					wg.Wait()
					ok <- struct{}{}
				}()
				select {
				case <-ok:
					log.Debugf("all services has been stopped")
				case <-time.After(conf.ShutdownTimeout):
					log.Errorf("shutdown timeout, exit directly")
				}
			} else {
				wg.Wait()
				log.Debugf("all services has been stopped")
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

			log.Debugf("cron task server has been started")

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

			log.Debugf("started glacier application in %v", time.Now().Sub(startupTs))

			return gf.Start()
		})

		if err != nil {
			return err
		}

		return nil
	}
}

type cronLogger struct{}

func (l cronLogger) Info(msg string, keysAndValues ...interface{}) {
	// Just drop it, we don't care
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.WithFields(logger.Fields{
		"arguments": keysAndValues,
	}).Errorf("%s: %v", msg, err)
}
