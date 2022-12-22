package app

import (
	"fmt"
	"os"
	"time"

	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/infra"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

type App struct {
	gcr infra.Glacier
	cli *cli.App
}

func (app *App) Cli() *cli.App {
	return app.cli
}

func (app *App) WithDescription(desc string) *App {
	app.cli.Description = desc
	return app
}

func (app *App) WithName(name string) *App {
	app.cli.Name = name
	return app
}

func (app *App) WithUsage(usage string) *App {
	app.cli.Usage = usage
	return app
}

func (app *App) WithUsageText(usageText string) *App {
	app.cli.UsageText = usageText
	return app
}

func (app *App) WithAuthor(name, email string) *App {
	app.cli.Authors = append(app.cli.Authors, &cli.Author{Name: name, Email: email})
	return app
}

func (app *App) WithAuthors(authors ...*cli.Author) *App {
	app.cli.Authors = append(app.cli.Authors, authors...)
	return app
}

func (app *App) WithCLIOptions(fn func(cliAPP *cli.App)) *App {
	fn(app.cli)
	return app
}

func (app *App) WithShutdownTimeoutFlag(timeout time.Duration) *App {
	return app.AddFlags(altsrc.NewDurationFlag(&cli.DurationFlag{
		Name:  glacier.ShutdownTimeoutOption,
		Usage: "set a shutdown timeout for each module",
		Value: timeout,
	}))
}

func (app *App) WithYAMLFlag(flagName string) *App {
	app.cli.Flags = append(app.cli.Flags, &cli.StringFlag{
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

func (app *App) WithLogger(logger infra.Logger) *App {
	app.gcr.SetLogger(logger)
	return app
}

func MustRun(app *App) {
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func MustStart(version string, asyncRunnerCount int, init func(app *App) error) {
	MustRun(CreateAndInit(version, asyncRunnerCount, init))
}

func CreateAndInit(version string, asyncRunnerCount int, init func(app *App) error) *App {
	app := Create(version, asyncRunnerCount)

	if init == nil {
		return app
	}

	if err := init(app); err != nil {
		panic(fmt.Errorf("application init failed: %v", err))
	}

	return app
}

func Default(version string) *App {
	return Create(version, 3)
}

func Create(version string, asyncRunnerCount int) *App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Version = version
	app.Flags = make([]cli.Flag, 0)

	glacierIns := glacier.New(version, asyncRunnerCount)
	app.Action = func(c *cli.Context) error {
		return glacierIns.Start(c)
	}

	return &App{
		gcr: glacierIns,
		cli: app,
	}
}

// Glacier glacierImpl return glacierImpl instance
func (app *App) Glacier() infra.Glacier {
	return app.gcr
}

// AddFlags add flags to cli
func (app *App) AddFlags(flags ...cli.Flag) *App {
	app.cli.Flags = append(app.cli.Flags, flags...)
	return app
}

func (app *App) AddIntFlag(name string, defaultVal int, usage string) *App {
	return app.AddFlags(IntFlag(name, defaultVal, usage))
}
func (app *App) AddFloat64Flag(name string, defaultVal float64, usage string) *App {
	return app.AddFlags(Float64Flag(name, defaultVal, usage))
}
func (app *App) AddStringSliceFlag(name string, defaultVal []string, usage string) *App {
	return app.AddFlags(StringSliceFlag(name, defaultVal, usage))
}
func (app *App) AddIntSliceFlag(name string, defaultVal []int, usage string) *App {
	return app.AddFlags(IntSliceFlag(name, defaultVal, usage))
}
func (app *App) AddStringFlag(name string, defaultVal string, usage string) *App {
	return app.AddFlags(StringFlag(name, defaultVal, usage))
}
func (app *App) AddBoolFlag(name string, usage string) *App {
	return app.AddFlags(BoolFlag(name, usage))
}
func (app *App) AddDurationFlag(name string, defaultVal time.Duration, usage string) *App {
	return app.AddFlags(DurationFlag(name, defaultVal, usage))
}

// Run start glacierImpl server
func (app *App) Run(args []string) error {
	return app.cli.Run(args)
}
