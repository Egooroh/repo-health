package scoring

import (
	"strings"
	"testing"
	"time"

	"repo-health/internal/domain"
	"repo-health/internal/methodology"
)

func TestCurrentMethodDoesNotRewardEmptyIssuesOrPulls(t *testing.T) {
	at := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	s := domain.Snapshot{Repository: domain.Repository{Ref: "l0bster007/messenger", Visibility: "public"}, CollectedAt: at,
		Sources: map[string]domain.SourceState{"files": {Status: "available"}, "issues": {Status: "available"}, "pulls": {Status: "available"}, "ci": {Status: "unavailable"}, "appsec": {Status: "not_connected"}},
		Files:   []string{"README.md"}, Issues: []domain.WorkItem{}, Pulls: []domain.WorkItem{},
	}
	r, err := CalculateCurrent(s, "public")
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != methodology.Version || r.Score != nil || r.Status != "insufficient" || r.Coverage != 0 || r.Eligible {
		t.Fatalf("unexpected score from empty cohorts: %+v", r)
	}
	if len(r.Categories) != 5 {
		t.Fatal("security included in public profile")
	}
	if r.Categories[4].Metrics[0].State != "insufficient" || r.Categories[3].Metrics[0].State != "insufficient" {
		t.Fatal("empty lists did not produce insufficient states")
	}
	if !strings.Contains(r.Categories[0].Metrics[0].Evidence, "README") {
		t.Fatal("documentation limitation not explained")
	}
	if ok, _ := CurrentEligibility(r, at); ok {
		t.Fatal("incomplete candidate entered ranking")
	}
	private, err := CalculateCurrent(s, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if len(private.Categories) != 6 || private.Categories[5].ID != "security" || private.Score != nil || private.Eligible {
		t.Fatal("personal profile claimed AppSec or public rank")
	}
}

func TestCurrentEligibilityRejectsHistoricalReports(t *testing.T) {
	now := time.Now().UTC()
	r := domain.Report{Version: Version, Score: ptr(100), Coverage: 90, AnalyzedAt: now, Repository: domain.Repository{Visibility: "public"}}
	if ok, reason := CurrentEligibility(r, now); ok || !strings.Contains(reason, "прежней") {
		t.Fatalf("historical report allowed: %s", reason)
	}
	r.Version = CurrentVersion
	r.Status = "complete"
	r.Coverage = 100
	if ok, _ := CurrentEligibility(r, now); !ok {
		t.Fatal("complete current report rejected")
	}
	if ok, _ := CurrentEligibility(r, now.Add(25*time.Hour)); ok {
		t.Fatal("stale current report allowed")
	}
	r.Repository.Visibility = "private"
	if ok, _ := CurrentEligibility(r, now); ok {
		t.Fatal("private report allowed")
	}
}

func ptr(v float64) *float64 { return &v }
