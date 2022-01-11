package service

import (
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
)

type Demo2Service struct {
	cc      infra.Resolver
	stopped chan interface{}
}

func (d *Demo2Service) Init(cc infra.Resolver) error {
	d.cc = cc
	d.stopped = make(chan interface{}, 0)
	return nil
}

func (d *Demo2Service) Name() string {
	return "demo2-service"
}

func (d *Demo2Service) Start() error {
	for {
		select {
		case <-d.stopped:
			log.Debug("service Demo2Service stopped")
			return nil
		default:
			time.Sleep(3 * time.Second)
			log.Infof("hello, world from %s", d.Name())
		}
	}
}

func (d *Demo2Service) Stop() {
	d.stopped <- struct{}{}
}

func (d *Demo2Service) Reload() {
	panic("implement me")
}
