package glacier

import (
	"time"

	"github.com/mylxsw/glacier/web"
	"github.com/urfave/cli"
)

type Config struct {
	HttpListen       string
	HttpWriteTimeout time.Duration
	HttpReadTimeout  time.Duration
	HttpIdleTimeout  time.Duration

	ShutdownTimeout time.Duration

	WebConfig *web.Config
}

func ConfigLoader(c *cli.Context) *Config {
	config := &Config{}
	config.HttpListen = c.String("listen")
	config.ShutdownTimeout = c.Duration("shutdown_timeout")

	config.HttpWriteTimeout = time.Second * 15
	config.HttpReadTimeout = time.Second * 15
	config.HttpIdleTimeout = time.Second * 60
	config.WebConfig = web.DefaultConfig()

	return config
}
