// Package scoring is a deterministic, versioned interpretation of collected facts.
package scoring

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"repo-health/internal/domain"
)

const Version = "mvp-1.0"
const LeaderboardMaxAge = 7 * 24 * time.Hour

func Calculate(s domain.Snapshot) domain.Report {
	r := domain.Report{Repository: s.Repository, Demo: s.Demo, Version: Version,
		AnalyzedAt: s.CollectedAt, Sources: s.Sources, Recommendations: []domain.Recommendation{}}
	docs := domain.Category{ID: "documentation", Title: "Документация", Weight: 30}
	for _, check := range []struct {
		id, title, action string
		weight            int
	}{
		{"readme", "README", "Добавьте README с назначением проекта и инструкцией запуска.", 15},
		{"license", "Лицензия", "Укажите условия использования проекта в файле LICENSE или COPYING.", 10},
		{"contributing", "Правила участия", "Добавьте CONTRIBUTING с порядком внесения изменений.", 5},
	} {
		m := unknown(check.id, check.title, check.weight, sourceDetail(s, "files"))
		if s.Sources["files"].Available() {
			found := false
			for _, path := range s.Files {
				name := strings.ToLower(path)
				name = strings.TrimPrefix(name, ".sourcecraft/")
				name = strings.TrimPrefix(name, ".github/")
				name = strings.TrimPrefix(name, "docs/")
				if matchDocument(name, check.id) {
					found = true
				}
			}
			value := 0.0
			evidence := "Файл не найден в проверенных каталогах: корень, docs, .sourcecraft, .github."
			if found {
				value = 100
				evidence = "Файл найден. Содержимое и качество текста в MVP не оцениваются."
			}
			m = measured(check.id, check.title, check.weight, value, evidence)
			if !found {
				r.Recommendations = append(r.Recommendations, domain.Recommendation{MetricID: check.id, Priority: "medium", Title: "Добавить: " + check.title, Reason: evidence, Action: check.action})
			}
		}
		docs.Metrics = append(docs.Metrics, m)
	}
	ci := unknown("ci_success", "Успешные запуски за 30 дней", 30, sourceDetail(s, "ci"))
	if s.Sources["ci"].Available() {
		total, success := 0, 0
		invalid := false
		for _, run := range s.Runs {
			if run.Status != "success" && run.Status != "failed" && run.Status != "timeout" {
				continue
			}
			if run.FinishedAt.IsZero() || run.FinishedAt.After(s.CollectedAt) {
				invalid = true
				continue
			}
			if run.FinishedAt.Before(s.CollectedAt.Add(-30 * 24 * time.Hour)) {
				continue
			}
			total++
			if run.Status == "success" {
				success++
			}
		}
		if invalid {
			ci.Evidence = "В истории CI есть завершённые запуски без корректного времени завершения."
		} else if total > 0 {
			value := 100 * float64(success) / float64(total)
			ci = measured(ci.ID, ci.Title, ci.Weight, value, fmt.Sprintf("За 30 дней: %d успешных из %d завершённых запусков (success, failed, timeout).", success, total))
			if value < 90 {
				r.Recommendations = append(r.Recommendations, domain.Recommendation{MetricID: ci.ID, Priority: "high", Title: "Разобрать сбои CI", Reason: ci.Evidence, Action: "Откройте неуспешные запуски в SourceCraft, установите причины и повторите проверки после исправления."})
			}
		} else {
			ci.Evidence = "За последние 30 дней нет завершённых запусков с результатом success, failed или timeout."
		}
	}
	issues := backlog(s, "issues", "issue_backlog", "Актуальность открытых issues", 20, 30)
	pulls := backlog(s, "pulls", "pull_backlog", "Актуальность запросов на слияние", 10, 14)
	for _, m := range []domain.Metric{issues, pulls} {
		if m.Score != nil && *m.Score < 100 {
			r.Recommendations = append(r.Recommendations, domain.Recommendation{MetricID: m.ID, Priority: "medium", Title: "Разобрать задачи без обновлений", Reason: m.Evidence, Action: "Проверьте актуальность незавершённых задач, уточните ответственных и следующий шаг. Закрывайте только действительно завершённые или ненужные задачи."})
		}
	}
	r.Categories = []domain.Category{docs,
		{ID: "ci", Title: "Надёжность CI", Weight: 30, Metrics: []domain.Metric{ci}},
		{ID: "maintenance", Title: "Поддержка проекта", Weight: 20, Metrics: []domain.Metric{issues}},
		{ID: "collaboration", Title: "Совместная разработка", Weight: 10, Metrics: []domain.Metric{pulls}},
		{ID: "security", Title: "Безопасность", Weight: 10, Metrics: []domain.Metric{unknown("appsec", "Результаты AppSec", 10, "AppSec не подключён. Состояние безопасности неизвестно.")}},
	}
	weighted, known := 0.0, 0
	for i := range r.Categories {
		c := &r.Categories[i]
		sum, cw := 0.0, 0
		for _, m := range c.Metrics {
			if m.Score != nil {
				sum += *m.Score * float64(m.Weight)
				cw += m.Weight
			}
		}
		if cw > 0 {
			v := round(sum / float64(cw))
			c.Score = &v
			weighted += sum
			known += cw
		}
	}
	r.Coverage = known
	if known > 0 && !s.Repository.Empty {
		v := round(weighted / float64(known))
		r.Score = &v
	}
	sort.SliceStable(r.Recommendations, func(i, j int) bool {
		return r.Recommendations[i].Priority == "high" && r.Recommendations[j].Priority != "high"
	})
	r.Eligible, r.EligibilityReason = Eligibility(r, s.CollectedAt)
	return r
}

func Eligibility(r domain.Report, now time.Time) (bool, string) {
	if r.Repository.Empty {
		return false, "Пустой репозиторий: недостаточно данных для оценки."
	}
	if r.Score == nil {
		return false, "Недостаточно данных для расчёта."
	}
	if r.Coverage < 60 {
		return false, "Для рейтинга нужна полнота данных не менее 60%."
	}
	if r.Version != Version {
		return false, "Нужен пересчёт по текущей методике."
	}
	if now.Sub(r.AnalyzedAt) > LeaderboardMaxAge {
		return false, "Анализ старше 7 дней: обновите оценку."
	}
	return true, "Достаточно данных для участия в рейтинге."
}

func backlog(s domain.Snapshot, source, id, title string, weight, days int) domain.Metric {
	m := unknown(id, title, weight, sourceDetail(s, source))
	if !s.Sources[source].Available() {
		return m
	}
	items := s.Issues
	if source == "pulls" {
		items = s.Pulls
	}
	open, stale := 0, 0
	for _, item := range items {
		active := false
		if source == "issues" {
			switch item.Status {
			case "initial", "in_progress", "paused":
				active = true
			case "completed", "cancelled":
				continue
			default:
				m.Evidence = "Источник вернул неизвестный статус issue."
				return m
			}
		} else {
			switch item.Status {
			case "open", "merging":
				active = true
			case "draft", "discarded", "merged":
				continue
			default:
				m.Evidence = "Источник вернул неизвестный статус запроса на слияние."
				return m
			}
		}
		if active {
			if item.UpdatedAt.IsZero() || item.UpdatedAt.After(s.CollectedAt) {
				m.Evidence = "Недостаточно корректных дат обновления задач."
				return m
			}
			open++
			if item.UpdatedAt.Before(s.CollectedAt.Add(-time.Duration(days) * 24 * time.Hour)) {
				stale++
			}
		}
	}
	value := 100.0
	if open > 0 {
		value = 100 * (1 - float64(stale)/float64(open))
	}
	return measured(id, title, weight, value, fmt.Sprintf("Незавершённых: %d; без обновлений более %d дней: %d. Это показатель очереди задач, а не частоты коммитов.", open, days, stale))
}

func matchDocument(name, id string) bool {
	if strings.Contains(name, "/") {
		return false
	}
	bases := []string{id}
	if id == "license" {
		bases = []string{"license", "licence", "copying"}
	}
	for _, base := range bases {
		for _, ext := range []string{"", ".md", ".markdown", ".rst", ".txt"} {
			if name == base+ext {
				return true
			}
		}
	}
	return false
}
func sourceDetail(s domain.Snapshot, key string) string {
	if state, ok := s.Sources[key]; ok && state.Detail != "" {
		return state.Detail
	}
	return "Данные источника недоступны."
}
func unknown(id, title string, weight int, evidence string) domain.Metric {
	return domain.Metric{ID: id, Title: title, Weight: weight, State: "unknown", Evidence: evidence}
}
func measured(id, title string, weight int, value float64, evidence string) domain.Metric {
	value = round(value)
	state := "pass"
	if value < 80 {
		state = "warning"
	}
	if value < 50 {
		state = "fail"
	}
	return domain.Metric{ID: id, Title: title, Weight: weight, Score: &value, State: state, Evidence: evidence}
}
func round(v float64) float64 { return math.Round(v*10) / 10 }
