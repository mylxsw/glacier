package infra

import (
	"context"
	"errors"
	"net"
	"reflect"
	"time"

	"github.com/mylxsw/go-ioc"
)

const (
	VersionKey     string = "version"
	StartupTimeKey string = "startup_time"
)

var (
	// DEBUG enable debug mode for glacier
	DEBUG = false
	// WARN enable warning log for glacier
	WARN = true
	// PrintGraph enable print graph
	PrintGraph = false
)

type Graceful interface {
	AddReloadHandler(h func())
	AddShutdownHandler(h func())
	// AddPreShutdownHandler 在所有服务停止之前执行，用于执行一些清理操作，该操作会阻塞服务停止，直到该操作完成，不受超时时间限制
	AddPreShutdownHandler(h func())
	Reload()
	Shutdown()
	Start() error
}

// Service is an interface for service
type Service interface {
	// Start service, not blocking
	Start() error
}

// CompleteService is an interface for a service which implements all service interface
type CompleteService interface {
	Service
	Initializer
	Stoppable
	Reloadable
	Nameable
}

// Initializer is an interface for service initializer
type Initializer interface {
	Init(resolver Resolver) error
}

// Stoppable is an interface for service that can be stopped
type Stoppable interface {
	Stop()
}

// Reloadable is an interface for reloadable service
type Reloadable interface {
	Reload()
}

// Nameable is an interface for service/provider name
type Nameable interface {
	Name() string
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
	OnServerReady(ffs ...interface{})

	// Start 应用入口
	Start(cliCtx FlagContext) error
	// Init Glacier 初始化之前执行，一般用于设置一些基本配置，比如日志等
	Init(f func(fc FlagContext) error) Glacier
	// BeforeServerStop 服务停止前的回调
	BeforeServerStop(f func(resolver Resolver) error) Glacier
	PreBind(fn func(binder Binder)) Glacier

	Singleton(ins ...interface{}) Glacier
	Prototype(ins ...interface{}) Glacier
	Resolve(resolver interface{}) error
	MustResolve(resolver interface{})
	Container() Container
	Resolver() Resolver
	Binder() Binder
}

type Container ioc.Container
type Binder ioc.Binder
type Resolver ioc.Resolver

type Hook interface {
	// OnServerReady call a function a server ready
	OnServerReady(ffs ...interface{})
}

func WithCondition(init interface{}, onCondition interface{}) ioc.Conditional {
	return ioc.WithCondition(init, onCondition)
}

// Autowire Automatically inject dependencies into obj and return obj for convenient chaining.
func Autowire[T any](resolver Resolver, obj T) T {
	if reflect.ValueOf(obj).Kind() != reflect.Ptr {
		panic(errors.New("obj must be a pointer"))
	}

	resolver.MustAutoWire(obj)
	return obj
}
