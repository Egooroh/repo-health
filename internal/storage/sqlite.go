// Package storage implements durable reports and jobs with SQLite transactions.
package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"repo-health/internal/domain"
)

//go:embed schema.sql
var schema string

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	fail := func(err error) (*Store, error) { db.Close(); return nil, err }
	if _, err = db.Exec("PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		return fail(err)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if version > 1 {
		return fail(errors.New("база создана более новой версией Repo Health"))
	}
	if version == 0 {
		tx, err := db.Begin()
		if err != nil {
			return fail(err)
		}
		if _, err = tx.Exec(schema); err != nil {
			tx.Rollback()
			return fail(err)
		}
		if err = tx.Commit(); err != nil {
			return fail(err)
		}
	}
	return &Store{db}, nil
}

func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *Store) Recover(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET status='queued', started_at=NULL WHERE status='running'")
	return err
}

const jobColumns = "j.id,j.repository_id,r.ref,r.demo,j.status,j.created_at,j.started_at,j.finished_at,j.error"

type scanner interface{ Scan(...any) error }

func scanJob(row scanner) (domain.Job, error) {
	var j domain.Job
	var created string
	var started, finished sql.NullString
	err := row.Scan(&j.ID, &j.RepositoryID, &j.Ref, &j.Demo, &j.Status, &created, &started, &finished, &j.Error)
	if errors.Is(err, sql.ErrNoRows) {
		return j, domain.ErrNotFound
	}
	if err != nil {
		return j, err
	}
	j.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return j, err
	}
	if started.Valid {
		v, e := time.Parse(time.RFC3339Nano, started.String)
		if e != nil {
			return j, e
		}
		j.StartedAt = &v
	}
	if finished.Valid {
		v, e := time.Parse(time.RFC3339Nano, finished.String)
		if e != nil {
			return j, e
		}
		j.FinishedAt = &v
	}
	return j, nil
}

func (s *Store) Enqueue(ctx context.Context, ref string, demo bool, now time.Time) (domain.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Job{}, err
	}
	defer tx.Rollback()
	repoID := domain.RepositoryID(ref, demo)
	latest, err := scanJob(tx.QueryRowContext(ctx, "SELECT "+jobColumns+" FROM jobs j JOIN repositories r ON r.id=j.repository_id WHERE r.id=? ORDER BY j.created_at DESC,j.rowid DESC LIMIT 1", repoID))
	if err == nil {
		if latest.Status == "queued" || latest.Status == "running" {
			return latest, nil
		}
		if now.Sub(latest.CreatedAt) < 30*time.Second {
			return domain.Job{}, domain.ErrCooldown
		}
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Job{}, err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE status IN ('queued','running')").Scan(&count); err != nil {
		return domain.Job{}, err
	}
	if count >= 100 {
		return domain.Job{}, domain.ErrQueueFull
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO repositories(id,ref,demo) VALUES(?,?,?) ON CONFLICT(id) DO NOTHING", repoID, ref, demo); err != nil {
		return domain.Job{}, err
	}
	bytes := make([]byte, 16)
	if _, err = rand.Read(bytes); err != nil {
		return domain.Job{}, err
	}
	j := domain.Job{ID: hex.EncodeToString(bytes), RepositoryID: repoID, Ref: ref, Demo: demo, Status: "queued", CreatedAt: now.UTC()}
	_, err = tx.ExecContext(ctx, "INSERT INTO jobs(id,repository_id,status,created_at) VALUES(?,?,?,?)", j.ID, repoID, j.Status, j.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return domain.Job{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Job{}, err
	}
	return j, nil
}

func (s *Store) Claim(ctx context.Context, now time.Time) (domain.Job, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `UPDATE jobs SET status='running',started_at=?
		WHERE id=(SELECT id FROM jobs WHERE status='queued' ORDER BY created_at,rowid LIMIT 1) RETURNING id`, now.UTC().Format(time.RFC3339Nano)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Job{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, id)
}
func (s *Store) GetJob(ctx context.Context, id string) (domain.Job, error) {
	return scanJob(s.db.QueryRowContext(ctx, "SELECT "+jobColumns+" FROM jobs j JOIN repositories r ON r.id=j.repository_id WHERE j.id=?", id))
}

func (s *Store) Complete(ctx context.Context, j domain.Job, snapshot domain.Snapshot, report domain.Report) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	facts, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO reports(repository_id,report_json,snapshot_json) VALUES(?,?,?)
		ON CONFLICT(repository_id) DO UPDATE SET report_json=excluded.report_json,snapshot_json=excluded.snapshot_json`, j.RepositoryID, string(raw), string(facts)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE jobs SET status='completed',finished_at=?,error='' WHERE id=?", time.Now().UTC().Format(time.RFC3339Nano), j.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Fail(ctx context.Context, j domain.Job, message string, invalidate bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if invalidate {
		if _, err = tx.ExecContext(ctx, "DELETE FROM reports WHERE repository_id=?", j.RepositoryID); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE jobs SET status='failed',finished_at=?,error=? WHERE id=?", time.Now().UTC().Format(time.RFC3339Nano), message, j.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetProject(ctx context.Context, id string) (domain.Project, error) {
	var p domain.Project
	var raw sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT r.id,r.ref,r.demo,p.report_json FROM repositories r
		LEFT JOIN reports p ON p.repository_id=r.id WHERE r.id=?`, id).Scan(&p.ID, &p.Ref, &p.Demo, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return p, domain.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if raw.Valid {
		if err = json.Unmarshal([]byte(raw.String), &p.Report); err != nil {
			return p, fmt.Errorf("read report: %w", err)
		}
	}
	j, err := scanJob(s.db.QueryRowContext(ctx, "SELECT "+jobColumns+" FROM jobs j JOIN repositories r ON r.id=j.repository_id WHERE r.id=? ORDER BY j.created_at DESC,j.rowid DESC LIMIT 1", id))
	if err == nil {
		p.Job = &j
	} else if !errors.Is(err, domain.ErrNotFound) {
		return p, err
	}
	return p, nil
}

func (s *Store) ListProjects(ctx context.Context, demo bool) ([]domain.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.id FROM repositories r LEFT JOIN jobs j ON j.id=
		(SELECT id FROM jobs WHERE repository_id=r.id ORDER BY created_at DESC,rowid DESC LIMIT 1)
		WHERE r.demo=? ORDER BY j.created_at DESC,r.ref LIMIT 1000`, demo)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	projects := make([]domain.Project, 0, len(ids))
	for _, id := range ids {
		p, err := s.GetProject(ctx, id)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}
