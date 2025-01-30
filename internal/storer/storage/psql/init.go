package psql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"log/slog"
	"project/internal/config"
	"project/internal/pkg/logger/sl"
	"project/pkg/e"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Storage struct {
	db        *sql.DB
	log       *slog.Logger
	notifiers *notifiers
}

type notifiers struct {
	m        map[string]chan *pq.Notification
	mu       sync.RWMutex
	listener *pq.Listener
}

func New(ctx context.Context, cfg *config.DB, migratePath string, log *slog.Logger) (*Storage, error) {
	const fn = "psql.New"

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Info("[OK] psql successfully connected")

	migrationDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	m, err := migrate.NewWithDatabaseInstance(migratePath, "postgres", migrationDriver)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	if err := m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("[OK] nothing to do")
		} else {
			return nil, e.Wrap(fn, err)
		}
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("[OK] Migrations: no change to apply")
		} else {
			return nil, e.Wrap(fn, err)
		}
	} else {
		log.Info("[OK] Migrations applied successfully!")
	}

	s := &Storage{
		db:  db,
		log: log,
		notifiers: &notifiers{
			m: make(map[string]chan *pq.Notification),
		},
	}

	s.notifiers.listener = s.listenNotifications(connStr)

	return s, nil
}

func (s *Storage) listenNotifications(connStr string) *pq.Listener {
	const fn = "psql.listenNotifications"

	listener := pq.NewListener(connStr, 10*time.Second, 60*time.Second, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			s.log.Error(fn, sl.Err(err))

			return
		}
	})

	go func() {
		for n := range listener.Notify {
			if n == nil {
				s.log.Warn("Connection re-established", slog.String("fn", fn))
				continue
			}

			s.notifiers.mu.RLock()

			notifyCh, ok := s.notifiers.m[n.Channel]
			if !ok {
				s.log.Warn("notify channel not found", slog.String("channel", n.Channel), slog.String("fn", fn))
				continue
			}

			select {
			case notifyCh <- n:
			default:
				s.log.Debug("notify chan overflow", slog.String("channel", n.Channel), slog.String("fn", fn))
			}

			s.notifiers.mu.RUnlock()
		}
	}()

	return listener
}
