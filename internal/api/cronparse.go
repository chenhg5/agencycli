package api

import (
	"fmt"

	"github.com/robfig/cron/v3"
)

// standardCronParser matches agencycli CLI 5-field crontab (+ descriptors).
var standardCronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func validateCronSchedule(expr string) error {
	if _, err := standardCronParser.Parse(expr); err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", expr, err)
	}
	return nil
}
