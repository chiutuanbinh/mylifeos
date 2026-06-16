package finance

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
