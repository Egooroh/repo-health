// Package app coordinates use cases through small interfaces owned by the consumer.
package app

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"repo-health/internal/domain"
	"repo-health/internal/scoring"
)

type Collector interface {
	Collect(context.Context, string, time.Time) (domain.Snapshot, error)
}
type Store interface {
	Enqueue(context.Context, string, bool, time.Time) (domain.Job, error)
	Claim(context.Context, time.Time) (domain.Job, error)
	Complete(context.Context, domain.Job, domain.Snapshot, domain.Report) error
	Fail(context.Context, domain.Job, string, bool) error
	Recover(context.Context) error
	GetJob(context.Context, string) (domain.Job, error)
	GetProject(context.Context, string) (domain.Project, error)
	ListProjects(context.Context, bool) ([]domain.Project, error)
}

type Service struct {
	store      Store
	live, demo Collector
	log        *slog.Logger
	wake       chan struct{}
	timeout    time.Duration
}

func New(store Store, live, demo Collector, log *slog.Logger) *Service {
	return &Service{store: store, live: live, demo: demo, log: log, wake: make(chan struct{}, 1), timeout: 2 * time.Minute}
}

func (s *Service) Analyze(ctx context.Context, input string, demo bool) (domain.Job, error) {
	ref, err := domain.ParseRef(input)
	if err != nil {
		return domain.Job{}, err
	}
	j, err := s.store.Enqueue(ctx, ref, demo, time.Now().UTC())
	if err == nil {
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
	return j, err
}

func (s *Service) Job(ctx context.Context, id string) (domain.Job, error) {
	return s.store.GetJob(ctx, id)
}
func (s *Service) Project(ctx context.Context, id string) (domain.Project, error) {
	p, err := s.store.GetProject(ctx, id)
	if p.Report != nil {
		p.Report.Eligible, p.Report.EligibilityReason = scoring.Eligibility(*p.Report, time.Now().UTC())
	}
	return p, err
}
func (s *Service) Projects(ctx context.Context, demo, leaderboard bool) ([]domain.Project, error) {
	projects, err := s.store.ListProjects(ctx, demo)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Project, 0, len(projects))
	now := time.Now().UTC()
	for _, p := range projects {
		if p.Report != nil {
			p.Report.Eligible, p.Report.EligibilityReason = scoring.Eligibility(*p.Report, now)
		}
		if !leaderboard || (p.Report != nil && p.Report.Eligible) {
			out = append(out, p)
		}
	}
	if leaderboard {
		sort.SliceStable(out, func(i, j int) bool {
			a, b := out[i].Report, out[j].Report
			if *a.Score != *b.Score {
				return *a.Score > *b.Score
			}
			if a.Coverage != b.Coverage {
				return a.Coverage > b.Coverage
			}
			return a.Repository.Ref < b.Repository.Ref
		})
	}
	return out, nil
}

// Run is the single durable-queue worker. Interrupted jobs are recovered at next start.
func (s *Service) Run(ctx context.Context) error {
	if err := s.store.Recover(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		j, err := s.store.Claim(ctx, time.Now().UTC())
		if err == nil {
			s.perform(ctx, j)
			continue
		}
		if !errors.Is(err, domain.ErrNotFound) && ctx.Err() == nil {
			s.log.Error("claim job", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-s.wake:
		}
	}
}

func (s *Service) perform(ctx context.Context, j domain.Job) {
	jobCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	collector := s.live
	if j.Demo {
		collector = s.demo
	}
	snapshot, err := collector.Collect(jobCtx, j.Ref, time.Now().UTC())
	// Keep running status on shutdown; Recover will retry without destroying an old report.
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		if saveErr := s.store.Fail(ctx, j, err.Error(), errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrNotPublic) || errors.Is(err, domain.ErrAccessDenied)); saveErr != nil {
			s.log.Error("persist job failure", "job", j.ID, "error", saveErr)
		}
		return
	}
	report := scoring.Calculate(snapshot)
	if err = s.store.Complete(ctx, j, snapshot, report); err != nil {
		s.log.Error("persist report", "job", j.ID, "error", err)
		if failErr := s.store.Fail(ctx, j, "Не удалось сохранить результат анализа. Повторите попытку.", false); failErr != nil {
			s.log.Error("persist failure", "error", failErr)
		}
	}
}
