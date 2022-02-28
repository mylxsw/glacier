//go:build windows
// +build windows

package graceful

import (
	"github.com/mylxsw/glacier/infra"
	"os"
	"time"
)

func NewWithDefault(perHandlerTimeout time.Duration) infra.Graceful {
	return NewWithSignal([]os.Signal{}, []os.Signal{os.Interrupt}, perHandlerTimeout)
}
