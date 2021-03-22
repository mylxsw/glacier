package scheduler

import (
	"context"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/graceful"
	cronV3 "github.com/robfig/cron/v3"
)

type provider struct {
	creator func(cc infra.Resolver, creator JobCreator)
	options []Option
}

func Provider(creator func(cc infra.Resolver, creator JobCreator), options ...Option) infra.DaemonProvider {
	return &provider{creator: creator, options: options}
}

func (p *provider) Register(app infra.Binder) {
	// 定时任务对象
	app.MustSingletonOverride(func(logger log.Logger) *cronV3.Cron {
		return cronV3.New(cronV3.WithSeconds(), cronV3.WithLogger(cronLogger{logger: logger}))
	})
	app.MustSingletonOverride(func(cc container.Container, logger log.Logger) Scheduler {
		cr := NewManager(cc, logger)
		for _, opt := range p.options {
			opt(cc, cr)
		}

		return cr
	})
	app.MustSingletonOverride(func(cr Scheduler) JobCreator { return cr })
}

func (p *provider) Boot(app infra.Resolver) {
	app.MustResolve(p.creator)
}

func (p *provider) Daemon(ctx context.Context, app infra.Resolver) {
	app.MustResolve(func(gf graceful.Graceful, cr Scheduler, logger log.Logger) {
		gf.AddShutdownHandler(cr.Stop)
		cr.Start()

		if logger.DebugEnabled() {
			logger.Debugf("cron task server has been started")
		}

		<-ctx.Done()
	})
}

type cronLogger struct {
	logger log.Logger
}

func (l cronLogger) Info(msg string, keysAndValues ...interface{}) {
	// Just drop it, we don't care
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.logger.WithFields(log.Fields{
		"arguments": keysAndValues,
	}).Errorf("%s: %v", msg, err)
}

// Option 定时任务配置型
type Option func(cc infra.Resolver, cr Scheduler)

// SetDistributeLockManagerOption 设置分布式锁管理器实现
func SetDistributeLockManagerOption(lockManager func(cc infra.Resolver) DistributeLockManager) Option {
	return func(cc infra.Resolver, cr Scheduler) {
		cr.DistributeLockManager(lockManager(cc))
	}
}
