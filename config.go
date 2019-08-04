package glacier

import (
	"time"

	"github.com/urfave/cli"
)

type Config struct {
	HttpListen       string
	HttpWriteTimeout time.Duration
	HttpReadTimeout  time.Duration
	HttpIdleTimeout  time.Duration
}

func ConfigLoader(c *cli.Context) *Config {
	config := &Config{}
	config.HttpListen = c.String("listen")

	config.HttpWriteTimeout = time.Second * 15
	config.HttpReadTimeout = time.Second * 15
	config.HttpIdleTimeout = time.Second * 60

	return config
}
