package handlers

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/models"
	"client-dashboard/internal/services"
	"client-dashboard/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
)

// ImportResult represents the result of CSV import
type ImportResult struct {
	Success      int      `json:"success"`
	Failed       int      `json:"failed"`
	Errors       []string `json:"errors"`
	ClientsAdded int      `json:"clients_added"`
	FundsAdded   int      `json:"funds_added"`
}

// ImportTransactionsHandler handles CSV file upload and import
func ImportTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse CSV
	rows, parseErrors, err := utils.ParseTransactionCSV(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := ImportResult{
		Errors: parseErrors,
		Failed: len(parseErrors),
	}

	clientCache := make(map[string]int)       // PAN -> ClientID
	fundCache := make(map[string]int)         // SchemeCode -> FundID

	// Process each row
	for _, row := range rows {
		// Get or create client
		clientID, exists := clientCache[row.ClientPAN]
		if !exists {
			// Check if client exists by PAN
			clients, _ := models.GetAllClients(database.DB)
			found := false
			for _, c := range clients {
				if c.PAN == row.ClientPAN {
					clientID = c.ID
					clientCache[row.ClientPAN] = clientID
					found = true
					break
				}
			}

			if !found {
				// Create new client
				client := &models.Client{
					Name:  row.ClientName,
					PAN:   row.ClientPAN,
					Email: row.ClientEmail,
					Phone: row.ClientPhone,
				}
				if err := models.CreateClient(database.DB, client); err != nil {
					result.Errors = append(result.Errors, "Failed to create client "+row.ClientName+": "+err.Error())
					result.Failed++
					continue
				}
				clientID = client.ID
				clientCache[row.ClientPAN] = clientID
				result.ClientsAdded++
			}
		}

		// Get or create fund
		fundID, exists := fundCache[row.SchemeCode]
		if !exists {
			// Check if fund exists
			fund, err := models.GetFundBySchemeCode(database.DB, row.SchemeCode)
			if err != nil {
				// Create new fund
				fund = &models.Fund{
					SchemeCode: row.SchemeCode,
					SchemeName: row.FundName,
					FundHouse:  row.FundHouse,
					Category:   row.Category,
				}

				// Set risk level based on category
				switch row.Category {
				case "Equity":
					fund.RiskLevel = "High"
				case "Debt":
					fund.RiskLevel = "Low"
				case "Hybrid":
					fund.RiskLevel = "Medium"
				}

				if err := models.CreateFund(database.DB, fund); err != nil {
					result.Errors = append(result.Errors, "Failed to create fund "+row.FundName+": "+err.Error())
					result.Failed++
					continue
				}
				result.FundsAdded++
			}
			fundID = fund.ID
			fundCache[row.SchemeCode] = fundID
		}

		// Calculate units
		units, nav, err := services.CalculateUnitsForTransaction(
			database.DB,
			fundID,
			row.SchemeCode,
			row.Amount,
			row.TransactionDate,
		)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to calculate units for transaction: "+err.Error())
			result.Failed++
			continue
		}

		// Skip if same transaction already exists (prevent duplicate imports)
		var existingCount int
		database.DB.QueryRow(
			`SELECT COUNT(*) FROM transactions WHERE client_id=$1 AND fund_id=$2 AND transaction_date=$3 AND amount=$4 AND transaction_type=$5`,
			clientID, fundID, row.TransactionDate, row.Amount, row.TransactionType,
		).Scan(&existingCount)
		if existingCount > 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Line skipped: duplicate transaction for %s on %s (%.0f)", row.ClientName, row.TransactionDate.Format("02-01-2006"), row.Amount))
			result.Failed++
			continue
		}

		// Create transaction
		txn := &models.Transaction{
			ClientID:        clientID,
			FundID:          fundID,
			TransactionType: row.TransactionType,
			TransactionDate: row.TransactionDate,
			Amount:          row.Amount,
			NAV:             nav,
			Units:           units,
			FolioNumber:     row.FolioNumber,
			Notes:           row.Notes,
		}

		if err := models.CreateTransaction(database.DB, txn); err != nil {
			result.Errors = append(result.Errors, "Failed to create transaction: "+err.Error())
			result.Failed++
			continue
		}

		result.Success++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetSampleCSVHandler returns a sample CSV file
func GetSampleCSVHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=sample_transactions.csv")
	w.Write([]byte(utils.GenerateSampleCSV()))
}
