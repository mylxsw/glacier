package application

import (
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
)

func (app *Application) PreBind(fn func(binder infra.Binder)) *Application {
	app.gcr.PreBind(fn)
	return app
}

func (app *Application) Provider(providers ...infra.Provider) *Application {
	app.gcr.Provider(providers...)
	return app
}

func (app *Application) Service(services ...infra.Service) *Application {
	app.gcr.Service(services...)
	return app
}

func (app *Application) Async(asyncJobs ...interface{}) *Application {
	app.gcr.Async(asyncJobs...)
	return app
}

func (app *Application) Graceful(builder func() infra.Graceful) *Application {
	app.gcr.Graceful(builder)
	return app
}

func (app *Application) Main(cliCtx infra.FlagContext) error {
	return app.gcr.Main(cliCtx)
}

func (app *Application) BeforeInitialize(f func(c infra.FlagContext) error) *Application {
	app.gcr.BeforeInitialize(f)
	return app
}

func (app *Application) AfterInitialized(f func(resolver infra.Resolver) error) *Application {
	app.gcr.AfterInitialized(f)
	return app
}

func (app *Application) BeforeServerStart(f func(cc container.Container) error) *Application {
	app.gcr.BeforeServerStart(f)
	return app
}

func (app *Application) AfterServerStart(f func(cc infra.Resolver) error) *Application {
	app.gcr.AfterServerStart(f)
	return app
}

func (app *Application) BeforeServerStop(f func(cc infra.Resolver) error) *Application {
	app.gcr.BeforeServerStop(f)
	return app
}

func (app *Application) AfterProviderBooted(f interface{}) *Application {
	app.gcr.AfterProviderBooted(f)
	return app
}

func (app *Application) Singleton(ins ...interface{}) *Application {
	app.gcr.Singleton(ins...)
	return app
}

func (app *Application) Prototype(ins ...interface{}) *Application {
	app.gcr.Prototype(ins...)
	return app
}

func (app *Application) ResolveWithError(resolver interface{}) error {
	return app.gcr.ResolveWithError(resolver)
}

func (app *Application) MustResolve(resolver interface{}) {
	app.gcr.MustResolve(resolver)
}

func (app *Application) Container() container.Container {
	return app.gcr.Container()
}
