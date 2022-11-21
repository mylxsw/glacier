package graceful

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/log"
)

type SignalHandler func(signalChan chan os.Signal, signals []os.Signal)

type gracefulImpl struct {
	lock sync.Mutex

	reloadSignals   []os.Signal
	shutdownSignals []os.Signal

	handlerTimeout time.Duration

	signalChan chan os.Signal

	signalHandler    SignalHandler
	reloadHandlers   []Handler
	shutdownHandlers []Handler
}

type Handler struct {
	handler     func()
	packagePath string
	filename    string
	line        int
}

func (h Handler) String() string {
	return fmt.Sprintf("%s(%s:%d)", h.packagePath, h.filename, h.line)
}

func NewWithSignal(reloadSignals []os.Signal, shutdownSignals []os.Signal, perHandlerTimeout time.Duration) infra.Graceful {
	return New(reloadSignals, shutdownSignals, perHandlerTimeout, func(signalChan chan os.Signal, signals []os.Signal) {
		signal.Notify(signalChan, signals...)
	})
}

func New(reloadSignals []os.Signal, shutdownSignals []os.Signal, handlerTimeout time.Duration, signalHandler SignalHandler) infra.Graceful {
	return &gracefulImpl{
		reloadSignals:    reloadSignals,
		shutdownSignals:  shutdownSignals,
		reloadHandlers:   make([]Handler, 0),
		shutdownHandlers: make([]Handler, 0),
		handlerTimeout:   handlerTimeout,
		signalChan:       make(chan os.Signal),
		signalHandler:    signalHandler,
	}
}

func (gf *gracefulImpl) AddReloadHandler(h func()) {
	handler := Handler{handler: h}
	pc, f, line, ok := runtime.Caller(1)
	if ok {
		handler.packagePath = runtime.FuncForPC(pc).Name()
		handler.filename = f
		handler.line = line
	}

	gf.lock.Lock()
	defer gf.lock.Unlock()

	gf.reloadHandlers = append(gf.reloadHandlers, handler)
}

func (gf *gracefulImpl) AddShutdownHandler(h func()) {
	handler := Handler{handler: h}
	pc, f, line, ok := runtime.Caller(1)
	if ok {
		handler.packagePath = runtime.FuncForPC(pc).Name()
		handler.filename = f
		handler.line = line
	}

	gf.lock.Lock()
	defer gf.lock.Unlock()

	gf.shutdownHandlers = append(gf.shutdownHandlers, handler)
}

func (gf *gracefulImpl) Reload() {
	if infra.DebugEnabled {
		log.Debug("graceful reloading...")
	}
	go gf.reload()
}

func (gf *gracefulImpl) Shutdown() {
	if infra.DebugEnabled {
		log.Debug("graceful closing...")
	}
	_ = gf.signalSelf(os.Interrupt)
}

func (gf *gracefulImpl) signalSelf(sig os.Signal) error {
	gf.signalChan <- sig
	return nil
}

func (gf *gracefulImpl) shutdown() {
	gf.lock.Lock()
	defer gf.lock.Unlock()

	handlerExecutedStat := make([]bool, len(gf.shutdownHandlers))
	for i := len(gf.shutdownHandlers) - 1; i >= 0; i-- {
		handlerExecutedStat[i] = false
	}

	var wg sync.WaitGroup
	wg.Add(len(gf.shutdownHandlers))
	for i := len(gf.shutdownHandlers) - 1; i >= 0; i-- {
		go func(i int, handler Handler) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("executing shutdown handler [%s] failed: %s", handler.String(), err)
				}

				handlerExecutedStat[i] = true
				wg.Done()
			}()

			handler.handler()
		}(i, gf.shutdownHandlers[i])
	}

	ok := make(chan interface{}, 0)
	defer close(ok)

	go func() {
		wg.Wait()
		ok <- struct{}{}
	}()

	select {
	case <-ok:
		if infra.DebugEnabled {
			log.Debug("all shutdown handlers executed")
		}
	case <-time.After(gf.handlerTimeout):
		log.Errorf("executing shutdown handlers timed out")
		for i, executed := range handlerExecutedStat {
			if executed {
				continue
			}

			log.Errorf("shutdown handler [%s] may not executed", gf.shutdownHandlers[i].String())
		}
	}
}

func (gf *gracefulImpl) reload() {
	gf.lock.Lock()
	defer gf.lock.Unlock()

	handlerExecutedStat := make([]bool, len(gf.shutdownHandlers))
	for i := len(gf.shutdownHandlers) - 1; i >= 0; i-- {
		handlerExecutedStat[i] = false
	}

	var wg sync.WaitGroup
	wg.Add(len(gf.reloadHandlers))
	for i := len(gf.reloadHandlers) - 1; i >= 0; i-- {
		go func(i int, handler Handler) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("executing reload handler failed: %s", err)
				}
				handlerExecutedStat[i] = true
				wg.Done()
			}()

			handler.handler()
		}(i, gf.reloadHandlers[i])
	}

	ok := make(chan interface{}, 0)
	defer close(ok)

	go func() {
		wg.Wait()
		ok <- struct{}{}
	}()

	select {
	case <-ok:
		if infra.DebugEnabled {
			log.Debug("all reload handlers executed")
		}
	case <-time.After(gf.handlerTimeout):
		log.Errorf("executing reload handlers timed out")
		for i, executed := range handlerExecutedStat {
			if executed {
				continue
			}

			log.Errorf("reload handler [%s] may not executed", gf.shutdownHandlers[i].String())
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
				log.Warningf("shutdown signal received: %s", sig.String())
				goto FINAL
			}
		}

		for _, s := range gf.reloadSignals {
			if s == sig {
				log.Warningf("reload signal received: %s", sig.String())
				gf.reload()
				break
			}
		}
	}
FINAL:
	gf.shutdown()

	return nil
}
