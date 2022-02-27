package main

import (
	"bytes"
	"github.com/mylxsw/glacier/log"
	"runtime"
	"strconv"
	"time"

	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/example/api"
	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/example/job"
	"github.com/mylxsw/glacier/example/service"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/starter/application"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
)

type CronEvent struct {
	GoroutineID uint64
}

func (c CronEvent) Async() bool {
	return true
}

func main() {
	//log.All().LogFormatter(formatter.NewJSONFormatter())

	//log.DefaultLogLevel(level.Error)
	//log.DefaultDynamicModuleName(true)
	//log.AddGlobalFilter(func(filter filter.Filter) filter.Filter {
	//	return func(f asteriaEvent.Event) {
	//		// 是否输出框架级别的debug日志
	//		//if f.Level == level.Debug && glacier.IsGlacierModuleLog(f.Module) {
	//		//	return
	//		//}
	//
	//		filter(f)
	//	}
	//})

	application.MustStart("1.0", run)
}

// run 后台持续运行的任务，除非手动触发退出，否则一直运行
func run(app *application.Application) error {
	app.WithDescription("Glacier 框架演示项目").
		WithName("glacier-example").
		WithUsage("这是一个示例项目").
		WithAuthor("管宜尧", "mylxsw@aicode.cc").
		WithCLIOptions(func(cliAPP *cli.App) {
			cliAPP.Copyright = "aicode.cc"
			cliAPP.UsageText = "这是 Usage Text"
		})

	app.WithLogger(log.StdLogger(log.DEBUG))

	app.WithFlagYAMLSupport("conf").WithShutdownTimeoutFlagSupport(5 * time.Second)

	app.AddFlags(application.StringFlag("listen", ":19945", "http listen addr"))
	app.AddBoolFlag("load-job", "")
	app.AddFlags(altsrc.NewBoolFlag(cli.BoolFlag{Name: "load-demoservice"}))

	app.Provider(job.ServiceProvider{}, api.ServiceProvider{})
	app.Service(&service.DemoService{}, &service.Demo2Service{})

	app.AfterInitialized(func(resolver infra.Resolver) error {
		return resolver.Resolve(func() {
			log.Debug("server initialized ...")
		})
	})

	//app.Provider(web.Provider(
	//	listener.FlagContext("listen"),
	//	// 设置该选项之后，路由匹配时将会忽略最末尾的 /
	//	// 路由 /aaa/bbb  匹配 /aaa/bbb, /aaa/bbb/
	//	// 路由 /aaa/bbb/ 匹配 /aaa/bbb, /aaa/bbb/
	//	// 默认为 false，匹配规则如下
	//	// 路由 /aaa/bbb 只匹配 /aaa/bbb 不匹配 /aaa/bbb/
	//	// 路由 /aaa/bbb/ 只匹配 /aaa/bbb/ 不匹配 /aaa/bbb
	//	web.SetIgnoreLastSlashOption(true),
	//	web.SetRouteHandlerOption(func(cc infra.Resolver, r web.Router, mw web.RequestMiddleware) {
	//		r.Get("/", func(webCtx web.Context) web.Response {
	//			return webCtx.JSON(web.M{"message": webCtx.Get("name")})
	//		})
	//	}),
	//))

	app.Provider(event.Provider(
		func(cc infra.Resolver, listener event.Listener) {
			listener.Listen(func(event CronEvent) {
				log.Debug("a new cron task executed")
				log.Debugf("event processed, listener_goroutine_id=%d, publisher_goroutine_id=%d", getGID(), event.GoroutineID)
			})
		},
		event.SetStoreOption(func(cc infra.Resolver) event.Store {
			return event.NewMemoryEventStore(true, 100)
		}),
	))

	app.PreBind(func(binder infra.Binder) {
		binder.MustSingleton(func(c infra.FlagContext) *config.Config {
			return &config.Config{
				DB:   "xxxxxx",
				Test: c.String("test"),
			}
		})
	})

	app.Async(func(conf *config.Config, publisher event.Publisher, gf infra.Graceful) {
		log.Debugf("config: %s", conf.Serialize())

		for i := 0; i < 10; i++ {
			publisher.Publish(CronEvent{GoroutineID: uint64(i)})
		}

		// 10s 后自动关闭服务
		time.AfterFunc(10*time.Second, gf.Shutdown)
	})

	return nil
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
