// Package demo provides explicitly fictional, offline examples in a separate catalogue.
package demo

import (
	"context"
	"fmt"
	"repo-health/internal/domain"
	"time"
)

var Refs = []string{"demo/atlas", "demo/pulse", "demo/seed"}

type Collector struct{}

func (Collector) Collect(ctx context.Context, ref string, now time.Time) (domain.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.Snapshot{}, err
	}
	s := domain.Snapshot{Demo: true, CollectedAt: now, Repository: domain.Repository{Ref: ref, DefaultBranch: "main", Language: "Go"},
		Sources: map[string]domain.SourceState{}, Files: []string{}, Issues: []domain.WorkItem{}, Pulls: []domain.WorkItem{}, Runs: []domain.CIRun{}}
	for _, key := range []string{"files", "issues", "pulls", "ci"} {
		s.Sources[key] = domain.SourceState{Status: "available", Detail: "Вымышленные данные демонстрационного примера."}
	}
	s.Sources["appsec"] = domain.SourceState{Status: "not_connected", Detail: "AppSec не подключён."}
	switch ref {
	case "demo/atlas":
		s.Repository.Description = "Вымышленная библиотека для работы с данными. Пример проекта с устойчивыми проверками и понятной документацией."
		s.Files = []string{"README.md", "LICENSE", "CONTRIBUTING.md"}
		for i := 0; i < 20; i++ {
			status := "success"
			if i == 1 {
				status = "failed"
			}
			s.Runs = append(s.Runs, domain.CIRun{Status: status, FinishedAt: now.Add(-time.Duration(i+1) * 12 * time.Hour)})
		}
		s.Issues = []domain.WorkItem{{Status: "in_progress", UpdatedAt: now.Add(-24 * time.Hour)}, {Status: "initial", UpdatedAt: now.Add(-72 * time.Hour)}}
		s.Pulls = []domain.WorkItem{{Status: "open", UpdatedAt: now.Add(-48 * time.Hour)}}
	case "demo/pulse":
		s.Repository.Description = "Вымышленный сервис мониторинга. Пример с нестабильным CI и задачами, которым требуется внимание."
		s.Repository.Language = "TypeScript"
		s.Files = []string{"README.md"}
		for i := 0; i < 10; i++ {
			status := "failed"
			if i < 6 {
				status = "success"
			}
			s.Runs = append(s.Runs, domain.CIRun{Status: status, FinishedAt: now.Add(-time.Duration(i+1) * 24 * time.Hour)})
		}
		s.Issues = []domain.WorkItem{{Status: "initial", UpdatedAt: now.Add(-50 * 24 * time.Hour)}, {Status: "initial", UpdatedAt: now.Add(-60 * 24 * time.Hour)}, {Status: "in_progress", UpdatedAt: now.Add(-48 * time.Hour)}, {Status: "completed", UpdatedAt: now.Add(-90 * 24 * time.Hour)}}
		s.Pulls = []domain.WorkItem{{Status: "open", UpdatedAt: now.Add(-21 * 24 * time.Hour)}, {Status: "open", UpdatedAt: now.Add(-24 * time.Hour)}}
	case "demo/seed":
		s.Repository.Description = "Вымышленный новый репозиторий. Показывает, как сервис ведёт себя, когда сведений недостаточно."
		s.Repository.Empty = true
		s.Sources["ci"] = domain.SourceState{Status: "unavailable", Detail: "Для этого примера история CI недоступна."}
	default:
		return domain.Snapshot{}, fmt.Errorf("%w: демонстрационный пример", domain.ErrNotFound)
	}
	return s, nil
}
