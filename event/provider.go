package event

import (
	"context"

	"github.com/mylxsw/glacier/infra"
)

type provider struct {
	evtStoreBuilder func(cc infra.Resolver) Store
	handler         func(cc infra.Resolver, listener Listener)
}

func (p *provider) Priority() int {
	return 10
}

// Provider create a event Provider
func Provider(handler func(resolver infra.Resolver, listener Listener), options ...Option) infra.Provider {
	p := &provider{handler: handler}
	for _, opt := range options {
		opt(p)
	}

	return p
}

func (p *provider) Register(app infra.Binder) {
	app.MustSingletonOverride(func(cc infra.Resolver) Store {
		if p.evtStoreBuilder != nil {
			return p.evtStoreBuilder(cc)
		}

		return NewMemoryEventStore(false, 20)
	})
	app.MustSingletonOverride(NewEventManager)
	app.MustSingletonOverride(func(manager Manager) Listener { return manager })
	app.MustSingletonOverride(func(manager Manager) Publisher { return manager })
}

func (p *provider) Boot(app infra.Resolver) {
	app.MustResolve(p.handler)
}

func (p *provider) Daemon(ctx context.Context, app infra.Resolver) {
	app.MustResolve(func(manager Manager) {
		<-manager.Start(ctx)
	})
}

type Option func(p *provider)

// SetStoreOption 设置底层存储实现
func SetStoreOption(h func(cc infra.Resolver) Store) Option {
	return func(p *provider) {
		p.evtStoreBuilder = h
	}
}
