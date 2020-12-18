package application

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/web"
	"github.com/mylxsw/graceful"
)

func (application *Application) Provider(provider infra.ServiceProvider) {
	application.glacier.Provider(provider)
}

func (application *Application) Service(service infra.Service) {
	application.glacier.Service(service)
}

func (application *Application) Graceful(builder func() graceful.Graceful) infra.Glacier {
	return application.glacier.Graceful(builder)
}

func (application *Application) WithHttpServer(builder infra.ListenerBuilder, options ...infra.WebServerOption) infra.Glacier {
	return application.glacier.WithHttpServer(builder, options...)
}

func (application *Application) WebAppInit(initFunc infra.InitWebAppHandler) infra.Glacier {
	return application.glacier.WebAppInit(initFunc)
}

func (application *Application) WebAppServerInit(handler infra.InitServerHandler) infra.Glacier {
	return application.glacier.WebAppServerInit(handler)
}

func (application *Application) WebAppRouter(handler infra.InitRouterHandler) infra.Glacier {
	return application.glacier.WebAppRouter(handler)
}

func (application *Application) WebAppMuxRouter(handler infra.InitMuxRouterHandler) infra.Glacier {
	return application.glacier.WebAppMuxRouter(handler)
}

func (application *Application) WebAppExceptionHandler(handler web.ExceptionHandler) infra.Glacier {
	return application.glacier.WebAppExceptionHandler(handler)
}

func (application *Application) HttpListenAddr() string {
	return application.glacier.HttpListenAddr()
}

func (application *Application) Handler() func(cliContext infra.FlagContext) error {
	return application.glacier.Handler()
}

func (application *Application) BeforeInitialize(f func(c infra.FlagContext) error) infra.Glacier {
	return application.glacier.BeforeInitialize(f)
}

func (application *Application) BeforeServerStart(f func(cc container.Container) error) infra.Glacier {
	return application.glacier.BeforeServerStart(f)
}

func (application *Application) AfterServerStart(f func(cc container.Container) error) infra.Glacier {
	return application.glacier.AfterServerStart(f)
}

func (application *Application) BeforeServerStop(f func(cc container.Container) error) infra.Glacier {
	return application.glacier.BeforeServerStop(f)
}

func (application *Application) Cron(f infra.CronTaskFunc) infra.Glacier {
	return application.glacier.Cron(f)
}

func (application *Application) EventListener(f infra.EventListenerFunc) infra.Glacier {
	return application.glacier.EventListener(f)
}

func (application *Application) Singleton(ins interface{}) infra.Glacier {
	return application.glacier.Singleton(ins)
}

func (application *Application) Prototype(ins interface{}) infra.Glacier {
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

func (application *Application) Main(f interface{}) infra.Glacier {
	return application.glacier.Main(f)
}

func (application *Application) Logger(logger log.Logger) infra.Glacier {
	return application.glacier.Logger(logger)
}
