package web

import (
	"context"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
	"github.com/mylxsw/go-ioc"
)

type provider struct {
	options         []Option
	listenerBuilder infra.ListenerBuilder
	repeatable      bool
}

func (p *provider) Priority() int {
	return -1
}

func DefaultProvider(routeHandler RouteHandler, options ...Option) infra.DaemonProvider {
	return Provider(listener.FlagContext("listen"), append(options, SetRouteHandlerOption(routeHandler))...)
}

func DefaultProviderWithListenerBuilder(listenerBuilder infra.ListenerBuilder, routeHandler RouteHandler, options ...Option) infra.DaemonProvider {
	return Provider(listenerBuilder, append(options, SetRouteHandlerOption(routeHandler))...)
}

func RepeatableProvider(builder infra.ListenerBuilder, options ...Option) infra.DaemonProvider {
	return &provider{
		options:         options,
		listenerBuilder: builder,
		repeatable:      true,
	}
}

func Provider(builder infra.ListenerBuilder, options ...Option) infra.DaemonProvider {
	return &provider{
		options:         options,
		listenerBuilder: builder,
	}
}

func (p *provider) Register(app infra.Binder) {
	if p.repeatable {
		return
	}

	app.MustSingletonOverride(func(cc ioc.Container) Server {
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
	if p.repeatable {
		app.MustResolve(func(app ioc.Container) {
			listenerBuilder := p.listenerBuilder
			if listenerBuilder == nil {
				listenerBuilder = listener.Default("127.0.0.1:8080")
			}

			l, err := listenerBuilder.Build(app)
			if err != nil {
				panic(err)
			}

			if err := NewServer(app, p.options...).Start(l); err != nil {
				panic(err)
			}
		})
	} else {
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
}
