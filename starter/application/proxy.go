package application

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/graceful"
)

func (application *Application) Provider(providers ...infra.Provider) *Application {
	application.glacier.Provider(providers...)
	return application
}

func (application *Application) Service(services ...infra.Service) *Application {
	application.glacier.Service(services...)
	return application
}

func (application *Application) Graceful(builder func() graceful.Graceful) *Application {
	application.glacier.Graceful(builder)
	return application
}

func (application *Application) Handler() func(cliContext infra.FlagContext) error {
	return application.glacier.Handler()
}

func (application *Application) BeforeInitialize(f func(c infra.FlagContext) error) *Application {
	application.glacier.BeforeInitialize(f)
	return application
}

func (application *Application) BeforeServerStart(f func(cc container.Container) error) *Application {
	application.glacier.BeforeServerStart(f)
	return application
}

func (application *Application) AfterServerStart(f func(cc infra.Resolver) error) *Application {
	application.glacier.AfterServerStart(f)
	return application
}

func (application *Application) BeforeServerStop(f func(cc infra.Resolver) error) *Application {
	application.glacier.BeforeServerStop(f)
	return application
}

func (application *Application) AfterProviderBooted(f interface{}) *Application {
	application.glacier.AfterProviderBooted(f)
	return application
}

func (application *Application) Singleton(ins ...interface{}) *Application {
	application.glacier.Singleton(ins...)
	return application
}

func (application *Application) Prototype(ins ...interface{}) *Application {
	application.glacier.Prototype(ins...)
	return application
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

func (application *Application) Main(f interface{}) *Application {
	application.glacier.Main(f)
	return application
}

func (application *Application) Logger(logger log.Logger) *Application {
	application.glacier.Logger(logger)
	return application
}
