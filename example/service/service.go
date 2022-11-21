package service

import (
	"github.com/mylxsw/glacier/log"
	"time"

	"github.com/mylxsw/glacier/infra"
)

type DemoService struct {
	cc      infra.Resolver
	stopped chan interface{}
}

func (d *DemoService) ShouldLoad(c infra.FlagContext) bool {
	return c.Bool("[example] load-demoservice")
}

func (d *DemoService) Init(cc infra.Resolver) error {
	d.cc = cc
	d.stopped = make(chan interface{}, 0)
	return nil
}

func (d *DemoService) Name() string {
	return "demo-service"
}

func (d *DemoService) Start() error {
	for {
		select {
		case <-d.stopped:
			log.Debug("[example] service DemoService stopped")
			return nil
		default:
			time.Sleep(5 * time.Second)
			log.Errorf("[example] hello, world from %s", d.Name())
		}
	}
}

func (d *DemoService) Stop() {
	d.stopped <- struct{}{}
}

func (d *DemoService) Reload() {
	panic("implement me")
}
