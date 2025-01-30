package psql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"project/internal/models"
	"project/internal/storer/storage"
	"project/pkg/e"
	"strconv"
	"strings"
)

func (s *Storage) GetUsersRole() (map[string]string, error) {
	const fn = "psql.GetUsersRole"

	q := `
	SELECT u.id, r.name
    FROM users u
    LEFT JOIN roles r ON r.id = u.role_id`

	rows, err := s.db.Query(q)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	res := map[string]string{}

	for rows.Next() {
		var (
			userID int64
			role   string
		)

		err := rows.Scan(&userID, &role)
		if err != nil {
			return nil, e.Wrap(fn, err)
		}

		res[strconv.FormatInt(userID, 10)] = role
	}

	if len(res) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return res, nil
}

func (s *Storage) GetRoleIDs() ([]models.Role, error) {
	const fn = "psql.GetRoleIDs"

	q := `SELECT * FROM roles`

	rows, err := s.db.Query(q)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	var roles []models.Role

	for rows.Next() {
		var (
			roleID   int64
			roleName string
		)

		err := rows.Scan(&roleID, &roleName)
		if err != nil {
			return nil, e.Wrap(fn, err)
		}

		roles = append(roles, models.Role{
			RoleID:   roleID,
			RoleName: roleName,
		})
	}

	return roles, nil
}

func (s *Storage) GetUserRole(ctx context.Context, userID int64) (string, error) {
	const fn = "psql.GetUserRole"

	q := `
	SELECT r.name
    FROM users u
    LEFT JOIN roles r ON r.id = u.role_id
    WHERE u.id = $1`

	var role string

	err := s.db.QueryRowContext(ctx, q, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrNoRecordsFound
		}

		return "", e.Wrap(fn, err)
	}

	return role, nil
}

// InsertUsers If the user already exists, the role will not change
func (s *Storage) InsertUsers(ctx context.Context, users []models.User) error {
	const fn = "psql.InsertUsers"

	var args []interface{}
	var sets []string
	idx := 1

	q := `INSERT INTO users (id, username, first_name, last_name, role_id) VALUES`

	for _, user := range users {
		sets = append(sets, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", idx, idx+1, idx+2, idx+3, idx+4))
		args = append(args, user.UserID, user.Username, user.FirstName, user.LastName, user.RoleID)
		idx += 5
	}

	q += strings.Join(sets, ", ")

	q += `
	ON CONFLICT (id)
	DO UPDATE SET
		username = EXCLUDED.username,
		first_name = EXCLUDED.first_name,
		last_name = EXCLUDED.last_name`

	_, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) CreateTgChannel(ctx context.Context, group *models.TgChannel) error {
	const fn = "psql.CreateTgChannel"

	q := `INSERT INTO tg_channels (id, name, description) VALUES ($1, $2, $3)
	ON CONFLICT (id)
	DO UPDATE SET
		name = EXCLUDED.name,
		description = EXCLUDED.description`

	_, err := s.db.ExecContext(ctx, q, group.ChannelID, group.Name, group.Description)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) CreateTgGroup(ctx context.Context, group *models.TgGroup) error {
	const fn = "psql.CreateTgGroup"

	q := `INSERT INTO tg_groups (id, name, description) VALUES ($1, $2, $3)
	ON CONFLICT (id)
	DO UPDATE SET
		name = EXCLUDED.name,
		description = EXCLUDED.description`

	_, err := s.db.ExecContext(ctx, q, group.GroupID, group.Name, group.Description)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) UpdateTgGroup(ctx context.Context, groupID int64, group *models.TgGroup) error {
	const fn = "psql.UpdateTgGroup"

	q := `
	 UPDATE tg_groups
	 SET id = $1, name = $2, description = $3
	 WHERE id = $4`

	res, err := s.db.ExecContext(ctx, q, group.GroupID, group.Name, group.Description, groupID)
	if err != nil {
		return e.Wrap(fn, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if rows == 0 {
		return e.Wrap(fn, storage.ErrNoRecordsFound)
	}

	return nil
}

func (s *Storage) TgGroupIsExists(ctx context.Context, groupID int64) (bool, error) {
	const fn = "psql.TgGroupIsExists"

	q := `SELECT EXISTS (SELECT 1 FROM tg_groups WHERE id = $1)`

	_, err := s.db.ExecContext(ctx, q, groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, e.Wrap(fn, err)
	}

	return true, nil
}

func (s *Storage) TgChannelIsExists(ctx context.Context, channelID int64) (bool, error) {
	const fn = "psql.TgGroupIsExists"

	q := `SELECT EXISTS (SELECT 1 FROM tg_channels WHERE id = $1)`

	_, err := s.db.ExecContext(ctx, q, channelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, e.Wrap(fn, err)
	}

	return true, nil
}

func (s *Storage) DeleteUser(ctx context.Context, userID int64) error {
	const fn = "psql.DeleteUser"

	q := `DELETE FROM users WHERE id = $1`

	res, err := s.db.ExecContext(ctx, q, userID)
	if err != nil {
		return e.Wrap(fn, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if rows == 0 {
		return e.Wrap(fn, storage.ErrNoRecordsFound)
	}

	return nil
}

func (s *Storage) InsertVkGroup(ctx context.Context, vkGroup *models.VkGroup) error {
	const fn = "psql.InsertVkGroup"

	q := `INSERT INTO vk_groups (id, name, domain) VALUES ($1, $2, $3)`

	_, err := s.db.ExecContext(ctx, q, vkGroup.ID, vkGroup.Name, vkGroup.Domain)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) UpdateTgGroupInfo(ctx context.Context, group *models.TgGroup) error {
	const fn = "psql.UpdateTgGroupInfo"

	q := `
	UPDATE tg_groups 
	SET name = $1, description = $2
	WHERE id = $3`

	res, err := s.db.ExecContext(ctx, q, group.Name, group.Description, group.GroupID)
	if err != nil {
		return e.Wrap(fn, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if rows == 0 {
		return e.Wrap(fn, storage.ErrNoRecordsFound)
	}

	return nil
}

func (s *Storage) DeleteTgChannel(ctx context.Context, channelID int64) error {
	const fn = "psql.DeleteTgChannel"

	q := `DELETE FROM tg_channels WHERE id = $1`

	res, err := s.db.ExecContext(ctx, q, channelID)
	if err != nil {
		return e.Wrap(fn, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if rows == 0 {
		return e.Wrap(fn, storage.ErrNoRecordsFound)
	}

	return nil
}

func (s *Storage) DeleteTgGroup(ctx context.Context, groupID int64) error {
	const fn = "psql.DeleteTgGroup"

	q := `DELETE FROM tg_groups WHERE id = $1`

	res, err := s.db.ExecContext(ctx, q, groupID)
	if err != nil {
		return e.Wrap(fn, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if rows == 0 {
		return e.Wrap(fn, storage.ErrNoRecordsFound)
	}

	return nil
}

func (s *Storage) InsertTgChannelMessages(ctx context.Context, msgs []models.TgChMessage) error {
	const fn = "psql.InsertTgChannelMessages"

	var args []interface{}
	var sets []string
	idx := 1

	q := `INSERT INTO tg_channel_messages (msg_id, channel_id, text, metadata, created_at) VALUES `

	for _, msg := range msgs {
		metadataSQL := sql.NullString{}

		if msg.Metadata != nil {
			metadataJSON, err := json.Marshal(msg.Metadata)
			if err != nil {
				return e.Wrap(fn, err)
			}

			metadataSQL = sql.NullString{
				String: string(metadataJSON),
				Valid:  true,
			}
		}

		createdAt := sql.NullTime{}

		if !msg.CreatedAt.IsZero() {
			createdAt = sql.NullTime{
				Valid: true,
				Time:  msg.CreatedAt,
			}
		}

		sets = append(sets,
			fmt.Sprintf(
				"($%d, $%d, $%d, $%d, COALESCE($%d, CURRENT_TIMESTAMP))",
				idx, idx+1, idx+2, idx+3, idx+4),
		)
		args = append(args, msg.MessageID, msg.ChannelID, msg.Text, metadataSQL, createdAt)
		idx += 5
	}

	q += strings.Join(sets, ", ")

	q += `
	ON CONFLICT (msg_id, channel_id) DO NOTHING`

	_, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) InsertTgGroupMessages(ctx context.Context, msgs []models.TgGroupMessage) error {
	const fn = "psql.InsertTgGroupMessages"

	var args []interface{}
	var sets []string
	idx := 1

	q := `INSERT INTO tg_group_messages (msg_id, group_id, username, text, metadata, created_at) VALUES `

	for _, msg := range msgs {
		metadataSQL := sql.NullString{}

		if msg.Metadata != nil {
			metadataJSON, err := json.Marshal(msg.Metadata)
			if err != nil {
				return e.Wrap(fn, err)
			}

			metadataSQL = sql.NullString{
				String: string(metadataJSON),
				Valid:  true,
			}
		}

		createdAt := sql.NullTime{}

		if !msg.CreatedAt.IsZero() {
			createdAt = sql.NullTime{
				Valid: true,
				Time:  msg.CreatedAt,
			}
		}

		sets = append(sets,
			fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, COALESCE($%d, CURRENT_TIMESTAMP))",
				idx, idx+1, idx+2, idx+3, idx+4, idx+5),
		)
		args = append(args, msg.MessageID, msg.GroupID, msg.Username, msg.Text, metadataSQL, createdAt)
		idx += 6
	}

	q += strings.Join(sets, ", ")

	q += `
	ON CONFLICT (msg_id, group_id) DO NOTHING`

	_, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) InsertWebMessages(ctx context.Context, msgs []models.WebMessage) error {
	const fn = "psql.InsertWebMessages"

	var args []interface{}
	var sets []string
	idx := 1

	q := `INSERT INTO web_messages (group_name, text, metadata, created_at, type) VALUES `

	for _, msg := range msgs {
		metadataSQL := sql.NullString{}

		if msg.Metadata != nil {
			metadataJSON, err := json.Marshal(msg.Metadata)
			if err != nil {
				return e.Wrap(fn, err)
			}

			metadataSQL = sql.NullString{
				String: string(metadataJSON),
				Valid:  true,
			}
		}

		sets = append(sets,
			fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d)",
				idx, idx+1, idx+2, idx+3, idx+4),
		)
		args = append(args, msg.GroupName, msg.Text, metadataSQL, msg.CreatedAt, msg.Type)
		idx += 5
	}

	q += strings.Join(sets, ", ")

	_, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) GetWebMessages(ctx context.Context, limit int, offset int) ([]models.WebMessage, error) {
	const fn = "psql.GetWebMessages"

	q := `SELECT * FROM web_messages ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	defer rows.Close()

	var msgs []models.WebMessage

	for rows.Next() {
		var msg models.WebMessage
		var metadataStr sql.NullString

		err := rows.Scan(&msg.ID, &msg.GroupName, &msg.Text, &metadataStr, &msg.CreatedAt, &msg.Type)
		if err != nil {
			return nil, err
		}

		if metadataStr.Valid {
			var metadata []models.MetaPair
			err = json.Unmarshal([]byte(metadataStr.String), &metadata)
			if err != nil {
				return nil, err
			}

			msg.Metadata = metadata
		}

		msgs = append(msgs, msg)
	}

	if len(msgs) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return msgs, nil
}

func (s *Storage) GetTgChannels(ctx context.Context) ([]models.TgChannel, error) {
	const fn = "psql.GetTgChannels"

	q := `SELECT * FROM tg_channels`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	defer rows.Close()

	var channels []models.TgChannel

	for rows.Next() {
		var (
			channelID      int64
			name           string
			description    string
			descriptionSQL sql.NullString
		)

		err := rows.Scan(&channelID, &name, &descriptionSQL)
		if err != nil {
			return nil, err
		}

		if descriptionSQL.Valid {
			description = descriptionSQL.String
		}

		channels = append(channels, models.TgChannel{
			ChannelID:   channelID,
			Name:        name,
			Description: description,
		})
	}

	if len(channels) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return channels, nil
}

func (s *Storage) GetTgGroups(ctx context.Context) ([]models.TgGroup, error) {
	const fn = "psql.GetTgGroups"

	q := `SELECT * FROM tg_groups`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	defer rows.Close()

	var groups []models.TgGroup

	for rows.Next() {
		var (
			groupID        int64
			groupName      string
			description    string
			descriptionSQL sql.NullString
		)

		err := rows.Scan(&groupID, &groupName, &descriptionSQL)
		if err != nil {
			return nil, err
		}

		if descriptionSQL.Valid {
			description = descriptionSQL.String
		}

		groups = append(groups, models.TgGroup{
			GroupID:     groupID,
			Name:        groupName,
			Description: description,
		})
	}

	if len(groups) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return groups, nil
}

func (s *Storage) InsertVkMessages(ctx context.Context, msgs []models.VkMessage) error {
	const fn = "psql.InsertVkMessages"

	var args []interface{}
	var sets []string
	idx := 1

	q := `INSERT INTO vk_messages (msg_id, group_id, text, metadata, created_at) VALUES `

	for _, msg := range msgs {
		metadataSQL := sql.NullString{}

		if msg.Metadata != nil {
			metadataJSON, err := json.Marshal(msg.Metadata)
			if err != nil {
				return e.Wrap(fn, err)
			}

			metadataSQL = sql.NullString{
				String: string(metadataJSON),
				Valid:  true,
			}
		}

		sets = append(sets,
			fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d)",
				idx, idx+1, idx+2, idx+3, idx+4),
		)
		args = append(args, msg.MessageID, msg.GroupID, msg.Text, metadataSQL, msg.CreatedAt)
		idx += 5
	}

	q += strings.Join(sets, ", ")

	q += ` ON CONFLICT DO NOTHING`

	_, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (s *Storage) GetVkGroups(ctx context.Context) ([]models.VkGroup, error) {
	const fn = "psql.GetVkGroups"

	q := `SELECT * FROM vk_groups`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	var groups []models.VkGroup

	for rows.Next() {
		var (
			vkID     int
			vkDomain string
			vkName   string
		)

		err := rows.Scan(&vkID, &vkName, &vkDomain)
		if err != nil {
			return nil, e.Wrap(fn, err)
		}

		groups = append(groups, models.VkGroup{
			ID:     vkID,
			Name:   vkName,
			Domain: vkDomain,
		})
	}

	if len(groups) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return groups, nil
}

func (s *Storage) DeleteVkGroup(ctx context.Context, vkDomain string) error {
	const fn = "psql.DeleteVkGroup"

	q := `DELETE FROM vk_groups WHERE domain = $1`

	res, err := s.db.ExecContext(ctx, q, vkDomain)
	if err != nil {
		return e.Wrap(fn, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if rows == 0 {
		return storage.ErrNoRecordsFound
	}

	return nil
}

func (s *Storage) AddNotifier(ctx context.Context, name string, buf uint) (<-chan *pq.Notification, error) {
	const fn = "psql.AddNotifier"

	_, err := s.db.ExecContext(ctx, fmt.Sprintf("LISTEN %s", name))
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	err = s.notifiers.listener.Listen(name)
	if err != nil {
		if errors.Is(err, pq.ErrChannelAlreadyOpen) {
			return nil, e.Wrap(fn, storage.ErrChannelAlreadyOpen)
		}

		return nil, e.Wrap(fn, err)
	}

	if buf == 0 {
		buf = 10
	}

	notifyChan := make(chan *pq.Notification, buf)

	s.notifiers.mu.Lock()
	s.notifiers.m[name] = notifyChan
	s.notifiers.mu.Unlock()

	return notifyChan, nil
}

func (s *Storage) GetNotifier(name string) (<-chan *pq.Notification, error) {
	const fn = "psql.GetNotifier"

	s.notifiers.mu.RLock()
	notifyCh, ok := s.notifiers.m[name]
	if !ok {
		return nil, e.Wrap(fn, storage.ErrChannelNotFound)
	}
	s.notifiers.mu.RUnlock()

	return notifyCh, nil
}

func (s *Storage) RemoveNotifier(ctx context.Context, name string) error {
	const fn = "psql.RemoveNotifier"

	_, err := s.db.ExecContext(ctx, fmt.Sprintf("UNLISTEN %s", name))
	if err != nil {
		return e.Wrap(fn, err)
	}

	s.notifiers.mu.RLock()
	_, ok := s.notifiers.m[name]
	if !ok {
		return e.Wrap(fn, storage.ErrChannelNotFound)
	}
	s.notifiers.mu.RUnlock()

	err = s.notifiers.listener.Unlisten(name)
	if err != nil {
		if errors.Is(err, pq.ErrChannelNotOpen) {
			return e.Wrap(fn, storage.ErrChannelNotFound)
		}

		return e.Wrap(fn, err)
	}

	s.notifiers.mu.Lock()
	delete(s.notifiers.m, name)
	s.notifiers.mu.Unlock()

	return nil
}
