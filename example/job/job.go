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
					log.Errorf("test-timeout-job skipped")
				}))
			},
			scheduler.SetDistributeLockManagerOption(func(cc infra.Resolver) scheduler.DistributeLockManager {
				return NewDistributeLockManager()
			}),
		),
	}
}

func (j ServiceProvider) ShouldLoad(c infra.FlagContext) bool {
	log.Debugf("call ShouldLoad for job.ServiceProvider")
	return c.Bool("load-job")
}

func (j ServiceProvider) Register(cc infra.Binder) {
	log.Debug("provider job.ServiceProvider loaded")
}

func (j ServiceProvider) Boot(app infra.Resolver) {
	app.MustResolve(func(sche scheduler.Scheduler) {
		job, _ := sche.Info("test-job")
		nextTs, _ := job.Next(5)
		for i, nt := range nextTs {
			log.Debugf("job test-job next %d ---> %s", i, nt)
		}
	})
}

type DistributeLockManager struct {
}

func NewDistributeLockManager() *DistributeLockManager {
	return &DistributeLockManager{}
}

func (manager *DistributeLockManager) TryLock() error {
	log.Debug("try lock ...")
	return nil
}

func (manager *DistributeLockManager) TryUnLock() error {
	log.Debug("try unlock ...")
	return nil
}

func (manager *DistributeLockManager) HasLock() bool {
	return true
}
