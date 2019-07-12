package main

import (
	"os"
	"time"

	"github.com/mylxsw/glacier"
	"github.com/mylxsw/go-toolkit/container"
	"github.com/mylxsw/go-toolkit/events"
	"github.com/mylxsw/go-toolkit/log"
	"github.com/mylxsw/go-toolkit/period_job"
	"github.com/robfig/cron"
)

var logger = log.Module("example")

type testJob struct{}

func (testJob) Handle() {
	logger.Info("Hello, test job!")

	glacier.Container().MustResolve(func(conf *Config) {
		logger.Infof("mysql_conn: %s", conf.MySQLURI)
	})
}

type CrontabEvent struct{}

type Config struct {
	MySQLURI string
}

func main() {
	g := glacier.Create("1.0")

	g.WithHttpServer(":19945")

	g.PeriodJob(func(pj *period_job.Manager, cc *container.Container) {
		pj.Run("test-job", testJob{}, 5*time.Second)

		for _, k := range cc.Keys() {
			logger.Debugf("-> %v", k)
		}
	})

	g.Crontab(func(cr *cron.Cron, cc *container.Container) error {
		if err := cr.AddFunc("@every 3s", func() {
			logger.Infof("hello, example!")

			_ = cc.Resolve(func(manager *events.EventManager) {
				manager.Publish(CrontabEvent{})
			})
		}); err != nil {
			return err
		}

		return nil
	})

	g.EventListener(func(listener *events.EventManager, cc *container.Container) {
		listener.Listen(func(event CrontabEvent) {
			logger.Debug("a new cron task executed")
		})
	})

	g.Singleton(func() *Config {
		return &Config{
			MySQLURI: "xxxxxx",
		}
	})


	g.Main(func(conf *Config) {
		logger.Errorf("main: %s", conf.MySQLURI)
	})

	if err := g.Run(os.Args); err != nil {
		panic(err)
	}
}
