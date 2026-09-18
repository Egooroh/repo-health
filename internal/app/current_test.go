package app_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"repo-health/internal/app"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"repo-health/internal/scoring"
	"repo-health/internal/storage"
)

type publicFixture struct{}

func (publicFixture) Collect(_ context.Context, ref string, now time.Time) (domain.Snapshot, error) {
	return domain.Snapshot{Repository: domain.Repository{Ref: ref, Visibility: "public", DefaultBranch: "main"}, CollectedAt: now,
		Sources: map[string]domain.SourceState{"files": {Status: "available"}, "issues": {Status: "available"}, "pulls": {Status: "available"}, "ci": {Status: "unavailable"}, "appsec": {Status: "not_connected"}},
		Files:   []string{"README.md"}, Issues: []domain.WorkItem{}, Pulls: []domain.WorkItem{}, Runs: []domain.CIRun{},
	}, nil
}

func TestWorkerPersistsCurrentMethodWithoutFictitiousScore(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := app.New(store, publicFixture{}, demo.Collector{}, log)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	job, err := service.Analyze(ctx, "l0bster007/messenger", false)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		project, err := service.Project(ctx, job.RepositoryID)
		if err != nil {
			t.Fatal(err)
		}
		if project.Job != nil && project.Job.Status == "completed" {
			if project.Report == nil || project.Report.Version != scoring.CurrentVersion || project.Report.Score != nil || project.Report.Coverage != 0 || project.Report.Eligible {
				t.Fatalf("unexpected report: %+v", project.Report)
			}
			items, err := service.Projects(ctx, false, true)
			if err != nil || len(items) != 0 {
				t.Fatalf("incomplete result in ranking: %+v %v", items, err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
