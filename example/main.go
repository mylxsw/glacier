package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	asteriaEvent "github.com/mylxsw/asteria/event"
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/example/api"
	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/example/job"
	"github.com/mylxsw/glacier/example/service"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/starter/application"
	"github.com/mylxsw/graceful"
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
			if f.Level == level.Debug && strings.HasPrefix(f.Module, "github.com.mylxsw.glacier.cron") {
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
	app.AddFlags(altsrc.NewBoolFlag(cli.BoolFlag{Name: "load-job"}))
	app.AddFlags(altsrc.NewBoolFlag(cli.BoolFlag{Name: "load-demoservice"}))

	// 设置该选项之后，路由匹配时将会忽略最末尾的 /
	// 路由 /aaa/bbb  匹配 /aaa/bbb, /aaa/bbb/
	// 路由 /aaa/bbb/ 匹配 /aaa/bbb, /aaa/bbb/
	// 默认为 false，匹配规则如下
	// 路由 /aaa/bbb 只匹配 /aaa/bbb 不匹配 /aaa/bbb/
	// 路由 /aaa/bbb/ 只匹配 /aaa/bbb/ 不匹配 /aaa/bbb
	app.Provider(job.ServiceProvider{}, api.ServiceProvider{})
	app.Service(&service.DemoService{}, &service.Demo2Service{})

	app.Provider(event.Provider(
		func(cc infra.Resolver, listener event.Listener) {
			listener.Listen(func(event CronEvent) {
				if log.DebugEnabled() {
					log.Debug("a new cron task executed")
				}

				log.Infof("event processed, listener_goroutine_id=%d, publisher_goroutine_id=%d", getGID(), event.GoroutineID)
			})
		},
		event.SetStoreOption(func(cc infra.Resolver) event.Store {
			return event.NewMemoryEventStore(true, 100)
		}),
	))

	app.Singleton(func(c infra.FlagContext) *config.Config {
		return &config.Config{
			DB:   "xxxxxx",
			Test: c.String("test"),
		}
	})

	app.Main(func(conf *config.Config, publisher event.Publisher, gf graceful.Graceful) {
		if log.DebugEnabled() {
			log.Debugf("config: %s", conf.Serialize())
		}

		for i := 0; i < 10; i++ {
			publisher.Publish(CronEvent{GoroutineID: uint64(i)})
		}

		// 5s 后自动关闭服务
		time.AfterFunc(5*time.Second, gf.Shutdown)
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
