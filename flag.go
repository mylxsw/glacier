package glacier

import (
	"time"

	"github.com/mylxsw/glacier/infra"
)

func (impl *framework) buildFlagContext(cliCtx infra.FlagContext) func() (infra.FlagContext, error) {
	return func() (infra.FlagContext, error) {
		res, err := impl.cc.CallWithProvider(impl.flagContextInit, impl.cc.Provider(func() infra.FlagContext {
			return cliCtx
		}))

		if err != nil {
			return nil, err
		}

		return res[0].(infra.FlagContext), nil
	}
}

type FlagContext struct {
	data map[string]interface{}
}

func (f *FlagContext) setData(name string, value interface{}) {
	f.data[name] = value
}

func (f *FlagContext) SetString(name string, value string)          { f.setData(name, value) }
func (f *FlagContext) SetStringSlice(name string, value []string)   { f.setData(name, value) }
func (f *FlagContext) SetBool(name string, value bool)              { f.setData(name, value) }
func (f *FlagContext) SetInt(name string, value int)                { f.setData(name, value) }
func (f *FlagContext) SetIntSlice(name string, value []int)         { f.setData(name, value) }
func (f *FlagContext) SetDuration(name string, value time.Duration) { f.setData(name, value) }
func (f *FlagContext) SetFloat64(name string, value float64)        { f.setData(name, value) }

func (f *FlagContext) String(name string) string {
	raw, ok := f.data[name]
	if !ok {
		return ""
	}

	val, ok := raw.(string)
	if !ok {
		return ""
	}

	return val
}

func (f *FlagContext) StringSlice(name string) []string {
	raw, ok := f.data[name]
	if !ok {
		return []string{}
	}

	val, ok := raw.([]string)
	if !ok {
		return []string{}
	}

	return val
}

func (f *FlagContext) Bool(name string) bool {
	raw, ok := f.data[name]
	if !ok {
		return false
	}

	val, ok := raw.(bool)
	if !ok {
		return false
	}

	return val
}

func (f *FlagContext) Int(name string) int {
	raw, ok := f.data[name]
	if !ok {
		return 0
	}

	val, ok := raw.(int)
	if !ok {
		return 0
	}

	return val
}

func (f *FlagContext) IntSlice(name string) []int {
	raw, ok := f.data[name]
	if !ok {
		return []int{}
	}

	val, ok := raw.([]int)
	if !ok {
		return []int{}
	}

	return val
}

func (f *FlagContext) Duration(name string) time.Duration {
	raw, ok := f.data[name]
	if !ok {
		return 0
	}

	val, ok := raw.(time.Duration)
	if !ok {
		return 0
	}

	return val
}

func (f *FlagContext) Float64(name string) float64 {
	raw, ok := f.data[name]
	if !ok {
		return 0
	}

	val, ok := raw.(float64)
	if !ok {
		return 0
	}

	return val
}

func (f *FlagContext) FlagNames() []string {
	names := make([]string, 0)
	for k := range f.data {
		names = append(names, k)
	}

	return names
}
