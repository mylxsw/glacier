package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	asteriaEvent "github.com/mylxsw/asteria/event"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/mylxsw/glacier/cron"
	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/example/api"
	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/example/job"
	"github.com/mylxsw/glacier/example/service"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/listener"
	"github.com/mylxsw/glacier/starter/application"
	"github.com/mylxsw/glacier/web"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

var Version = "1.0"
var GitCommit = "aabbccddeeffgghhiijjkk"

type CronEvent struct {
	GoroutineID uint64
}

func (c CronEvent) Async() bool {
	return true
}

func main() {
	//log.All().LogFormatter(formatter.NewJSONFormatter())

	//log.DefaultLogLevel(level.Error)
	log.DefaultDynamicModuleName(true)
	log.AddGlobalFilter(func(filter log.Filter) log.Filter {
		return func(f asteriaEvent.Event) {
			if strings.HasPrefix(f.Module, "github.com.mylxsw.glacier.cron") {
				return
			}

			filter(f)
		}
	})

	app := application.Create(fmt.Sprintf("%s (%s)", Version, GitCommit[:8]))

	app.AddFlags(altsrc.NewStringFlag(cli.StringFlag{
		Name:  "listen",
		Usage: "http listen addr",
		Value: ":19945",
	}))

	// 设置该选项之后，路由匹配时将会忽略最末尾的 /
	// 路由 /aaa/bbb  匹配 /aaa/bbb, /aaa/bbb/
	// 路由 /aaa/bbb/ 匹配 /aaa/bbb, /aaa/bbb/
	// 默认为 false，匹配规则如下
	// 路由 /aaa/bbb 只匹配 /aaa/bbb 不匹配 /aaa/bbb/
	// 路由 /aaa/bbb/ 只匹配 /aaa/bbb/ 不匹配 /aaa/bbb
	app.WithHttpServer(listener.FlagContext("listen"), infra.SetIgnoreLastSlashOption(true))

	app.WebAppExceptionHandler(func(ctx web.Context, err interface{}) web.Response {
		log.Errorf("stack: %s", debug.Stack())
		return nil
	})

	app.Provider(job.ServiceProvider{})
	app.Provider(api.ServiceProvider{})

	app.Service(&service.DemoService{})
	app.Service(&service.Demo2Service{})

	app.Cron(func(cr cron.Manager, cc container.Container) error {
		if err := cr.Add("hello", "@every 15s", func(manager event.Manager) {
			log.Infof("hello, example!")
			manager.Publish(CronEvent{GoroutineID: getGID()})
		}); err != nil {
			return err
		}

		return nil
	})

	app.EventListener(func(listener event.Manager, cc container.Container) {
		listener.Listen(func(event CronEvent) {
			if log.DebugEnabled() {
				log.Debug("a new cron task executed")
			}

			log.Infof("event processed, listener_goroutine_id=%d, publisher_goroutine_id=%d", getGID(), event.GoroutineID)
		})
	})

	app.Singleton(func(c infra.FlagContext) *config.Config {
		return &config.Config{
			DB:   "xxxxxx",
			Test: c.String("test"),
		}
	})

	app.Main(func(conf *config.Config, router *mux.Router) {
		if log.DebugEnabled() {
			log.Debugf("config: %s", conf.Serialize())
			for _, r := range web.GetAllRoutes(router) {
				log.Debugf("route: %s -> %s | %s | %s", r.Name, r.Methods, r.PathTemplate, r.PathRegexp)
			}
		}
	})

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
