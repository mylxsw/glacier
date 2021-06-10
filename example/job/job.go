package job

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/scheduler"
)

type ServiceProvider struct{}

func (j ServiceProvider) Aggregates() []infra.Provider {
	return []infra.Provider{
		scheduler.Provider(
			func(cc infra.Resolver, creator scheduler.JobCreator) {
				//_ = cr.Add("test-job", "@every 10s", TestJob)
				_ = creator.Add("test-timeout-job", "@every 5s", scheduler.WithoutOverlap(TestTimeoutJob).SkipCallback(func() {
					log.Errorf("test-timeout-job skipped")
				}))
				cc.Resolve(func(hook infra.Hook) {
					hook.OnReady(func() {
						log.Debugf("on ready hook called")
					})
				})
			},
			scheduler.SetDistributeLockManagerOption(func(cc infra.Resolver) scheduler.DistributeLockManager {
				return NewDistributeLockManager()
			}),
		),
	}
}

func (j ServiceProvider) ShouldLoad(c infra.FlagContext) bool {
	return c.Bool("load-job")
}

func (j ServiceProvider) Register(cc infra.Binder) {
}

func (j ServiceProvider) Boot(app infra.Resolver) {
	app.MustResolve(func(sche scheduler.Scheduler) {
		if log.DebugEnabled() {
			job, _ := sche.Info("test-job")
			nextTs, _ := job.Next(5)
			for i, nt := range nextTs {
				log.Debugf("job test-job next %d ---> %s", i, nt)
			}
		}
	})
}

type DistributeLockManager struct {
}

func NewDistributeLockManager() *DistributeLockManager {
	return &DistributeLockManager{}
}

func (manager *DistributeLockManager) TryLock() error {
	if log.DebugEnabled() {
		log.Debug("try lock ...")
	}
	return nil
}

func (manager *DistributeLockManager) TryUnLock() error {
	if log.DebugEnabled() {
		log.Debug("try unlock ...")
	}
	return nil
}

func (manager *DistributeLockManager) HasLock() bool {
	return true
}
