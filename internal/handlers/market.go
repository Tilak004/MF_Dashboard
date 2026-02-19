package handlers

import (
	"client-dashboard/internal/utils"
	"encoding/csv"
	"encoding/json"
	"fmt"
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

// ── NSE India ─────────────────────────────────────────────────────────────

type nseAllIndicesResp struct {
	Data []struct {
		IndexSymbol   string  `json:"indexSymbol"`
		Index         string  `json:"index"`
		Last          float64 `json:"last"`
		Variation     float64 `json:"variation"`
		PercentChange float64 `json:"percentChange"`
	} `json:"data"`
	Timestamp    string `json:"timestamp"`
	MarketStatus struct {
		MarketState string `json:"marketState"`
	} `json:"marketStatus"`
}

// wantedNSEIndices maps NSE indexSymbol → display name
// Sensex is a BSE index and is not available from NSE's allIndices API.
var wantedNSEIndices = map[string]string{
	"NIFTY 50":   "Nifty 50",
	"NIFTY BANK": "Bank Nifty",
}

// fetchNSEIndices hits NSE's allIndices public API.
func fetchNSEIndices() ([]MarketIndex, bool, error) {
	client := &http.Client{Timeout: 12 * time.Second}

	apiReq, _ := http.NewRequest("GET", "https://www.nseindia.com/api/allIndices", nil)
	apiReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	apiReq.Header.Set("Accept", "application/json, text/plain, */*")
	apiReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
	apiReq.Header.Set("Referer", "https://www.nseindia.com/")

	apiResp, err := client.Do(apiReq)
	if err != nil {
		return nil, false, fmt.Errorf("NSE allIndices fetch failed: %w", err)
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("NSE allIndices returned HTTP %d", apiResp.StatusCode)
	}

	var nse nseAllIndicesResp
	if err := json.NewDecoder(apiResp.Body).Decode(&nse); err != nil {
		return nil, false, fmt.Errorf("failed to decode NSE response: %w", err)
	}

	isOpen := nse.MarketStatus.MarketState == "Open"
	ist := time.FixedZone("IST", 5*3600+30*60)
	lastUpdated := time.Now().In(ist).Format("02-Jan 15:04")

	var result []MarketIndex
	for _, d := range nse.Data {
		displayName, ok := wantedNSEIndices[d.IndexSymbol]
		if !ok {
			continue
		}
		result = append(result, MarketIndex{
			Name:          displayName,
			Symbol:        d.IndexSymbol,
			Price:         d.Last,
			Change:        d.Variation,
			ChangePercent: d.PercentChange,
			LastUpdated:   lastUpdated,
			IsMarketOpen:  isOpen,
		})
	}

	return result, isOpen, nil
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

	// Fetch NSE indices (Nifty 50, Bank Nifty, Sensex if available)
	wg.Add(1)
	go func() {
		defer wg.Done()
		indices, _, err := fetchNSEIndices()
		if err != nil {
			utils.LogError("Market", fmt.Errorf("NSE fetch failed: %w", err))
			return
		}
		mu.Lock()
		nseIndices = indices
		mu.Unlock()
		utils.LogInfo("Market", fmt.Sprintf("NSE returned %d indices", len(indices)))
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
