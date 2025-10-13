package filter

import (
	"testing"
	"time"
)

func TestInRange(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneHourLater := now.Add(1 * time.Hour)

	testCases := []struct {
		name     string
		t        time.Time
		from     *time.Time
		to       *time.Time
		expected bool
	}{
		{
			name:     "Time is within the range",
			t:        now,
			from:     &oneHourAgo,
			to:       &oneHourLater,
			expected: true,
		},
		{
			name:     "Time is before the from time",
			t:        oneHourAgo.Add(-1 * time.Millisecond),
			from:     &oneHourAgo,
			to:       &oneHourLater,
			expected: false,
		},
		{
			name:     "Time is after the to time",
			t:        oneHourLater.Add(1 * time.Millisecond),
			from:     &oneHourAgo,
			to:       &oneHourLater,
			expected: false,
		},
		{
			name:     "from is nil",
			t:        now,
			from:     nil,
			to:       &oneHourLater,
			expected: true,
		},
		{
			name:     "to is nil",
			t:        now,
			from:     &oneHourAgo,
			to:       nil,
			expected: true,
		},
		{
			name:     "Both from and to are nil",
			t:        now,
			from:     nil,
			to:       nil,
			expected: true,
		},
		{
			name:     "from and to are the same as the time being checked",
			t:        now,
			from:     &now,
			to:       &now,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := InRange(tc.t, tc.from, tc.to)
			if actual != tc.expected {
				t.Errorf("InRange(%v, %v, %v) = %v; expected %v", tc.t, tc.from, tc.to, actual, tc.expected)
			}
		})
	}
}
