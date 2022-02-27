package infra

import (
	"context"
	"net"
	"time"

	"github.com/mylxsw/container"
)

const (
	VersionKey     string = "version"
	StartupTimeKey string = "startup_time"
)

type Graceful interface {
	AddReloadHandler(h func())
	AddShutdownHandler(h func())
	Reload()
	Shutdown()
	Start() error
}

// Service is an interface for service
type Service interface {
	// Init initialize the service
	Init(resolver Resolver) error
	// Name return service name
	Name() string
	// Start service, not blocking
	Start() error
	// Stop the service
	Stop()
	// Reload reload service
	Reload()
}

type Provider interface {
	// Register add some dependency for current module
	// this method is called one by one synchronous
	// service provider don't autowired in this stage
	Register(binder Binder)
}

// Priority 优先级接口
// 实现该接口后，在加载 Provider/Service 时，会按照 Priority 大小依次加载（值越小越先加载）
type Priority interface {
	Priority() int
}

type ProviderBoot interface {
	// Boot starts the module
	// this method is called one by one synchronous after all register methods called
	// service provider has been autowired in this stage
	Boot(resolver Resolver)
}

type DaemonProvider interface {
	Provider
	// Daemon is an async method called after boot
	// this method is called asynchronous and concurrent
	Daemon(ctx context.Context, resolver Resolver)
}

// ProviderAggregate Provider 聚合，所有实现该接口的 Provider 在加载之前将会先加载该集合中的 Provider
type ProviderAggregate interface {
	Aggregates() []Provider
}

type ListenerBuilder interface {
	Build(resolver Resolver) (net.Listener, error)
}

type FlagContext interface {
	String(name string) string
	StringSlice(name string) []string
	Bool(name string) bool
	Int(name string) int
	IntSlice(name string) []int
	Duration(name string) time.Duration
	Float64(name string) float64
	FlagNames() (names []string)
}

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Warning(v ...interface{})
	Warningf(format string, v ...interface{})
	// Critical 关键性错误，遇到该日志输出时，应用直接退出
	Critical(v ...interface{})
	// Criticalf 关键性错误，遇到该日志输出时，应用直接退出
	Criticalf(format string, v ...interface{})
}

type Glacier interface {
	SetLogger(logger Logger) Glacier

	// WithFlagContext 设置 FlagContext，支持覆盖 FlagContext 默认实现
	// 参数 fn 只支持 `func(...) infra.FlagContext` 形式
	WithFlagContext(fn interface{}) Glacier

	// Provider 注册一个模块
	Provider(providers ...Provider)
	// Service 注册一个 Service
	Service(services ...Service)
	// Async 注册一个异步任务
	Async(asyncJobs ...interface{})

	// Graceful 设置优雅停机实现
	Graceful(builder func() Graceful) Glacier

	// OnServerReady call a function a server ready
	OnServerReady(f interface{})

	// Main 应用入口
	Main(cliCtx FlagContext) error
	// BeforeInitialize Glacier 初始化之前执行，一般用于设置一些基本配置，比如日志等
	BeforeInitialize(f func(c FlagContext) error) Glacier
	// AfterInitialized Glacier 初始化之后执行，所有的实例绑定都可以使用了
	AfterInitialized(f func(resolver Resolver) error) Glacier

	// BeforeServerStart 此时所有对象已经注册完毕，但是服务启动前执行
	BeforeServerStart(f func(cc container.Container) error) Glacier
	// AfterServerStart 此时所有服务都已经启动（Main 除外）
	AfterServerStart(f func(cc Resolver) error) Glacier
	// BeforeServerStop 服务停止前的回调
	BeforeServerStop(f func(cc Resolver) error) Glacier
	// AfterProviderBooted 所有的 providers 都已经完成 boot 之后执行
	AfterProviderBooted(f interface{}) Glacier

	PreBind(fn func(binder Binder)) Glacier
	Singleton(ins ...interface{}) Glacier
	Prototype(ins ...interface{}) Glacier
	ResolveWithError(resolver interface{}) error
	MustResolve(resolver interface{})
	Container() container.Container
}

type Binder container.Binder
type Resolver container.Resolver

type Hook interface {
	// OnServerReady call a function a server ready
	OnServerReady(f interface{})
}

func WithCondition(init interface{}, onCondition interface{}) container.Conditional {
	return container.WithCondition(init, onCondition)
}
