package scheduler

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/container"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

// JobCreator is a creator for cron job
type JobCreator interface {
	// Add add a cron job
	Add(name string, plan string, handler interface{}) error
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

	// Start start cron manager
	Start()
	// Stop stop cron job manager
	Stop()

	// DistributeLockManager is a setter method for distribute lock manager
	DistributeLockManager(lockManager DistributeLockManager)
}

// DistributeLockManager is a distribute lock manager interface
type DistributeLockManager interface {
	// TryLock try to get lock
	// this method will be called every 60s
	// you should set a ttl for lock since unlock method may be not be called in some case
	TryLock() error
	// TryUnlock try to release the lock
	TryUnLock() error
	// HasLock return whether manager has lock
	HasLock() bool
}

type schedulerImpl struct {
	lock sync.RWMutex
	cc   container.Container
	cr   *cron.Cron

	distributeLockManager DistributeLockManager

	jobs   map[string]*Job
	logger log.Logger
}

// Job is a job object
type Job struct {
	ID      cron.EntryID
	Name    string
	Plan    string
	handler func()
	Paused  bool
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
func NewManager(cc container.Container, logger log.Logger) Scheduler {
	m := schedulerImpl{cc: cc, jobs: make(map[string]*Job), logger: logger}
	cc.MustResolve(func(cr *cron.Cron) { m.cr = cr })

	return &m
}

func (c *schedulerImpl) DistributeLockManager(lockManager DistributeLockManager) {
	c.distributeLockManager = lockManager
}

func (c *schedulerImpl) Add(name string, plan string, handler interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if reg, existed := c.jobs[name]; existed {
		return fmt.Errorf("job with name [%s] has existed: %d | %s", name, reg.ID, reg.Plan)
	}

	hh, ok := handler.(JobHandler)
	if !ok {
		hh = newHandler(handler)
	}

	jobHandler := func() {
		if c.distributeLockManager != nil && !c.distributeLockManager.HasLock() {
			if c.logger.DebugEnabled() {
				c.logger.Debugf("cron job [%s] can not start because it doesn't get the lock", name)
			}
			return
		}

		if c.logger.DebugEnabled() {
			c.logger.Debugf("cron job [%s] running", name)
		}
		startTs := time.Now()
		defer func() {
			if err := recover(); err != nil {
				c.logger.Errorf("cron job [%s] stopped with some errors: %v, elapse %s", name, err, time.Now().Sub(startTs))
			} else {
				if c.logger.DebugEnabled() {
					c.logger.Debugf("cron job [%s] stopped, elapse %s", name, time.Now().Sub(startTs))
				}
			}
		}()
		if err := c.cc.ResolveWithError(hh.Handle); err != nil {
			c.logger.Errorf("cron job [%s] failed, Err: %v, Stack: \n%s", name, err, debug.Stack())
		}
	}
	id, err := c.cr.AddFunc(plan, jobHandler)

	if err != nil {
		return errors.Wrap(err, "add cron job failed")
	}

	c.jobs[name] = &Job{
		ID:      id,
		Name:    name,
		Plan:    plan,
		handler: jobHandler,
		Paused:  false,
	}
	if c.logger.DebugEnabled() {
		c.logger.Debugf("add job [%s] to cron manager with plan %s", name, plan)
	}

	return nil
}

func (c *schedulerImpl) Remove(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	reg, exist := c.jobs[name]
	if !exist {
		return errors.Errorf("job with name [%s] not found", name)
	}

	delete(c.jobs, name)
	if !reg.Paused {
		c.cr.Remove(reg.ID)
	}

	if c.logger.DebugEnabled() {
		c.logger.Debugf("remove job [%s] from cron manager", name)
	}
	return nil
}

func (c *schedulerImpl) Pause(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	reg, exist := c.jobs[name]
	if !exist {
		return errors.Errorf("job with name [%s] not found", name)
	}

	if reg.Paused {
		return nil
	}

	c.cr.Remove(reg.ID)
	reg.Paused = true

	if c.logger.DebugEnabled() {
		c.logger.Debugf("change job [%s] to paused", name)
	}

	return nil
}

func (c *schedulerImpl) Continue(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	reg, exist := c.jobs[name]
	if !exist {
		return errors.Errorf("job with name [%s] not found", name)
	}

	if !reg.Paused {
		return nil
	}

	id, err := c.cr.AddFunc(reg.Plan, reg.handler)
	if err != nil {
		return errors.Wrap(err, "change job from paused to continue failed")
	}

	reg.Paused = false
	reg.ID = id

	if c.logger.DebugEnabled() {
		c.logger.Debugf("change job [%s] to continue", name)
	}

	return nil
}

func (c *schedulerImpl) Info(name string) (Job, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if job, ok := c.jobs[name]; ok {
		return *job, nil
	}

	return Job{}, fmt.Errorf("job with name [%s] not found", name)
}

func (c *schedulerImpl) Start() {
	if c.distributeLockManager != nil {
		getDistributeLock := func() {
			if err := c.distributeLockManager.TryLock(); err != nil {
				if c.logger.WarningEnabled() {
					c.logger.Warningf("try to get distribute lock failed: %v", err)
				}
			}
		}

		getDistributeLock()
		if _, err := c.cr.AddFunc("@every 60s", getDistributeLock); err != nil {
			c.logger.Errorf("initialize cron failed: can not create distribute lock task: %v", err)
		}
	}

	c.cr.Start()
}

func (c *schedulerImpl) Stop() {
	c.cr.Stop()
	if c.distributeLockManager != nil {
		if err := c.distributeLockManager.TryUnLock(); err != nil {
			if c.logger.WarningEnabled() {
				c.logger.Warningf("try to release distribute lock failed: %v", err)
			}
		}
	}
}
