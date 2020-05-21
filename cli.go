package glacier

import (
	"net"
	"time"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/graceful"
)

const (
	VersionKey     string = "version"
	StartupTimeKey string = "startup_time"
)

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
	Provider(provider ServiceProvider)
	// Service 注册一个 Service
	Service(service Service)

	// TCPListener 设置 net.Listener 对象，优先顺序为 TCPListener > TCPListenerAddr > Config.HttpListen
	TCPListener(listenerBuilder func() net.Listener) Glacier
	// WithListenerAddr 设置 Http 服务监听地址，优先顺序为 TCPListener > TCPListenerAddr > Config.HttpListen
	TCPListenerAddr(addr string) Glacier

	// WithHttpServer 初始化 Http Server
	WithHttpServer() Glacier
	// WebAppInit web app 初始化阶段，web 应用对象还没有创建，在这里可以更新 web 配置
	WebAppInit(initFunc InitWebAppHandler) Glacier
	// WebAppServerInit web 服务初始化阶段，web 服务对象已经创建，此时不能再更新 web 配置了
	// 此时 web 服务还没有启动，可以通过 handler 修改 server 对象和 tcpListenerBuilder 对象
	WebAppServerInit(handler InitServerHandler) Glacier
	// WebAppRouter 路由注册 Handler，在该 Handler 中注册 API 路由规则
	WebAppRouter(handler InitRouterHandler) Glacier
	// WebAppMuxRouter 路由注册 Handler，该方法获取到的是底层的 Gorilla Mux 对象
	// 一般用来注册静态资源路由
	// router.PathPrefix("/dist/").Handler(http.StripPrefix("/dist/", http.FileServer(FS(false)))).Name("assets")
	WebAppMuxRouter(handler InitMuxRouterHandler) Glacier
	// WebAppExceptionHandler 设置 Web APP 异常处理器
	WebAppExceptionHandler(handler web.ExceptionHandler) Glacier
	// HttpListenAddr 返回 HTTP 监听地址
	HttpListenAddr() string

	// Graceful 设置优雅停机实现
	Graceful(builder func() graceful.Graceful) Glacier

	Handler() func(cliContext FlagContext) error
	// BeforeInitialize Glacier 初始化之前执行，一般用于设置一些基本配置，比如日志等
	BeforeInitialize(f func(c FlagContext) error) Glacier

	// BeforeServerStart 此时所有对象已经注册完毕，但是服务启动前执行
	BeforeServerStart(f func(cc container.Container) error) Glacier
	// AfterServerStart 此时所有服务都已经启动（Main 除外）
	AfterServerStart(f func(cc container.Container) error) Glacier
	// BeforeServerStop 服务停止前的回调
	BeforeServerStop(f func(cc container.Container) error) Glacier

	// 设置定时任务
	Cron(f CronTaskFunc) Glacier
	EventListener(f EventListenerFunc) Glacier
	Singleton(ins interface{}) Glacier
	Prototype(ins interface{}) Glacier
	ResolveWithError(resolver interface{}) error
	MustResolve(resolver interface{})
	Container() container.Container
	// Main 函数，在 App 启动的最后执行该函数
	Main(f interface{}) Glacier
}
