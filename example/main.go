package main

import (
	"os"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/example/job"
	"github.com/mylxsw/go-toolkit/container"
	"github.com/mylxsw/go-toolkit/events"
	"github.com/robfig/cron"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

type CrontabEvent struct{}

func main() {
	g := glacier.Create("1.0")
	g.WithHttpServer(":19945")
	g.AddFlags(altsrc.NewStringFlag(cli.StringFlag{
		Name:  "test",
		Value: "",
	}))

	g.Provider(job.ServiceProvider{})

	g.Crontab(func(cr *cron.Cron, cc *container.Container) error {
		if err := cr.AddFunc("@every 3s", func() {
			log.Infof("hello, example!")

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
			log.Debug("a new cron task executed")
		})
	})

	g.Singleton(func(c *cli.Context) *config.Config {
		return &config.Config{
			MySQLURI: "xxxxxx",
			Test:     c.String("test"),
		}
	})

	g.Main(func(conf *config.Config) {
		log.Errorf("main: %s", conf.Test)
	})

	if err := g.Run(os.Args); err != nil {
		panic(err)
	}
}
