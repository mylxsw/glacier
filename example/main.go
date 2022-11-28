package main

import (
	"bytes"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/glacier/starter/app"
	"github.com/mylxsw/glacier/web"

	"net/http"
	_ "net/http/pprof"

	"github.com/mylxsw/glacier/event"
	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/example/job"
	"github.com/mylxsw/glacier/example/service"
	"github.com/mylxsw/glacier/infra"
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

	runtime.SetBlockProfileRate(1)

	infra.DEBUG = true
	infra.PrintGraph = true

	app.MustStart("1.0", 3, run)
}

// run 后台持续运行的任务，除非手动触发退出，否则一直运行
func run(ins *app.App) error {
	ins.WithDescription("Glacier 框架演示项目").
		WithName("glacier-example").
		WithUsage("这是一个示例项目").
		WithAuthor("管宜尧", "mylxsw@aicode.cc").
		WithCLIOptions(func(cliAPP *cli.App) {
			cliAPP.Copyright = "aicode.cc"
			cliAPP.UsageText = "这是 Usage Text"
		})

	ins.WithLogger(log.StdLogger())

	ins.WithYAMLFlag("conf").WithShutdownTimeoutFlag(3 * time.Second)

	ins.AddFlags(app.StringFlag("listen", ":19945", "http listen addr"))
	ins.AddBoolFlag("load-job", "")
	ins.AddFlags(altsrc.NewBoolFlag(cli.BoolFlag{Name: "load-demoservice"}))

	ins.Provider(job.ServiceProvider{})
	ins.Service(&service.DemoService{}, &service.Demo2Service{})

	ins.Init(func(c infra.FlagContext) error {
		log.Debug("init")
		return nil
	})

	ins.BeforeServerStop(func(cc infra.Resolver) error {
		return nil
	})

	// ins.Provider(api.ServiceProvider{})
	ins.Provider(web.DefaultProvider(
		func(resolver infra.Resolver, router web.Router, mw web.RequestMiddleware) {
			router.Get("/", func(ctx web.Context) web.Response {
				return ctx.JSON(web.M{"hello": ctx.InputWithDefault("name", "world")})
			})
		},
		web.SetMuxRouteHandlerOption(func(resolver infra.Resolver, router *mux.Router) {
			router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
		}),
		web.SetServerConfigOption(func(server *http.Server, listener net.Listener) {
			//server.ReadTimeout = 10 * time.Second
		}),
	))

	//ins.Provider(web.Provider(
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

	ins.Provider(event.Provider(
		func(cc infra.Resolver, listener event.Listener) {
			listener.Listen(func(event CronEvent) {
				log.Debug("[example] a new cron task executed")
				log.Debugf("[example] event processed, listener_goroutine_id=%d, publisher_goroutine_id=%d", getGID(), event.GoroutineID)
			})
		},
		event.SetStoreOption(func(cc infra.Resolver) event.Store {
			return event.NewMemoryEventStore(true, 100)
		}),
	))

	ins.PreBind(func(binder infra.Binder) {
		binder.MustSingleton(func(c infra.FlagContext) *config.Config {
			return &config.Config{
				DB:   "xxxxxx",
				Test: c.String("test"),
			}
		})
	})

	ins.Async(func(conf *config.Config, publisher event.Publisher, gf infra.Graceful) {
		log.Debugf("[example] config: %s", conf.Serialize())

		for i := 0; i < 10; i++ {
			publisher.Publish(CronEvent{GoroutineID: uint64(i)})
		}

		// 10s 后自动关闭服务
		// go time.AfterFunc(10*time.Second, gf.Shutdown)
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
