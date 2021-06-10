package infra

import (
	"context"
	"net"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/graceful"
)

const (
	VersionKey     string = "version"
	StartupTimeKey string = "startup_time"
)

// Service is a interface for service
type Service interface {
	// Init initialize the service
	Init(resolver Resolver) error
	// Name return service name
	Name() string
	// Start start service, not blocking
	Start() error
	// Stop stop the service
	Stop()
	// Reload reload service
	Reload()
}

// ModuleLoadPolicy 实现该接口用于判断当前模块（Service/Provider/DaemonProvider）是否加载
type ModuleLoadPolicy interface {
	// ShouldLoad 如果返回 true，则加载该模块，否则跳过
	ShouldLoad(c FlagContext) bool
}

type Provider interface {
	// Register add some dependency for current module
	// this method is called one by one synchronous
	// service provider don't autowired in this stage
	Register(app Binder)
	// Boot start the module
	// this method is called one by one synchronous after all register methods called
	// service provider has been autowired in this stage
	Boot(app Resolver)
}

type DaemonProvider interface {
	Provider
	// Daemon is a async method called after boot
	// this method is called asynchronous and concurrent
	Daemon(ctx context.Context, app Resolver)
}

// ProviderAggregate Provider 聚合，所有实现该接口的 Provider 在加载之前将会先加载该集合中的 Provider
type ProviderAggregate interface {
	Aggregates() []Provider
}

type ListenerBuilder interface {
	Build(cc Resolver) (net.Listener, error)
}

type FlagContext interface {
	String(name string) string
	GlobalString(name string) string
	StringSlice(name string) []string
	GlobalStringSlice(name string) []string

	Bool(name string) bool
	GlobalBool(name string) bool
	BoolT(name string) bool
	GlobalBoolT(name string) bool

	Int64(name string) int64
	GlobalInt64(name string) int64
	Int(name string) int
	GlobalInt(name string) int
	IntSlice(name string) []int
	GlobalIntSlice(name string) []int
	Uint64(name string) uint64
	GlobalUint64(name string) uint64
	Uint(name string) uint
	GlobalUint(name string) uint
	Int64Slice(name string) []int64
	GlobalInt64Slice(name string) []int64

	Duration(name string) time.Duration
	GlobalDuration(name string) time.Duration

	Float64(name string) float64
	GlobalFloat64(name string) float64

	Generic(name string) interface{}
	GlobalGeneric(name string) interface{}

	FlagNames() (names []string)
	GlobalFlagNames() (names []string)
}

type Glacier interface {
	// Provider 注册一个模块
	Provider(providers ...Provider)
	// Service 注册一个 Service
	Service(services ...Service)

	// Graceful 设置优雅停机实现
	Graceful(builder func() graceful.Graceful) Glacier

	// OnServerReady call a function a server ready
	OnServerReady(f interface{})

	Handler() func(cliContext FlagContext) error
	// BeforeInitialize Glacier 初始化之前执行，一般用于设置一些基本配置，比如日志等
	BeforeInitialize(f func(c FlagContext) error) Glacier

	// BeforeServerStart 此时所有对象已经注册完毕，但是服务启动前执行
	BeforeServerStart(f func(cc container.Container) error) Glacier
	// AfterServerStart 此时所有服务都已经启动（Main 除外）
	AfterServerStart(f func(cc Resolver) error) Glacier
	// BeforeServerStop 服务停止前的回调
	BeforeServerStop(f func(cc Resolver) error) Glacier
	// AfterProviderBooted 所有的 providers 都已经完成 boot 之后执行
	AfterProviderBooted(f interface{}) Glacier

	Logger(logger log.Logger) Glacier
	Singleton(ins ...interface{}) Glacier
	Prototype(ins ...interface{}) Glacier
	ResolveWithError(resolver interface{}) error
	MustResolve(resolver interface{})
	Container() container.Container
	// Main 函数，在 App 启动的最后执行该函数
	Main(f interface{}) Glacier
}

type Binder container.Binder
type Resolver container.Resolver

type Hook interface {
	// OnServerReady call a function a server ready
	OnServerReady(f interface{})
}
