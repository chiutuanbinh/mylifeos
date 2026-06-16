package postgres

import (
	"context"
	"encoding/json"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/settings"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgSettingsRepo struct{ db *pgxpool.Pool }

func NewSettingsRepo(db *pgxpool.Pool) repository.SettingsRepo { return &pgSettingsRepo{db} }

func (r *pgSettingsRepo) Get(ctx context.Context, userID string) (settings.UserSettings, error) {
	row := r.db.QueryRow(ctx,
		`SELECT user_id, notifications, modules_enabled FROM user_settings WHERE user_id = $1`, userID)
	var s settings.UserSettings
	var notifBytes, modulesBytes []byte
	if err := row.Scan(&s.UserID, &notifBytes, &modulesBytes); err != nil {
		s.UserID = userID
		s.Notifications = map[string]any{"email": true, "push": false}
		s.ModulesEnabled = map[string]any{"finance": true, "health": true, "goals": true, "notes": true, "calendar": true, "inventory": true}
		return s, nil
	}
	json.Unmarshal(notifBytes, &s.Notifications)
	json.Unmarshal(modulesBytes, &s.ModulesEnabled)
	return s, nil
}

func (r *pgSettingsRepo) Upsert(ctx context.Context, s settings.UserSettings) (settings.UserSettings, error) {
	notifJSON, _ := json.Marshal(s.Notifications)
	modulesJSON, _ := json.Marshal(s.ModulesEnabled)
	row := r.db.QueryRow(ctx,
		`INSERT INTO user_settings (user_id, notifications, modules_enabled)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		   SET notifications = EXCLUDED.notifications,
		       modules_enabled = EXCLUDED.modules_enabled
		 RETURNING user_id, notifications, modules_enabled`,
		s.UserID, notifJSON, modulesJSON)
	var out settings.UserSettings
	var notifBytes, modulesBytes []byte
	if err := row.Scan(&out.UserID, &notifBytes, &modulesBytes); err != nil {
		return out, err
	}
	json.Unmarshal(notifBytes, &out.Notifications)
	json.Unmarshal(modulesBytes, &out.ModulesEnabled)
	return out, nil
}
