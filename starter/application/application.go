package application

import (
	"time"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

var _app *Application

// App return glacierImpl instance you created
func App() *Application {
	if _app == nil {
		panic("you should create a glacierImpl application by call CreateGlacier function first!")
	}

	return _app
}

// Container return container instance for glacier
func Container() container.Container {
	return App().glacier.Container()
}

type Application struct {
	glacier glacier.Glacier
	cli     *cli.App
}

func (application *Application) Cli() *cli.App {
	return application.cli
}

func Create(version string, flags ...cli.Flag) *Application {
	if _app != nil {
		panic("a glacier application has been created")
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

	glacierIns := glacier.CreateGlacier(version)
	app.Action = func(c *cli.Context) error {
		return glacierIns.Handler()(c)
	}

	_app = &Application{
		glacier: glacierIns,
		cli:     app,
	}

	return _app
}

// glacierImpl return glacierImpl instance
func (application *Application) Glacier() glacier.Glacier {
	return application.glacier
}

// AddFlags add flags to cli
func (application *Application) AddFlags(flags ...cli.Flag) *Application {
	application.cli.Flags = append(application.cli.Flags, flags...)
	return application
}

// Run start glacierImpl server
func (application *Application) Run(args []string) error {
	if application.glacier.HttpListenAddr() != "" {
		application.cli.Flags = append(
			application.cli.Flags,
			altsrc.NewStringFlag(cli.StringFlag{
				Name:  glacier.HttpListenOption,
				Value: application.glacier.HttpListenAddr(),
				Usage: "http server listen address",
			}),
			altsrc.NewStringFlag(cli.StringFlag{
				Name:  glacier.WebTemplatePrefixOption,
				Usage: "web template path prefix",
				Value: "",
			}),
			altsrc.NewInt64Flag(cli.Int64Flag{
				Name:  glacier.WebMultipartFormMaxMemory,
				Usage: "multipart form max memory size in bytes",
				Value: int64(10 << 20),
			}))
	}

	return application.cli.Run(args)
}

// StringFlag create a string flag
func StringFlag(name string, defaultValue string, usage string) *altsrc.StringFlag {
	return altsrc.NewStringFlag(cli.StringFlag{
		Name:  name,
		Usage: usage,
		Value: defaultValue,
	})
}

// IntFlag create a int flag
func IntFlag(name string, defaultValue int, usage string) *altsrc.IntFlag {
	return altsrc.NewIntFlag(cli.IntFlag{
		Name:  name,
		Usage: usage,
		Value: defaultValue,
	})
}
