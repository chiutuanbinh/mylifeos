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
