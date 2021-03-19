package event

import (
	"context"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
)

type provider struct {
	evtStore Store
	handler  func(cc container.Container, listener Listener)
}

// Provider create a event Provider
func Provider(handler func(cc container.Container, listener Listener), options ...Option) infra.Provider {
	p := &provider{handler: handler}
	for _, opt := range options {
		opt(p)
	}

	return p
}

func (p *provider) Register(app container.Container) {
	app.MustSingletonOverride(func() Store {
		if p.evtStore != nil {
			return p.evtStore
		}

		return NewMemoryEventStore(false, 20)
	})
	app.MustSingletonOverride(NewEventManager)
	app.MustSingletonOverride(func(manager Manager) Listener { return manager })
	app.MustSingletonOverride(func(manager Manager) Publisher { return manager })
}

func (p *provider) Boot(app infra.Glacier) {
	app.MustResolve(p.handler)
}

func (p *provider) Daemon(ctx context.Context, app infra.Glacier) {
	app.MustResolve(func(manager Manager) {
		<-manager.Start(ctx)
	})
}

type Option func(p *provider)

// SetStoreOption 设置底层存储实现
func SetStoreOption(store Store) Option {
	return func(p *provider) {
		p.evtStore = store
	}
}
