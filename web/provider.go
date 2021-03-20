package web

import (
	"context"

	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
)

type provider struct {
	options         []Option
	listenerBuilder infra.ListenerBuilder
}

func Provider(builder infra.ListenerBuilder, options ...Option) infra.DaemonProvider {
	return &provider{
		options:         options,
		listenerBuilder: builder,
	}
}

func (p *provider) Register(app infra.Binder) {
	app.MustSingletonOverride(func(cc container.Container) Server {
		return NewServer(cc, p.options...)
	})
	app.MustSingletonOverride(func() infra.ListenerBuilder {
		if p.listenerBuilder == nil {
			return listener.Default("127.0.0.1:8080")
		}

		return p.listenerBuilder
	})
}

func (p *provider) Boot(app infra.Resolver) {
}

func (p *provider) Daemon(ctx context.Context, app infra.Resolver) {
	app.MustResolve(func(server Server, listenerBuilder infra.ListenerBuilder) {
		l, err := listenerBuilder.Build(app)
		if err != nil {
			panic(err)
		}

		if err := server.Start(l); err != nil {
			panic(err)
		}
	})
}
