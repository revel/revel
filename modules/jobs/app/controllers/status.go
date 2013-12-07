package controllers

import (
	"github.com/robfig/cron"
	"github.com/robfig/revel"
	"github.com/robfig/revel/modules/jobs/app/jobs"
	"strings"
)

type Jobs struct {
	*revel.Controller
}

func (c Jobs) Status() revel.Result {
	if _, ok := c.Request.Header["X-Forwarded-For"]; ok || !strings.HasPrefix(c.Request.RemoteAddr, "127.0.0.1:") {
		return c.Forbidden("%s is not local", c.Request.RemoteAddr)
	}
	entries := jobs.MainCron.Entries()
	return c.Render(entries)
}

func init() {
	revel.TemplateFuncs["castjob"] = func(job cron.Job) *jobs.Job {
		return job.(*jobs.Job)
	}
}
