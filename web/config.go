package web

type Config struct {
	MultipartFormMaxMemory int64  // Multipart-form 解析占用最大内存
	ViewTemplatePathPrefix string // 视图模板目录
	TempDir                string // 临时目录，用于上传文件等
	TempFilePattern        string // 临时文件规则
}

// DefaultConfig create a default config
func DefaultConfig() *Config {
	return &Config{
		MultipartFormMaxMemory: int64(10 << 20), // 10M
		ViewTemplatePathPrefix: "",
		TempDir:                "/tmp",
		TempFilePattern:        "hades-files-",
	}
}
