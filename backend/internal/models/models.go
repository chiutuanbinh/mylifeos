package models

import "time"

type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Date        string    `json:"date"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Amount      float64   `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type Budget struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Category     string    `json:"category"`
	MonthlyLimit float64   `json:"monthly_limit"`
	CreatedAt    time.Time `json:"created_at"`
}

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

type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Title         string  `json:"title"`
	StartAt       string  `json:"start_at"`
	EndAt         string  `json:"end_at"`
	Color         string  `json:"color"`
	AllDay        bool    `json:"all_day"`
	GoogleEventID *string `json:"google_event_id,omitempty"`
}

type Asset struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Value            float64  `json:"value"`
	PurchasedAt      *string  `json:"purchased_at"`
	Notes            string   `json:"notes"`
	PurchaseValue    *float64 `json:"purchase_value"`
	DepreciationRate float64  `json:"depreciation_rate"`
	CurrentValue     float64  `json:"current_value"`
}

type UserSettings struct {
	UserID         string         `json:"user_id"`
	Notifications  map[string]any `json:"notifications"`
	ModulesEnabled map[string]any `json:"modules_enabled"`
}

type NetWorthSnapshot struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	SnapshotDate string  `json:"snapshot_date"`
	AssetsValue  float64 `json:"assets_value"`
	CashPosition float64 `json:"cash_position"`
	NetWorth     float64 `json:"net_worth"`
}

type DashboardSummary struct {
	NetWorthTrend    []float64     `json:"net_worth_trend"`
	NetWorth         float64       `json:"net_worth"`
	HabitsTotal      int           `json:"habits_total"`
	HabitsDoneToday  int           `json:"habits_done_today"`
	GoalsAvgProgress int           `json:"goals_avg_progress"`
	BudgetTotal      float64       `json:"budget_total"`
	BudgetSpent      float64       `json:"budget_spent"`
	RecentTx         []Transaction `json:"recent_transactions"`
}
