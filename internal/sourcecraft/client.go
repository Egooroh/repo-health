// Package sourcecraft adapts the official SourceCraft REST API to domain facts.
package sourcecraft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"repo-health/internal/domain"
)

const BaseURL = "https://api.sourcecraft.tech"

type Client struct {
	baseURL, token string
	http           *http.Client
	maxPages       int
}

func New(token string) *Client {
	return NewWithHTTP(BaseURL, token, &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, 5)
}

// NewWithHTTP allows isolated contract tests without calling SourceCraft.
func NewWithHTTP(baseURL, token string, client *http.Client, maxPages int) *Client {
	if maxPages < 1 {
		maxPages = 1
	}
	return &Client{strings.TrimRight(baseURL, "/"), token, client, maxPages}
}

type apiError struct{ status int }

func (e apiError) Error() string {
	switch e.status {
	case 401, 403:
		return "SourceCraft отклонил доступ. Проверьте серверный SOURCECRAFT_TOKEN и его права."
	case 404:
		return "Репозиторий или источник данных не найден либо недоступен."
	case 429:
		return "SourceCraft временно ограничил частоту запросов. Повторите анализ позже."
	default:
		return fmt.Sprintf("SourceCraft вернул HTTP %d. Повторите анализ позже.", e.status)
	}
}

func (c *Client) get(ctx context.Context, path string, q url.Values, dst any) error {
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+q.Encode(), nil)
		if err != nil {
			return errors.New("не удалось подготовить запрос SourceCraft")
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "RepoHealth-MVP/1.0")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return errors.New("анализ отменён или превысил отведённое время")
			}
			return errors.New("не удалось подключиться к SourceCraft; проверьте сеть и повторите анализ")
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
		resp.Body.Close()
		if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < 2 {
			delay := time.Duration(attempt+1) * 250 * time.Millisecond
			if sec, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && sec > 0 {
				delay = time.Duration(min(sec, 2)) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				continue
			}
		}
		if resp.StatusCode != http.StatusOK {
			return apiError{resp.StatusCode}
		}
		if readErr != nil || len(data) > 4*1024*1024 {
			return errors.New("ответ SourceCraft слишком большой или не прочитан полностью")
		}
		if json.Unmarshal(data, dst) != nil {
			return errors.New("формат ответа SourceCraft не соответствует ожидаемому JSON")
		}
		return nil
	}
	return errors.New("SourceCraft временно недоступен")
}

// list reads complete pages. A truncated source never becomes a misleading empty list.
func (c *Client) list(ctx context.Context, path, key string, q url.Values) ([]json.RawMessage, domain.SourceState) {
	if q == nil {
		q = url.Values{}
	}
	q.Set("page_size", "100")
	out := []json.RawMessage{}
	seen := map[string]bool{}
	for page := 0; page < c.maxPages; page++ {
		var envelope map[string]json.RawMessage
		if err := c.get(ctx, path, q, &envelope); err != nil {
			return nil, domain.SourceState{Status: "unavailable", Detail: err.Error()}
		}
		var batch []json.RawMessage
		// Empty repeated fields may be omitted by protobuf JSON encoders.
		if raw, ok := envelope[key]; ok && json.Unmarshal(raw, &batch) != nil {
			return nil, domain.SourceState{Status: "unavailable", Detail: "Неожиданный формат списка SourceCraft."}
		}
		out = append(out, batch...)
		var token string
		if raw, ok := envelope["next_page_token"]; ok && json.Unmarshal(raw, &token) != nil {
			return nil, domain.SourceState{Status: "unavailable", Detail: "Некорректный указатель следующей страницы."}
		}
		if token == "" {
			return out, domain.SourceState{Status: "available", Detail: fmt.Sprintf("Источник прочитан полностью: %d записей.", len(out))}
		}
		if seen[token] {
			return nil, domain.SourceState{Status: "truncated", Detail: "SourceCraft повторил указатель страницы; расчёт по неполной выборке отключён."}
		}
		seen[token] = true
		q.Set("page_token", token)
	}
	return nil, domain.SourceState{Status: "truncated", Detail: fmt.Sprintf("Достигнут лимит %d страниц по 100 записей. Источник не участвует в оценке.", c.maxPages)}
}

func (c *Client) Collect(ctx context.Context, ref string, now time.Time) (domain.Snapshot, error) {
	canonical, err := domain.ParseRef(ref)
	if err != nil {
		return domain.Snapshot{}, err
	}
	path := "/repos/" + canonical
	var meta struct {
		ID            string `json:"id"`
		Slug          string `json:"slug"`
		Visibility    string `json:"visibility"`
		Description   string `json:"description"`
		DefaultBranch string `json:"default_branch"`
		Empty         bool   `json:"is_empty"`
		Language      struct {
			Name string `json:"name"`
		} `json:"language"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
	}
	if err = c.get(ctx, path, nil, &meta); err != nil {
		var upstream apiError
		if errors.As(err, &upstream) && upstream.status == 404 {
			return domain.Snapshot{}, fmt.Errorf("%w: репозиторий SourceCraft не найден или недоступен", domain.ErrNotFound)
		}
		if errors.As(err, &upstream) && (upstream.status == 401 || upstream.status == 403) {
			return domain.Snapshot{}, fmt.Errorf("%w: %s", domain.ErrAccessDenied, err)
		}
		return domain.Snapshot{}, err
	}
	if meta.Visibility != "public" {
		return domain.Snapshot{}, domain.ErrNotPublic
	}
	if meta.ID == "" || meta.Slug == "" {
		return domain.Snapshot{}, errors.New("SourceCraft вернул неполное описание репозитория")
	}
	s := domain.Snapshot{Repository: domain.Repository{Ref: canonical, URL: "https://sourcecraft.dev/" + canonical,
		Description: meta.Description, Language: meta.Language.Name, DefaultBranch: meta.DefaultBranch, Empty: meta.Empty, Fork: meta.Parent != nil},
		CollectedAt: now, Sources: map[string]domain.SourceState{"appsec": {Status: "not_connected", Detail: "AppSec не подключён в этой версии."}},
		Files: []string{}, Issues: []domain.WorkItem{}, Pulls: []domain.WorkItem{}, Runs: []domain.CIRun{}}
	if meta.Empty {
		s.Sources["files"] = domain.SourceState{Status: "available", Detail: "Репозиторий пуст."}
	} else {
		s.Files, s.Sources["files"] = c.files(ctx, path)
	}
	for _, source := range []string{"issues", "pulls", "ci"} {
		suffix, key := "/"+source, source
		if source == "pulls" {
			key = "pull_requests"
		}
		if source == "ci" {
			suffix = "/cicd/runs"
			key = "runs"
		}
		query := url.Values{}
		if source == "issues" {
			query.Set("filter", "visibility=public")
		}
		items, state := c.list(ctx, path+suffix, key, query)
		s.Sources[source] = state
		if !state.Available() {
			continue
		}
		for _, raw := range items {
			var decodeErr error
			switch source {
			case "issues":
				var item struct {
					Status struct {
						Type string `json:"status_type"`
					} `json:"status"`
					UpdatedAt  time.Time `json:"updated_at"`
					Visibility string    `json:"visibility"`
				}
				decodeErr = json.Unmarshal(raw, &item)
				if decodeErr == nil && item.Visibility != "" && item.Visibility != "public" {
					continue
				}
				s.Issues = append(s.Issues, domain.WorkItem{Status: item.Status.Type, UpdatedAt: item.UpdatedAt})
			case "pulls":
				var item struct {
					Status    string    `json:"status"`
					UpdatedAt time.Time `json:"updated_at"`
				}
				decodeErr = json.Unmarshal(raw, &item)
				s.Pulls = append(s.Pulls, domain.WorkItem{Status: item.Status, UpdatedAt: item.UpdatedAt})
			case "ci":
				var item struct {
					Status string `json:"status"`
					Dates  struct {
						FinishedAt time.Time `json:"finished_at"`
					} `json:"dates"`
				}
				decodeErr = json.Unmarshal(raw, &item)
				s.Runs = append(s.Runs, domain.CIRun{Status: item.Status, FinishedAt: item.Dates.FinishedAt})
			}
			if decodeErr != nil {
				s.Sources[source] = domain.SourceState{Status: "unavailable", Detail: "Источник содержит некорректные записи или даты."}
				break
			}
		}
	}
	if ctx.Err() != nil {
		return domain.Snapshot{}, errors.New("истекло время анализа; повторите попытку")
	}
	return s, nil
}

func (c *Client) files(ctx context.Context, path string) ([]string, domain.SourceState) {
	root, state := c.list(ctx, path+"/trees", "trees", url.Values{"recursive": {"false"}})
	if !state.Available() {
		return nil, state
	}
	files := []string{}
	dirs := []string{}
	read := func(items []json.RawMessage, prefix string) bool {
		for _, raw := range items {
			var item struct {
				Name string `json:"name"`
				Path string `json:"path"`
				Type string `json:"type"`
			}
			if json.Unmarshal(raw, &item) != nil || item.Name == "" {
				return false
			}
			if item.Type == "file" || item.Type == "executable" {
				files = append(files, prefix+item.Name)
			}
			if prefix == "" && (item.Name == "docs" || item.Name == ".sourcecraft" || item.Name == ".github") && item.Type == "dir" {
				dirs = append(dirs, item.Name)
			}
		}
		return true
	}
	if !read(root, "") {
		return nil, domain.SourceState{Status: "unavailable", Detail: "Не удалось прочитать дерево файлов."}
	}
	for _, dir := range dirs {
		items, sub := c.list(ctx, path+"/trees", "trees", url.Values{"path": {dir}, "recursive": {"false"}})
		if !sub.Available() {
			return nil, sub
		}
		if !read(items, dir+"/") {
			return nil, domain.SourceState{Status: "unavailable", Detail: "Некорректное дерево документов."}
		}
	}
	return files, domain.SourceState{Status: "available", Detail: "Проверены корень и присутствующие каталоги docs, .sourcecraft, .github."}
}
