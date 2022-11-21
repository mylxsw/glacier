package application

import (
	"fmt"
	"os"
	"time"

	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/infra"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

type Application struct {
	glacier infra.Glacier
	cli     *cli.App
}

func (application *Application) Cli() *cli.App {
	return application.cli
}

func (application *Application) WithDescription(desc string) *Application {
	application.cli.Description = desc
	return application
}

func (application *Application) WithName(name string) *Application {
	application.cli.Name = name
	return application
}

func (application *Application) WithUsage(usage string) *Application {
	application.cli.Usage = usage
	return application
}

func (application *Application) WithUsageText(usageText string) *Application {
	application.cli.UsageText = usageText
	return application
}

func (application *Application) WithAuthor(name, email string) *Application {
	application.cli.Authors = append(application.cli.Authors, cli.Author{Name: name, Email: email})
	return application
}

func (application *Application) WithAuthors(authors ...cli.Author) *Application {
	application.cli.Authors = append(application.cli.Authors, authors...)
	return application
}

func (application *Application) WithCLIOptions(fn func(cliAPP *cli.App)) *Application {
	fn(application.cli)
	return application
}

func (application *Application) WithShutdownTimeoutFlagSupport(timeout time.Duration) *Application {
	return application.AddFlags(altsrc.NewDurationFlag(cli.DurationFlag{
		Name:  glacier.ShutdownTimeoutOption,
		Usage: "set a shutdown timeout for each module",
		Value: timeout,
	}))
}

func (application *Application) WithFlagYAMLSupport(flagName string) *Application {
	application.cli.Flags = append(application.cli.Flags, cli.StringFlag{
		Name:  flagName,
		Value: "",
		Usage: "configuration file path",
	})

	application.cli.Before = func(c *cli.Context) error {
		conf := c.String(flagName)
		if conf == "" {
			return nil
		}

		inputSource, err := altsrc.NewYamlSourceFromFile(conf)
		if err != nil {
			return err
		}

		return altsrc.ApplyInputSourceValues(c, inputSource, c.App.Flags)
	}

	return application
}

func (application *Application) WithLogger(logger infra.Logger) *Application {
	application.glacier.SetLogger(logger)
	return application
}

func MustRun(app *Application) {
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func MustStart(version string, init func(app *Application) error) {
	MustRun(CreateAndInit(version, init))
}

func CreateAndInit(version string, init func(app *Application) error) *Application {
	app := Create(version)

	if init == nil {
		return app
	}

	if err := init(app); err != nil {
		panic(fmt.Errorf("application init failed: %v", err))
	}

	return app
}

func Create(version string) *Application {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Version = version
	app.Flags = make([]cli.Flag, 0)

	glacierIns := glacier.CreateGlacier(version, 3)
	app.Action = func(c *cli.Context) error {
		return glacierIns.Main(c)
	}

	return &Application{
		glacier: glacierIns,
		cli:     app,
	}
}

// Glacier glacierImpl return glacierImpl instance
func (application *Application) Glacier() infra.Glacier {
	return application.glacier
}

// AddFlags add flags to cli
func (application *Application) AddFlags(flags ...cli.Flag) *Application {
	application.cli.Flags = append(application.cli.Flags, flags...)
	return application
}

func (application *Application) AddIntFlag(name string, defaultVal int, usage string) *Application {
	return application.AddFlags(IntFlag(name, defaultVal, usage))
}
func (application *Application) AddFloat64Flag(name string, defaultVal float64, usage string) *Application {
	return application.AddFlags(Float64Flag(name, defaultVal, usage))
}
func (application *Application) AddStringSliceFlag(name string, defaultVal []string, usage string) *Application {
	return application.AddFlags(StringSliceFlag(name, defaultVal, usage))
}
func (application *Application) AddIntSliceFlag(name string, defaultVal []int, usage string) *Application {
	return application.AddFlags(IntSliceFlag(name, defaultVal, usage))
}
func (application *Application) AddStringFlag(name string, defaultVal string, usage string) *Application {
	return application.AddFlags(StringFlag(name, defaultVal, usage))
}
func (application *Application) AddBoolFlag(name string, usage string) *Application {
	return application.AddFlags(BoolFlag(name, usage))
}
func (application *Application) AddDurationFlag(name string, defaultVal time.Duration, usage string) *Application {
	return application.AddFlags(DurationFlag(name, defaultVal, usage))
}

// Run start glacierImpl server
func (application *Application) Run(args []string) error {
	return application.cli.Run(args)
}
