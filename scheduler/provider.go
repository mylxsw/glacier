package scheduler

import (
	"context"

	"github.com/mylxsw/glacier/log"

	"github.com/mylxsw/glacier/infra"
	cronV3 "github.com/robfig/cron/v3"
)

type provider struct {
	creator func(cc infra.Resolver, creator JobCreator)
	options []Option
}

func (p *provider) Priority() int {
	return -1
}

func Provider(creator func(cc infra.Resolver, creator JobCreator), options ...Option) infra.DaemonProvider {
	return &provider{creator: creator, options: options}
}

func (p *provider) Register(app infra.Binder) {
	// 定时任务对象
	app.MustSingletonOverride(func() *cronV3.Cron {
		return cronV3.New(cronV3.WithSeconds(), cronV3.WithLogger(cronLogger{}))
	})
	app.MustSingletonOverride(func(resolver infra.Resolver) Scheduler {
		cr := NewManager(resolver)
		for _, opt := range p.options {
			opt(resolver, cr)
		}

		return cr
	})
	app.MustSingletonOverride(func(cr Scheduler) JobCreator { return cr })
}

func (p *provider) Boot(app infra.Resolver) {
	app.MustResolve(p.creator)
}

func (p *provider) Daemon(ctx context.Context, app infra.Resolver) {
	app.MustResolve(func(gf infra.Graceful, cr Scheduler, logger infra.Logger) {
		gf.AddShutdownHandler(cr.Stop)
		cr.Start()
		<-ctx.Done()
	})
}

type cronLogger struct {
}

func (l cronLogger) Info(msg string, keysAndValues ...interface{}) {
	// Just drop it, we don't care
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.Errorf("[glacier] %s: %v", msg, err)
}

// Option 定时任务配置型
type Option func(cc infra.Resolver, cr Scheduler)

// SetDistributeLockManagerOption 设置分布式锁管理器实现
func SetDistributeLockManagerOption(lockManager func(cc infra.Resolver) DistributeLockManager) Option {
	return func(cc infra.Resolver, cr Scheduler) {
		cr.DistributeLockManager(lockManager(cc))
	}
}
