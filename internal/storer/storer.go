package storer

import (
	"context"
	"errors"
	"log/slog"
	"project/internal/config"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	app_cache "project/internal/storer/app-cache"
	"project/internal/storer/cache"
	"project/internal/storer/files"
	"project/internal/storer/storage"
	"project/pkg/e"
	"sync/atomic"
	"time"
)

type Storer struct {
	DB     storage.Storage
	CDB    cache.Cache
	FDB    files.Files
	MM     app_cache.AppCache
	log    *slog.Logger
	cdbErr cdbErr
}

type cdbErr struct {
	statusChan chan struct{}
	status     atomic.Bool
}

func New(storage storage.Storage, cache cache.Cache, files files.Files, mm app_cache.AppCache, cfg *config.Config, log *slog.Logger) (*Storer, error) {
	store := &Storer{
		DB:  storage,
		CDB: cache,
		FDB: files,
		MM:  mm,
		log: log,
		cdbErr: cdbErr{
			statusChan: make(chan struct{}),
			status:     atomic.Bool{},
		},
	}

	if err := store.prepareCache(); err != nil {
		return nil, e.Wrap("prepare cache error", err)
	}

	if err := store.initSubMaps(); err != nil {
		return nil, e.Wrap("init sub maps", err)
	}

	go store.cacheRestorer(cfg.Redis.RestoreTimeout)

	return store, nil
}

func (s *Storer) initSubMaps() error {

	s.MM.CreateMap(models.MediaGroupMapName)

	s.MM.CreateMap(models.VkNewsGroupMapName)

	s.MM.CreateMap(models.RoleIDsMapName)

	roles, err := s.DB.GetRoleIDs()
	if err != nil {
		return err
	}

	for _, role := range roles {
		s.MM.SetToMap(models.RoleIDsMapName, role.RoleName, role.RoleID, 0)
	}

	return nil
}

func (s *Storer) cacheRestorer(timeout time.Duration) {
	const fn = "[CACHE RESTORER]"

	for {
		<-s.cdbErr.statusChan
		s.cdbErr.status.Store(true)
		s.log.Warn(fn, slog.String("msg", "cache restorer started"))

		for {
			if err := s.CDB.Ping(context.Background()); err != nil {
				s.log.Error(fn, sl.Err(err))
				time.Sleep(timeout)

				continue
			}

			s.log.Info(fn, slog.String("msg", "redis successfully reconnected"))

			if err := s.prepareCache(); err != nil {
				s.log.Error(fn, sl.Err(err))
				time.Sleep(timeout)

				continue
			}

			s.cdbErr.status.Store(false)

			break
		}
	}
}

func (s *Storer) prepareCache() error {
	usersRole, err := s.DB.GetUsersRole()
	if err != nil {
		if errors.Is(err, storage.ErrNoRecordsFound) {
			return errors.New("[STORAGE] no user records found")
		}

		return err
	}

	if err := s.CDB.MSet(context.Background(), usersRole); err != nil {
		return err
	}

	s.log.Info("prepared cache compiled successfully")

	s.log.Debug("data prepared", slog.Any("users role", usersRole))

	return nil
}

func (s *Storer) ReportCdbErr() {
	select {
	case s.cdbErr.statusChan <- struct{}{}:
	default:
	}
}

func (s *Storer) CdbErrStatus() bool {
	return s.cdbErr.status.Load()
}
