package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/glacier/log"

	"github.com/mylxsw/glacier/infra"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

// JobCreator is a creator for cron job
type JobCreator interface {
	// Add a cron job
	Add(name string, plan string, handler interface{}) error
	// AddAndRunOnServerReady add a cron job, and trigger it immediately when server is ready
	AddAndRunOnServerReady(name string, plan string, handler interface{}) error

	// MustAdd add a cron job
	MustAdd(name string, plan string, handler interface{})
	// MustAddAndRunOnServerReady add a cron job, and trigger it immediately when server is ready
	MustAddAndRunOnServerReady(name string, plan string, handler interface{})
}

// Scheduler is a manager object to manage cron jobs
type Scheduler interface {
	JobCreator
	// Remove remove a cron job
	Remove(name string) error
	// Pause set job status to paused
	Pause(name string) error
	// Continue set job status to continue
	Continue(name string) error
	// Info get job info
	Info(name string) (Job, error)

	// Start cron manager
	Start()
	// Stop cron job manager
	Stop()

	LockManagerBuilder(builder LockManagerBuilder)
}

type LockManager interface {
	TryLock(ctx context.Context) error
	Release(ctx context.Context) error
}

var ErrLockFailed = errors.New("lock failed")

type LockManagerBuilder func(name string) LockManager

type schedulerImpl struct {
	lock     sync.RWMutex
	resolver infra.Resolver
	cr       *cron.Cron

	lockManagerBuilder LockManagerBuilder

	jobs map[string]*Job
}

// Job is a job object
type Job struct {
	ID          cron.EntryID
	Name        string
	Plan        string
	handler     func()
	Paused      bool
	lockManager LockManager
}

// Next get execute plan for job
func (job Job) Next(nextNum int) ([]time.Time, error) {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sc, err := parser.Parse(job.Plan)
	if err != nil {
		return nil, err
	}

	results := make([]time.Time, nextNum)
	lastTs := time.Now()
	for i := 0; i < nextNum; i++ {
		lastTs = sc.Next(lastTs)
		results[i] = lastTs
	}

	return results, nil
}

// NewManager create a new Scheduler
func NewManager(resolver infra.Resolver) Scheduler {
	m := schedulerImpl{resolver: resolver, jobs: make(map[string]*Job)}
	resolver.MustResolve(func(cr *cron.Cron) { m.cr = cr })

	return &m
}

func (c *schedulerImpl) LockManagerBuilder(builder LockManagerBuilder) {
	c.lockManagerBuilder = builder
}

func (c *schedulerImpl) MustAddAndRunOnServerReady(name string, plan string, handler interface{}) {
	if err := c.AddAndRunOnServerReady(name, plan, handler); err != nil {
		panic(err)
	}
}

func (c *schedulerImpl) AddAndRunOnServerReady(name string, plan string, handler interface{}) error {
	if err := c.Add(name, plan, handler); err != nil {
		return err
	}

	hh, ok := handler.(JobHandler)
	if ok {
		handler = hh.Handle
	}

	return c.resolver.Resolve(func(hook infra.Hook) {
		hook.OnServerReady(handler)
	})
}

func (c *schedulerImpl) MustAdd(name string, plan string, handler interface{}) {
	if err := c.Add(name, plan, handler); err != nil {
		panic(err)
	}
}

func (c *schedulerImpl) Add(name string, plan string, handler interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if reg, existed := c.jobs[name]; existed {
		return fmt.Errorf("job with name [%s] already existed: %d | %s", name, reg.ID, reg.Plan)
	}

	hh, ok := handler.(JobHandler)
	if !ok {
		hh = newHandler(handler)
	}

	var lockManager LockManager
	if c.lockManagerBuilder != nil {
		lockManager = c.lockManagerBuilder(name)
	}

	jobHandler := func() {
		if lockManager != nil {
			if err := lockManager.TryLock(context.TODO()); err != nil {
				if errors.Is(err, ErrLockFailed) {
					if infra.DEBUG {
						log.Debugf("[glacier] cron job [%s] can not start because it doesn't get the lock", name)
					}

					return
				}

				log.Errorf("[glacier] cron job [%s] can not start because it can not get the lock: %v", name, err)
				return
			}
		}

		if infra.DEBUG {
			log.Debugf("[glacier] cron job [%s] running", name)
		}

		startTs := time.Now()
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("[glacier] cron job [%s] stopped with some errors: %v, took %s", name, err, time.Since(startTs))
			} else {
				if infra.DEBUG {
					log.Debugf("[glacier] cron job [%s] stopped, took %s", name, time.Since(startTs))
				}
			}
		}()
		if err := c.resolver.Resolve(hh.Handle); err != nil {
			log.Errorf("[glacier] cron job [%s] failed, Err: %v, Stack: \n%s", name, err, debug.Stack())
		}
	}
	id, err := c.cr.AddFunc(plan, jobHandler)

	if err != nil {
		return errors.Wrap(err, "[glacier] add cron job failed")
	}

	c.jobs[name] = &Job{
		ID:          id,
		Name:        name,
		Plan:        plan,
		handler:     jobHandler,
		Paused:      false,
		lockManager: lockManager,
	}

	if infra.DEBUG {
		log.Debugf("[glacier] add job [%s] to scheduler(%s)", name, plan)
	}

	return nil
}

func (c *schedulerImpl) Remove(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	reg, exist := c.jobs[name]
	if !exist {
		return errors.Errorf("[glacier] job with name [%s] not found", name)
	}

	if reg.lockManager != nil {
		if err := reg.lockManager.Release(context.TODO()); err != nil {
			log.Errorf("[glacier] cron job [%s] can not release lock: %v", name, err)
		}
	}

	delete(c.jobs, name)
	if !reg.Paused {
		c.cr.Remove(reg.ID)
	}

	if infra.DEBUG {
		log.Debugf("[glacier] remove job [%s] from scheduler", name)
	}

	return nil
}

func (c *schedulerImpl) Pause(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	reg, exist := c.jobs[name]
	if !exist {
		return errors.Errorf("[glacier] job with name [%s] not found", name)
	}

	if reg.Paused {
		return nil
	}

	c.cr.Remove(reg.ID)
	reg.Paused = true

	if infra.DEBUG {
		log.Debugf("[glacier] change job [%s] to paused", name)
	}

	return nil
}

func (c *schedulerImpl) Continue(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	reg, exist := c.jobs[name]
	if !exist {
		return errors.Errorf("[glacier] job with name [%s] not found", name)
	}

	if !reg.Paused {
		return nil
	}

	id, err := c.cr.AddFunc(reg.Plan, reg.handler)
	if err != nil {
		return errors.Wrap(err, "[glacier] change job from paused to continue failed")
	}

	reg.Paused = false
	reg.ID = id

	if infra.DEBUG {
		log.Debugf("[glacier] change job [%s] to continue", name)
	}

	return nil
}

func (c *schedulerImpl) Info(name string) (Job, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if job, ok := c.jobs[name]; ok {
		return *job, nil
	}

	return Job{}, fmt.Errorf("[glacier] job with name [%s] not found", name)
}

func (c *schedulerImpl) Start() {
	c.cr.Start()
}

func (c *schedulerImpl) Stop() {
	if c.lockManagerBuilder != nil {
		for _, job := range c.jobs {
			if job.lockManager != nil {
				if err := job.lockManager.Release(context.TODO()); err != nil {
					log.Errorf("[glacier] cron job [%s] can not release lock: %v", job.Name, err)
				}
			}
		}
	}

	c.cr.Stop()
}
