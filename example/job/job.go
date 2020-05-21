package job

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/cron"
)

type ServiceProvider struct{}

func (j ServiceProvider) Register(cc container.Container) {
}

func (j ServiceProvider) Boot(app glacier.Glacier) {
	app.Cron(func(cr cron.Manager, cc container.Container) error {
		cr.DistributeLockManager(NewDistributeLockManager())

		_ = cr.Add("test-job", "@every 10s", TestJob)

		job, _ := cr.Info("test-job")
		nextTs, _ := job.Next(5)
		for i, nt := range nextTs {
			log.Debugf("job test-job next %d ---> %s", i, nt)
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
	log.Debug("try lock ...")
	return nil
}

func (manager *DistributeLockManager) TryUnLock() error {
	log.Debug("try unlock ...")
	return nil
}

func (manager *DistributeLockManager) HasLock() bool {
	return false
}

