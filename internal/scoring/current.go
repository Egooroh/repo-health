package scoring

import (
	"errors"
	"fmt"
	"math"
	"time"

	"repo-health/internal/domain"
	"repo-health/internal/methodology"
)

const CurrentVersion = methodology.Version

// CalculateCurrent projects only verified collector facts into rhs-2.0-rc1.
// The old list-of-files and updated-at fields cannot establish the new
// documentation, response, CI or activity measurements. Unknown is explicit.
func CalculateCurrent(s domain.Snapshot, profile string) (domain.Report, error) {
	if profile != "public" && profile != "personal" {
		return domain.Report{}, errors.New("неизвестный профиль оценки")
	}
	values := map[string]methodology.Value{}
	states := map[string]methodology.State{}
	reasons := map[string]string{}
	set := func(id string, state methodology.State, reason string) {
		states[id] = state
		values[id] = methodology.Unknown(state)
		reasons[id] = reason
	}
	if s.Sources["files"].Available() {
		set("documentation", methodology.Incomplete, "Прочитан список файлов. Содержание, примеры и инструкции ещё не проверяются по правилам rhs-2.0-rc1; наличие README не доказывает полноту документации.")
	} else {
		set("documentation", sourceState(s.Sources["files"]), "Список файлов не получен полностью; содержание документации не проверено.")
	}
	if s.Sources["ci"].Available() {
		set("ci", methodology.Incomplete, "Есть история запусков, но не подтверждены основная ветка, ревизия, pipeline и первая попытка для каждой пары. Доля успешных запусков пока не рассчитывается.")
	} else {
		set("ci", sourceState(s.Sources["ci"]), "История CI недоступна или неполна.")
	}
	set("code", methodology.Missing, "Нет проверенных счётчиков сложности функций и дублирующихся строк. Исходный код не анализировался.")
	if s.Sources["pulls"].Available() {
		state := methodology.Incomplete
		if len(s.Pulls) == 0 {
			state = methodology.Insufficient
		}
		set("activity", state, "Список запросов на слияние получен, но нет проверенной истории содержательных коммитов и ответов сопровождающих. Пустой список не означает хорошую активность.")
	} else {
		set("activity", sourceState(s.Sources["pulls"]), "История содержательных изменений и ответов на предложения изменений не получена.")
	}
	if s.Sources["issues"].Available() {
		state := methodology.Incomplete
		if len(s.Issues) == 0 {
			state = methodology.Insufficient
		}
		set("issues", state, fmt.Sprintf("Получено issues: %d. Не проверены роли отвечавших и даты содержательных ответов. Отсутствие открытых задач не даёт 100 баллов.", len(s.Issues)))
	} else {
		set("issues", sourceState(s.Sources["issues"]), "Issues недоступны или список неполон; качество ответов не установлено.")
	}
	securityState := methodology.Missing
	if profile == "personal" {
		securityState = sourceState(s.Sources["appsec"])
		if securityState == methodology.Missing {
			reasons["security"] = "AppSec ещё не подключён. Отсутствие результата не означает отсутствие уязвимостей."
		} else {
			reasons["security"] = "Результат AppSec не получен и не подтверждён для текущей ревизии."
		}
		states["security"] = securityState
	}
	calculated, err := methodology.Evaluate(methodology.Snapshot{
		Profile: profile, At: s.CollectedAt, AsOf: s.CollectedAt,
		PublicRepository: s.Repository.Visibility == "public",
		Categories:       values, SecurityState: securityState,
	})
	if err != nil {
		return domain.Report{}, err
	}
	r := domain.Report{Repository: s.Repository, Demo: s.Demo, Version: CurrentVersion,
		AnalyzedAt: s.CollectedAt, Score: calculated.Score, Coverage: int(math.Round(100 * calculated.Coverage)),
		Status: calculated.Status, Sources: s.Sources,
		Categories: []domain.Category{}, Recommendations: []domain.Recommendation{},
	}
	for _, item := range []struct {
		id, title                    string
		publicWeight, personalWeight int
	}{
		{"documentation", "Документация", 20, 16},
		{"ci", "Надёжность CI", 25, 20},
		{"code", "Состояние кода", 25, 20},
		{"activity", "Активность проекта", 15, 12},
		{"issues", "Работа с задачами", 15, 12},
		{"security", "Безопасность", 0, 20},
	} {
		if item.id == "security" && profile == "public" {
			continue
		}
		weight := item.publicWeight
		if profile == "personal" {
			weight = item.personalWeight
		}
		v := calculated.Categories[item.id]
		state := states[item.id]
		if state == "" {
			state = v.State
		}
		c := domain.Category{ID: item.id, Title: item.title, Weight: weight, Score: v.Score,
			Metrics: []domain.Metric{{ID: item.id + "_evidence", Title: "Достаточность проверенных фактов", Weight: weight, Score: v.Score,
				State: string(state), Evidence: reasons[item.id]}},
		}
		r.Categories = append(r.Categories, c)
	}
	r.Eligible, r.EligibilityReason = CurrentEligibility(r, s.CollectedAt)
	if profile == "personal" {
		r.Eligible = false
		r.EligibilityReason = "Личный отчёт не публикуется в общем рейтинге."
	}
	return r, nil
}

func sourceState(s domain.SourceState) methodology.State {
	switch s.Status {
	case "truncated":
		return methodology.Incomplete
	case "unavailable":
		return methodology.Failed
	case "not_connected", "":
		return methodology.Missing
	default:
		return methodology.Incomplete
	}
}

func CurrentEligibility(r domain.Report, now time.Time) (bool, string) {
	if r.Version != CurrentVersion {
		return false, "Исторический отчёт рассчитан по прежней методике. Запустите анализ повторно."
	}
	if r.Repository.Visibility != "public" {
		return false, "Личный или закрытый репозиторий не публикуется в рейтинге."
	}
	if r.Repository.Empty {
		return false, "Пустой репозиторий нельзя оценить."
	}
	if r.Score == nil {
		return false, "Для rhs-2.0-rc1 пока недостаточно проверенных данных; отсутствие issues и запросов на слияние не считается успехом."
	}
	if now.Before(r.AnalyzedAt) || now.Sub(r.AnalyzedAt) > 24*time.Hour {
		return false, "Отчёт старше 24 часов; нужен новый анализ."
	}
	if r.Status != "complete" || r.Coverage != 100 {
		return false, "Для рейтинга нужен полный отчёт по публичным категориям rhs-2.0-rc1."
	}
	return true, "Полный актуальный публичный отчёт."
}
