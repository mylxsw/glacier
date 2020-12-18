package infra

import "github.com/mylxsw/glacier/web"

// SetMultipartFormMaxMemoryOption Multipart-form 解析占用最大内存
func SetMultipartFormMaxMemoryOption(max int64) WebServerOption {
	return func(conf *web.Config) {
		conf.MultipartFormMaxMemory = max
	}
}

// SetIgnoreLastSlashOption 忽略路由地址末尾的 /
func SetIgnoreLastSlashOption(ignore bool) WebServerOption {
	return func(conf *web.Config) {
		conf.IgnoreLastSlash = ignore
	}
}

// SetTempFileOption 设置临时文件规则
func SetTempFileOption(tempDir, tempFilePattern string) WebServerOption {
	return func(conf *web.Config) {
		if tempDir != "" {
			conf.TempDir = tempDir
		}

		if tempFilePattern != "" {
			conf.TempFilePattern = tempFilePattern
		}
	}
}
