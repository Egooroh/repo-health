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

func TestEnqueueStale(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC()
	old := now.Add(-25 * time.Hour)
	fresh := now.Add(-time.Hour)

	mustComplete := func(ref string, demo bool, createdAt time.Time) domain.Job {
		t.Helper()
		j, err := s.Enqueue(ctx, ref, demo, createdAt)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Complete(ctx, j, domain.Snapshot{}, domain.Report{}); err != nil {
			t.Fatal(err)
		}
		return j
	}

	staleJob := mustComplete("org/stale", false, old)
	freshJob := mustComplete("org/fresh", false, fresh)
	demoJob := mustComplete("demo/atlas", true, old)
	activeJob, err := s.Enqueue(ctx, "org/active", false, old)
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.EnqueueStale(ctx, now, 24*time.Hour, 10)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected exactly one repository requeued, got %d", n)
	}
	if p, err := s.GetProject(ctx, staleJob.RepositoryID); err != nil || p.Job.Status != "queued" || p.Job.ID == staleJob.ID {
		t.Fatal("stale repository was not requeued", err)
	}
	if p, err := s.GetProject(ctx, freshJob.RepositoryID); err != nil || p.Job.ID != freshJob.ID {
		t.Fatal("recently analyzed repository should not be requeued", err)
	}
	if p, err := s.GetProject(ctx, demoJob.RepositoryID); err != nil || p.Job.ID != demoJob.ID {
		t.Fatal("demo repository should not be auto-refreshed", err)
	}
	if p, err := s.GetProject(ctx, activeJob.RepositoryID); err != nil || p.Job.ID != activeJob.ID {
		t.Fatal("repository with an active job should not be double-queued", err)
	}
}
