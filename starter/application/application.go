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
	gcr infra.Glacier
	cli *cli.App
}

func (app *Application) Cli() *cli.App {
	return app.cli
}

func (app *Application) WithDescription(desc string) *Application {
	app.cli.Description = desc
	return app
}

func (app *Application) WithName(name string) *Application {
	app.cli.Name = name
	return app
}

func (app *Application) WithUsage(usage string) *Application {
	app.cli.Usage = usage
	return app
}

func (app *Application) WithUsageText(usageText string) *Application {
	app.cli.UsageText = usageText
	return app
}

func (app *Application) WithAuthor(name, email string) *Application {
	app.cli.Authors = append(app.cli.Authors, cli.Author{Name: name, Email: email})
	return app
}

func (app *Application) WithAuthors(authors ...cli.Author) *Application {
	app.cli.Authors = append(app.cli.Authors, authors...)
	return app
}

func (app *Application) WithCLIOptions(fn func(cliAPP *cli.App)) *Application {
	fn(app.cli)
	return app
}

func (app *Application) WithShutdownTimeoutFlagSupport(timeout time.Duration) *Application {
	return app.AddFlags(altsrc.NewDurationFlag(cli.DurationFlag{
		Name:  glacier.ShutdownTimeoutOption,
		Usage: "set a shutdown timeout for each module",
		Value: timeout,
	}))
}

func (app *Application) WithYAMLFlag(flagName string) *Application {
	app.cli.Flags = append(app.cli.Flags, cli.StringFlag{
		Name:  flagName,
		Value: "",
		Usage: "configuration file path",
	})

	app.cli.Before = func(c *cli.Context) error {
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

	return app
}

func (app *Application) WithLogger(logger infra.Logger) *Application {
	app.gcr.SetLogger(logger)
	return app
}

func MustRun(app *Application) {
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func MustStart(version string, asyncRunnerCount int, init func(app *Application) error) {
	MustRun(CreateAndInit(version, asyncRunnerCount, init))
}

func CreateAndInit(version string, asyncRunnerCount int, init func(app *Application) error) *Application {
	app := Create(version, asyncRunnerCount)

	if init == nil {
		return app
	}

	if err := init(app); err != nil {
		panic(fmt.Errorf("application init failed: %v", err))
	}

	return app
}

func Create(version string, asyncRunnerCount int) *Application {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Version = version
	app.Flags = make([]cli.Flag, 0)

	glacierIns := glacier.CreateGlacier(version, asyncRunnerCount)
	app.Action = func(c *cli.Context) error {
		return glacierIns.Main(c)
	}

	return &Application{
		gcr: glacierIns,
		cli: app,
	}
}

// Glacier glacierImpl return glacierImpl instance
func (app *Application) Glacier() infra.Glacier {
	return app.gcr
}

// AddFlags add flags to cli
func (app *Application) AddFlags(flags ...cli.Flag) *Application {
	app.cli.Flags = append(app.cli.Flags, flags...)
	return app
}

func (app *Application) AddIntFlag(name string, defaultVal int, usage string) *Application {
	return app.AddFlags(IntFlag(name, defaultVal, usage))
}
func (app *Application) AddFloat64Flag(name string, defaultVal float64, usage string) *Application {
	return app.AddFlags(Float64Flag(name, defaultVal, usage))
}
func (app *Application) AddStringSliceFlag(name string, defaultVal []string, usage string) *Application {
	return app.AddFlags(StringSliceFlag(name, defaultVal, usage))
}
func (app *Application) AddIntSliceFlag(name string, defaultVal []int, usage string) *Application {
	return app.AddFlags(IntSliceFlag(name, defaultVal, usage))
}
func (app *Application) AddStringFlag(name string, defaultVal string, usage string) *Application {
	return app.AddFlags(StringFlag(name, defaultVal, usage))
}
func (app *Application) AddBoolFlag(name string, usage string) *Application {
	return app.AddFlags(BoolFlag(name, usage))
}
func (app *Application) AddDurationFlag(name string, defaultVal time.Duration, usage string) *Application {
	return app.AddFlags(DurationFlag(name, defaultVal, usage))
}

// Run start glacierImpl server
func (app *Application) Run(args []string) error {
	return app.cli.Run(args)
}
