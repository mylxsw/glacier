package glacier

import "github.com/mylxsw/glacier/infra"

// Provider add a service provider
func (glacier *glacierImpl) Provider(providers... infra.Provider) {
	glacier.providers = append(glacier.providers, providers...)
}

// Service add a service
func (glacier *glacierImpl) Service(services... infra.Service) {
	glacier.services = append(glacier.services, services...)
}