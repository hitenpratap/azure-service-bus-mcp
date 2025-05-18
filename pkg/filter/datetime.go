package filter

import "time"

// InRange checks if t is between from and to (inclusive). Nil bounds are ignored.
func InRange(t time.Time, from, to *time.Time) bool {
	if from != nil && t.Before(*from) {
		return false
	}
	if to != nil && t.After(*to) {
		return false
	}
	return true
}
