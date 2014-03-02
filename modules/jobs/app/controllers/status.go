package controllers

import (
	"github.com/revel/revel"
	"github.com/revel/revel/modules/jobs/app/jobs"
	"github.com/robfig/cron"
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
