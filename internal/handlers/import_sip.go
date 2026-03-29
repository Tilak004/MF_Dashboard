package handlers

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/models"
	"client-dashboard/internal/utils"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SIPImportResult struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	ClientsAdded int      `json:"clients_added"`
	FundsAdded   int      `json:"funds_added"`
	Errors       []string `json:"errors"`
}

// ImportSIPSchedulesHandler imports SIP schedules from CSV file
func ImportSIPSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	result := SIPImportResult{
		Errors: []string{},
	}

	// Track created entities
	clientsMap := make(map[string]int)    // PAN -> ClientID
	fundsMap := make(map[string]int)      // SchemeCode -> FundID

	// Skip header row
	if _, err := reader.Read(); err != nil {
		http.Error(w, "Invalid CSV format", http.StatusBadRequest)
		return
	}

	// Process each row
	lineNum := 1
	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Failed to parse CSV row", lineNum))
			result.FailedCount++
			continue
		}

		// Expected columns: Client Name, PAN, Fund Name, Scheme Code, Amount, Start Date, Frequency, Day of Month
		if len(record) < 8 {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Insufficient columns", lineNum))
			result.FailedCount++
			continue
		}

		clientName := strings.TrimSpace(record[0])
		pan := strings.TrimSpace(record[1])
		fundName := strings.TrimSpace(record[2])
		schemeCode := strings.TrimSpace(record[3])
		amountStr := strings.TrimSpace(record[4])
		startDateStr := strings.TrimSpace(record[5])
		frequency := strings.TrimSpace(record[6])
		dayOfMonthStr := strings.TrimSpace(record[7])

		// Skip empty lines
		if clientName == "" && pan == "" && fundName == "" && amountStr == "" {
			continue
		}

		// Parse amount
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Invalid amount '%s'", lineNum, amountStr))
			result.FailedCount++
			continue
		}

		// Parse start date (DD/MM/YYYY)
		startDate, err := time.Parse("02/01/2006", startDateStr)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Invalid date format '%s'. Use DD/MM/YYYY", lineNum, startDateStr))
			result.FailedCount++
			continue
		}

		// Parse day of month
		dayOfMonth, err := strconv.Atoi(dayOfMonthStr)
		if err != nil || dayOfMonth < 1 || dayOfMonth > 31 {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Invalid day of month '%s'", lineNum, dayOfMonthStr))
			result.FailedCount++
			continue
		}

		// Get or create client
		clientID, ok := clientsMap[pan]
		if !ok {
			// Try to find existing client by PAN
			existingClients, err := models.GetAllClients(database.DB)
			var foundClient *models.Client
			if err == nil {
				for _, c := range existingClients {
					if c.PAN == pan {
						foundClient = &c
						break
					}
				}
			}

			if foundClient != nil {
				clientID = foundClient.ID
				clientsMap[pan] = clientID
			} else {
				// Create new client
				newClient := &models.Client{
					Name: clientName,
					PAN:  pan,
				}
				if err := models.CreateClient(database.DB, newClient); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Failed to create client: %v", lineNum, err))
					result.FailedCount++
					continue
				}
				clientID = newClient.ID
				clientsMap[pan] = clientID
				result.ClientsAdded++
				utils.LogInfo("Import", fmt.Sprintf("Created client: %s (ID: %d)", clientName, clientID))
			}
		}

		// Get or create fund
		fundID, ok := fundsMap[schemeCode]
		if !ok {
			fund, err := models.GetFundBySchemeCode(database.DB, schemeCode)
			if err != nil {
				// Create new fund
				// Determine category from fund name
				category := determineCategoryFromName(fundName)

				newFund := &models.Fund{
					SchemeCode: schemeCode,
					SchemeName: fundName,
					Category:   category,
				}
				if err := models.CreateFund(database.DB, newFund); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Failed to create fund: %v", lineNum, err))
					result.FailedCount++
					continue
				}
				fundID = newFund.ID
				fundsMap[schemeCode] = fundID
				result.FundsAdded++
				utils.LogInfo("Import", fmt.Sprintf("Created fund: %s (ID: %d)", fundName, fundID))
			} else {
				fundID = fund.ID
				fundsMap[schemeCode] = fundID
			}
		}

		// Skip if a SIP already exists for same client + fund + start date (prevent duplicate imports)
		var existingCount int
		database.DB.QueryRow(
			`SELECT COUNT(*) FROM sip_schedules WHERE client_id=$1 AND fund_id=$2 AND start_date=$3`,
			clientID, fundID, startDate,
		).Scan(&existingCount)
		if existingCount > 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: SIP already exists for %s - %s (skipped)", lineNum, clientName, fundName))
			result.FailedCount++
			continue
		}

		// Create SIP schedule
		sipSchedule := &models.SIPSchedule{
			ClientID:   clientID,
			FundID:     fundID,
			Amount:     amount,
			StartDate:  startDate,
			EndDate:    nil, // No end date
			Frequency:  frequency,
			DayOfMonth: dayOfMonth,
			IsActive:   true,
		}

		if err := models.CreateSIPSchedule(database.DB, sipSchedule); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Failed to create SIP schedule: %v", lineNum, err))
			result.FailedCount++
			continue
		}

		result.SuccessCount++
		utils.LogInfo("Import", fmt.Sprintf("Created SIP schedule: %s - %s - ₹%.2f/month", clientName, fundName, amount))
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// determineCategoryFromName determines fund category from fund name
func determineCategoryFromName(name string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "equity") || strings.Contains(nameLower, "large cap") ||
		strings.Contains(nameLower, "mid cap") || strings.Contains(nameLower, "small cap") ||
		strings.Contains(nameLower, "flexi cap") || strings.Contains(nameLower, "multi cap") {
		return "Equity"
	}

	if strings.Contains(nameLower, "debt") || strings.Contains(nameLower, "bond") ||
		strings.Contains(nameLower, "gilt") || strings.Contains(nameLower, "liquid") {
		return "Debt"
	}

	if strings.Contains(nameLower, "balanced") || strings.Contains(nameLower, "hybrid") ||
		strings.Contains(nameLower, "multi asset") || strings.Contains(nameLower, "gold") {
		return "Hybrid"
	}

	return "Other"
}
