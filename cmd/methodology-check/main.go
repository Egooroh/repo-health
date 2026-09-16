// Command methodology-check verifies fixed synthetic scenarios and prints Markdown.
package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"repo-health/internal/methodology"
)

type expected struct {
	base, score  *float64
	coverage     float64
	status, gate string
	rank         bool
}
type scenario struct {
	name string
	edit func(*methodology.Snapshot)
	want expected
}

func number(n float64) *float64 { return &n }
func value(n float64) methodology.Value {
	return methodology.Value{State: methodology.Known, Score: number(n), Coverage: 1}
}
func display(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", *p)
}
func equal(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return math.Abs(*a-*b) < 1e-9
}
func main() {
	at := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	cases := []scenario{
		{"Полные публичные данные", func(s *methodology.Snapshot) {}, expected{number(90), number(90), 1, "complete", "", true}},
		{"Плохой CI", func(s *methodology.Snapshot) { s.Categories["ci"] = value(20) }, expected{number(72.5), number(72.5), 1, "complete", "", true}},
		{"Доступ к CI пропал", func(s *methodology.Snapshot) { s.Categories["ci"] = methodology.Unknown(methodology.Denied) }, expected{number(90), number(90), .75, "partial", "", false}},
		{"Только документация", func(s *methodology.Snapshot) {
			s.Categories = map[string]methodology.Value{"documentation": value(100)}
		}, expected{nil, nil, .2, "insufficient", "", false}},
		{"Часть Code Health неизвестна", func(s *methodology.Snapshot) { v := value(90); v.Coverage = .5; s.Categories["code"] = v }, expected{number(90), number(90), .875, "partial", "", false}},
		{"Security недоступен лично", func(s *methodology.Snapshot) { s.Profile = "personal"; s.SecurityState = methodology.Denied }, expected{number(90), number(90), .8, "partial", "", false}},
		{"Личный отчёт: критическая находка", func(s *methodology.Snapshot) {
			s.Profile = "personal"
			s.SecurityState = methodology.Known
			s.Findings = []methodology.Finding{{ID: "a", Severity: "critical", Status: "open"}}
		}, expected{number(80), number(59), 1, "complete", "critical_appsec", false}},
		{"Принятый критический риск", func(s *methodology.Snapshot) {
			s.Profile = "personal"
			s.SecurityState = methodology.Known
			s.Findings = []methodology.Finding{{ID: "a", Severity: "critical", Status: "accepted"}}
		}, expected{number(80), number(59), 1, "complete", "critical_appsec", false}},
		{"После ограничения потерян доступ", func(s *methodology.Snapshot) {
			s.Profile = "personal"
			s.SecurityState = methodology.Denied
			s.PreviousSecurityBlock = true
		}, expected{number(90), nil, .8, "blocked_unverified", "security_status_unverified", false}},
		{"Новое чистое сканирование", func(s *methodology.Snapshot) {
			s.Profile = "personal"
			s.SecurityState = methodology.Known
			s.PreviousSecurityBlock = true
		}, expected{number(92), number(92), 1, "complete", "", false}},
		{"Приватные поля не влияют на публичный отчёт", func(s *methodology.Snapshot) {
			s.SecurityState = "invalid"
			s.Findings = []methodology.Finding{{ID: "private", Severity: "invalid", Status: "invalid"}}
			s.PreviousSecurityBlock = true
		}, expected{number(90), number(90), 1, "complete", "", true}},
		{"Давность ровно 24 часа", func(s *methodology.Snapshot) { s.AsOf = s.At.Add(24 * time.Hour) }, expected{number(90), number(90), 1, "complete", "", true}},
		{"Давность превышена", func(s *methodology.Snapshot) { s.AsOf = s.At.Add(24*time.Hour + time.Second) }, expected{nil, nil, 1, "stale", "", false}},
	}
	fmt.Printf("# Контрольные сценарии %s\n\n", methodology.Version)
	fmt.Println("Синтетические нормализованные данные, не результаты анализа SourceCraft. Начальные оценки пяти публичных категорий — 90. Время снимка фиксировано: 2026-09-16 12:00 UTC.")
	fmt.Println()
	fmt.Println("| Сценарий | База | Итог | Полнота | Статус | Ограничение | В рейтинг | Проверка |")
	fmt.Println("| --- | ---: | ---: | ---: | --- | --- | --- | --- |")
	failures := 0
	for _, c := range cases {
		s := methodology.Snapshot{Profile: "public", At: at, AsOf: at, PublicRepository: true, Categories: map[string]methodology.Value{}}
		for _, id := range []string{"documentation", "ci", "code", "activity", "issues"} {
			s.Categories[id] = value(90)
		}
		c.edit(&s)
		r, err := methodology.Evaluate(s)
		ok := err == nil && equal(r.Base, c.want.base) && equal(r.Score, c.want.score) && math.Abs(r.Coverage-c.want.coverage) < 1e-9 && r.Status == c.want.status && r.Gate == c.want.gate && r.Rankable == c.want.rank
		check := "PASS"
		if !ok {
			check = "FAIL"
			failures++
			fmt.Fprintf(os.Stderr, "%s: got %+v; error %v\n", c.name, r, err)
		}
		fmt.Printf("| %s | %s | %s | %.2f%% | %s | %s | %t | %s |\n", c.name, display(r.Base), display(r.Score), 100*r.Coverage, r.Status, r.Gate, r.Rankable, check)
	}
	fmt.Printf("\nПроверено %d сценариев; ошибок: %d.\n", len(cases), failures)
	fmt.Println("\n## Контрпримеры исходного PDF")
	fmt.Println("\n| Проверка | Исходная формула | rhs-2.0-rc1 | Проверка |")
	fmt.Println("| --- | --- | --- | --- |")
	response, _ := methodology.Response(5, 10)
	backlog, _ := methodology.Backlog(0, 10)
	issues, _ := methodology.Issues(response, backlog)
	code, _ := methodology.Code(100, 1000, 40, 1000)
	oldIssues := .5*60 + (100.-0)*40
	oldTODOBefore := 100. - (1*2 + 1*50)
	oldTODOAfter := 100. - (10*2 + .1*50)
	if oldIssues != 4030 || !equal(issues.Score, number(70)) || oldTODOBefore != 48 || oldTODOAfter != 75 || !equal(code.Score, number(86.5)) {
		failures++
		fmt.Fprintln(os.Stderr, "counterexample mismatch")
	}
	fmt.Printf("| Единицы Issues | %.0f, выходит за 100 | %.2f при 5/10 ответов и отсутствии завала | %t |\n", oldIssues, *issues.Score, equal(issues.Score, number(70)))
	fmt.Printf("| Добавление 9 новых TODO к одному старому | %.0f → %.0f | %.2f → %.2f при неизменных структурных счётчиках | %t |\n", oldTODOBefore, oldTODOAfter, *code.Score, *code.Score, equal(code.Score, number(86.5)))
	fmt.Println("| Cap только категории Security | 86 при Security=30 и остальных=100 | Общий cap=59 проверен личным сценарием выше | проверено сценарием |")
	fmt.Println("\nПроверка TODO относится к арифметике нормализованных структурных счётчиков; автоматический парсер кода пока не подключён. В новом Issues используется своевременность ответа, а не доля закрытых задач. Численное совпадение 70 не означает эквивалентность этих метрик.")
	if failures > 0 {
		os.Exit(1)
	}
}
