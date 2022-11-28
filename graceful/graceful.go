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

	signalHandler       SignalHandler
	reloadHandlers      []Handler
	shutdownHandlers    []Handler
	preShutdownHandlers []Handler
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

func (gf *gracefulImpl) AddPreShutdownHandler(h func()) {
	handler := Handler{handler: h}
	pc, f, line, ok := runtime.Caller(1)
	if ok {
		handler.packagePath = runtime.FuncForPC(pc).Name()
		handler.filename = f
		handler.line = line
	}

	gf.lock.Lock()
	defer gf.lock.Unlock()

	gf.preShutdownHandlers = append(gf.preShutdownHandlers, handler)
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
	if infra.DEBUG {
		log.Debug("[glacier] graceful reloading...")
	}
	go gf.reload()
}

func (gf *gracefulImpl) Shutdown() {
	if infra.DEBUG {
		log.Debug("[glacier] graceful closing...")
	}
	_ = gf.signalSelf(os.Interrupt)
}

func (gf *gracefulImpl) signalSelf(sig os.Signal) error {
	gf.signalChan <- sig
	return nil
}

func (gf *gracefulImpl) shutdown() {
	startTs := time.Now()

	gf.lock.Lock()
	defer gf.lock.Unlock()

	for _, handler := range gf.preShutdownHandlers {
		if infra.DEBUG {
			log.Debugf("[glacier] pre shutdown handler: %s", handler.String())
		}

		handler.handler()
	}

	handlerExecutedStat := make([]bool, len(gf.shutdownHandlers))
	for i := len(gf.shutdownHandlers) - 1; i >= 0; i-- {
		handlerExecutedStat[i] = false
	}

	var wg sync.WaitGroup
	wg.Add(len(gf.shutdownHandlers))
	for i := len(gf.shutdownHandlers) - 1; i >= 0; i-- {
		go func(i int, handler Handler) {
			startTs := time.Now()
			if infra.DEBUG {
				log.Debugf("[glacier] executing shutdown handler [%s]", handler.String())
			}

			defer func() {
				if err := recover(); err != nil {
					log.Errorf("[glacier] executing shutdown handler [%s] failed: %s", handler.String(), err)
				}

				if infra.DEBUG {
					log.Debugf("[glacier] shutdown handler [%s] finished, took %s", handler.String(), time.Since(startTs).String())
				}

				handlerExecutedStat[i] = true
				wg.Done()
			}()

			handler.handler()
		}(i, gf.shutdownHandlers[i])
	}

	ok := make(chan interface{})
	defer close(ok)

	go func() {
		wg.Wait()
		ok <- struct{}{}
	}()

	select {
	case <-ok:
		if infra.DEBUG {
			log.Debugf("[glacier] all shutdown handlers executed, took %s", time.Since(startTs))
		}
	case <-time.After(gf.handlerTimeout):
		log.Errorf("[glacier] executing shutdown handlers timed out, took %s", time.Since(startTs))
		for i, executed := range handlerExecutedStat {
			if executed {
				continue
			}

			log.Errorf("[glacier] shutdown handler [%s] may not finished", gf.shutdownHandlers[i].String())
		}
	}
}

func (gf *gracefulImpl) reload() {
	startTs := time.Now()

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
			startTs := time.Now()
			if infra.DEBUG {
				log.Debugf("[glacier] executing reload handler [%s]", handler.String())
			}

			defer func() {
				if err := recover(); err != nil {
					log.Errorf("[glacier] executing reload handler failed: %s", err)
				}

				if infra.DEBUG {
					log.Debugf("[glacier] reload handler [%s] finished, took %s", handler.String(), time.Since(startTs).String())
				}

				handlerExecutedStat[i] = true
				wg.Done()
			}()

			handler.handler()
		}(i, gf.reloadHandlers[i])
	}

	ok := make(chan interface{})
	defer close(ok)

	go func() {
		wg.Wait()
		ok <- struct{}{}
	}()

	select {
	case <-ok:
		if infra.DEBUG {
			log.Debugf("[glacier] all reload handlers executed, took %s", time.Since(startTs))
		}
	case <-time.After(gf.handlerTimeout):
		log.Errorf("[glacier] executing reload handlers timed out, took %s", time.Since(startTs))
		for i, executed := range handlerExecutedStat {
			if executed {
				continue
			}

			log.Errorf("[glacier] reload handler [%s] may not finished", gf.shutdownHandlers[i].String())
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
				if infra.WARN {
					log.Warningf("[glacier] shutdown signal received: %s", sig.String())
				}
				goto FINAL
			}
		}

		for _, s := range gf.reloadSignals {
			if s == sig {
				if infra.WARN {
					log.Warningf("[glacier] reload signal received: %s", sig.String())
				}
				gf.reload()
				break
			}
		}
	}
FINAL:
	gf.shutdown()

	return nil
}
