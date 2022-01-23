package glacier

import (
	"strings"
	"time"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-utils/str"
)

const (
	// ShutdownTimeoutOption 优雅停机超时时间命令行选型名称
	ShutdownTimeoutOption = "shutdown-timeout"
)

// Config 框架级配置
type Config struct {
	ShutdownTimeout time.Duration
}

// ConfigLoader 框架级配置实例创建
func ConfigLoader(c infra.FlagContext) *Config {
	config := &Config{}

	config.ShutdownTimeout = c.Duration(ShutdownTimeoutOption)
	if config.ShutdownTimeout.Microseconds() == 0 {
		config.ShutdownTimeout = 15 * time.Second
	}

	return config
}

// IsGlacierModuleLog 判断模块名称是否是 Glacier 框架内部模块
func IsGlacierModuleLog(module string) bool {
	if strings.HasPrefix(module, "github.com.mylxsw") {
		return str.HasPrefixes(module, []string{
			"github.com.mylxsw.glacier",
			"github.com.mylxsw.graceful",
		})
	}

	if strings.HasPrefix(module, "g.c.m") {
		return str.HasPrefixes(module, []string{
			"g.c.m.glacier",
			"g.c.m.graceful",
			"g.c.m.g.event",
			"g.c.m.g.scheduler",
			"g.c.m.g.web",
		})
	}

	return false

}
