package glacier

import (
	"context"

	"github.com/mylxsw/go-toolkit/container"
	"github.com/mylxsw/go-toolkit/events"
	"github.com/mylxsw/go-toolkit/graceful"
	"github.com/mylxsw/go-toolkit/log"
	"github.com/mylxsw/go-toolkit/period_job"
	"github.com/mylxsw/go-toolkit/web"
	"github.com/robfig/cron"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

var logger = log.Module("glacier")

// Glacier is the server
type Glacier struct {
	appName   string
	version   string
	app       *cli.App
	container *container.Container

	beforeServerStart func(cc *container.Container) error
	beforeServerStop  func(cc *container.Container) error

	webAppInitFunc   interface{}
	webAppRouterFunc InitRouterHandler

	cronTaskFunc      CronTaskFunc
	eventListenerFunc EventListenerFunc
	periodJobFunc     PeriodJobFunc

	httpListenAddr string

	singletons []interface{}
	prototypes []interface{}
}

type CronTaskFunc func(cr *cron.Cron, cc *container.Container) error
type EventListenerFunc func(listener *events.EventManager, cc *container.Container)
type PeriodJobFunc func(pj *period_job.Manager, cc *container.Container)

var glacierInstance *Glacier

// App return Glacier instance you created
func App() *Glacier {
	if glacierInstance == nil {
		panic("you should create a Glacier by calling Create function first!")
	}

	return glacierInstance
}

// Container return container instance for glacier
func Container() *container.Container {
	return App().Container()
}

// Create a new Glacier server
func Create(version string, flags ...cli.Flag) *Glacier {
	if glacierInstance != nil {
		panic("a glacier instance has been created already")
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
			Usage: "specify log level",
		}),
		altsrc.NewBoolTFlag(cli.BoolTFlag{
			Name:  "log_colorful",
			Usage: "set true if you want to hav colorful log output",
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

		return altsrc.ApplyInputSourceValues(c, inputSource, serverFlags)
	}
	app.Flags = serverFlags

	glacierInstance = &Glacier{}
	glacierInstance.app = app
	glacierInstance.webAppInitFunc = func() error { return nil }
	glacierInstance.webAppRouterFunc = func(router *web.Router, mw web.RequestMiddleware) {}
	glacierInstance.singletons = make([]interface{}, 0)
	glacierInstance.prototypes = make([]interface{}, 0)

	app.Action = createServer(glacierInstance)

	return glacierInstance
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

// BeforeServerStart set a hook func executed before server start
func (glacier *Glacier) BeforeServerStart(f func(cc *container.Container) error) *Glacier {
	glacier.beforeServerStart = f
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

// WebAppRouter add routes for http server
func (glacier *Glacier) WebAppRouter(handler InitRouterHandler) *Glacier {
	glacier.webAppRouterFunc = handler
	return glacier
}

// Crontab add cron tasks
func (glacier *Glacier) Crontab(f CronTaskFunc) *Glacier {
	glacier.cronTaskFunc = f
	return glacier
}

// EventListener add event listeners
func (glacier *Glacier) EventListener(f EventListenerFunc) *Glacier {
	glacier.eventListenerFunc = f
	return glacier
}

// PeriodJob add period jobs
func (glacier *Glacier) PeriodJob(f PeriodJobFunc) *Glacier {
	glacier.periodJobFunc = f
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

// Container return container instance
func (glacier *Glacier) Container() *container.Container {
	return glacier.container
}

// Run start Glacier server
func (glacier *Glacier) Run(args []string) error {
	if glacier.httpListenAddr != "" {
		glacier.app.Flags = append(glacier.app.Flags, altsrc.NewStringFlag(cli.StringFlag{
			Name:  "listen",
			Value: glacier.httpListenAddr,
			Usage: "http server listen address",
		}))
	}

	return glacier.app.Run(args)
}

func createServer(glacier *Glacier) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		log.SetDefaultLevel(log.GetLevelByName(c.String("log_level")))
		log.SetDefaultColorful(c.Bool("log_colorful"))

		logger.Infof("server starting, version=%s", glacier.version)

		ctx, cancel := context.WithCancel(context.Background())
		cc := container.NewWithContext(ctx)
		glacier.container = cc

		cc.MustBindValue("version", glacier.version)
		cc.MustSingleton(func() *cli.Context {
			return c
		})
		cc.MustSingleton(ConfigLoader)
		cc.MustSingleton(func() events.EventStore {
			return events.NewMemoryEventStore(false)
		})
		cc.MustSingleton(events.NewEventManager)
		cc.MustSingleton(graceful.NewWithDefault)
		cc.MustSingleton(func() *WebApp {
			return NewWebApp(cc, glacier.webAppRouterFunc)
		})

		cc.MustSingleton(cron.New)
		cc.MustSingleton(period_job.NewManager)

		for _, i := range glacier.singletons {
			cc.MustSingleton(i)
		}

		for _, i := range glacier.prototypes {
			cc.MustPrototype(i)
		}

		defer cc.MustResolve(func(cr *cron.Cron, pj *period_job.Manager) {
			cancel()

			if glacier.beforeServerStop != nil {
				_ = glacier.beforeServerStop(cc)
			}

			cr.Stop()
			pj.Wait()
			logger.Debugf("all services has been stopped")
		})

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

		if glacier.httpListenAddr != "" {
			if err := cc.ResolveWithError(func(webApp *WebApp) error {
				if glacier.webAppInitFunc != nil {
					if err := webApp.Init(glacier.webAppInitFunc); err != nil {
						return err
					}
				}

				if err := webApp.Start(); err != nil {
					return err
				}

				return nil
			}); err != nil {
				return err
			}
		}

		err := cc.ResolveWithError(func(cr *cron.Cron, gf *graceful.Graceful) error {
			if glacier.cronTaskFunc != nil {
				if err := glacier.cronTaskFunc(cr, cc); err != nil {
					return err
				}
			}

			// start cron task server
			cr.Start()

			// start period jobs
			if glacier.periodJobFunc != nil {
				if err := cc.Resolve(glacier.periodJobFunc); err != nil {
					return err
				}
			}

			return gf.Start()
		})

		if err != nil {
			return err
		}

		return nil
	}
}
