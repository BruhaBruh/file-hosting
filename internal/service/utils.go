package service

import "time"

const defaultFileDuration = time.Hour

func parseDuration(raw string, allowInfinite ...bool) time.Duration {
	isAllowedInfinite := false
	if len(allowInfinite) > 0 {
		isAllowedInfinite = allowInfinite[0]
	}

	if raw == "-1" && isAllowedInfinite {
		return 0
	} else if raw == "5m" {
		return 5 * time.Minute
	} else if raw == "30m" {
		return 30 * time.Minute
	} else if raw == "1h" || raw == "60m" {
		return time.Hour
	} else if raw == "1d" || raw == "24h" {
		return 24 * time.Hour
	} else if raw == "1w" || raw == "7d" {
		return 7 * 24 * time.Hour
	}

	return defaultFileDuration
}
