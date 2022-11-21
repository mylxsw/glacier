package service

import (
	"github.com/mylxsw/glacier/log"
	"time"
)

type Demo2Service struct {
	stopped chan interface{}
}

func (d *Demo2Service) Start() error {
	d.stopped = make(chan interface{}, 0)
	for {
		select {
		case <-d.stopped:
			log.Debug("[example] service Demo2Service stopped")
			return nil
		default:
			time.Sleep(3 * time.Second)
		}
	}
}

func (d *Demo2Service) Stop() {
	d.stopped <- struct{}{}
	close(d.stopped)
}
