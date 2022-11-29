package glacier

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/glacier/graceful"
	"github.com/mylxsw/glacier/log"
	"github.com/mylxsw/go-ioc"

	"github.com/mylxsw/glacier/infra"
)

// initStage 框架初始化阶段
func (impl *framework) initStage(flagCtx infra.FlagContext) error {
	if infra.DEBUG {
		impl.pushGraphvizNode("initStage", false).Type = infra.GraphvizNodeTypeClusterStart
		defer func() {
			impl.pushGraphvizNode("initStage", false).Type = infra.GraphvizNodeTypeClusterEnd
		}()
	}

	// 执行初始化钩子，用于在框架运行前执行一系列的前置操作
	if impl.init != nil {
		if infra.DEBUG {
			impl.pushGraphvizNode("invoke init hook", false).Style = infra.GraphvizNodeStyleHook
		}

		if err := impl.init(flagCtx); err != nil {
			return err
		}

	}

	// 初始化日志实现
	if impl.logger != nil {
		if infra.DEBUG {
			impl.pushGraphvizNode("init logger", false)
		}
		log.SetDefaultLogger(impl.logger)
	}

	return nil
}

// diBindStage 初始化依赖注入容器阶段
func (impl *framework) diBindStage(ctx context.Context, flagCtx infra.FlagContext) error {
	if infra.DEBUG {
		impl.pushGraphvizNode("diBindStage", false).Type = infra.GraphvizNodeTypeClusterStart
		defer func() {
			impl.pushGraphvizNode("diBindStage", false).Type = infra.GraphvizNodeTypeClusterEnd
		}()

		impl.pushGraphvizNode("create container", false)
	}

	impl.cc = ioc.NewWithContext(ctx)

	impl.cc.MustBindValue(infra.VersionKey, impl.version)
	impl.cc.MustBindValue(infra.StartupTimeKey, impl.startTime)
	impl.cc.MustSingleton(impl.buildFlagContext(flagCtx))
	impl.cc.MustSingletonOverride(func() infra.Resolver { return impl.cc })
	impl.cc.MustSingletonOverride(func() infra.Binder { return impl.cc })
	impl.cc.MustSingletonOverride(func() infra.Hook { return impl })

	// 基本配置加载
	impl.cc.MustSingletonOverride(ConfigLoader)
	impl.cc.MustSingletonOverride(log.Default)

	// 优雅停机
	impl.cc.MustSingletonOverride(func(conf *Config) infra.Graceful {
		if impl.gracefulBuilder != nil {
			return impl.gracefulBuilder()
		}
		return graceful.NewWithDefault(conf.ShutdownTimeout)
	})

	// 注册全局对象
	if infra.DEBUG {
		impl.pushGraphvizNode("add singletons to container", false)
	}
	for _, i := range impl.singletons {
		impl.cc.MustSingletonOverride(i)
	}

	if infra.DEBUG {
		impl.pushGraphvizNode("add prototypes to container", false)
	}
	for _, i := range impl.prototypes {
		impl.cc.MustPrototypeOverride(i)
	}

	// 完成预绑定对象的绑定
	if impl.preBinder != nil {
		if infra.DEBUG {
			impl.pushGraphvizNode("invoke preBind hook", false).Style = infra.GraphvizNodeStyleHook
			log.Debugf("[glacier] invoke pre-bind hook")
		}
		impl.preBinder(impl.cc)
	}

	return nil
}

func (impl *framework) Start(flagCtx infra.FlagContext) error {
	// 全局异常处理
	defer func() {
		if err := recover(); err != nil {
			if infra.DEBUG {
				impl.pushGraphvizNode("global panic recover", false).Style = infra.GraphvizNodeStyleError
			}
			log.Criticalf("[glacier] application initialize failed with a panic, Err: %s, Stack: \n%s", err, debug.Stack())
		}

		if infra.DEBUG && infra.PrintGraph {
			impl.pushGraphvizNode("shutdownStage", false).Type = infra.GraphvizNodeTypeClusterEnd
			impl.nodeLock.Lock()
			fmt.Println(impl.nodes.Draw())
			impl.nodeLock.Unlock()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	impl.initStage(flagCtx)
	impl.diBindStage(ctx, flagCtx)

	return impl.cc.Resolve(func(resolver infra.Resolver, gf infra.Graceful, conf *Config) error {
		gf.AddShutdownHandler(cancel)

		// 设置服务关闭钩子
		if impl.beforeServerStop != nil {
			gf.AddShutdownHandler(func() {
				if infra.DEBUG {
					impl.pushGraphvizNode("invoke beforeServerStop hook", false).Style = infra.GraphvizNodeStyleHook
					log.Debugf("[glacier] invoke beforeServerStop hook")
				}
				_ = impl.beforeServerStop(resolver)
			})
		}

		impl.updateGlacierStatus(Initialized)

		if infra.DEBUG {
			impl.pushGraphvizNode("diBindStage", false).Type = infra.GraphvizNodeTypeClusterStart
			defer func() {
				impl.pushGraphvizNode("diBindStage", false).Type = infra.GraphvizNodeTypeClusterEnd
			}()
		}

		var wg sync.WaitGroup
		var bootStage = func() error {
			if infra.DEBUG {
				impl.pushGraphvizNode("bootStage", false).Type = infra.GraphvizNodeTypeClusterStart
				defer func() {
					impl.pushGraphvizNode("bootStage", false).Type = infra.GraphvizNodeTypeClusterEnd
				}()
			}

			// 注册 Providers & Services
			if err := impl.registerProviders(); err != nil {
				return err
			}

			if err := impl.registerServices(); err != nil {
				return err
			}

			// 启动 asyncRunners
			stop := impl.startAsyncRunners()
			impl.consumeAsyncJobs()

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-stop
			}()

			// 初始化 Services
			if err := impl.initServices(); err != nil {
				return err
			}

			// 启动 Providers
			if err := impl.bootProviders(); err != nil {
				return err
			}

			// 启动 Daemon Providers
			if err := impl.startDaemonProviders(ctx, &wg); err != nil {
				return err
			}

			// 启动 Services
			if err := impl.startServices(ctx, &wg); err != nil {
				return err
			}

			return nil
		}
		if err := bootStage(); err != nil {
			return err
		}

		impl.updateGlacierStatus(Started)
		impl.readyStage(resolver, gf)

		defer impl.shutdownHandler(conf, &wg)
		if infra.DEBUG {
			gf.AddPreShutdownHandler(func() {
				impl.pushGraphvizNode("shutdownStage", false).Type = infra.GraphvizNodeTypeClusterStart
			})
		}
		return gf.Start()
	})
}

func (impl *framework) readyStage(resolver infra.Resolver, gf infra.Graceful) {
	if infra.DEBUG {
		impl.pushGraphvizNode("readyStage", false).Type = infra.GraphvizNodeTypeClusterStart
		defer func() {
			impl.pushGraphvizNode("readyStage", false).Type = infra.GraphvizNodeTypeClusterEnd
		}()
	}

	var childGraphNodes []*infra.GraphvizNode
	if len(impl.onServerReadyHooks) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(impl.onServerReadyHooks))

		var parentGraphNode *infra.GraphvizNode
		if infra.DEBUG {
			parentGraphNode = impl.pushGraphvizNode("invoke onServerReady hooks", true)
			parentGraphNode.Style = infra.GraphvizNodeStyleHook
		}

		for _, hook := range impl.onServerReadyHooks {
			if infra.DEBUG {
				childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode("invoke onServerReady hook: "+hook.name, true, parentGraphNode))
				log.Debugf("[glacier] invoke onServerReady hook [%s]", hook.name)
			}

			go func(hook namedFunc) {
				defer wg.Done()
				if err := resolver.Resolve(hook.fn); err != nil {
					log.Errorf("[glacier] onServerReady hook [%s] failed: %v", hook.name, err)
				}
			}(hook)
		}

		gf.AddShutdownHandler(wg.Wait)
	}

	if infra.DEBUG {
		impl.pushGraphvizNode("launched", false, childGraphNodes...)
		log.Debugf("[glacier] application launched successfully, took %s", time.Since(impl.startTime))
	}
}

func (impl *framework) shutdownHandler(conf *Config, wg *sync.WaitGroup) {
	if infra.DEBUG {
		impl.pushGraphvizNode("shutdown", false)
	}

	if conf.ShutdownTimeout > 0 {
		ok := make(chan interface{})
		go func() {
			wg.Wait()
			ok <- struct{}{}
		}()
		select {
		case <-ok:
			if infra.DEBUG {
				log.Debugf("[glacier] all modules has been stopped, application will exit safely")
			}
		case <-time.After(conf.ShutdownTimeout):
			log.Errorf("[glacier] shutdown timeout, exit directly")
		}
	} else {
		wg.Wait()
		if infra.DEBUG {
			log.Debugf("[glacier] all modules has been stopped")
		}
	}

}
