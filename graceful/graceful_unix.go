//go:build !windows
// +build !windows

package graceful

import (
	"github.com/mylxsw/glacier/infra"
	"os"
	"syscall"
	"time"
)

func NewWithDefault(perHandlerTimeout time.Duration) infra.Graceful {
	return NewWithSignal([]os.Signal{syscall.SIGUSR2}, []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT}, perHandlerTimeout)
}
