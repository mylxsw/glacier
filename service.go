package glacier

import (
	"github.com/mylxsw/container"
)

// Service is a interface for service
type Service interface {
	// Init initialize the service
	Init(cc *container.Container) error
	// Name return service name
	Name() string
	// Start start service, not blocking
	Start() error
	// Stop stop the service
	Stop()
	// Reload reload service
	Reload()
}
