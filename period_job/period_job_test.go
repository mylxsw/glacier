package period_job_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mylxsw/container"
	"github.com/mylxsw/go-toolkit/period_job"
)

type DemoJob struct {
	lock    sync.Mutex
	counter int
}

func (job *DemoJob) Handle() {
	job.lock.Lock()
	defer job.lock.Unlock()

	job.counter++
}

func (job *DemoJob) Count() int {
	job.lock.Lock()
	defer job.lock.Unlock()

	return job.counter
}

func TestJob(t *testing.T) {
	cc := container.New()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	manager := period_job.NewManager(ctx, cc)

	job1 := &DemoJob{}
	job2 := &DemoJob{}

	manager.Run("Test", job1, 10*time.Millisecond)
	manager.Run("Test2", job2, 20*time.Millisecond)

	manager.Wait()

	if job1.Count() <= job2.Count() {
		t.Error("test failed")
	}

	if job1.Count() < 5 {
		t.Error("test failed")
	}

	if job2.Count() < 3 {
		t.Error("test failed")
	}
}
