# Wealth Trends Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Trends tab to WealthPage showing net worth history vs Vietnamese economic benchmarks (VN-Index, SJC gold, CPI, bank interest rates) with a finance news feed and manual backfill for past net worth data.

**Architecture:** Backend daily cron scrapes public Vietnamese sources (cafef.vn RSS, sjc.com.vn XML, Yahoo Finance for VN-Index, bank rate pages) and stores results in `benchmark_data` and `news_cache` tables. New endpoints serve this data. Frontend Trends tab renders line charts via Recharts with overlay toggles and a bank rates table.

**Tech Stack:** Go/chi/pgx (backend), `golang.org/x/net/html` for scraping, Recharts (frontend), React Query.

---

## File Map

| Action | Path |
|--------|------|
| Create | `supabase/migrations/20260614000002_wealth_trends.sql` |
| Create | `backend/internal/migrate/005_wealth_trends.sql` |
| Modify | `backend/internal/models/models.go` |
| Create | `backend/internal/repo/trends.go` |
| Create | `backend/internal/scraper/scraper.go` |
| Create | `backend/internal/handlers/trends.go` |
| Create | `backend/internal/handlers/trends_test.go` |
| Modify | `backend/cmd/server/main.go` |
| Modify | `frontend/src/api/types.ts` |
| Modify | `frontend/src/api/endpoints.ts` |
| Create | `frontend/src/components/NetWorthChart.tsx` |
| Modify | `frontend/src/pages/WealthPage.tsx` |

---

## Task 1: DB Migration — Wealth Trends

**Files:**
- Create: `supabase/migrations/20260614000002_wealth_trends.sql`
- Create: `backend/internal/migrate/005_wealth_trends.sql`

- [ ] **Step 1: Write the migration**

Create `supabase/migrations/20260614000002_wealth_trends.sql`:

```sql
-- Add note column for manual backfill entries
ALTER TABLE net_worth_snapshots
  ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';

-- Benchmark time series (VN-Index, SJC gold, CPI, bank rates)
CREATE TABLE IF NOT EXISTS benchmark_data (
  id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,
  date   DATE NOT NULL,
  value  NUMERIC(18,4) NOT NULL,
  UNIQUE(source, date)
);
ALTER TABLE benchmark_data ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS benchmark_data_read ON benchmark_data;
CREATE POLICY benchmark_data_read ON benchmark_data FOR SELECT USING (true);

-- Finance news cache
CREATE TABLE IF NOT EXISTS news_cache (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source       TEXT NOT NULL,
  published_at TIMESTAMPTZ NOT NULL,
  title        TEXT NOT NULL,
  url          TEXT NOT NULL,
  fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE news_cache ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS news_cache_read ON news_cache;
CREATE POLICY news_cache_read ON news_cache FOR SELECT USING (true);
```

Copy the same content to `backend/internal/migrate/005_wealth_trends.sql`.

- [ ] **Step 2: Verify files exist**

```bash
ls supabase/migrations/20260614000002_wealth_trends.sql
ls backend/internal/migrate/005_wealth_trends.sql
```

- [ ] **Step 3: Commit**

```bash
git add supabase/migrations/20260614000002_wealth_trends.sql backend/internal/migrate/005_wealth_trends.sql
git commit -m "chore: db migration — benchmark_data, news_cache, net_worth note column"
```

---

## Task 2: Update Backend Models

**Files:**
- Modify: `backend/internal/models/models.go`

- [ ] **Step 1: Update `NetWorthSnapshot` to include Note and manual flag**

In `backend/internal/models/models.go`, replace `NetWorthSnapshot`:

```go
type NetWorthSnapshot struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	SnapshotDate string  `json:"snapshot_date"`
	AssetsValue  float64 `json:"assets_value"`
	CashPosition float64 `json:"cash_position"`
	NetWorth     float64 `json:"net_worth"`
	Note         string  `json:"note"`
}
```

- [ ] **Step 2: Add BenchmarkData, BankRate, NewsItem models**

Add after `NetWorthSnapshot`:

```go
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
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/models/models.go
git commit -m "feat: add BenchmarkData, BankRate, NewsItem models"
```

---

## Task 3: Create Trends Repo

**Files:**
- Create: `backend/internal/repo/trends.go`

- [ ] **Step 1: Create `backend/internal/repo/trends.go`**

```go
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
	placeholders := make([]string, len(sources))
	args := []interface{}{from, to}
	for i, s := range sources {
		placeholders[i] = "$" + string(rune('0'+i+3))
		args = append(args, s)
	}
	q := `SELECT id, source, date, value FROM benchmark_data
	      WHERE date BETWEEN $1::date AND $2::date`
	if len(sources) > 0 {
		q += ` AND source = ANY($3::text[])`
		args = []interface{}{from, to, sources}
	}
	q += ` ORDER BY source, date`

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
	// Store each rate as benchmark_data with source = "bankrate_<bank>_saving" / "bankrate_<bank>_lending"
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
	// Get the most recent saving+lending rate per bank
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

	type rawRate struct {
		source string
		value  float64
		date   string
	}
	var raws []rawRate
	for rows.Next() {
		var rr rawRate
		var d time.Time
		rows.Scan(&rr.source, &rr.value, &d)
		rr.date = d.Format("2006-01-02")
		raws = append(raws, rr)
	}

	// Build BankRate map: bankrate_<bank>_saving / bankrate_<bank>_lending
	bankMap := map[string]*models.BankRate{}
	for _, rr := range raws {
		parts := strings.SplitN(rr.source, "_", 3) // ["bankrate", bank, "saving"|"lending"]
		if len(parts) != 3 {
			continue
		}
		bank := parts[1]
		if bankMap[bank] == nil {
			bankMap[bank] = &models.BankRate{Bank: bank, FetchedDate: rr.date}
		}
		switch parts[2] {
		case "saving":
			bankMap[bank].Saving12m = rr.value
		case "lending":
			bankMap[bank].Lending = rr.value
		}
	}

	order := []string{"vcb", "bidv", "agribank", "tcb"}
	var out []models.BankRate
	for _, bank := range order {
		if r, ok := bankMap[bank]; ok {
			out = append(out, *r)
		}
	}
	// Include any bank not in the order list
	for bank, r := range bankMap {
		inOrder := false
		for _, o := range order {
			if o == bank {
				inOrder = true
				break
			}
		}
		if !inOrder {
			out = append(out, *r)
		}
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
			 ON CONFLICT DO NOTHING`,
			n.Source, n.PublishedAt, n.Title, n.URL)
		if err != nil {
			return err
		}
	}
	// Trim to last 50 items
	_, err := r.db.Exec(ctx,
		`DELETE FROM news_cache WHERE id NOT IN (
		   SELECT id FROM news_cache ORDER BY published_at DESC LIMIT 50
		 )`)
	return err
}
```

- [ ] **Step 2: Verify package compiles**

```bash
cd backend && go build ./internal/repo/... 2>&1
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repo/trends.go
git commit -m "feat: TrendsRepo — snapshots, benchmarks, bank rates, news"
```

---

## Task 4: Create Scraper

**Files:**
- Create: `backend/internal/scraper/scraper.go`

The scraper fetches: VN-Index (Yahoo Finance unofficial API), SJC gold (sjc.com.vn XML), bank rates (Vietcombank/BIDV/Agribank/Techcombank public pages), and news (cafef.vn RSS). All operations are best-effort — failures are logged, not fatal.

- [ ] **Step 1: Add `golang.org/x/net` dependency**

```bash
cd backend && go get golang.org/x/net/html
```

- [ ] **Step 2: Create `backend/internal/scraper/scraper.go`**

```go
package scraper

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

// Run fetches all benchmarks and news, storing into the TrendsRepo.
// All individual fetch errors are logged and skipped — never fatal.
func Run(ctx context.Context, r repo.TrendsRepo) {
	today := time.Now().Format("2006-01-02")

	if v, err := fetchVNIndex(ctx); err != nil {
		log.Printf("scraper: vn_index: %v", err)
	} else {
		r.UpsertBenchmark(ctx, models.BenchmarkData{Source: "vn_index", Date: today, Value: v})
		log.Printf("scraper: vn_index=%.2f", v)
	}

	if v, err := fetchSJCGold(ctx); err != nil {
		log.Printf("scraper: sjc_gold: %v", err)
	} else {
		r.UpsertBenchmark(ctx, models.BenchmarkData{Source: "sjc_gold", Date: today, Value: v})
		log.Printf("scraper: sjc_gold=%.0f", v)
	}

	for _, rate := range fetchBankRates(ctx) {
		rate.FetchedDate = today
		if err := r.UpsertBankRate(ctx, rate); err != nil {
			log.Printf("scraper: bank_rate %s: %v", rate.Bank, err)
		} else {
			log.Printf("scraper: %s saving=%.2f%% lending=%.2f%%", rate.Bank, rate.Saving12m, rate.Lending)
		}
	}

	if news, err := fetchCafefNews(ctx); err != nil {
		log.Printf("scraper: news: %v", err)
	} else {
		if err := r.UpsertNews(ctx, news); err != nil {
			log.Printf("scraper: upsert news: %v", err)
		} else {
			log.Printf("scraper: news=%d items", len(news))
		}
	}
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyLifeOS/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// fetchVNIndex uses Yahoo Finance chart API for ^VNINDEX (free, no auth required).
func fetchVNIndex(ctx context.Context) (float64, error) {
	body, err := httpGet(ctx, "https://query1.finance.yahoo.com/v8/finance/chart/%5EVNINDEX?range=1d&interval=1d")
	if err != nil {
		return 0, err
	}
	// Extract "regularMarketPrice" from JSON via simple regex (avoids encoding/json import cycle)
	re := regexp.MustCompile(`"regularMarketPrice":\s*([0-9]+\.?[0-9]*)`)
	m := re.FindSubmatch(body)
	if m == nil {
		return 0, fmt.Errorf("regularMarketPrice not found in response")
	}
	return strconv.ParseFloat(string(m[1]), 64)
}

// fetchSJCGold parses the SJC XML feed for the "SJC" buy price (VND per tael).
func fetchSJCGold(ctx context.Context) (float64, error) {
	body, err := httpGet(ctx, "https://sjc.com.vn/xml/tygiavang.xml")
	if err != nil {
		return 0, err
	}
	type Item struct {
		Type string `xml:"type,attr"`
		Buy  string `xml:"buy,attr"`
	}
	type Root struct {
		Items []Item `xml:"ratelist>item"`
	}
	var root Root
	if err := xml.Unmarshal(body, &root); err != nil {
		return 0, fmt.Errorf("xml decode: %w", err)
	}
	for _, item := range root.Items {
		if strings.Contains(item.Type, "SJC") {
			s := strings.ReplaceAll(item.Buy, ",", "")
			return strconv.ParseFloat(strings.TrimSpace(s), 64)
		}
	}
	return 0, fmt.Errorf("SJC item not found")
}

// fetchBankRates scrapes standard lending + 12-month saving rates from major VN banks.
// Rates are extracted via regex from the bank's public rate page.
// Returns only banks that were successfully scraped.
func fetchBankRates(ctx context.Context) []models.BankRate {
	type bankTarget struct {
		id      string
		url     string
		saving  *regexp.Regexp
		lending *regexp.Regexp
	}

	targets := []bankTarget{
		{
			id:      "vcb",
			url:     "https://www.vietcombank.com.vn/vi-VN/KHCN/Cong-cu-Tien-ich/Lai-suat",
			saving:  regexp.MustCompile(`12\s*tháng[^%]*?(\d+[,.]?\d*)\s*%`),
			lending: regexp.MustCompile(`Lãi suất cho vay[^%]*?(\d+[,.]?\d*)\s*%`),
		},
		{
			id:      "bidv",
			url:     "https://www.bidv.com.vn/vi/lai-suat-tiet-kiem",
			saving:  regexp.MustCompile(`12\s*tháng[^%]*?(\d+[,.]?\d*)\s*%`),
			lending: regexp.MustCompile(`cho vay[^%]*?(\d+[,.]?\d*)\s*%`),
		},
		{
			id:      "agribank",
			url:     "https://www.agribank.com.vn/vn/lai-suat/lai-suat-huy-dong",
			saving:  regexp.MustCompile(`12\s*tháng[^%]*?(\d+[,.]?\d*)\s*%`),
			lending: regexp.MustCompile(`(\d+[,.]?\d*)\s*%.*cho vay`),
		},
		{
			id:      "tcb",
			url:     "https://techcombank.com/khach-hang-ca-nhan/tiet-kiem/lai-suat-tiet-kiem",
			saving:  regexp.MustCompile(`12\s*tháng[^%]*?(\d+[,.]?\d*)\s*%`),
			lending: regexp.MustCompile(`cho vay[^%]*?(\d+[,.]?\d*)\s*%`),
		},
	}

	parseRate := func(s string) float64 {
		s = strings.ReplaceAll(s, ",", ".")
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	var out []models.BankRate
	for _, t := range targets {
		body, err := httpGet(ctx, t.url)
		if err != nil {
			log.Printf("scraper: bank %s fetch: %v", t.id, err)
			continue
		}
		sm := t.saving.FindSubmatch(body)
		lm := t.lending.FindSubmatch(body)
		if sm == nil || lm == nil {
			log.Printf("scraper: bank %s rate parse failed", t.id)
			continue
		}
		out = append(out, models.BankRate{
			Bank:      t.id,
			Saving12m: parseRate(string(sm[1])),
			Lending:   parseRate(string(lm[1])),
		})
	}
	return out
}

// fetchCafefNews parses cafef.vn chứng khoán RSS feed.
func fetchCafefNews(ctx context.Context) ([]models.NewsItem, error) {
	body, err := httpGet(ctx, "https://cafef.vn/rss/chung-khoan.rss")
	if err != nil {
		return nil, err
	}
	type RSSItem struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
	}
	type RSS struct {
		Items []RSSItem `xml:"channel>item"`
	}
	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("rss decode: %w", err)
	}
	var out []models.NewsItem
	for _, item := range rss.Items {
		pub, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if pub.IsZero() {
			pub, _ = time.Parse(time.RFC1123, item.PubDate)
		}
		out = append(out, models.NewsItem{
			Source:      "cafef",
			PublishedAt: pub.UTC().Format(time.RFC3339),
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.Link),
		})
	}
	return out, nil
}
```

- [ ] **Step 3: Build the scraper package**

```bash
cd backend && go build ./internal/scraper/... 2>&1
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/scraper/scraper.go backend/go.mod backend/go.sum
git commit -m "feat: scraper for VN-Index, SJC gold, bank rates, cafef news"
```

---

## Task 5: Create Trends Handler

**Files:**
- Create: `backend/internal/handlers/trends.go`

- [ ] **Step 1: Create `backend/internal/handlers/trends.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
	"github.com/chiutuanbinh/mylifeos/backend/internal/scraper"
)

type TrendsHandler struct {
	repo     repo.TrendsRepo
	assetRepo repo.AssetRepo
}

func NewTrendsHandler(r repo.TrendsRepo, a repo.AssetRepo) *TrendsHandler {
	return &TrendsHandler{repo: r, assetRepo: a}
}

func (h *TrendsHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	snaps, err := h.repo.ListSnapshots(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snaps)
}

func (h *TrendsHandler) AddSnapshot(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Date     string  `json:"date"`
		NetWorth float64 `json:"net_worth"`
		Note     string  `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Date == "" {
		http.Error(w, `{"error":"date and net_worth required"}`, 400)
		return
	}
	snap, err := h.repo.UpsertSnapshot(r.Context(), models.NetWorthSnapshot{
		UserID:       uid,
		SnapshotDate: body.Date,
		NetWorth:     body.NetWorth,
		Note:         body.Note,
	})
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(snap)
}

func (h *TrendsHandler) ListBenchmarks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sources := strings.Split(q.Get("sources"), ",")
	from := q.Get("from")
	to := q.Get("to")
	if from == "" {
		from = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	var filteredSources []string
	for _, s := range sources {
		if s != "" {
			filteredSources = append(filteredSources, s)
		}
	}
	data, err := h.repo.ListBenchmarks(r.Context(), filteredSources, from, to)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *TrendsHandler) ListBankRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.repo.LatestBankRates(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rates)
}

func (h *TrendsHandler) ListNews(w http.ResponseWriter, r *http.Request) {
	news, err := h.repo.ListNews(r.Context(), 20)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

// TriggerScrape manually triggers the scraper (useful for testing, called once on startup).
func (h *TrendsHandler) TriggerScrape(w http.ResponseWriter, r *http.Request) {
	go scraper.Run(r.Context(), h.repo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "scrape started"})
}
```

- [ ] **Step 2: Build the handlers package**

```bash
cd backend && go build ./internal/handlers/... 2>&1
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handlers/trends.go
git commit -m "feat: TrendsHandler — snapshots, benchmarks, bank rates, news, scrape trigger"
```

---

## Task 6: Write Trends Handler Tests

**Files:**
- Create: `backend/internal/handlers/trends_test.go`

- [ ] **Step 1: Create `backend/internal/handlers/trends_test.go`**

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockTrendsRepo struct{}

func (m *mockTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]models.NetWorthSnapshot, error) {
	return []models.NetWorthSnapshot{
		{ID: "s-1", SnapshotDate: "2026-06-14", NetWorth: 500000000, Note: ""},
	}, nil
}
func (m *mockTrendsRepo) UpsertSnapshot(_ context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error) {
	s.ID = "s-new"
	return s, nil
}
func (m *mockTrendsRepo) UpsertBenchmark(_ context.Context, _ models.BenchmarkData) error { return nil }
func (m *mockTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]models.BenchmarkData, error) {
	return []models.BenchmarkData{
		{ID: "b-1", Source: "vn_index", Date: "2026-06-14", Value: 1250.5},
	}, nil
}
func (m *mockTrendsRepo) LatestBankRates(_ context.Context) ([]models.BankRate, error) {
	return []models.BankRate{
		{Bank: "vcb", Saving12m: 5.5, Lending: 9.0, FetchedDate: "2026-06-14"},
	}, nil
}
func (m *mockTrendsRepo) UpsertBankRate(_ context.Context, _ models.BankRate) error { return nil }
func (m *mockTrendsRepo) ListNews(_ context.Context, _ int) ([]models.NewsItem, error) {
	return []models.NewsItem{
		{ID: "n-1", Source: "cafef", Title: "VN-Index tăng mạnh", URL: "https://cafef.vn/1"},
	}, nil
}
func (m *mockTrendsRepo) UpsertNews(_ context.Context, _ []models.NewsItem) error { return nil }

func TestTrendsListSnapshots(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListSnapshots))

	req := httptest.NewRequest("GET", "/api/v1/net-worth-snapshots", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var snaps []models.NetWorthSnapshot
	if err := json.NewDecoder(w.Body).Decode(&snaps); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ID != "s-1" {
		t.Fatalf("unexpected: %+v", snaps)
	}
}

func TestTrendsAddSnapshot(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"date": "2026-01-01", "net_worth": 400000000, "note": "backfill"})
	req := httptest.NewRequest("POST", "/api/v1/net-worth-snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTrendsAddSnapshot_MissingDate(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"net_worth": 400000000})
	req := httptest.NewRequest("POST", "/api/v1/net-worth-snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrendsListBenchmarks(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	req := httptest.NewRequest("GET", "/api/v1/benchmarks?sources=vn_index&from=2026-01-01&to=2026-06-14", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data []models.BenchmarkData
	json.NewDecoder(w.Body).Decode(&data)
	if len(data) != 1 || data[0].Source != "vn_index" {
		t.Fatalf("unexpected: %+v", data)
	}
}

func TestTrendsListBankRates(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBankRates))

	req := httptest.NewRequest("GET", "/api/v1/bank-rates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rates []models.BankRate
	json.NewDecoder(w.Body).Decode(&rates)
	if len(rates) != 1 || rates[0].Bank != "vcb" {
		t.Fatalf("unexpected: %+v", rates)
	}
}

func TestTrendsListNews(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListNews))

	req := httptest.NewRequest("GET", "/api/v1/news", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var news []models.NewsItem
	json.NewDecoder(w.Body).Decode(&news)
	if len(news) != 1 || news[0].Source != "cafef" {
		t.Fatalf("unexpected: %+v", news)
	}
}
```

Note: `mockAssetRepo` already exists in `assets_test.go` in the same test package.

- [ ] **Step 2: Run tests**

```bash
cd backend && go test ./internal/handlers/... -v -run TestTrends 2>&1 | tail -20
```

Expected: all `TestTrends*` tests PASS.

- [ ] **Step 3: Run full coverage check**

```bash
cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash scripts/hooks/pre-commit
```

Expected: `✓ Coverage OK`.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handlers/trends_test.go
git commit -m "test: TrendsHandler tests"
```

---

## Task 7: Wire Trends Routes + Startup Scrape

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add TrendsHandler to main.go**

In `backend/cmd/server/main.go`, add after the existing handler setup:

```go
trendsRepo    := repo.NewTrendsRepo(db)
trendsHandler := handlers.NewTrendsHandler(trendsRepo, repo.NewAssetRepo(db))
```

Add routes inside the `/api/v1` block:

```go
r.Get("/net-worth-snapshots",  trendsHandler.ListSnapshots)
r.Post("/net-worth-snapshots", trendsHandler.AddSnapshot)
r.Get("/benchmarks",           trendsHandler.ListBenchmarks)
r.Get("/bank-rates",           trendsHandler.ListBankRates)
r.Get("/news",                 trendsHandler.ListNews)
r.Post("/scrape",              trendsHandler.TriggerScrape)
```

- [ ] **Step 2: Add startup scrape (runs once on boot, non-blocking)**

After all handler wiring and before `http.ListenAndServe`, add:

```go
// Kick off a scrape in the background on startup so data is fresh after deploy.
go func() {
	ctx := context.Background()
	scraper.Run(ctx, trendsRepo)
}()
```

Add `"github.com/chiutuanbinh/mylifeos/backend/internal/scraper"` to the import block.

- [ ] **Step 3: Build and verify**

```bash
cd backend && go build ./... 2>&1
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire trends routes, startup scrape"
```

---

## Task 8: Frontend Types + Endpoints

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/api/endpoints.ts`

- [ ] **Step 1: Add new types to `types.ts`**

Append to `frontend/src/api/types.ts`:

```typescript
export interface NetWorthSnapshot {
  id: string
  user_id: string
  snapshot_date: string
  assets_value: number
  cash_position: number
  net_worth: number
  note: string
}

export interface BenchmarkData {
  id: string
  source: string
  date: string
  value: number
}

export interface BankRate {
  bank: string
  saving_12m: number
  lending: number
  fetched_date: string
}

export interface NewsItem {
  id: string
  source: string
  published_at: string
  title: string
  url: string
}
```

- [ ] **Step 2: Add new endpoints to `endpoints.ts`**

Add import of the new types and append the endpoint functions:

```typescript
import type {
  // existing imports...
  NetWorthSnapshot, BenchmarkData, BankRate, NewsItem,
} from './types'

export const getNetWorthSnapshots = () =>
  apiClient.get<NetWorthSnapshot[]>('/net-worth-snapshots').then(r => r.data)

export const addNetWorthSnapshot = (data: { date: string; net_worth: number; note?: string }) =>
  apiClient.post<NetWorthSnapshot>('/net-worth-snapshots', data).then(r => r.data)

export const getBenchmarks = (sources: string[], from: string, to: string) =>
  apiClient.get<BenchmarkData[]>('/benchmarks', { params: { sources: sources.join(','), from, to } }).then(r => r.data)

export const getBankRates = () =>
  apiClient.get<BankRate[]>('/bank-rates').then(r => r.data)

export const getNews = () =>
  apiClient.get<NewsItem[]>('/news').then(r => r.data)
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/api/endpoints.ts
git commit -m "feat: add NetWorthSnapshot, BenchmarkData, BankRate, NewsItem types + endpoints"
```

---

## Task 9: Install Recharts

- [ ] **Step 1: Install**

```bash
cd frontend && npm install recharts
```

- [ ] **Step 2: Verify**

```bash
cd frontend && npm run build 2>&1 | tail -5
```

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add frontend/package.json frontend/package-lock.json
git commit -m "chore: add recharts"
```

---

## Task 10: Create NetWorthChart Component

**Files:**
- Create: `frontend/src/components/NetWorthChart.tsx`

- [ ] **Step 1: Create `frontend/src/components/NetWorthChart.tsx`**

```typescript
import { useMemo, useState } from 'react'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer,
} from 'recharts'
import { Button, Space, Checkbox } from 'antd'
import type { NetWorthSnapshot, BenchmarkData } from '../api/types'

interface Props {
  snapshots: NetWorthSnapshot[]
  benchmarks: BenchmarkData[]
}

type Range = '1M' | '3M' | '6M' | '1Y' | 'ALL'

const BENCHMARK_META: Record<string, { label: string; color: string }> = {
  vn_index: { label: 'VN-Index', color: '#f5a623' },
  sjc_gold: { label: 'SJC Gold', color: '#e8c14a' },
  gso_cpi:  { label: 'CPI',      color: '#7ed321' },
}

function filterByRange(dates: string[], range: Range): string {
  const now = new Date()
  const cutoffs: Record<Range, Date> = {
    '1M':  new Date(now.getFullYear(), now.getMonth() - 1, now.getDate()),
    '3M':  new Date(now.getFullYear(), now.getMonth() - 3, now.getDate()),
    '6M':  new Date(now.getFullYear(), now.getMonth() - 6, now.getDate()),
    '1Y':  new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()),
    'ALL': new Date(0),
  }
  return cutoffs[range].toISOString().split('T')[0]
}

export function NetWorthChart({ snapshots, benchmarks }: Props) {
  const [range, setRange] = useState<Range>('1Y')
  const [activeOverlays, setActiveOverlays] = useState<string[]>(['vn_index', 'sjc_gold'])

  const cutoff = filterByRange([], range)

  const filteredSnaps = snapshots.filter(s => s.snapshot_date >= cutoff)

  const chartData = useMemo(() => {
    if (filteredSnaps.length === 0) return []

    // Base values at start of range for % normalization
    const baseNetWorth = filteredSnaps[0].net_worth
    const baseBySource: Record<string, number> = {}

    // Build a date-keyed map
    const byDate: Record<string, Record<string, number>> = {}

    filteredSnaps.forEach(s => {
      byDate[s.snapshot_date] = { net_worth_pct: ((s.net_worth - baseNetWorth) / baseNetWorth) * 100 }
    })

    benchmarks
      .filter(b => activeOverlays.includes(b.source) && b.date >= cutoff)
      .forEach(b => {
        if (baseBySource[b.source] === undefined) {
          baseBySource[b.source] = b.value
        }
        if (!byDate[b.date]) byDate[b.date] = {}
        const base = baseBySource[b.source]
        byDate[b.date][b.source + '_pct'] = base > 0 ? ((b.value - base) / base) * 100 : 0
      })

    return Object.entries(byDate)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, vals]) => ({ date, ...vals }))
  }, [filteredSnaps, benchmarks, activeOverlays, cutoff])

  const toggleOverlay = (source: string) => {
    setActiveOverlays(prev =>
      prev.includes(source) ? prev.filter(s => s !== source) : [...prev, source]
    )
  }

  const formatPct = (v: number) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Space>
          {(['1M', '3M', '6M', '1Y', 'ALL'] as Range[]).map(r => (
            <Button
              key={r} size="small"
              type={range === r ? 'primary' : 'default'}
              onClick={() => setRange(r)}
            >{r}</Button>
          ))}
        </Space>
        <Space>
          {Object.entries(BENCHMARK_META).map(([source, meta]) => (
            <Checkbox
              key={source}
              checked={activeOverlays.includes(source)}
              onChange={() => toggleOverlay(source)}
              style={{ fontSize: 12 }}
            >
              <span style={{ color: meta.color }}>{meta.label}</span>
            </Checkbox>
          ))}
        </Space>
      </div>

      {chartData.length === 0 ? (
        <div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>
          No data yet. Snapshots accumulate daily.
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={280}>
          <LineChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" tick={{ fontSize: 11 }} tickFormatter={d => d.slice(5)} />
            <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `${v.toFixed(0)}%`} />
            <Tooltip
              formatter={(v: number, name: string) => [formatPct(v), name]}
              labelFormatter={l => `Date: ${l}`}
            />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            <Line
              type="monotone" dataKey="net_worth_pct" name="Net Worth"
              stroke="#1677ff" strokeWidth={2} dot={false} connectNulls
            />
            {activeOverlays.map(source => (
              <Line
                key={source}
                type="monotone"
                dataKey={source + '_pct'}
                name={BENCHMARK_META[source]?.label ?? source}
                stroke={BENCHMARK_META[source]?.color ?? '#aaa'}
                strokeWidth={1.5} dot={false} connectNulls strokeDasharray="4 2"
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Build check**

```bash
cd frontend && npm run build 2>&1 | tail -10
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/NetWorthChart.tsx
git commit -m "feat: NetWorthChart component with range selector + benchmark overlays"
```

---

## Task 11: Add Trends Tab to WealthPage

**Files:**
- Modify: `frontend/src/pages/WealthPage.tsx`

- [ ] **Step 1: Add Trends tab to WealthPage**

In `frontend/src/pages/WealthPage.tsx`:

1. Add imports at the top:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { LineChartOutlined } from '@ant-design/icons'
import type { BankRate, NewsItem } from '../api/types'
import {
  getNetWorthSnapshots, addNetWorthSnapshot,
  getBenchmarks, getBankRates, getNews,
} from '../api/endpoints'
import { NetWorthChart } from '../components/NetWorthChart'
```

2. Add a `TrendsTab` function component before `WealthPage`:

```typescript
const BANK_DISPLAY: Record<string, string> = {
  vcb: 'Vietcombank', bidv: 'BIDV', agribank: 'Agribank', tcb: 'Techcombank',
}

function TrendsTab() {
  const [backfillOpen, setBackfillOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const now = new Date()
  const yearAgo = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()).toISOString().split('T')[0]
  const todayStr = now.toISOString().split('T')[0]

  const { data: snapshots = [] } = useQuery({
    queryKey: ['net-worth-snapshots'],
    queryFn: getNetWorthSnapshots,
  })

  const { data: benchmarks = [] } = useQuery({
    queryKey: ['benchmarks', yearAgo, todayStr],
    queryFn: () => getBenchmarks(['vn_index', 'sjc_gold', 'gso_cpi'], yearAgo, todayStr),
  })

  const { data: bankRates = [] } = useQuery({
    queryKey: ['bank-rates'],
    queryFn: getBankRates,
  })

  const { data: news = [] } = useQuery({
    queryKey: ['news'],
    queryFn: getNews,
  })

  const addMutation = useMutation({
    mutationFn: addNetWorthSnapshot,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['net-worth-snapshots'] })
      setBackfillOpen(false)
      form.resetFields()
    },
  })

  // Summary stats: latest vs 30 days ago
  const latest = snapshots[snapshots.length - 1]
  const thirtyDaysAgo = new Date(now)
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
  const cutoff30 = thirtyDaysAgo.toISOString().split('T')[0]
  const snap30 = snapshots.filter(s => s.snapshot_date <= cutoff30).slice(-1)[0]

  const pctChange = (curr: number, prev?: number) =>
    prev && prev !== 0 ? ((curr - prev) / prev * 100).toFixed(1) : null

  const latestBenchmark = (source: string) => {
    const pts = benchmarks.filter(b => b.source === source).sort((a, b) => a.date.localeCompare(b.date))
    return { latest: pts[pts.length - 1], oldest: pts[0] }
  }

  const vnidx = latestBenchmark('vn_index')
  const gold = latestBenchmark('sjc_gold')

  return (
    <div>
      {/* Summary cards */}
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>Net Worth (30d)</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: '#1677ff' }}>
              {latest ? `₫${(latest.net_worth / 1e6).toFixed(1)}M` : '—'}
            </div>
            {snap30 && latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(latest.net_worth, snap30.net_worth)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(latest.net_worth, snap30.net_worth)}% vs 30d ago
              </div>
            )}
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>VN-Index (1Y)</div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>
              {vnidx.latest ? vnidx.latest.value.toFixed(0) : '—'}
            </div>
            {vnidx.oldest && vnidx.latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(vnidx.latest.value, vnidx.oldest.value)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(vnidx.latest.value, vnidx.oldest.value)}% vs 1Y ago
              </div>
            )}
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>SJC Gold (1Y)</div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>
              {gold.latest ? `${(gold.latest.value / 1e6).toFixed(1)}M/lượng` : '—'}
            </div>
            {gold.oldest && gold.latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(gold.latest.value, gold.oldest.value)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(gold.latest.value, gold.oldest.value)}% vs 1Y ago
              </div>
            )}
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Button size="small" icon={<PlusOutlined />} onClick={() => setBackfillOpen(true)}>
              Add past data point
            </Button>
          </Card>
        </Col>
      </Row>

      {/* Main chart */}
      <Card size="small" title="Net Worth Trend vs Benchmarks (% change from start)" style={{ marginBottom: 16 }}>
        <NetWorthChart snapshots={snapshots} benchmarks={benchmarks} />
      </Card>

      {/* Bank rates */}
      <Card size="small" title="Bank Interest Rates" style={{ marginBottom: 16 }}>
        {bankRates.length === 0 ? (
          <div style={{ color: '#bbb', fontSize: 12 }}>Rates fetched daily. Check back tomorrow.</div>
        ) : (
          <>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <th style={{ padding: '6px 8px', textAlign: 'left', color: '#999', fontWeight: 500 }}>Bank</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', color: '#999', fontWeight: 500 }}>Saving 12m</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', color: '#999', fontWeight: 500 }}>Lending</th>
                </tr>
              </thead>
              <tbody>
                {bankRates.map((r: BankRate) => (
                  <tr key={r.bank} style={{ borderBottom: '1px solid #f5f5f5' }}>
                    <td style={{ padding: '6px 8px' }}>{BANK_DISPLAY[r.bank] ?? r.bank}</td>
                    <td style={{ padding: '6px 8px', textAlign: 'right', color: '#52c41a' }}>{r.saving_12m}%</td>
                    <td style={{ padding: '6px 8px', textAlign: 'right', color: '#ff4d4f' }}>{r.lending}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {bankRates[0] && (
              <div style={{ fontSize: 11, color: '#bbb', marginTop: 6 }}>Updated: {bankRates[0].fetched_date}</div>
            )}
          </>
        )}
      </Card>

      {/* News */}
      <Card size="small" title="Finance News (cafef.vn)">
        {news.length === 0 ? (
          <div style={{ color: '#bbb', fontSize: 12 }}>News fetched daily.</div>
        ) : (
          news.slice(0, 10).map((n: NewsItem) => (
            <div key={n.id} style={{ padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <a href={n.url} target="_blank" rel="noopener noreferrer"
                style={{ fontSize: 13, color: '#1677ff', textDecoration: 'none' }}>
                {n.title}
              </a>
              <div style={{ fontSize: 11, color: '#bbb', marginTop: 2 }}>
                {new Date(n.published_at).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
              </div>
            </div>
          ))
        )}
      </Card>

      {/* Backfill modal */}
      <Modal title="Add Past Net Worth" open={backfillOpen} onCancel={() => setBackfillOpen(false)} footer={null}>
        <Form form={form} layout="vertical"
          onFinish={values => addMutation.mutate({ date: values.date, net_worth: values.net_worth, note: values.note })}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="net_worth" label="Net Worth (₫)" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} min={0} step={1000000} />
          </Form.Item>
          <Form.Item name="note" label="Note (optional)"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

3. Add the Trends tab to the `WealthPage` `Tabs` items:

```typescript
export function WealthPage() {
  return (
    <Tabs
      defaultActiveKey="transactions"
      items={[
        { key: 'transactions', label: 'Transactions', children: <TransactionsTab /> },
        { key: 'budgets',      label: 'Budgets',      children: <BudgetsTab /> },
        { key: 'assets',       label: 'Assets',       children: <AssetsTab /> },
        { key: 'trends',       label: <><LineChartOutlined /> Trends</>, children: <TrendsTab /> },
      ]}
    />
  )
}
```

- [ ] **Step 2: Run lint + build**

```bash
cd frontend && npm run lint && npm run build 2>&1 | tail -15
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/WealthPage.tsx
git commit -m "feat: WealthPage Trends tab — net worth chart, bank rates, news, backfill"
```

---

## Task 12: Final Verification

- [ ] **Step 1: Backend tests + coverage**

```bash
cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash scripts/hooks/pre-commit
```

Expected: `✓ Coverage OK`, all files ≥80%.

- [ ] **Step 2: Frontend lint + build**

```bash
cd frontend && npm run lint && npm run build 2>&1 | tail -10
```

Expected: clean.

- [ ] **Step 3: Integration smoke test**

```bash
bash scripts/integration-test.sh
```

Expected: pages load, no JS crashes.

- [ ] **Step 4: Create and merge PR**

```bash
git push -u origin feat/wealth-trends
gh pr create --title "feat: Wealth Trends tab — net worth history, VN benchmarks, bank rates, news" \
  --body "$(cat <<'EOF'
## Summary
- New Trends tab on Wealth page with net worth history line chart vs VN-Index, SJC gold, CPI (% change normalized)
- Daily backend scraper: VN-Index (Yahoo Finance), SJC gold (sjc.com.vn XML), bank rates (VCB/BIDV/Agribank/TCB), cafef.vn news RSS
- Bank interest rates table (saving 12m + lending per bank), updated daily
- Finance news feed (cafef.vn), 10 latest items
- Manual backfill modal for past net worth data points
- DB: `benchmark_data`, `news_cache` tables + `note` column on `net_worth_snapshots`

## Test plan
- [ ] Navigate to Wealth → Trends tab
- [ ] Chart renders (may be empty on first boot — run `POST /api/v1/scrape` to populate)
- [ ] Add a past data point via backfill modal — appears in chart
- [ ] Bank rates table shows data after first scrape
- [ ] News feed shows headlines with links
- [ ] Range selector (1M/3M/6M/1Y/ALL) filters chart data
- [ ] VN-Index / SJC Gold overlay toggles work
- [ ] Backend tests pass at ≥80% per file

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```
