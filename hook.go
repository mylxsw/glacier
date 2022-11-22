package glacier

import (
	"fmt"
	"reflect"

	"github.com/mylxsw/glacier/infra"
)

// PreBind 设置预绑定实例，这里会确保在容器中第一次进行对象实例化之前完成实例绑定
func (impl *framework) PreBind(fn func(binder infra.Binder)) infra.Glacier {
	impl.preBinder = fn
	return impl
}

// Init set a hook func executed before server initialize
// Usually, we use this method to initialize the log configuration
func (impl *framework) Init(f func(c infra.FlagContext) error) infra.Glacier {
	impl.init = f
	return impl
}

// OnServerReady call a function on server ready
func (impl *framework) OnServerReady(ffs ...interface{}) {
	impl.lock.Lock()
	defer impl.lock.Unlock()

	if impl.status == Started {
		panic(fmt.Errorf("[glacier] can not call OnServerReady since server has been started"))
	}

	for _, f := range ffs {
		fn := newNamedFunc(f)
		if reflect.TypeOf(f).Kind() != reflect.Func {
			panic(fmt.Errorf("[glacier] argument for OnServerReady [%s] must be a callable function", fn.name))
		}

		impl.onServerReadyHooks = append(impl.onServerReadyHooks, fn)
	}
}

// BeforeServerStop set a hook func executed before server stop
func (impl *framework) BeforeServerStop(f func(cc infra.Resolver) error) infra.Glacier {
	impl.beforeServerStop = f
	return impl
}
