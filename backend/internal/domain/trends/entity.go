package trends

type NetWorthSnapshot struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	SnapshotDate string  `json:"snapshot_date"`
	AssetsValue  float64 `json:"assets_value"`
	CashPosition float64 `json:"cash_position"`
	NetWorth     float64 `json:"net_worth"`
	Note         string  `json:"note"`
}

type BenchmarkData struct {
	ID     string  `json:"id"`
	Source string  `json:"source"`
	Date   string  `json:"date"`
	Value  float64 `json:"value"`
}

type BankRate struct {
	Bank        string  `json:"bank"`
	Saving12m   float64 `json:"saving_12m"`
	Lending     float64 `json:"lending"`
	FetchedDate string  `json:"fetched_date"`
}

type NewsItem struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Title       string `json:"title"`
	URL         string `json:"url"`
}
