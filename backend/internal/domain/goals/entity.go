package goals

import "time"

type Goal struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	TargetDate  *string     `json:"target_date"`
	Progress    int         `json:"progress"`
	Color       string      `json:"color"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	KeyResults  []KeyResult `json:"key_results,omitempty"`
}

type KeyResult struct {
	ID           string  `json:"id"`
	GoalID       string  `json:"goal_id"`
	UserID       string  `json:"user_id"`
	Description  string  `json:"description"`
	Done         bool    `json:"done"`
	Recurring    bool    `json:"recurring"`
	ReminderTime *string `json:"reminder_time,omitempty"`
}

type KRLog struct {
	ID         string `json:"id"`
	KRID       string `json:"kr_id"`
	UserID     string `json:"user_id"`
	LoggedDate string `json:"logged_date"`
	Done       bool   `json:"done"`
}
