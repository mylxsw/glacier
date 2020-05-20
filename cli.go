package glacier

import (
	"time"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/web"
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
	Provider(provider ServiceProvider)
	Service(service Service)
	WithHttpServer(listenAddr string) Glacier
	WebAppInit(initFunc interface{}) Glacier
	WebAppServerInit(handler InitServerHandler) Glacier
	WebAppRouter(handler InitRouterHandler) Glacier
	WebAppMuxRouter(handler InitMuxRouterHandler) Glacier
	WebAppExceptionHandler(handler web.ExceptionHandler) Glacier
	HttpListenAddr() string
	Handler() func(cliContext FlagContext) error
	BeforeInitialize(f func(c FlagContext) error) Glacier
	BeforeServerStart(f func(cc container.Container) error) Glacier
	AfterServerStart(f func(cc container.Container) error) Glacier
	BeforeServerStop(f func(cc container.Container) error) Glacier
	Cron(f CronTaskFunc) Glacier
	EventListener(f EventListenerFunc) Glacier
	Singleton(ins interface{}) Glacier
	Prototype(ins interface{}) Glacier
	ResolveWithError(resolver interface{}) error
	MustResolve(resolver interface{})
	Container() container.Container
	Main(f interface{}) Glacier
}
