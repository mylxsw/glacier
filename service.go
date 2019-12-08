package glacier

import (
	"context"

	"github.com/mylxsw/container"
)

// Service is a interface for service
type Service interface {
	// Init initialize the service
	Init(cc container.Container) error
	// Name return service name
	Name() string
	// Start start service, not blocking
	Start() error
	// Stop stop the service
	Stop()
	// Reload reload service
	Reload()
}


type ServiceProvider interface {
	// Register add some dependency for current module
	// this method is called one by one synchronous
	Register(app container.Container)
	// Boot start the module
	// this method is called one by one synchronous after all register methods called
	Boot(app Glacier)
}

type DaemonServiceProvider interface {
	ServiceProvider
	// Daemon is a async method called after boot
	// this method is called asynchronous and concurrent
	Daemon(ctx context.Context, app Glacier)
}

// Provider add a service provider
func (glacier *glacierImpl) Provider(provider ServiceProvider) {
	glacier.providers = append(glacier.providers, provider)
}

// Service add a service
func (glacier *glacierImpl) Service(service Service) {
	glacier.services = append(glacier.services, service)
}