package application

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/graceful"
)

func (application *Application) Provider(providers ...infra.Provider) {
	application.glacier.Provider(providers...)
}

func (application *Application) Service(services ...infra.Service) {
	application.glacier.Service(services...)
}

func (application *Application) Graceful(builder func() graceful.Graceful) infra.Glacier {
	return application.glacier.Graceful(builder)
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

func (application *Application) AfterServerStart(f func(cc infra.Resolver) error) infra.Glacier {
	return application.glacier.AfterServerStart(f)
}

func (application *Application) BeforeServerStop(f func(cc infra.Resolver) error) infra.Glacier {
	return application.glacier.BeforeServerStop(f)
}

func (application *Application) AfterProviderBooted(f interface{}) infra.Glacier {
	return application.glacier.AfterProviderBooted(f)
}

func (application *Application) Singleton(ins ...interface{}) infra.Glacier {
	return application.glacier.Singleton(ins...)
}

func (application *Application) Prototype(ins ...interface{}) infra.Glacier {
	return application.glacier.Prototype(ins...)
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
