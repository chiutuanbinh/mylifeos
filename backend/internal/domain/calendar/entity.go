package calendar

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
