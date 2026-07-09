package entity

import "time"

// ParseEstimateDuration parses a stored estimate_duration value into a duration.
func ParseEstimateDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}
