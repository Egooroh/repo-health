package scoring

import (
	"context"
	"reflect"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"testing"
	"time"
)

var now = time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)

func TestUnavailableIsNotZeroOrSuccess(t *testing.T) {
	s := domain.Snapshot{CollectedAt: now, Repository: domain.Repository{Ref: "a/b"}, Sources: map[string]domain.SourceState{"files": {Status: "available"}}, Files: []string{"README.md", "LICENSE", "docs/CONTRIBUTING.md"}}
	r := Calculate(s)
	if r.Score == nil || *r.Score != 100 || r.Coverage != 30 || r.Eligible {
		t.Fatalf("unexpected incomplete result: %+v", r)
	}
	for _, c := range r.Categories[1:] {
		if c.Score != nil {
			t.Errorf("unknown %s scored", c.ID)
		}
	}
	s.Sources["files"] = domain.SourceState{Status: "unavailable"}
	r = Calculate(s)
	if r.Score != nil || r.Coverage != 0 {
		t.Fatalf("no data should not score: %+v", r)
	}
}

func TestDeterministicDemoAndEmpty(t *testing.T) {
	s, err := (demo.Collector{}).Collect(context.Background(), "demo/atlas", now)
	if err != nil {
		t.Fatal(err)
	}
	a, b := Calculate(s), Calculate(s)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("non-deterministic score")
	}
	if a.Score == nil || *a.Score != 98.3 || a.Coverage != 90 || !a.Eligible {
		t.Fatalf("unexpected atlas: %+v", a)
	}
	s.Repository.Empty = true
	empty := Calculate(s)
	if empty.Score != nil || empty.Eligible {
		t.Fatal("empty repository was scored")
	}
}

func TestCIWindowAndBacklogStates(t *testing.T) {
	s := domain.Snapshot{CollectedAt: now, Sources: map[string]domain.SourceState{"ci": {Status: "available"}, "issues": {Status: "available"}, "pulls": {Status: "available"}},
		Runs:   []domain.CIRun{{Status: "success", FinishedAt: now.Add(-time.Hour)}, {Status: "failed", FinishedAt: now.Add(-time.Hour)}, {Status: "success", FinishedAt: now.Add(-31 * 24 * time.Hour)}, {Status: "canceled", FinishedAt: now.Add(-time.Hour)}},
		Issues: []domain.WorkItem{{Status: "initial", UpdatedAt: now.Add(-31 * 24 * time.Hour)}, {Status: "completed", UpdatedAt: now.Add(-90 * 24 * time.Hour)}, {Status: "in_progress", UpdatedAt: now.Add(-time.Hour)}},
		Pulls:  []domain.WorkItem{{Status: "draft", UpdatedAt: now.Add(-60 * 24 * time.Hour)}, {Status: "open", UpdatedAt: now.Add(-time.Hour)}}}
	r := Calculate(s)
	for index, want := range map[int]float64{1: 50, 2: 50, 3: 100} {
		if r.Categories[index].Score == nil || *r.Categories[index].Score != want {
			t.Errorf("category %d: %+v", index, r.Categories[index])
		}
	}
	s.Issues[0].UpdatedAt = time.Time{}
	if Calculate(s).Categories[2].Score != nil {
		t.Fatal("invalid date should exclude metric")
	}
	s.Sources["ci"] = domain.SourceState{Status: "truncated"}
	if Calculate(s).Categories[1].Score != nil {
		t.Fatal("truncated CI should not score")
	}
}

func TestEligibilityRefreshAndVersion(t *testing.T) {
	s, _ := (demo.Collector{}).Collect(context.Background(), "demo/atlas", now)
	r := Calculate(s)
	if ok, _ := Eligibility(r, now.Add(8*24*time.Hour)); ok {
		t.Fatal("stale report admitted")
	}
	r.Version = "old"
	if ok, _ := Eligibility(r, now); ok {
		t.Fatal("old methodology admitted")
	}
}
