package watcher

import (
	"github.com/robfig/cron/v3"
)

// StandardParser is a cron parser that supports standard 5-field cron expressions
// plus descriptors like @hourly, @daily, etc.
var StandardParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// ParseSchedule parses a cron schedule expression and returns the schedule.
func ParseSchedule(expr string) (cron.Schedule, error) {
	return StandardParser.Parse(expr)
}
