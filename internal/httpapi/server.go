package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"repo-health/internal/app"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"repo-health/internal/scoring"
	"repo-health/web"
)

type Server struct {
	app             *app.Service
	log             *slog.Logger
	tokenConfigured bool
	mu              sync.Mutex
	window          time.Time
	writes          int
}

func New(service *app.Service, log *slog.Logger, tokenConfigured bool) http.Handler {
	s := &Server{app: service, log: log, tokenConfigured: tokenConfigured}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/v1/meta", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"version": scoring.CurrentVersion, "token_configured": s.tokenConfigured, "appsec_connected": false})
	})
	mux.HandleFunc("GET /api/v1/repositories", s.list(false))
	mux.HandleFunc("GET /api/v1/leaderboard", s.list(true))
	mux.HandleFunc("GET /api/v1/repositories/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := s.app.Project(r.Context(), r.PathValue("id"))
		if err != nil {
			s.failure(w, err)
			return
		}
		writeJSON(w, 200, p)
	})
	mux.HandleFunc("GET /api/v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		j, err := s.app.Job(r.Context(), r.PathValue("id"))
		if err != nil {
			s.failure(w, err)
			return
		}
		writeJSON(w, 200, j)
	})
	mux.HandleFunc("POST /api/v1/analyses", s.analyze)
	mux.HandleFunc("POST /api/v1/demo", s.seedDemo)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, "Метод API не найден.") })
	assets := http.FileServer(http.FS(web.Files))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			writeError(w, 405, "Метод не поддерживается.")
			return
		}
		if r.URL.Path != "/" && r.URL.Path != "/app.js" && r.URL.Path != "/access.js" && r.URL.Path != "/styles.css" {
			http.NotFound(w, r)
			return
		}
		assets.ServeHTTP(w, r)
	})
	return s.middleware(mux)
}

func (s *Server) list(leaderboard bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := s.app.Projects(r.Context(), r.URL.Query().Get("demo") == "true", leaderboard)
		if err != nil {
			s.failure(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": projects, "count": len(projects)})
	}
}

func (s *Server) analyze(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ref  string `json:"ref"`
		Demo bool   `json:"demo"`
	}
	if !decode(w, r, &body) {
		return
	}
	ref, err := domain.ParseRef(body.Ref)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if body.Demo {
		allowed := false
		for _, candidate := range demo.Refs {
			if ref == candidate {
				allowed = true
			}
		}
		if !allowed {
			writeError(w, 400, "Неизвестный демонстрационный пример.")
			return
		}
	}
	j, err := s.app.Analyze(r.Context(), ref, body.Demo)
	if err != nil {
		s.failure(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/jobs/"+j.ID)
	writeJSON(w, http.StatusAccepted, j)
}

func (s *Server) seedDemo(w http.ResponseWriter, r *http.Request) {
	var body struct{}
	if !decode(w, r, &body) {
		return
	}
	for _, ref := range demo.Refs {
		_, err := s.app.Project(r.Context(), domain.RepositoryID(ref, true))
		if err == nil {
			continue
		}
		if !errors.Is(err, domain.ErrNotFound) {
			s.failure(w, err)
			return
		}
		if _, err = s.app.Analyze(r.Context(), ref, true); err != nil {
			s.failure(w, err)
			return
		}
	}
	writeJSON(w, 202, map[string]string{"status": "queued"})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(dst) != nil {
		writeError(w, 400, "Некорректный JSON запроса.")
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, 400, "Ожидается один JSON-объект.")
		return false
	}
	return true
}

func (s *Server) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, 404, "Результат ещё не создан или не найден.")
	case errors.Is(err, domain.ErrCooldown), errors.Is(err, domain.ErrQueueFull):
		w.Header().Set("Retry-After", "30")
		writeError(w, 429, err.Error())
	default:
		s.log.Error("request failed", "error", err)
		writeError(w, 500, "Не удалось выполнить запрос. Повторите попытку.")
	}
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeError(w, 403, "Запрос должен исходить из интерфейса приложения.")
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				if err != nil || u.Host != r.Host || u.Scheme != scheme {
					writeError(w, 403, "Недопустимый источник запроса.")
					return
				}
			}
			if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
				writeError(w, 415, "Используйте Content-Type: application/json.")
				return
			}
			s.mu.Lock()
			if time.Since(s.window) >= time.Minute {
				s.window = time.Now()
				s.writes = 0
			}
			s.writes++
			limited := s.writes > 30
			s.mu.Unlock()
			if limited {
				w.Header().Set("Retry-After", "60")
				writeError(w, 429, "Слишком много запросов. Подождите минуту.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
