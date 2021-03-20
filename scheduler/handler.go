package scheduler

import (
	"github.com/mylxsw/glacier/infra"
)

// JobHandler 定时任务 Job 处理接口，所有的任务都要实现该接口
type JobHandler interface {
	Handle(resolver infra.Resolver) error
}

type jobHandlerImpl struct {
	handler interface{}
}

func newHandler(handler interface{}) JobHandler {
	return jobHandlerImpl{handler: handler}
}

func (h jobHandlerImpl) Handle(resolver infra.Resolver) error {
	return resolver.ResolveWithError(h.handler)
}

// WithoutOverlap 可以避免当前任务执行时间过长时，同一任务同时存在多个运行实例的问题
// 当任务还在执行时，下一次调度将会被取消
func WithoutOverlap(handler interface{}) *OverlapJobHandler {
	return &OverlapJobHandler{
		handler:   handler,
		executing: make(chan interface{}, 1),
	}
}

// OverlapJobHandler 是一个 Job Handler，可以避免当前任务执行时间过长时，同一任务同时存在多个运行实例的问题
// 当任务还在执行时，下一次调度将会被取消
type OverlapJobHandler struct {
	handler      interface{}
	skipCallback func()
	executing    chan interface{}
}

func (handler *OverlapJobHandler) SkipCallback(fn func()) *OverlapJobHandler {
	handler.skipCallback = fn
	return handler
}

func (handler *OverlapJobHandler) Handle(resolver infra.Resolver) error {
	select {
	case handler.executing <- struct{}{}:
		defer func() { <-handler.executing }()
		return resolver.ResolveWithError(handler.handler)
	default:
		if handler.skipCallback != nil {
			handler.skipCallback()
		}
	}

	return nil
}
