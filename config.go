package glacier

import (
	"time"

	"github.com/mylxsw/glacier/infra"
)

const (
	ShutdownTimeoutOption = "shutdown_timeout"
)

type Config struct {
	ShutdownTimeout time.Duration
}

func ConfigLoader(c infra.FlagContext) *Config {
	config := &Config{}
	config.ShutdownTimeout = c.Duration(ShutdownTimeoutOption)

	return config
}
