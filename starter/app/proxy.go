package app

import (
	"github.com/mylxsw/glacier/infra"
)

func (app *App) PreBind(fn func(binder infra.Binder)) *App {
	app.gcr.PreBind(fn)
	return app
}

func (app *App) Provider(providers ...infra.Provider) *App {
	app.gcr.Provider(providers...)
	return app
}

func (app *App) Service(services ...infra.Service) *App {
	app.gcr.Service(services...)
	return app
}

func (app *App) Async(asyncJobs ...interface{}) *App {
	app.gcr.Async(asyncJobs...)
	return app
}

func (app *App) Graceful(builder func() infra.Graceful) *App {
	app.gcr.Graceful(builder)
	return app
}

func (app *App) Start(cliCtx infra.FlagContext) error {
	return app.gcr.Start(cliCtx)
}

func (app *App) Init(f func(c infra.FlagContext) error) *App {
	app.gcr.Init(f)
	return app
}

func (app *App) OnServerReady(ffs ...interface{}) {
	app.gcr.OnServerReady(ffs...)
}

func (app *App) BeforeServerStop(f func(cc infra.Resolver) error) *App {
	app.gcr.BeforeServerStop(f)
	return app
}

func (app *App) Singleton(ins ...interface{}) *App {
	app.gcr.Singleton(ins...)
	return app
}

func (app *App) Prototype(ins ...interface{}) *App {
	app.gcr.Prototype(ins...)
	return app
}

func (app *App) Resolve(resolver interface{}) error {
	return app.gcr.Resolve(resolver)
}

func (app *App) MustResolve(resolver interface{}) {
	app.gcr.MustResolve(resolver)
}

func (app *App) Container() infra.Container {
	return app.gcr.Container()
}

func (app *App) Resolver() infra.Resolver {
	return app.gcr.Container()
}
func (app *App) Binder() infra.Binder {
	return app.gcr.Container()
}
