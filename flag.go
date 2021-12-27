package glacier

import (
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
	"time"
)

func BoolFlag(name string, defaultVal bool, usage string) cli.Flag {
	return BoolEnvFlag(name, defaultVal, usage, "")
}

func BoolEnvFlag(name string, defaultVal bool, usage string, envName string) cli.Flag {
	if defaultVal {
		return altsrc.NewBoolTFlag(cli.BoolTFlag{
			Name:   name,
			Usage:  usage,
			EnvVar: envName,
		})
	}

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

func Int64Flag(name string, defaultVal int64, usage string) cli.Flag {
	return Int64EnvFlag(name, defaultVal, usage, "")
}

func Int64EnvFlag(name string, defaultVal int64, usage string, envName string) cli.Flag {
	return altsrc.NewInt64Flag(cli.Int64Flag{
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

func UintFlag(name string, defaultVal uint, usage string) cli.Flag {
	return UintEnvFlag(name, defaultVal, usage, "")
}

func UintEnvFlag(name string, defaultVal uint, usage string, envName string) cli.Flag {
	return altsrc.NewUintFlag(cli.UintFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  defaultVal,
	})
}

func Uint64Flag(name string, defaultVal uint64, usage string) cli.Flag {
	return Uint64EnvFlag(name, defaultVal, usage, "")
}

func Uint64EnvFlag(name string, defaultVal uint64, usage string, envName string) cli.Flag {
	return altsrc.NewUint64Flag(cli.Uint64Flag{
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

func Int64SliceFlag(name string, defaultVal []int64, usage string) cli.Flag {
	return Int64SliceEnvFlag(name, defaultVal, usage, "")
}

func Int64SliceEnvFlag(name string, defaultVal []int64, usage string, envName string) cli.Flag {
	defaultValS := cli.Int64Slice(defaultVal)
	return altsrc.NewInt64SliceFlag(cli.Int64SliceFlag{
		Name:   name,
		Usage:  usage,
		EnvVar: envName,
		Value:  &defaultValS,
	})
}
