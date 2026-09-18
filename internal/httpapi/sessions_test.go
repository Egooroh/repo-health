package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"repo-health/internal/app"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"repo-health/internal/storage"
)

type sessionSource struct {
	token   string
	revoked *atomic.Bool
}

func (s *sessionSource) ValidateToken(context.Context) error {
	if s.token == "invalid" {
		return errors.New("upstream secret must not be echoed")
	}
	return nil
}
func (s *sessionSource) CheckAccess(context.Context, string) error {
	if s.revoked.Load() {
		return domain.ErrAccessDenied
	}
	return nil
}
func (s *sessionSource) Collect(ctx context.Context, ref string, now time.Time) (domain.Snapshot, error) {
	snapshot, err := (demo.Collector{}).Collect(ctx, "demo/atlas", now)
	snapshot.Repository.Ref = ref
	snapshot.Repository.Description = "private result for " + s.token
	return snapshot, err
}

func TestPersonalSessionsIsolationRevocationAndLogout(t *testing.T) {
	store, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	public := app.New(store, demo.Collector{}, demo.Collector{}, log)
	var revoked atomic.Bool
	h, closeSessions := newSessionHandler(context.Background(), public, log, false, func(token string) personalSource { return &sessionSource{token, &revoked} })
	defer closeSessions()
	request := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		if method == "POST" {
			r.Header.Set("Content-Type", "application/json")
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	connect := func(token string, old *http.Cookie) *http.Cookie {
		w := request("POST", "/api/v1/session", `{"token":"`+token+`"}`, old)
		if w.Code != 200 {
			t.Fatalf("connect: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), token) {
			t.Fatal("token leaked")
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].MaxAge != 3600 {
			t.Fatal("unsafe session cookie")
		}
		return cookies[0]
	}
	a := connect("alice-secret", nil)
	b := connect("bob-secret", nil)
	w := request("POST", "/api/v1/personal/analyses", `{"ref":"org/private"}`, a)
	if w.Code != 202 {
		t.Fatalf("analyze: %d %s", w.Code, w.Body.String())
	}
	var job domain.Job
	if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.Header().Get("Location"), "/personal/jobs/") {
		t.Fatal("wrong job scope")
	}
	path := "/api/v1/personal/repositories/" + job.RepositoryID
	deadline := time.Now().Add(5 * time.Second)
	for {
		w = request("GET", path, "", a)
		if w.Code != 200 {
			t.Fatalf("own read: %d %s", w.Code, w.Body.String())
		}
		var p domain.Project
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		if p.Report != nil {
			if p.Report.Eligible || p.Report.Repository.Description != "private result for alice-secret" {
				t.Fatal("wrong personal result")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, route := range []string{path, "/api/v1/personal/jobs/" + job.ID} {
		if got := request("GET", route, "", b); got.Code != 404 {
			t.Fatalf("other browser read: %d", got.Code)
		}
		if got := request("GET", route, "", nil); got.Code != 401 {
			t.Fatal("anonymous personal access")
		}
	}
	for _, route := range []string{"/api/v1/repositories/" + job.RepositoryID, "/api/v1/jobs/" + job.ID} {
		if got := request("GET", route, "", a); got.Code != 404 {
			t.Fatal("personal result appeared in public storage")
		}
	}
	w = request("GET", "/api/v1/leaderboard", "", a)
	if w.Code != 200 || strings.Contains(w.Body.String(), "org/private") {
		t.Fatal("private ranking leak")
	}
	if got := request("GET", "/api/v1/personal/leaderboard", "", a); got.Code != 404 {
		t.Fatal("personal ranking enabled")
	}
	// Same repo, different token: independent result rather than a shared cache.
	w = request("POST", "/api/v1/personal/analyses", `{"ref":"org/private"}`, b)
	if w.Code != 202 {
		t.Fatal("second owner cannot independently analyze")
	}
	if got := request("GET", "/api/v1/personal/jobs/"+job.ID, "", b); got.Code != 404 {
		t.Fatal("job crossed owners")
	}
	revoked.Store(true)
	for _, route := range []string{path, "/api/v1/personal/repositories", "/api/v1/personal/jobs/" + job.ID} {
		w = request("GET", route, "", a)
		if w.Code != 403 || strings.Contains(w.Body.String(), "alice-secret") || strings.Contains(w.Body.String(), "org/private") {
			t.Fatal("cached data survived revoked access")
		}
	}
	revoked.Store(false)
	a2 := connect("alice-replacement", a)
	if got := request("GET", path, "", a); got.Code != 401 {
		t.Fatal("old cookie still works")
	}
	if got := request("GET", path, "", a2); got.Code != 404 {
		t.Fatal("replacement inherited old reports")
	}
	w = request("POST", "/api/v1/session/disconnect", "{}", a2)
	if w.Code != 200 || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("disconnect cookie not cleared")
	}
	if got := request("GET", path, "", a2); got.Code != 401 {
		t.Fatal("logged out cookie still works")
	}
}

func TestSessionInputAndOriginBoundaries(t *testing.T) {
	store, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	var revoked atomic.Bool
	h, closeSessions := newSessionHandler(context.Background(), app.New(store, demo.Collector{}, demo.Collector{}, log), log, false, func(token string) personalSource { return &sessionSource{token, &revoked} })
	defer closeSessions()
	cases := []struct {
		name, url, body, origin, content string
		status                           int
	}{
		{"empty", "http://localhost", `{"token":""}`, "", "application/json", 400},
		{"newline", "http://localhost", `{"token":"one\ntwo"}`, "", "application/json", 400},
		{"unknown", "http://localhost", `{"token":"ok","extra":true}`, "", "application/json", 400},
		{"oversize", "http://localhost", `{"token":"` + strings.Repeat("a", 5000) + `"}`, "", "application/json", 400},
		{"invalid", "http://localhost", `{"token":"invalid"}`, "", "application/json", 422},
		{"csrf", "http://localhost", `{"token":"ok"}`, "https://attacker.invalid", "application/json", 403},
		{"form", "http://localhost", `{"token":"ok"}`, "", "text/plain", 415},
		{"remote http", "http://host.example", `{"token":"ok"}`, "", "application/json", 400},
		{"tls", "https://host.example", `{"token":"ok"}`, "https://host.example", "application/json", 200},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", test.url+"/api/v1/session", strings.NewReader(test.body))
			r.Header.Set("Content-Type", test.content)
			r.Header.Set("Origin", test.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != test.status {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "upstream secret") {
				t.Fatal("raw upstream error exposed")
			}
			if test.name == "tls" && !w.Result().Cookies()[0].Secure {
				t.Fatal("TLS cookie not Secure")
			}
		})
	}
}

func TestExpiredSessionCannotReadAndCleanupStopsWorker(t *testing.T) {
	store, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := app.New(store, demo.Collector{}, demo.Collector{}, log)
	ctx, cancel := context.WithCancel(context.Background())
	s := &personalSession{store: store, service: service, cancel: cancel, done: make(chan struct{}), expires: time.Now().Add(-time.Second)}
	go func() { defer close(s.done); _ = service.Run(ctx) }()
	m := &sessionManager{entries: map[string]*personalSession{"expired": s}, log: log}
	r := httptest.NewRequest("GET", "http://localhost/api/v1/personal/repositories", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "expired"})
	w := httptest.NewRecorder()
	m.personal(w, r)
	if w.Code != 401 {
		t.Fatal("expired session accepted")
	}
	m.expire(false)
	if len(m.entries) != 0 || !s.closed || s.service != nil || s.source != nil {
		t.Fatal("expired session not released")
	}
	select {
	case <-s.done:
	default:
		t.Fatal("worker leaked")
	}
}
