package job

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/cron"
	"github.com/mylxsw/glacier/infra"
)

type ServiceProvider struct{}

func (j ServiceProvider) Register(cc container.Container) {
}

func (j ServiceProvider) Boot(app infra.Glacier) {
	app.Cron(func(cr cron.Manager, cc container.Container) error {
		cr.DistributeLockManager(NewDistributeLockManager())

		//_ = cr.Add("test-job", "@every 10s", TestJob)
		_ = cr.Add("test-timeout-job", "@every 5s", cron.WithoutOverlap(TestTimeoutJob).SkipCallback(func() {
			log.Errorf("test-timeout-job skipped")
		}))

		if log.DebugEnabled() {
			job, _ := cr.Info("test-job")
			nextTs, _ := job.Next(5)
			for i, nt := range nextTs {
				log.Debugf("job test-job next %d ---> %s", i, nt)
			}
		}

		return nil
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
