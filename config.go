package glacier

import (
	"time"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/web"
)

const (
	ShutdownTimeoutOption = "shutdown_timeout"
)

type Config struct {
	HttpWriteTimeout time.Duration
	HttpReadTimeout  time.Duration
	HttpIdleTimeout  time.Duration
	ShutdownTimeout  time.Duration

	WebConfig *web.Config
}

func ConfigLoader(c infra.FlagContext) *Config {
	config := &Config{}
	config.ShutdownTimeout = c.Duration(ShutdownTimeoutOption)

	config.HttpWriteTimeout = time.Second * 15
	config.HttpReadTimeout = time.Second * 15
	config.HttpIdleTimeout = time.Second * 60
	config.WebConfig = web.DefaultConfig()

	return config
}
