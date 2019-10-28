/**
Package period_job 实现了周期性任务的定时触发执行。

	manager := period_job.NewManager(ctx, cc)

	job1 := &DemoJob{}
	job2 := &DemoJob{}

	manager.Run("Test", job1, 10*time.Millisecond)
	manager.Run("Test2", job2, 20*time.Millisecond)

	manager.Wait()

*/
package period_job

