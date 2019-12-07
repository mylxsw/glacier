package glacier

import (
	"time"
)

type FlagContext interface {
	String(name string) string
	GlobalString(name string) string
	StringSlice(name string) []string
	GlobalStringSlice(name string) []string

	Bool(name string) bool
	GlobalBool(name string) bool
	BoolT(name string) bool
	GlobalBoolT(name string) bool

	Int64(name string) int64
	GlobalInt64(name string) int64
	Int(name string) int
	GlobalInt(name string) int
	IntSlice(name string) []int
	GlobalIntSlice(name string) []int
	Uint64(name string) uint64
	GlobalUint64(name string) uint64
	Uint(name string) uint
	GlobalUint(name string) uint
	Int64Slice(name string) []int64
	GlobalInt64Slice(name string) []int64

	Duration(name string) time.Duration
	GlobalDuration(name string) time.Duration

	Float64(name string) float64
	GlobalFloat64(name string) float64

	Generic(name string) interface{}
	GlobalGeneric(name string) interface{}

	FlagNames() (names []string)
	GlobalFlagNames() (names []string)
}
