package mqtt

import (
	"testing"
	"time"
)

func TestStartupBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 10 * time.Second},
		{attempt: 2, want: 20 * time.Second},
		{attempt: 3, want: 40 * time.Second},
		{attempt: 4, want: 60 * time.Second},
		{attempt: 5, want: 60 * time.Second},
	}

	for _, tc := range cases {
		got := startupBackoff(tc.attempt)
		if got != tc.want {
			t.Fatalf("attempt %d: expected %s, got %s", tc.attempt, tc.want, got)
		}
	}
}
