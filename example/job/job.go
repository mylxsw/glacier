package job

import (
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/glacier/scheduler"
)

type ServiceProvider struct{}

func (j ServiceProvider) Aggregates() []infra.Provider {
	return []infra.Provider{
		scheduler.Provider(
			func(cc infra.Resolver, creator scheduler.JobCreator) {
				//_ = cr.Add("test-job", "@every 10s", TestJob)
				_ = creator.AddAndRunOnServerReady("test-timeout-job", "@every 5s", scheduler.WithoutOverlap(TestTimeoutJob).SkipCallback(func() {
					log.Errorf("[example] test-timeout-job skipped")
				}))
			},
			scheduler.SetLockManagerOption(func(_ infra.Resolver) scheduler.LockManagerBuilder {
				return NewDistributeLockManager
			}),
		),
	}
}

func (j ServiceProvider) ShouldLoad(c infra.FlagContext) bool {
	log.Debugf("[example] call ShouldLoad for job.ServiceProvider")
	return c.Bool("load-job")
}

func (j ServiceProvider) Register(cc infra.Binder) {
	log.Debug("[example] provider job.ServiceProvider loaded")
}

func (j ServiceProvider) Boot(app infra.Resolver) {
	app.MustResolve(func(sche scheduler.Scheduler) {
		job, _ := sche.Info("test-job")
		nextTs, _ := job.Next(5)
		for i, nt := range nextTs {
			log.Debugf("[example] job test-job next %d ---> %s", i, nt)
		}
	})
}

type DistributeLockManager struct {
	name string
}

func NewDistributeLockManager(name string) scheduler.LockManager {
	return &DistributeLockManager{name: name}
}

func (manager *DistributeLockManager) TryLock() error {
	log.Debugf("[example] try lock for %s ...", manager.name)
	return nil
}

func (manager *DistributeLockManager) Release() error {
	log.Debugf("[example] try release lock for %s ...", manager.name)
	return nil
}
