package storage

import (
	"context"
	"errors"
	"path/filepath"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"repo-health/internal/scoring"
	"testing"
	"time"
)

func TestDurableQueueReportAndInvalidation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "health.db")
	now := time.Now().UTC()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.Enqueue(ctx, "demo/atlas", true, now)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := s.Enqueue(ctx, "demo/atlas", true, now)
	if err != nil || duplicate.ID != j.ID {
		t.Fatal("active job was duplicated", err)
	}
	claimed, err := s.Claim(ctx, now)
	if err != nil || claimed.ID != j.ID {
		t.Fatal("claim", err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	claimed, err = s.Claim(ctx, now)
	if err != nil || claimed.ID != j.ID {
		t.Fatal("lost interrupted job", err)
	}
	facts, _ := (demo.Collector{}).Collect(ctx, "demo/atlas", now)
	if err = s.Complete(ctx, claimed, facts, scoring.Calculate(facts)); err != nil {
		t.Fatal(err)
	}
	p, err := s.GetProject(ctx, j.RepositoryID)
	if err != nil || p.Report == nil || p.Job.Status != "completed" {
		t.Fatal("report missing", err)
	}
	live, err := s.ListProjects(ctx, false)
	if err != nil || len(live) != 0 {
		t.Fatal("demo leaked into real catalogue", err)
	}
	if _, err = s.Enqueue(ctx, "demo/atlas", true, now); !errors.Is(err, domain.ErrCooldown) {
		t.Fatal("cooldown missing", err)
	}
	retry, err := s.Enqueue(ctx, "demo/atlas", true, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Fail(ctx, retry, "temporarily unavailable", false); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetProject(ctx, j.RepositoryID)
	if p.Report == nil {
		t.Fatal("transient failure destroyed prior result")
	}
	if err = s.Fail(ctx, retry, "no longer public", true); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetProject(ctx, j.RepositoryID)
	if p.Report != nil {
		t.Fatal("private result remains visible")
	}
}
