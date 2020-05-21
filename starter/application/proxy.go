package application

import (
	"net"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/web"
)

func (application *Application) Provider(provider glacier.ServiceProvider) {
	application.glacier.Provider(provider)
}

func (application *Application) Service(service glacier.Service) {
	application.glacier.Service(service)
}

func (application *Application) TCPListener(ln net.Listener) glacier.Glacier {
	return application.glacier.TCPListener(ln)
}

func (application *Application) TCPListenerAddr(addr string) glacier.Glacier {
	return application.glacier.TCPListenerAddr(addr)
}

func (application *Application) WithHttpServer() glacier.Glacier {
	return application.glacier.WithHttpServer()
}

func (application *Application) WebAppInit(initFunc glacier.InitWebAppHandler) glacier.Glacier {
	return application.glacier.WebAppInit(initFunc)
}

func (application *Application) WebAppServerInit(handler glacier.InitServerHandler) glacier.Glacier {
	return application.glacier.WebAppServerInit(handler)
}

func (application *Application) WebAppRouter(handler glacier.InitRouterHandler) glacier.Glacier {
	return application.glacier.WebAppRouter(handler)
}

func (application *Application) WebAppMuxRouter(handler glacier.InitMuxRouterHandler) glacier.Glacier {
	return application.glacier.WebAppMuxRouter(handler)
}

func (application *Application) WebAppExceptionHandler(handler web.ExceptionHandler) glacier.Glacier {
	return application.glacier.WebAppExceptionHandler(handler)
}

func (application *Application) HttpListenAddr() string {
	return application.glacier.HttpListenAddr()
}

func (application *Application) Handler() func(cliContext glacier.FlagContext) error {
	return application.glacier.Handler()
}

func (application *Application) BeforeInitialize(f func(c glacier.FlagContext) error) glacier.Glacier {
	return application.glacier.BeforeInitialize(f)
}

func (application *Application) BeforeServerStart(f func(cc container.Container) error) glacier.Glacier {
	return application.glacier.BeforeServerStart(f)
}

func (application *Application) AfterServerStart(f func(cc container.Container) error) glacier.Glacier {
	return application.glacier.AfterServerStart(f)
}

func (application *Application) BeforeServerStop(f func(cc container.Container) error) glacier.Glacier {
	return application.glacier.BeforeServerStop(f)
}

func (application *Application) Cron(f glacier.CronTaskFunc) glacier.Glacier {
	return application.glacier.Cron(f)
}

func (application *Application) EventListener(f glacier.EventListenerFunc) glacier.Glacier {
	return application.glacier.EventListener(f)
}

func (application *Application) Singleton(ins interface{}) glacier.Glacier {
	return application.glacier.Singleton(ins)
}

func (application *Application) Prototype(ins interface{}) glacier.Glacier {
	return application.glacier.Prototype(ins)
}

func (application *Application) ResolveWithError(resolver interface{}) error {
	return application.glacier.ResolveWithError(resolver)
}

func (application *Application) MustResolve(resolver interface{}) {
	application.glacier.MustResolve(resolver)
}

func (application *Application) Container() container.Container {
	return application.glacier.Container()
}

func (application *Application) Main(f interface{}) glacier.Glacier {
	return application.glacier.Main(f)
}

