package application

import (
	"time"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/infra"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

var _app *Application

// App return glacierImpl instance you created
func App() *Application {
	if _app == nil {
		panic("you should create a Glacier application by call Create function first!")
	}

	return _app
}

// Container return container instance for glacier
func Container() container.Container {
	return App().glacier.Container()
}

type Application struct {
	glacier infra.Glacier
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
func (application *Application) Glacier() infra.Glacier {
	return application.glacier
}

// AddFlags add flags to cli
func (application *Application) AddFlags(flags ...cli.Flag) *Application {
	application.cli.Flags = append(application.cli.Flags, flags...)
	return application
}

// Run start glacierImpl server
func (application *Application) Run(args []string) error {
	return application.cli.Run(args)
}
