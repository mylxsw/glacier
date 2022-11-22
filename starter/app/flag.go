package app

import (
	"time"

	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

func BoolFlag(name string, usage string) cli.Flag {
	return BoolEnvFlag(name, usage, "")
}

func BoolEnvFlag(name string, usage string, envName string) cli.Flag {
	return altsrc.NewBoolFlag(cli.BoolFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
	})
}

func StringFlag(name string, defaultVal string, usage string) cli.Flag {
	return StringEnvFlag(name, defaultVal, usage, "")
}

func StringEnvFlag(name string, defaultVal string, usage string, envName string) cli.Flag {
	return altsrc.NewStringFlag(cli.StringFlag{
		Name:   name,
		Usage:  usage,
		Value:  defaultVal,
		EnvVar: envName,
	})
}

func DurationFlag(name string, defaultVal time.Duration, usage string) cli.Flag {
	return DurationEnvFlag(name, defaultVal, usage, "")
}

func DurationEnvFlag(name string, defaultVal time.Duration, usage string, envName string) cli.Flag {
	return altsrc.NewDurationFlag(cli.DurationFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  defaultVal,
	})
}

func IntFlag(name string, defaultVal int, usage string) cli.Flag {
	return IntEnvFlag(name, defaultVal, usage, "")
}

func IntEnvFlag(name string, defaultVal int, usage string, envName string) cli.Flag {
	return altsrc.NewIntFlag(cli.IntFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  defaultVal,
	})
}

func Float64Flag(name string, defaultVal float64, usage string) cli.Flag {
	return Float64EnvFlag(name, defaultVal, usage, "")
}

func Float64EnvFlag(name string, defaultVal float64, usage string, envName string) cli.Flag {
	return altsrc.NewFloat64Flag(cli.Float64Flag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  defaultVal,
	})
}

func StringSliceFlag(name string, defaultVal []string, usage string) cli.Flag {
	return StringSliceEnvFlag(name, defaultVal, usage, "")
}

func StringSliceEnvFlag(name string, defaultVal []string, usage string, envName string) cli.Flag {
	defaultValS := cli.StringSlice(defaultVal)
	return altsrc.NewStringSliceFlag(cli.StringSliceFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  &defaultValS,
	})
}

func IntSliceFlag(name string, defaultVal []int, usage string) cli.Flag {
	return IntSliceEnvFlag(name, defaultVal, usage, "")
}

func IntSliceEnvFlag(name string, defaultVal []int, usage string, envName string) cli.Flag {
	defaultValS := cli.IntSlice(defaultVal)
	return altsrc.NewIntSliceFlag(cli.IntSliceFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  &defaultValS,
	})
}
