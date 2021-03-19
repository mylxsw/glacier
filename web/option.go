package web

import "time"

// SetMultipartFormMaxMemoryOption Multipart-form 解析占用最大内存
func SetMultipartFormMaxMemoryOption(max int64) Option {
	return func(conf *Config) {
		conf.MultipartFormMaxMemory = max
	}
}

// SetIgnoreLastSlashOption 忽略路由地址末尾的 /
func SetIgnoreLastSlashOption(ignore bool) Option {
	return func(conf *Config) {
		conf.IgnoreLastSlash = ignore
	}
}

// SetTempFileOption 设置临时文件规则
func SetTempFileOption(tempDir, tempFilePattern string) Option {
	return func(conf *Config) {
		if tempDir != "" {
			conf.TempDir = tempDir
		}

		if tempFilePattern != "" {
			conf.TempFilePattern = tempFilePattern
		}
	}
}

// SetInitHandlerOption 初始化阶段，web 应用对象还没有创建，在这里可以更新 web 配置
func SetInitHandlerOption(h InitHandler) Option {
	return func(conf *Config) {
		conf.initHandler = h
	}
}

// SetListenerHandlerOption 服务初始化阶段，web 服务对象已经创建，此时不能再更新 web 配置了
func SetListenerHandlerOption(h ListenerHandler) Option {
	return func(conf *Config) {
		conf.listenerHandler = h
	}
}

// SetRouteHandlerOption 路由注册 Handler，在该 Handler 中注册 API 路由规则
func SetRouteHandlerOption(h RouteHandler) Option {
	return func(conf *Config) {
		conf.routeHandler = h
	}
}

// SetExceptionHandlerOption 设置 Server APP 异常处理器
func SetExceptionHandlerOption(h ExceptionHandler) Option {
	return func(conf *Config) {
		conf.exceptionHandler = h
	}
}

// SetMuxRouteHandlerOption 路由注册 Handler，该方法获取到的是底层的 Gorilla Mux 对象
func SetMuxRouteHandlerOption(h MuxRouteHandler) Option {
	return func(conf *Config) {
		conf.muxRouteHandler = h
	}
}

// SetHttpWriteTimeoutOption set Http write timeout
func SetHttpWriteTimeoutOption(t time.Duration) Option {
	return func(conf *Config) {
		conf.HttpWriteTimeout = t
	}
}

// SetHttpReadTimeoutOption set Http read timeout
func SetHttpReadTimeoutOption(t time.Duration) Option {
	return func(conf *Config) {
		conf.HttpReadTimeout = t
	}
}

// SetHttpIdleTimeoutOption set Http idle timeout
func SetHttpIdleTimeoutOption(t time.Duration) Option {
	return func(conf *Config) {
		conf.HttpIdleTimeout = t
	}
}

