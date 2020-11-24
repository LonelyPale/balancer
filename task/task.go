package task

import (
	"github.com/robfig/cron/v3"
)

var (
	crontab *cron.Cron
)

func Start(jobs ...*Job) error {
	if len(jobs) == 0 {
		return nil
	}

	crontab = cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),
	))

	for _, job := range jobs {
		eid, err := crontab.AddJob(job.spec, job)
		if err != nil {
			return err
		}

		cron.DefaultLogger.Info("Task EntryID: %d, %s\n", eid, job.spec)
	}

	crontab.Start()
	return nil
}

func Stop() {
	<-crontab.Stop().Done()
}
