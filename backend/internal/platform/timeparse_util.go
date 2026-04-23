package util

import (
	"strings"
	"time"
)

func ParseTimeBestEffort(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t
	}

	// Common variants
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"02/01/2006 15:04",
		"02-01-2006 15:04",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, raw); err == nil {
			return &t
		}
	}

	return nil
}
