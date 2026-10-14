package cli

import (
	"testing"
	"time"
)

func TestQuietPeriodSkipsFreshSnapshot(t *testing.T) {
	s := &autoState{quiet: time.Hour}
	now := time.Now()
	if s.allow(now, now.Add(-time.Minute)) {
		t.Fatalf("snapshot 1m ago must be within quiet hour")
	}
	if !s.allow(now, now.Add(-2*time.Hour)) {
		t.Fatalf("snapshot 2h ago must allow")
	}
	if !s.allow(now, time.Time{}) {
		t.Fatalf("no prior snapshot must allow")
	}
}

func TestFailLogRateLimited(t *testing.T) {
	s := &autoState{}
	now := time.Now()
	if !s.shouldLogFail(now) {
		t.Fatalf("first failure must log")
	}
	if s.shouldLogFail(now.Add(10 * time.Second)) {
		t.Fatalf("second failure within a minute must not log")
	}
	if !s.shouldLogFail(now.Add(61 * time.Second)) {
		t.Fatalf("failure after a minute must log")
	}
}
