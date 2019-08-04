package job

import (
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/go-toolkit/container"
	"github.com/mylxsw/go-toolkit/period_job"
)

type ServiceProvider struct{}

func (j ServiceProvider) Register(cc *container.Container) {
	cc.MustSingleton(NewTestJob)
}

func (j ServiceProvider) Boot(app *glacier.Glacier) {
	app.PeriodJob(func(pj *period_job.Manager, cc *container.Container) {
		cc.MustResolve(func(testJob *TestJob) {
			for _, k := range cc.Keys() {
				log.Debugf("-> %v", k)
			}
			pj.Run("test-job", testJob, 5*time.Second)
		})
	})
}

