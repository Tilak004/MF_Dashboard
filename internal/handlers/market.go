package handlers

import (
	"client-dashboard/internal/utils"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Simple in-memory cache — avoids hitting APIs on every dashboard load
var (
	marketCache     []MarketIndex
	marketCacheTime time.Time
	marketCacheMu   sync.Mutex
	marketCacheTTL  = 5 * time.Minute
)

// MarketIndex holds data for a single market index or instrument
type MarketIndex struct {
	Name          string  `json:"name"`
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	LastUpdated   string  `json:"last_updated"`
	IsMarketOpen  bool    `json:"is_market_open"`
}

// ── Yahoo Finance ──────────────────────────────────────────────────────────

// yahooChartResp is the response from Yahoo Finance's v8 chart API.
type yahooChartResp struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

// indianIndices lists the Yahoo Finance symbols to fetch.
var indianIndices = []struct {
	symbol      string
	displayName string
}{
	{"^NSEI", "Nifty 50"},
	{"^NSEBANK", "Bank Nifty"},
	{"^BSESN", "Sensex"},
}

// fetchYahooIndices fetches Indian indices from Yahoo Finance's v8 chart API.
// Uses query2 + v8/chart which works from cloud servers without session cookies.
func fetchYahooIndices() ([]MarketIndex, error) {
	ist := time.FixedZone("IST", 5*3600+30*60)
	now := time.Now().In(ist)
	isOpen := isNSEMarketOpen(now)
	lastUpdated := now.Format("02-Jan 15:04")

	var wg sync.WaitGroup
	var mu sync.Mutex
	var result []MarketIndex

	for _, idx := range indianIndices {
		idx := idx
		wg.Add(1)
		go func() {
			defer wg.Done()

			// ^NSEI → %5ENSEI (^ must be percent-encoded in the path)
			url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%%5E%s?interval=1d&range=2d",
				idx.symbol[1:])

			client := &http.Client{Timeout: 12 * time.Second}
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != http.StatusOK {
				log.Printf("Yahoo Finance fetch failed for %s: %v", idx.symbol, err)
				return
			}
			defer resp.Body.Close()

			var yResp yahooChartResp
			if err := json.NewDecoder(resp.Body).Decode(&yResp); err != nil || len(yResp.Chart.Result) == 0 {
				log.Printf("Failed to decode Yahoo Finance response for %s: %v", idx.symbol, err)
				return
			}

			meta := yResp.Chart.Result[0].Meta
			change := meta.RegularMarketPrice - meta.ChartPreviousClose
			changePct := 0.0
			if meta.ChartPreviousClose > 0 {
				changePct = (change / meta.ChartPreviousClose) * 100
			}

			mu.Lock()
			result = append(result, MarketIndex{
				Name:          idx.displayName,
				Symbol:        idx.symbol,
				Price:         meta.RegularMarketPrice,
				Change:        change,
				ChangePercent: changePct,
				LastUpdated:   lastUpdated,
				IsMarketOpen:  isOpen,
			})
			mu.Unlock()
		}()
	}

	wg.Wait()
	return result, nil
}

// ── Stooq CSV ─────────────────────────────────────────────────────────────

// fetchStooqQuote fetches a single quote from Stooq's CSV API.
// Format: ?f=sd2t2ohlcvn => Symbol, Date, Time, Open, High, Low, Close, Volume, Name
func fetchStooqQuote(name, code string) (*MarketIndex, error) {
	url := fmt.Sprintf("https://stooq.com/q/l/?s=%s&f=sd2t2ohlcvn&h&e=csv", code)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stooq request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stooq returned HTTP %d for %s", resp.StatusCode, code)
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, fmt.Errorf("stooq CSV parse error for %s: %v", code, err)
	}

	// Column order: Symbol(0), Date(1), Time(2), Open(3), High(4), Low(5), Close(6), Volume(7), Name(8)
	row := records[1]
	if len(row) < 7 {
		return nil, fmt.Errorf("stooq: insufficient columns for %s", code)
	}

	openPrice, _ := strconv.ParseFloat(row[3], 64)
	closePrice, _ := strconv.ParseFloat(row[6], 64)
	if closePrice == 0 { // "N/D" parses to 0
		return nil, fmt.Errorf("stooq: no price data for %s (N/D)", code)
	}

	change := closePrice - openPrice
	changePct := 0.0
	if openPrice > 0 {
		changePct = (change / openPrice) * 100
	}

	dateStr := row[1]
	timeStr := row[2]
	updated := dateStr
	if len(timeStr) >= 5 && timeStr != "N/D" {
		updated = dateStr + " " + timeStr[:5]
	}

	ist := time.FixedZone("IST", 5*3600+30*60)
	now := time.Now().In(ist)
	isOpen := isNSEMarketOpen(now)

	return &MarketIndex{
		Name:          name,
		Symbol:        code,
		Price:         closePrice,
		Change:        change,
		ChangePercent: changePct,
		LastUpdated:   updated,
		IsMarketOpen:  isOpen,
	}, nil
}

// isNSEMarketOpen returns true during NSE trading hours (9:15 AM – 3:30 PM IST, Mon–Fri).
func isNSEMarketOpen(t time.Time) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 9*60+15 && mins <= 15*60+30
}

// ── Handler ───────────────────────────────────────────────────────────────

// GetMarketOverviewHandler fetches live market data and returns it as JSON.
// Indian indices come from NSE's API; Gold and USD/INR come from Stooq.
// Results are cached for 5 minutes.
func GetMarketOverviewHandler(w http.ResponseWriter, r *http.Request) {
	// Serve from cache if still fresh
	marketCacheMu.Lock()
	if marketCache != nil && time.Since(marketCacheTime) < marketCacheTTL {
		cached := marketCache
		marketCacheMu.Unlock()
		utils.LogInfo("Market", "Serving market data from cache")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}
	marketCacheMu.Unlock()

	utils.LogInfo("Market", "Fetching fresh market data (NSE + Stooq)")

	var wg sync.WaitGroup
	var mu sync.Mutex
	var nseIndices []MarketIndex
	var stooqResults []*MarketIndex

	// Fetch Indian indices from Yahoo Finance (works from cloud servers)
	wg.Add(1)
	go func() {
		defer wg.Done()
		indices, err := fetchYahooIndices()
		if err != nil {
			utils.LogError("Market", fmt.Errorf("Yahoo Finance fetch failed: %w", err))
			return
		}
		mu.Lock()
		nseIndices = indices
		mu.Unlock()
		utils.LogInfo("Market", fmt.Sprintf("Yahoo Finance returned %d indices", len(indices)))
	}()

	// Fetch Gold from Stooq
	wg.Add(1)
	go func() {
		defer wg.Done()
		mi, err := fetchStooqQuote("Gold (USD/oz)", "xauusd")
		if err != nil {
			utils.LogError("Market", fmt.Errorf("Stooq gold failed: %w", err))
			return
		}
		mu.Lock()
		stooqResults = append(stooqResults, mi)
		mu.Unlock()
	}()

	// Fetch USD/INR from Stooq
	wg.Add(1)
	go func() {
		defer wg.Done()
		mi, err := fetchStooqQuote("USD / INR", "usdinr")
		if err != nil {
			utils.LogError("Market", fmt.Errorf("Stooq USD/INR failed: %w", err))
			return
		}
		mu.Lock()
		stooqResults = append(stooqResults, mi)
		mu.Unlock()
	}()

	wg.Wait()

	// Combine: NSE indices first, then commodities/forex
	var indices []MarketIndex
	indices = append(indices, nseIndices...)
	for _, mi := range stooqResults {
		if mi != nil {
			indices = append(indices, *mi)
		}
	}

	if len(indices) > 0 {
		marketCacheMu.Lock()
		marketCache = indices
		marketCacheTime = time.Now()
		marketCacheMu.Unlock()
	}

	utils.LogInfo("Market", fmt.Sprintf("Returning %d market instruments", len(indices)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(indices)
}
