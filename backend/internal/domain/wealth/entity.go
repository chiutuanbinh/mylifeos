package wealth

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

type Liability struct {
	ID                string   `json:"id"`
	UserID            string   `json:"user_id"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Balance           float64  `json:"balance"`
	OriginalPrincipal *float64 `json:"original_principal"`
	InterestRate      *float64 `json:"interest_rate"`
	StartedAt         *string  `json:"started_at"`
	DueAt             *string  `json:"due_at"`
	Notes             string   `json:"notes"`
}
