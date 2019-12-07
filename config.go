package glacier

import (
	"time"

	"github.com/mylxsw/glacier/web"
)

const (
	HttpListenOption          = "listen"
	ShutdownTimeoutOption     = "shutdown_timeout"
	WebTemplatePrefixOption   = "web_template_prefix"
	WebMultipartFormMaxMemory = "web_multipart_form_max_memory"
)

type Config struct {
	HttpListen       string
	HttpWriteTimeout time.Duration
	HttpReadTimeout  time.Duration
	HttpIdleTimeout  time.Duration
	ShutdownTimeout  time.Duration

	WebConfig *web.Config
}

func ConfigLoader(c FlagContext) *Config {
	config := &Config{}
	config.HttpListen = c.String(HttpListenOption)
	config.ShutdownTimeout = c.Duration(ShutdownTimeoutOption)

	config.HttpWriteTimeout = time.Second * 15
	config.HttpReadTimeout = time.Second * 15
	config.HttpIdleTimeout = time.Second * 60
	config.WebConfig = web.DefaultConfig()

	return config
}
