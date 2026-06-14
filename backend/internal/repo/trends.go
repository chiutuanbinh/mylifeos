package repo

import (
	"context"
	"strings"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrendsRepo interface {
	ListSnapshots(ctx context.Context, userID string) ([]models.NetWorthSnapshot, error)
	UpsertSnapshot(ctx context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error)
	UpsertBenchmark(ctx context.Context, b models.BenchmarkData) error
	ListBenchmarks(ctx context.Context, sources []string, from, to string) ([]models.BenchmarkData, error)
	LatestBankRates(ctx context.Context) ([]models.BankRate, error)
	UpsertBankRate(ctx context.Context, b models.BankRate) error
	ListNews(ctx context.Context, limit int) ([]models.NewsItem, error)
	UpsertNews(ctx context.Context, items []models.NewsItem) error
}

type pgTrendsRepo struct{ db *pgxpool.Pool }

func NewTrendsRepo(db *pgxpool.Pool) TrendsRepo { return &pgTrendsRepo{db} }

func (r *pgTrendsRepo) ListSnapshots(ctx context.Context, userID string) ([]models.NetWorthSnapshot, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, snapshot_date, assets_value, cash_position, net_worth, note
		 FROM net_worth_snapshots WHERE user_id = $1 ORDER BY snapshot_date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.NetWorthSnapshot
	for rows.Next() {
		var s models.NetWorthSnapshot
		var d time.Time
		rows.Scan(&s.ID, &s.UserID, &d, &s.AssetsValue, &s.CashPosition, &s.NetWorth, &s.Note)
		s.SnapshotDate = d.Format("2006-01-02")
		out = append(out, s)
	}
	if out == nil {
		out = []models.NetWorthSnapshot{}
	}
	return out, rows.Err()
}

func (r *pgTrendsRepo) UpsertSnapshot(ctx context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO net_worth_snapshots (user_id, snapshot_date, assets_value, cash_position, net_worth, note)
		 VALUES ($1, $2::date, $3, $4, $5, $6)
		 ON CONFLICT (user_id, snapshot_date)
		 DO UPDATE SET assets_value=$3, cash_position=$4, net_worth=$5, note=$6
		 RETURNING id, user_id, snapshot_date, assets_value, cash_position, net_worth, note`,
		s.UserID, s.SnapshotDate, s.AssetsValue, s.CashPosition, s.NetWorth, s.Note)
	var out models.NetWorthSnapshot
	var d time.Time
	err := row.Scan(&out.ID, &out.UserID, &d, &out.AssetsValue, &out.CashPosition, &out.NetWorth, &out.Note)
	out.SnapshotDate = d.Format("2006-01-02")
	return out, err
}

func (r *pgTrendsRepo) UpsertBenchmark(ctx context.Context, b models.BenchmarkData) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO benchmark_data (source, date, value)
		 VALUES ($1, $2::date, $3)
		 ON CONFLICT (source, date) DO UPDATE SET value=$3`,
		b.Source, b.Date, b.Value)
	return err
}

func (r *pgTrendsRepo) ListBenchmarks(ctx context.Context, sources []string, from, to string) ([]models.BenchmarkData, error) {
	var q string
	var args []interface{}
	if len(sources) > 0 {
		q = `SELECT id, source, date, value FROM benchmark_data
		     WHERE date BETWEEN $1::date AND $2::date AND source = ANY($3::text[])
		     ORDER BY source, date`
		args = []interface{}{from, to, sources}
	} else {
		q = `SELECT id, source, date, value FROM benchmark_data
		     WHERE date BETWEEN $1::date AND $2::date
		     ORDER BY source, date`
		args = []interface{}{from, to}
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.BenchmarkData
	for rows.Next() {
		var b models.BenchmarkData
		var d time.Time
		rows.Scan(&b.ID, &b.Source, &d, &b.Value)
		b.Date = d.Format("2006-01-02")
		out = append(out, b)
	}
	if out == nil {
		out = []models.BenchmarkData{}
	}
	return out, rows.Err()
}

func (r *pgTrendsRepo) UpsertBankRate(ctx context.Context, b models.BankRate) error {
	today := b.FetchedDate
	if today == "" {
		today = time.Now().Format("2006-01-02")
	}
	savingSource := "bankrate_" + strings.ToLower(b.Bank) + "_saving"
	lendingSource := "bankrate_" + strings.ToLower(b.Bank) + "_lending"
	_, err := r.db.Exec(ctx,
		`INSERT INTO benchmark_data (source, date, value) VALUES ($1, $2::date, $3)
		 ON CONFLICT (source, date) DO UPDATE SET value=$3`,
		savingSource, today, b.Saving12m)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO benchmark_data (source, date, value) VALUES ($1, $2::date, $3)
		 ON CONFLICT (source, date) DO UPDATE SET value=$3`,
		lendingSource, today, b.Lending)
	return err
}

func (r *pgTrendsRepo) LatestBankRates(ctx context.Context) ([]models.BankRate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT source, value, date
		 FROM benchmark_data
		 WHERE source LIKE 'bankrate_%'
		   AND date = (SELECT MAX(date) FROM benchmark_data b2 WHERE b2.source = benchmark_data.source)
		 ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bankMap := map[string]*models.BankRate{}
	for rows.Next() {
		var src string
		var val float64
		var d time.Time
		rows.Scan(&src, &val, &d)
		dateStr := d.Format("2006-01-02")
		// src format: bankrate_<bank>_saving or bankrate_<bank>_lending
		parts := strings.SplitN(strings.TrimPrefix(src, "bankrate_"), "_", 2)
		if len(parts) != 2 {
			continue
		}
		bank, kind := parts[0], parts[1]
		if bankMap[bank] == nil {
			bankMap[bank] = &models.BankRate{Bank: bank, FetchedDate: dateStr}
		}
		switch kind {
		case "saving":
			bankMap[bank].Saving12m = val
		case "lending":
			bankMap[bank].Lending = val
		}
	}

	order := []string{"vcb", "bidv", "agribank", "tcb"}
	var out []models.BankRate
	seen := map[string]bool{}
	for _, bank := range order {
		if r, ok := bankMap[bank]; ok {
			out = append(out, *r)
			seen[bank] = true
		}
	}
	for bank, r := range bankMap {
		if !seen[bank] {
			out = append(out, *r)
		}
	}
	if out == nil {
		out = []models.BankRate{}
	}
	return out, nil
}

func (r *pgTrendsRepo) ListNews(ctx context.Context, limit int) ([]models.NewsItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, source, published_at, title, url
		 FROM news_cache ORDER BY published_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.NewsItem
	for rows.Next() {
		var n models.NewsItem
		var pub time.Time
		rows.Scan(&n.ID, &n.Source, &pub, &n.Title, &n.URL)
		n.PublishedAt = pub.Format(time.RFC3339)
		out = append(out, n)
	}
	if out == nil {
		out = []models.NewsItem{}
	}
	return out, rows.Err()
}

func (r *pgTrendsRepo) UpsertNews(ctx context.Context, items []models.NewsItem) error {
	for _, n := range items {
		_, err := r.db.Exec(ctx,
			`INSERT INTO news_cache (source, published_at, title, url)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (url) DO NOTHING`,
			n.Source, n.PublishedAt, n.Title, n.URL)
		if err != nil {
			return err
		}
	}
	_, err := r.db.Exec(ctx,
		`DELETE FROM news_cache WHERE id NOT IN (
		   SELECT id FROM news_cache ORDER BY published_at DESC LIMIT 50
		 )`)
	return err
}
