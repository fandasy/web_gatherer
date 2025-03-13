package sup

import (
	"context"
	"project/internal/models"
	"project/pkg/e"
	"strconv"
)

func (h *Handler) SaveAdminUser(ctx context.Context, user models.User) error {
	const fn = "init.PrepareAdmin"

	adminRoleID, _ := h.ac.GetFromMap(models.RoleIDsMapName, models.AdminRole)

	user.RoleID = adminRoleID.(int64)

	if err := h.db.InsertUsers(ctx, []models.User{user}); err != nil {
		return e.Wrap(fn, err)
	}

	userIdStr := strconv.FormatInt(user.UserID, 10)

	if err := h.cdb.Set(ctx, userIdStr, models.AdminRole); err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}
