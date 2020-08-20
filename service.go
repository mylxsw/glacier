package glacier

import "github.com/mylxsw/glacier/infra"

// Provider add a service provider
func (glacier *glacierImpl) Provider(provider infra.ServiceProvider) {
	glacier.providers = append(glacier.providers, provider)
}

// Service add a service
func (glacier *glacierImpl) Service(service infra.Service) {
	glacier.services = append(glacier.services, service)
}