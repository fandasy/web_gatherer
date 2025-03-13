package server

import (
	"context"
	"project/internal/clients/vk"
	"project/internal/models"
	"time"
)

type Storage interface {
	GetRoleIDs(ctx context.Context) ([]models.Role, error)
}

type AppCache interface {
	CreateMap(name string)
	SetToMap(name, key string, value any, TTL time.Duration) bool
}

func Prepare(ctx context.Context, db Storage, ac AppCache, vk *vk.Handler) error {
	ac.CreateMap(models.MediaGroupMapName)

	ac.CreateMap(models.RoleIDsMapName)

	roles, err := db.GetRoleIDs(ctx)
	if err != nil {
		return err
	}

	for _, role := range roles {
		ac.SetToMap(models.RoleIDsMapName, role.RoleName, role.RoleID, 0)
	}

	if err := vk.PrepareNewsGatherer(ctx); err != nil {
		return err
	}

	return nil
}
