package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"repo-health/internal/app"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"repo-health/internal/sourcecraft"
	"repo-health/internal/storage"
)

const sessionCookie = "repo_health_session"
const sessionLifetime = time.Hour
const maxSessions = 16
const maxPersonalRepositories = 20

type personalSource interface {
	app.Collector
	ValidateToken(context.Context) error
	CheckAccess(context.Context, string) error
}

// Each browser owns its collector, queue and in-memory database. None of these
// references are installed in the public service or its persistent database.
type personalSession struct {
	mu      sync.Mutex
	service *app.Service
	source  personalSource
	store   *storage.Store
	cancel  context.CancelFunc
	done    chan struct{}
	expires time.Time
	closed  bool
}

func (s *personalSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		s.cancel()
		<-s.done
		s.store.Close()
		s.source = nil
		s.service = nil
	}
}

type sessionManager struct {
	mu        sync.Mutex
	entries   map[string]*personalSession
	ctx       context.Context
	log       *slog.Logger
	newSource func(string) personalSource
}

// NewWithSessions adds temporary personal access to the existing public API.
// Calling closeSessions stops workers and releases all personal databases.
func NewWithSessions(ctx context.Context, service *app.Service, log *slog.Logger, tokenConfigured bool) (http.Handler, func()) {
	return newSessionHandler(ctx, service, log, tokenConfigured, func(token string) personalSource { return sourcecraft.NewPersonal(token) })
}

func newSessionHandler(ctx context.Context, service *app.Service, log *slog.Logger, tokenConfigured bool, factory func(string) personalSource) (http.Handler, func()) {
	ctx, cancel := context.WithCancel(ctx)
	m := &sessionManager{entries: make(map[string]*personalSession), ctx: ctx, log: log, newSource: factory}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.expire(false)
			case <-ctx.Done():
				m.expire(true)
				return
			}
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/session", m.connect)
	mux.HandleFunc("GET /api/v1/session", m.status)
	mux.HandleFunc("POST /api/v1/session/disconnect", m.disconnect)
	mux.HandleFunc("/api/v1/personal/", m.personal)
	mux.Handle("/", New(service, log, tokenConfigured))
	guard := &Server{log: log}
	return guard.middleware(mux), func() { cancel(); <-done }
}

func (m *sessionManager) expire(all bool) {
	m.mu.Lock()
	var expired []*personalSession
	for id, s := range m.entries {
		if all || !time.Now().Before(s.expires) {
			delete(m.entries, id)
			expired = append(expired, s)
		}
	}
	m.mu.Unlock()
	for _, s := range expired {
		s.close()
	}
}

func (m *sessionManager) lookup(r *http.Request) *personalSession {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	m.mu.Lock()
	s := m.entries[cookie.Value]
	m.mu.Unlock()
	return s
}

func sessionCookieValue(w http.ResponseWriter, r *http.Request, id string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: id, Path: "/api/v1/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}

func (m *sessionManager) status(w http.ResponseWriter, r *http.Request) {
	if s := m.lookup(r); s != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.closed && time.Now().Before(s.expires) {
			writeJSON(w, 200, map[string]any{"connected": true, "expires_at": s.expires, "appsec_connected": false})
			return
		}
	}
	writeJSON(w, 200, map[string]any{"connected": false, "appsec_connected": false})
}

func (m *sessionManager) connect(w http.ResponseWriter, r *http.Request) {
	// An insecure remote origin must never collect credentials. Local development
	// works over HTTP; deployed instances must terminate TLS at this server.
	host := r.URL.Hostname()
	if host == "" {
		host = strings.Split(r.Host, ":")[0]
	}
	if r.TLS == nil && host != "localhost" && host != "127.0.0.1" && r.Host != "[::1]" && !strings.HasPrefix(r.Host, "[::1]:") {
		writeError(w, 400, "Для подключения токена нужен HTTPS. HTTP допускается только на localhost.")
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &body) {
		return
	}
	token := strings.TrimSpace(body.Token)
	if token == "" || len(token) > 2048 || strings.IndexFunc(token, func(r rune) bool { return r <= 32 || r >= 127 }) >= 0 {
		writeError(w, 400, "Введите токен SourceCraft без пробелов и переносов строк.")
		return
	}
	m.expire(false)
	m.mu.Lock()
	full := len(m.entries) >= maxSessions
	m.mu.Unlock()
	if full {
		writeError(w, 503, "Достигнут лимит личных сессий. Повторите позже.")
		return
	}
	source := m.newSource(token)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	err := source.ValidateToken(ctx)
	cancel()
	if err != nil {
		writeError(w, 422, "Не удалось подтвердить доступ к SourceCraft. Проверьте токен, его права и соединение; повторите попытку.")
		return
	}
	store, err := storage.Open(":memory:")
	if err != nil {
		writeError(w, 500, "Не удалось создать личную сессию.")
		return
	}
	workerCtx, stop := context.WithCancel(m.ctx)
	s := &personalSession{source: source, store: store, service: app.NewPersonal(store, source, demo.Collector{}, m.log), cancel: stop, done: make(chan struct{}), expires: time.Now().Add(sessionLifetime)}
	idBytes := make([]byte, 32)
	if _, err := rand.Read(idBytes); err != nil {
		stop()
		store.Close()
		writeError(w, 500, "Не удалось создать сессию.")
		return
	}
	id := hex.EncodeToString(idBytes)
	m.mu.Lock()
	if len(m.entries) >= maxSessions || m.ctx.Err() != nil {
		m.mu.Unlock()
		stop()
		store.Close()
		writeError(w, 503, "Сессии временно недоступны.")
		return
	}
	m.entries[id] = s
	// Start before releasing the manager lock: shutdown may already be pending.
	go func() { defer close(s.done); _ = s.service.Run(workerCtx) }()
	m.mu.Unlock()
	m.remove(r)
	sessionCookieValue(w, r, id, int(sessionLifetime.Seconds()))
	writeJSON(w, 200, map[string]any{"connected": true, "expires_at": s.expires, "appsec_connected": false})
}

func (m *sessionManager) remove(r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return
	}
	m.mu.Lock()
	s := m.entries[cookie.Value]
	delete(m.entries, cookie.Value)
	m.mu.Unlock()
	if s != nil {
		s.close()
	}
}

func (m *sessionManager) disconnect(w http.ResponseWriter, r *http.Request) {
	var body struct{}
	if !decode(w, r, &body) {
		return
	}
	m.remove(r)
	sessionCookieValue(w, r, "", -1)
	writeJSON(w, 200, map[string]bool{"connected": false})
}

func (m *sessionManager) personal(w http.ResponseWriter, r *http.Request) {
	s := m.lookup(r)
	if s == nil {
		writeError(w, 401, "Подключите токен SourceCraft на сайте.")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !time.Now().Before(s.expires) {
		writeError(w, 401, "Личная сессия завершена. Подключите токен снова.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/personal/")
	helper := &Server{app: s.service, log: m.log}
	check := func(ref string) bool {
		if err := s.source.CheckAccess(ctx, ref); err != nil {
			writeError(w, 403, "Доступ к репозиторию не подтверждён. Проверьте права токена и соединение с SourceCraft.")
			return false
		}
		return true
	}
	switch {
	case r.Method == "GET" && path == "repositories":
		projects, err := s.service.Projects(ctx, false, false)
		if err != nil {
			helper.failure(w, err)
			return
		}
		// Fail closed: no cached names, counts or scores are sent if access cannot
		// be confirmed. A session holds at most 20 repositories.
		for i := range projects {
			if !check(projects[i].Ref) {
				return
			}
			markPersonal(&projects[i])
		}
		writeJSON(w, 200, map[string]any{"items": projects, "count": len(projects), "scope": "personal"})
	case r.Method == "GET" && strings.HasPrefix(path, "repositories/"):
		p, err := s.service.Project(ctx, strings.TrimPrefix(path, "repositories/"))
		if err != nil {
			helper.failure(w, err)
			return
		}
		if !check(p.Ref) {
			return
		}
		markPersonal(&p)
		writeJSON(w, 200, p)
	case r.Method == "GET" && strings.HasPrefix(path, "jobs/"):
		j, err := s.service.Job(ctx, strings.TrimPrefix(path, "jobs/"))
		if err != nil {
			helper.failure(w, err)
			return
		}
		if !check(j.Ref) {
			return
		}
		writeJSON(w, 200, j)
	case r.Method == "POST" && path == "analyses":
		var body struct {
			Ref  string `json:"ref"`
			Demo bool   `json:"demo"`
		}
		if !decode(w, r, &body) {
			return
		}
		ref, err := domain.ParseRef(body.Ref)
		if err != nil || body.Demo {
			writeError(w, 400, "Укажите реальный репозиторий SourceCraft.")
			return
		}
		projects, err := s.service.Projects(ctx, false, false)
		if err != nil {
			helper.failure(w, err)
			return
		}
		found := false
		for _, p := range projects {
			if p.Ref == ref {
				found = true
			}
		}
		if !found && len(projects) >= maxPersonalRepositories {
			writeError(w, 409, "Лимит сессии — 20 репозиториев. Отключите токен и подключите его заново для новой сессии.")
			return
		}
		if !check(ref) {
			return
		}
		j, err := s.service.Analyze(ctx, ref, false)
		if err != nil {
			helper.failure(w, err)
			return
		}
		w.Header().Set("Location", "/api/v1/personal/jobs/"+j.ID)
		writeJSON(w, 202, j)
	default:
		writeError(w, 404, "Метод личного API не найден.")
	}
}

func markPersonal(p *domain.Project) {
	if p.Report != nil {
		p.Report.Eligible = false
		p.Report.EligibilityReason = "Личный отчёт: не публикуется и не участвует в общем рейтинге. AppSec пока не подключён."
	}
}
