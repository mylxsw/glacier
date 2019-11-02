package job

import (
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/cron"
)

type ServiceProvider struct{}

func (j ServiceProvider) Register(cc *container.Container) {
}

func (j ServiceProvider) Boot(app *glacier.Glacier) {
	app.Cron(func(cr cron.Manager, cc *container.Container) error {
		_ = cr.Add("test-job", "@every 30s", TestJob)
		return nil
	})
}
