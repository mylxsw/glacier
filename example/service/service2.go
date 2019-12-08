package service

import (
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
)

type Demo2Service struct {
	cc      container.Container
	stopped chan interface{}
}

func (d *Demo2Service) Init(cc container.Container) error {
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
			return nil
		default:
			time.Sleep(1 * time.Second)
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
