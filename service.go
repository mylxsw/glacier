package glacier

import (
	"fmt"
	"reflect"

	"github.com/mylxsw/glacier/infra"
)

// Provider add a service provider
func (glacier *glacierImpl) Provider(providers ...infra.Provider) {
	glacier.providers = append(glacier.providers, providers...)
}

// Service add a service
func (glacier *glacierImpl) Service(services ...infra.Service) {
	glacier.services = append(glacier.services, services...)
}

type asyncJob struct {
	fn interface{}
}

func (aj asyncJob) Call(resolver infra.Resolver) error {
	return resolver.ResolveWithError(aj.fn)
}

// Async add a async function
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
