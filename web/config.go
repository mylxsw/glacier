package web

import (
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/infra"
)

type RouteHandler func(cc infra.Resolver, router Router, mw RequestMiddleware)
type MuxRouteHandler func(cc infra.Resolver, router *mux.Router)
type ListenerHandler func(server *http.Server, listener net.Listener)
type InitHandler func(cc infra.Resolver, webServer Server, conf *Config) error

type Config struct {
	routeHandler     RouteHandler
	listenerHandler  ListenerHandler
	muxRouteHandler  MuxRouteHandler
	initHandler      InitHandler
	exceptionHandler ExceptionHandler

	MultipartFormMaxMemory int64  // Multipart-form 解析占用最大内存
	ViewTemplatePathPrefix string // 视图模板目录
	TempDir                string // 临时目录，用于上传文件等
	TempFilePattern        string // 临时文件规则
	IgnoreLastSlash        bool   // 是否忽略 URL 末尾的 /

	HttpWriteTimeout time.Duration
	HttpReadTimeout  time.Duration
	HttpIdleTimeout  time.Duration
}

// DefaultConfig create a default config
func DefaultConfig() *Config {
	return &Config{
		MultipartFormMaxMemory: int64(10 << 20), // 10M
		ViewTemplatePathPrefix: "",
		TempDir:                "/tmp",
		TempFilePattern:        "glacier-files-",
		IgnoreLastSlash:        false,
		HttpWriteTimeout:       time.Second * 15,
		HttpReadTimeout:        time.Second * 15,
		HttpIdleTimeout:        time.Second * 60,
	}
}
