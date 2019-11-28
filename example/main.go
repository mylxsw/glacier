package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gorilla/mux"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/cron"
	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/example/api"
	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/example/job"
	"github.com/mylxsw/glacier/example/service"
	"github.com/mylxsw/glacier/web"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

var Version string
var GitCommit string

type CronEvent struct{}

func main() {
	g := glacier.Create(fmt.Sprintf("%s (%s)", Version, GitCommit[:8]))
	g.WithHttpServer(":19945")
	g.AddFlags(altsrc.NewStringFlag(cli.StringFlag{
		Name:  "test",
		Value: "",
	}))

	g.WebAppExceptionHandler(func(ctx web.Context, err interface{}) web.Response {
		log.Errorf("stack: %s", debug.Stack())
		return nil
	})

	g.Provider(job.ServiceProvider{})
	g.Provider(api.ServiceProvider{})

	g.Service(&service.DemoService{})
	g.Service(&service.Demo2Service{})

	g.Cron(func(cr cron.Manager, cc *container.Container) error {
		if err := cr.Add("hello", "@every 15s", func(manager event.Manager) {
			log.Infof("hello, example!")
			manager.Publish(CronEvent{})
		}); err != nil {
			return err
		}

		return nil
	})

	g.EventListener(func(listener event.Manager, cc *container.Container) {
		listener.Listen(func(event CronEvent) {
			log.Debug("a new cron task executed")
		})
	})

	g.Singleton(func(c *cli.Context) *config.Config {
		return &config.Config{
			DB:   "xxxxxx",
			Test: c.String("test"),
		}
	})

	g.Main(func(conf *config.Config, router *mux.Router) {
		log.Debugf("config: %s", conf.Serialize())
		for _, r := range web.GetAllRoutes(router) {
			log.Debugf("route: %s -> %s | %s | %s", r.Name, r.Methods, r.PathTemplate, r.PathRegexp)
		}
	})

	if err := g.Run(os.Args); err != nil {
		panic(err)
	}
}
