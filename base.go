package glacier

import (
	"fmt"
	"reflect"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-utils/ternary"
)

// Status 当前 Glacier 的状态
type Status int

func (s Status) String() string {
	switch s {
	case Initialized:
		return "Initialized"
	case Started:
		return "Started"
	}

	return "Unknown"
}

const (
	Unknown     Status = 0
	Initialized Status = 1
	Started     Status = 2
)

type namedFunc struct {
	name string
	fn   interface{}
}

func newNamedFunc(fn interface{}) namedFunc {
	return namedFunc{fn: fn, name: resolveNameable(fn)}
}

var errorKind = reflect.TypeOf((*error)(nil)).Elem()

func resolveNameable(item interface{}) string {
	var name string
	if p, ok := item.(infra.Nameable); ok {
		name = p.Name()
	}

	if name == "" {
		pf := reflect.TypeOf(item)
		name = fmt.Sprintf("%s:%s", ternary.If(pf.PkgPath() == "", ".", pf.PkgPath()), pf.String())
	}
	return name
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
