package jobs

import (
	"github.com/revel/revel"
	"github.com/robfig/cron"
)

const DEFAULT_JOB_POOL_SIZE = 10

var (
	// Singleton instance of the underlying job scheduler.
	MainCron *cron.Cron

	// This limits the number of jobs allowed to run concurrently.
	workPermits chan struct{}

	// Is a single job allowed to run concurrently with itself?
	selfConcurrent bool
)

func init() {
	MainCron = cron.New()
	revel.OnAppStart(func() {
		if size := revel.Config.IntDefault("jobs.pool", DEFAULT_JOB_POOL_SIZE); size > 0 {
			workPermits = make(chan struct{}, size)
		}
		selfConcurrent = revel.Config.BoolDefault("jobs.selfconcurrent", false)
		MainCron.Start()
		revel.INFO.Print("Go to /@jobs to see job status.")
	})
}
