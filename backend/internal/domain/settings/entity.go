package settings

type UserSettings struct {
	UserID         string         `json:"user_id"`
	Notifications  map[string]any `json:"notifications"`
	ModulesEnabled map[string]any `json:"modules_enabled"`
}
