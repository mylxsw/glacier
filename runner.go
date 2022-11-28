package glacier

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/log"
)

type asyncJob struct {
	fn interface{}
}

func (aj asyncJob) Call(resolver infra.Resolver) error {
	return resolver.Resolve(aj.fn)
}

// Async 添加一个异步执行函数
func (impl *framework) Async(fns ...interface{}) {
	for i, fn := range fns {
		if reflect.TypeOf(fn).Kind() != reflect.Func {
			panic(fmt.Errorf("invalid argument: fn at %d must be a func", i))
		}

		impl.lock.Lock()
		if impl.status == Started {
			impl.asyncJobChannel <- asyncJob{fn: fn}
		} else {
			impl.asyncJobs = append(impl.asyncJobs, asyncJob{fn: fn})
		}
		impl.lock.Unlock()
	}
}

func (impl *framework) startAsyncRunners() <-chan interface{} {
	stop := make(chan interface{})

	var parentGraphNode *infra.GraphvizNode
	var childGraphNodes []*infra.GraphvizNode

	if infra.DEBUG {
		parentGraphNode = impl.pushGraphvizNode("start async runners", true)
		parentGraphNode.Style = infra.GraphvizNodeStyleImportant
	}

	impl.asyncJobChannel = make(chan asyncJob)
	impl.cc.MustResolve(func(gf infra.Graceful) {
		gf.AddShutdownHandler(func() {
			close(impl.asyncJobChannel)
		})
	})

	var wg sync.WaitGroup
	wg.Add(impl.asyncRunnerCount)

	for i := 0; i < impl.asyncRunnerCount; i++ {
		if infra.DEBUG {
			childGraphNodes = append(childGraphNodes, impl.pushGraphvizNode(fmt.Sprintf("start async runner %d", i), false, parentGraphNode))
			log.Debugf("[glacier] async runner %d starting ...", i)
		}

		go func(i int) {
			defer wg.Done()

			for job := range impl.asyncJobChannel {
				if err := job.Call(impl.cc); err != nil {
					log.Errorf("[glacier] async runner [async-runner-%d] failed: %v", i, err)
				}
			}

			if infra.DEBUG {
				log.Debugf("[glacier] async runner [async-runner-%d] stopping...", i)
			}
		}(i)
	}

	if infra.DEBUG {
		impl.pushGraphvizNode("all async runners started", false, childGraphNodes...)
	}

	go func() {
		wg.Wait()

		if infra.DEBUG {
			impl.pushGraphvizNode("all async runners stopped", false)
			log.Debug("[glacier] all async runners stopped")
		}

		close(stop)
	}()

	return stop
}

func (impl *framework) consumeAsyncJobs() {
	impl.lock.Lock()
	defer impl.lock.Unlock()

	for _, job := range impl.asyncJobs {
		impl.asyncJobChannel <- job
	}
	impl.asyncJobs = nil
}
