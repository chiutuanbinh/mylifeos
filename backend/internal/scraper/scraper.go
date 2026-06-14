package scraper

import (
	"context"
	"encoding/json"
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

	if news, err := fetchNews(ctx); err != nil {
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

// fetchVNIndex uses CafeF real-time prices API.
// Returns error when market is closed (Price=0) so DB retains last known value.
func fetchVNIndex(ctx context.Context) (float64, error) {
	body, err := httpGet(ctx, "https://cafef.vn/du-lieu/Ajax/PageNew/RealtimePricesHeader.ashx?symbols=VNINDEX")
	if err != nil {
		return 0, err
	}
	var data map[string]struct {
		Price float64 `json:"Price"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("json decode: %w", err)
	}
	v, ok := data["VNINDEX"]
	if !ok {
		return 0, fmt.Errorf("VNINDEX not found in response")
	}
	if v.Price == 0 {
		return 0, fmt.Errorf("market closed or no data")
	}
	return v.Price, nil
}

// fetchSJCGold fetches SJC buy price (VND per tael) from PNJ gold API.
// PNJ returns prices in thousands VND (e.g. "144.000" = 144,000 thousand = 144,000,000 VND).
func fetchSJCGold(ctx context.Context) (float64, error) {
	body, err := httpGet(ctx, "https://edge-cf-api.pnj.io/ecom-frontend/v3/get-gold-price")
	if err != nil {
		return 0, err
	}
	var resp struct {
		Locations []struct {
			GoldType []struct {
				Name   string `json:"name"`
				GiaMua string `json:"gia_mua"`
			} `json:"gold_type"`
		} `json:"locations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("json decode: %w", err)
	}
	for _, loc := range resp.Locations {
		for _, gt := range loc.GoldType {
			if strings.Contains(gt.Name, "SJC") {
				// "144.000" VN format: dot is thousands separator → 144000 thousand VND → × 1000 = VND
				s := strings.ReplaceAll(gt.GiaMua, ".", "")
				v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
				if err != nil {
					return 0, fmt.Errorf("parse price %q: %w", gt.GiaMua, err)
				}
				return v * 1000, nil
			}
		}
	}
	return 0, fmt.Errorf("SJC not found in PNJ response")
}

// fetchBankRates scrapes 12-month saving rates from VNExpress's regularly-updated
// bank rate article. Individual bank sites use JS rendering so are unreliable.
//
// Source: https://vnexpress.net/chu-de/lai-suat-ngan-hang-3210
func fetchBankRates(ctx context.Context) []models.BankRate {
	parseRate := func(s string) float64 {
		s = strings.ReplaceAll(s, ",", ".")
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	// Step 1: fetch the topic page to find the latest rate article URL.
	topicBody, err := httpGet(ctx, "https://vnexpress.net/chu-de/lai-suat-ngan-hang-3210")
	if err != nil {
		log.Printf("scraper: bank rates topic fetch: %v", err)
		return nil
	}
	articleRe := regexp.MustCompile(`href="(https://vnexpress\.net/lai-suat-tiet-kiem-ngan-hang-nao-cao-nhat-\d+\.html)"`)
	am := articleRe.FindSubmatch(topicBody)
	if am == nil {
		log.Printf("scraper: bank rates article link not found on topic page")
		return nil
	}
	articleURL := string(am[1])

	// Step 2: fetch the article body.
	articleBody, err := httpGet(ctx, articleURL)
	if err != nil {
		log.Printf("scraper: bank rates article fetch %s: %v", articleURL, err)
		return nil
	}
	text := string(articleBody)

	// Step 3: extract rates from article text.
	// Range pattern like "6,6-6,8" → use average. Single like "6,5" → use as-is.
	rateRe := regexp.MustCompile(`(\d+[,\.]\d+)(?:-(\d+[,\.]\d+))?\s*%`)

	extractRate := func(context string) float64 {
		m := rateRe.FindStringSubmatch(context)
		if m == nil {
			return 0
		}
		lo := parseRate(m[1])
		if m[2] != "" {
			hi := parseRate(m[2])
			return (lo + hi) / 2
		}
		return lo
	}

	// Map Vietnamese bank name patterns to IDs.
	type bankMapping struct {
		id      string
		pattern *regexp.Regexp
	}
	mappings := []bankMapping{
		{"vcb", regexp.MustCompile(`(?i)Vietcombank|VCB`)},
		{"bidv", regexp.MustCompile(`(?i)BIDV`)},
		{"agribank", regexp.MustCompile(`(?i)Agribank`)},
		{"tcb", regexp.MustCompile(`(?i)Techcombank|TCB`)},
	}

	found := map[string]float64{}

	// Split text into sentences and look for bank + rate co-occurrence.
	sentences := strings.Split(text, ".")
	for _, sent := range sentences {
		for _, bm := range mappings {
			if bm.pattern.MatchString(sent) {
				if r := extractRate(sent); r > 0 {
					if _, already := found[bm.id]; !already {
						found[bm.id] = r
					}
				}
			}
		}
	}

	// Fallback: if state-owned banks mention a group rate, apply to unmapped ones.
	// Article often says "BIDV, Agribank, VietinBank trả X-Y%".
	stateRe := regexp.MustCompile(`(?i)(BIDV|Agribank|VietinBank)[^\n.]*?(\d+[,\.]\d+(?:-\d+[,\.]\d+)?)\s*%`)
	if sm := stateRe.FindStringSubmatch(text); sm != nil {
		groupRate := extractRate(sm[2] + "%")
		for _, id := range []string{"vcb", "bidv", "agribank"} {
			if _, ok := found[id]; !ok && groupRate > 0 {
				found[id] = groupRate
			}
		}
	}

	if len(found) == 0 {
		log.Printf("scraper: bank rates no rates parsed from %s", articleURL)
		return nil
	}

	var out []models.BankRate
	for id, rate := range found {
		out = append(out, models.BankRate{
			Bank:      id,
			Saving12m: rate,
			Lending:   0, // VNExpress article focuses on saving rates
		})
		log.Printf("scraper: bank %s saving12m=%.2f%%", id, rate)
	}
	return out
}

// fetchNews parses VnEconomy chứng khoán RSS feed.
func fetchNews(ctx context.Context) ([]models.NewsItem, error) {
	body, err := httpGet(ctx, "https://vneconomy.vn/chung-khoan.rss")
	if err != nil {
		return nil, err
	}
	type RSSItem struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		GUID    string `xml:"guid"`
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
		link := strings.TrimSpace(item.Link)
		if link == "" {
			link = strings.TrimSpace(item.GUID)
		}
		if link == "" {
			continue
		}
		out = append(out, models.NewsItem{
			Source:      "vneconomy",
			PublishedAt: pub.UTC().Format(time.RFC3339),
			Title:       strings.TrimSpace(item.Title),
			URL:         link,
		})
	}
	return out, nil
}
