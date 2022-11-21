package glacier

import (
	"fmt"
	"github.com/mylxsw/go-utils/array"
	"reflect"

	"github.com/mylxsw/glacier/infra"
)

var errorKind = reflect.TypeOf((*error)(nil)).Elem()

// Provider add a service provider
func (glacier *glacierImpl) Provider(providers ...infra.Provider) {
	for _, p := range providers {
		validateShouldLoadMethod(reflect.TypeOf(p))
	}

	glacier.providers = append(glacier.providers, providers...)
}

func validateShouldLoadMethod(pType reflect.Type) {
	if method, ok := pType.MethodByName("ShouldLoad"); ok {
		returnValueCount := method.Type.NumOut()
		if method.Type.Out(0).Kind() != reflect.Bool {
			panic(fmt.Errorf("invalid provider %s: the first return value for ShouldLoad method must a bool", pType.String()))
		}
		if returnValueCount == 0 || returnValueCount > 2 {
			panic(fmt.Errorf("invalid provider %s: ShouldLoad method must be func(...) bool or func(...) (bool, error)", pType.String()))
		} else if returnValueCount == 2 {
			if !method.Type.Out(1).Implements(errorKind) {
				panic(fmt.Errorf("invalid provider %s: the second return value for ShouldLoad method must be an error", pType.String()))
			}
		}
	}
}

// Service add a service
func (glacier *glacierImpl) Service(services ...infra.Service) {
	for _, p := range services {
		validateShouldLoadMethod(reflect.TypeOf(p))
	}

	glacier.services = append(glacier.services, array.Map(services, func(srv infra.Service) serviceRegister {
		var name string
		if srvn, ok := srv.(infra.Nameable); ok {
			name = srvn.Name()
		}

		if name == "" {
			name = reflect.TypeOf(srv).String()
		}

		return serviceRegister{service: srv, name: name}
	})...)
}

type asyncJob struct {
	fn interface{}
}

func (aj asyncJob) Call(resolver infra.Resolver) error {
	return resolver.ResolveWithError(aj.fn)
}

// Async 添加一个异步执行函数
func (glacier *glacierImpl) Async(fns ...interface{}) {
	for i, fn := range fns {
		if reflect.TypeOf(fn).Kind() != reflect.Func {
			panic(fmt.Errorf("invalid argument: fn at %d must be a func", i))
		}

		glacier.lock.Lock()
		if glacier.status == Started {
			glacier.asyncJobChannel <- asyncJob{fn: fn}
		} else {
			glacier.asyncJobs = append(glacier.asyncJobs, asyncJob{fn: fn})
		}
		glacier.lock.Unlock()
	}
}
