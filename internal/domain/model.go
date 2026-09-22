package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("не найдено")
	ErrNotPublic    = errors.New("репозиторий не является открытым")
	ErrAccessDenied = errors.New("нет доступа к данным репозитория")
	ErrQueueFull    = errors.New("очередь заполнена, повторите позже")
	ErrCooldown     = errors.New("повторный анализ доступен через 30 секунд")
)

var slugPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,255}$`)

func ParseRef(input string) (string, error) {
	s := strings.TrimSpace(input)
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" ||
			(u.Hostname() != "sourcecraft.dev" && u.Hostname() != "git.sourcecraft.dev") ||
			u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
			return "", errors.New("укажите HTTPS-адрес репозитория SourceCraft или организация/репозиторий")
		}
		s = strings.Trim(u.Path, "/")
	}
	s = strings.TrimSuffix(s, ".git")
	parts := strings.Split(s, "/")
	if len(parts) != 2 || !slugPattern.MatchString(parts[0]) || !slugPattern.MatchString(parts[1]) {
		return "", errors.New("нужен адрес вида организация/репозиторий без дополнительных разделов")
	}
	return strings.ToLower(s), nil
}

func RepositoryID(ref string, demo bool) string {
	if demo {
		ref = "demo:" + ref
	}
	hash := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(hash[:16])
}

type SourceState struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func (s SourceState) Available() bool { return s.Status == "available" }

type Repository struct {
	Ref           string `json:"ref"`
	Description   string `json:"description"`
	URL           string `json:"url"`
	Language      string `json:"language"`
	DefaultBranch string `json:"default_branch"`
	Revision      string `json:"revision,omitempty"`
	Empty         bool   `json:"empty"`
	Fork          bool   `json:"fork"`
	Visibility    string `json:"visibility,omitempty"`
}

type WorkItem struct {
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CIRun struct {
	Status     string    `json:"status"`
	FinishedAt time.Time `json:"finished_at"`
}

type Snapshot struct {
	Repository  Repository             `json:"repository"`
	CollectedAt time.Time              `json:"collected_at"`
	Demo        bool                   `json:"demo"`
	Sources     map[string]SourceState `json:"sources"`
	Files       []string               `json:"files"`
	Issues      []WorkItem             `json:"issues"`
	Pulls       []WorkItem             `json:"pulls"`
	Runs        []CIRun                `json:"runs"`
}

type Metric struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Weight   int      `json:"weight"`
	Score    *float64 `json:"score"`
	State    string   `json:"state"`
	Evidence string   `json:"evidence"`
}

type Category struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Weight  int      `json:"weight"`
	Score   *float64 `json:"score"`
	Metrics []Metric `json:"metrics"`
}

type Recommendation struct {
	MetricID string `json:"metric_id"`
	Priority string `json:"priority"`
	Title    string `json:"title"`
	Reason   string `json:"reason"`
	Action   string `json:"action"`
}

type Report struct {
	Repository        Repository             `json:"repository"`
	Demo              bool                   `json:"demo"`
	Version           string                 `json:"version"`
	Status            string                 `json:"status,omitempty"`
	AnalyzedAt        time.Time              `json:"analyzed_at"`
	Score             *float64               `json:"score"`
	Coverage          int                    `json:"coverage"`
	Eligible          bool                   `json:"eligible"`
	EligibilityReason string                 `json:"eligibility_reason"`
	Categories        []Category             `json:"categories"`
	Recommendations   []Recommendation       `json:"recommendations"`
	Sources           map[string]SourceState `json:"sources"`
}

type Job struct {
	ID           string     `json:"id"`
	RepositoryID string     `json:"repository_id"`
	Ref          string     `json:"ref"`
	Demo         bool       `json:"demo"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Error        string     `json:"error,omitempty"`
}

type Project struct {
	ID     string  `json:"id"`
	Ref    string  `json:"ref"`
	Demo   bool    `json:"demo"`
	Report *Report `json:"report"`
	Job    *Job    `json:"job"`
}
