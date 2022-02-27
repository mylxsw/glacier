package graceful

import (
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/log"
	"os"
	"os/signal"
	"sync"
	"time"
)

type SignalHandler func(signalChan chan os.Signal, signals []os.Signal)

type gracefulImpl struct {
	lock sync.Mutex

	reloadSignals   []os.Signal
	shutdownSignals []os.Signal

	perHandlerTimeout time.Duration

	signalChan chan os.Signal

	signalHandler    SignalHandler
	reloadHandlers   []func()
	shutdownHandlers []func()
}

func NewWithSignal(reloadSignals []os.Signal, shutdownSignals []os.Signal, perHandlerTimeout time.Duration) infra.Graceful {
	return New(reloadSignals, shutdownSignals, perHandlerTimeout, func(signalChan chan os.Signal, signals []os.Signal) {
		signal.Notify(signalChan, signals...)
	})
}

func New(reloadSignals []os.Signal, shutdownSignals []os.Signal, perHandlerTimeout time.Duration, signalHandler SignalHandler) infra.Graceful {
	return &gracefulImpl{
		reloadSignals:     reloadSignals,
		shutdownSignals:   shutdownSignals,
		reloadHandlers:    make([]func(), 0),
		shutdownHandlers:  make([]func(), 0),
		perHandlerTimeout: perHandlerTimeout,
		signalChan:        make(chan os.Signal),
		signalHandler:     signalHandler,
	}
}

func (gf *gracefulImpl) AddReloadHandler(h func()) {
	gf.lock.Lock()
	defer gf.lock.Unlock()

	gf.reloadHandlers = append(gf.reloadHandlers, h)
}

func (gf *gracefulImpl) AddShutdownHandler(h func()) {
	gf.lock.Lock()
	defer gf.lock.Unlock()

	gf.shutdownHandlers = append(gf.shutdownHandlers, h)
}

func (gf *gracefulImpl) Reload() {
	log.Debug("execute reload...")
	go gf.reload()
}

func (gf *gracefulImpl) Shutdown() {
	log.Debug("shutdown...")

	_ = gf.signalSelf(os.Interrupt)
}

func (gf *gracefulImpl) signalSelf(sig os.Signal) error {
	gf.signalChan <- sig
	return nil
}

func (gf *gracefulImpl) shutdown() {
	gf.lock.Lock()
	defer gf.lock.Unlock()

	ok := make(chan interface{}, 0)
	defer close(ok)
	for i := len(gf.shutdownHandlers) - 1; i >= 0; i-- {
		go func(handler func()) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("execute shutdown handler failed: %s", err)
				}
				safeSendChanel(ok, struct{}{})
			}()

			handler()
		}(gf.shutdownHandlers[i])

		select {
		case <-ok:
		case <-time.After(gf.perHandlerTimeout):
			log.Errorf("execute shutdown handler timeout")
		}
	}
}

func (gf *gracefulImpl) reload() {
	gf.lock.Lock()
	defer gf.lock.Unlock()

	ok := make(chan interface{}, 0)
	defer close(ok)
	for i := len(gf.reloadHandlers) - 1; i >= 0; i-- {
		go func(handler func()) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("execute reload handler failed: %s", err)
				}
				safeSendChanel(ok, struct{}{})
			}()
			handler()
		}(gf.reloadHandlers[i])

		select {
		case <-ok:
		case <-time.After(gf.perHandlerTimeout):
			log.Errorf("execute reload handler timeout")
		}
	}
}

func (gf *gracefulImpl) Start() error {
	signals := make([]os.Signal, 0)
	signals = append(signals, gf.reloadSignals...)
	signals = append(signals, gf.shutdownSignals...)
	gf.signalHandler(gf.signalChan, signals)

	for {
		sig := <-gf.signalChan

		for _, s := range gf.shutdownSignals {
			if s == sig {
				goto FINAL
			}
		}

		for _, s := range gf.reloadSignals {
			if s == sig {
				log.Debugf("received a reload signal %s", sig.String())
				gf.reload()
				break
			}
		}
	}
FINAL:

	log.Debug("received a shutdown signal")

	gf.shutdown()

	return nil
}

func safeSendChanel(c chan interface{}, data interface{}) {
	defer func() { recover() }()
	c <- data
}
