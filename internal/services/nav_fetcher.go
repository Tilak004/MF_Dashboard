package services

import (
	"client-dashboard/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// MFAPIResponse represents the response from mfapi.in
type MFAPIResponse struct {
	Meta struct {
		FundHouse  string      `json:"fund_house"`
		SchemeType string      `json:"scheme_type"`
		SchemeCode interface{} `json:"scheme_code"` // Can be string or number
		SchemeName string      `json:"scheme_name"`
	} `json:"meta"`
	Data []struct {
		Date string `json:"date"`
		NAV  string `json:"nav"`
	} `json:"data"`
	Status string `json:"status"`
}

const mfapiBaseURL = "https://api.mfapi.in/mf"

// FetchNAVForFund fetches the latest NAV for a specific fund from mfapi.in
func FetchNAVForFund(db *sql.DB, fundID int, schemeCode string) error {
	url := fmt.Sprintf("%s/%s", mfapiBaseURL, schemeCode)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch NAV from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var apiResp MFAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode API response: %w", err)
	}

	if apiResp.Status != "SUCCESS" || len(apiResp.Data) == 0 {
		return fmt.Errorf("no NAV data available for scheme code %s", schemeCode)
	}

	// Save ALL historical NAV data (mfapi returns up to 5 years)
	// We need this for calculating units from old transactions
	savedCount := 0

	for _, navData := range apiResp.Data {

		// Parse date (format: DD-MM-YYYY)
		navDate, err := time.Parse("02-01-2006", navData.Date)
		if err != nil {
			log.Printf("Failed to parse date %s: %v", navData.Date, err)
			continue
		}

		// Parse NAV value
		var navValue float64
		_, err = fmt.Sscanf(navData.NAV, "%f", &navValue)
		if err != nil {
			log.Printf("Failed to parse NAV value %s: %v", navData.NAV, err)
			continue
		}

		// Save to database (CreateNAVHistory handles duplicates)
		nav := &models.NAVHistory{
			FundID:   fundID,
			NAVDate:  navDate,
			NAVValue: navValue,
		}

		if err := models.CreateNAVHistory(db, nav); err != nil {
			// Log error but continue with other records
			log.Printf("Failed to save NAV for %s: %v", navDate.Format("02-01-2006"), err)
			continue
		}

		savedCount++
	}

	if savedCount == 0 {
		return fmt.Errorf("failed to save any NAV data for scheme %s", schemeCode)
	}

	log.Printf("Fetched %d NAV records for fund ID %d (scheme: %s)", savedCount, fundID, schemeCode)
	return nil
}

// FetchNAVForAllActiveFunds fetches NAV for all funds that have transactions
func FetchNAVForAllActiveFunds(db *sql.DB) error {
	funds, err := models.GetActiveFunds(db)
	if err != nil {
		return fmt.Errorf("failed to get active funds: %w", err)
	}

	log.Printf("Starting NAV fetch for %d active funds", len(funds))

	successCount := 0
	errorCount := 0

	for _, fund := range funds {
		if err := FetchNAVForFund(db, fund.ID, fund.SchemeCode); err != nil {
			log.Printf("Error fetching NAV for %s (%s): %v", fund.SchemeName, fund.SchemeCode, err)
			errorCount++
		} else {
			successCount++
		}

		// Sleep for 100ms to avoid overwhelming the API
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("NAV fetch completed: %d successful, %d errors", successCount, errorCount)

	if errorCount > 0 && successCount == 0 {
		return fmt.Errorf("all NAV fetches failed")
	}

	return nil
}

// GetOrFetchNAV retrieves NAV from database or fetches from API if not available
// This is used during CSV import to automatically fetch historical NAV
func GetOrFetchNAV(db *sql.DB, fundID int, schemeCode string, date time.Time) (float64, error) {
	// First try to get from database (exact date)
	nav, err := models.GetNAVByDate(db, fundID, date)
	if err == nil {
		return nav.NAVValue, nil
	}

	// Try to get from ±3 days (market holidays, weekends)
	for i := 1; i <= 3; i++ {
		// Try previous days
		prevDate := date.AddDate(0, 0, -i)
		nav, err = models.GetNAVByDate(db, fundID, prevDate)
		if err == nil {
			log.Printf("Using NAV from %s (requested %s)", prevDate.Format("02-01-2006"), date.Format("02-01-2006"))
			return nav.NAVValue, nil
		}

		// Try next days
		nextDate := date.AddDate(0, 0, i)
		nav, err = models.GetNAVByDate(db, fundID, nextDate)
		if err == nil {
			log.Printf("Using NAV from %s (requested %s)", nextDate.Format("02-01-2006"), date.Format("02-01-2006"))
			return nav.NAVValue, nil
		}
	}

	// NAV not found in database - fetch ALL historical data for this fund
	log.Printf("NAV not found for fund ID %d on date %s, fetching historical data...", fundID, date.Format("02-01-2006"))

	if err := FetchNAVForFund(db, fundID, schemeCode); err != nil {
		return 0, fmt.Errorf("failed to fetch historical NAV: %w", err)
	}

	// Try again to get NAV from database (with ±3 days tolerance)
	nav, err = models.GetNAVByDate(db, fundID, date)
	if err == nil {
		return nav.NAVValue, nil
	}

	for i := 1; i <= 3; i++ {
		prevDate := date.AddDate(0, 0, -i)
		nav, err = models.GetNAVByDate(db, fundID, prevDate)
		if err == nil {
			return nav.NAVValue, nil
		}
	}

	// If still not found, return error
	return 0, fmt.Errorf("NAV not available for fund ID %d on date %s (even after fetching)", fundID, date.Format("02-01-2006"))
}

// FetchRecentNAV fetches only the last 30 days of NAV data for a fund
// This is used by the "Refresh NAV" button to update current values
func FetchRecentNAV(db *sql.DB, fundID int, schemeCode string) error {
	url := fmt.Sprintf("%s/%s", mfapiBaseURL, schemeCode)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch NAV from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var apiResp MFAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode API response: %w", err)
	}

	if apiResp.Status != "SUCCESS" || len(apiResp.Data) == 0 {
		return fmt.Errorf("no NAV data available for scheme code %s", schemeCode)
	}

	// Save only last 30 days of NAV data
	savedCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, navData := range apiResp.Data {
		// Parse date (format: DD-MM-YYYY)
		navDate, err := time.Parse("02-01-2006", navData.Date)
		if err != nil {
			log.Printf("Failed to parse date %s: %v", navData.Date, err)
			continue
		}

		// Only save NAV from last 30 days
		if navDate.Before(thirtyDaysAgo) {
			break // API returns data in descending order (newest first)
		}

		// Parse NAV value
		var navValue float64
		_, err = fmt.Sscanf(navData.NAV, "%f", &navValue)
		if err != nil {
			log.Printf("Failed to parse NAV value %s: %v", navData.NAV, err)
			continue
		}

		// Save to database (CreateNAVHistory handles duplicates)
		nav := &models.NAVHistory{
			FundID:   fundID,
			NAVDate:  navDate,
			NAVValue: navValue,
		}

		if err := models.CreateNAVHistory(db, nav); err != nil {
			// Log error but continue with other records
			log.Printf("Failed to save NAV for %s: %v", navDate.Format("02-01-2006"), err)
			continue
		}

		savedCount++
	}

	if savedCount == 0 {
		log.Printf("No new NAV data to save for fund ID %d (might be up to date)", fundID)
		return nil // Not an error - might already have the data
	}

	log.Printf("Fetched %d recent NAV records for fund ID %d (scheme: %s)", savedCount, fundID, schemeCode)
	return nil
}
