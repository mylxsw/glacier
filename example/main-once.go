package main

import (
	"fmt"

	"github.com/mylxsw/glacier/example/config"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/glacier/starter/app"
)

func main() {
	infra.DEBUG = true
	infra.PrintGraph = true
	app.MustStart("1.0", 3, runOnce)
}

// runOnce 执行一次性任务，执行完毕自动推出
func runOnce(ins *app.App) error {
	//log.AddGlobalFilter(func(filter filter.Filter) filter.Filter {
	//	return func(f asteriaEvent.Event) {
	//		if glacier.IsGlacierModuleLog(f.Module) {
	//			return
	//		}
	//
	//		filter(f)
	//	}
	//})

	ins.Singleton(func() *config.Config {
		log.Debugf("[example] create config ...")
		return &config.Config{DB: "demo", Test: "test str"}
	})

	ins.Async(func(gf infra.Graceful, conf *config.Config) {
		defer gf.Shutdown()

		fmt.Println(conf.Serialize())
	})

	return nil
}
