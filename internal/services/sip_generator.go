package services

import (
	"client-dashboard/internal/models"
	"database/sql"
	"time"
)

// GeneratedSIPTransaction represents a transaction generated from a SIP schedule
type GeneratedSIPTransaction struct {
	SIPScheduleID  int       `json:"sip_schedule_id"`
	ClientID       int       `json:"client_id"`
	FundID         int       `json:"fund_id"`
	Amount         float64   `json:"amount"`
	ExpectedDate   time.Time `json:"expected_date"`
	TransactionID  *int      `json:"transaction_id,omitempty"`  // Null if not executed
	ActualDate     *time.Time `json:"actual_date,omitempty"`
	IsMissed       bool      `json:"is_missed"`
}

// GenerateExpectedInstallments generates all expected SIP installments from start date to today
// Only returns installments that should have been executed (past dates + today)
func GenerateExpectedInstallments(schedule *models.SIPSchedule) []time.Time {
	var installments []time.Time

	if !schedule.IsActive {
		return installments
	}

	today := time.Now()
	current := schedule.StartDate

	// Generate installments until today
	for !current.After(today) {
		// Check if we've reached end date (if specified)
		if schedule.EndDate != nil && current.After(*schedule.EndDate) {
			break
		}

		installments = append(installments, current)

		// Calculate next installment date based on frequency
		switch schedule.Frequency {
		case "MONTHLY":
			current = current.AddDate(0, 1, 0)
		case "QUARTERLY":
			current = current.AddDate(0, 3, 0)
		default:
			// Default to monthly
			current = current.AddDate(0, 1, 0)
		}
	}

	return installments
}

// GetSIPGeneratedTransactions generates all SIP transactions for a client
// This creates virtual transactions from SIP schedules for portfolio calculation
func GetSIPGeneratedTransactions(db *sql.DB, clientID int) ([]models.Transaction, error) {
	// Get all SIP schedules for client
	schedules, err := models.GetSIPSchedulesByClient(db, clientID)
	if err != nil {
		return nil, err
	}

	var transactions []models.Transaction

	for _, schedule := range schedules {
		if !schedule.IsActive {
			continue
		}

		// Generate expected installments
		installments := GenerateExpectedInstallments(&schedule)

		// Create transactions for each installment
		for _, date := range installments {
			// Check if actual transaction exists for this date (±3 days tolerance)
			hasActual, actualTxn := checkActualTransaction(db, schedule.ClientID, schedule.FundID, date)

			if hasActual {
				// Use actual transaction
				transactions = append(transactions, *actualTxn)
			} else {
				// Create virtual transaction from SIP schedule
				// We'll fetch NAV and calculate units when needed
				transactions = append(transactions, models.Transaction{
					ID:              0, // Virtual transaction (ID = 0)
					ClientID:        schedule.ClientID,
					FundID:          schedule.FundID,
					TransactionType: "SIP",
					TransactionDate: date,
					Amount:          schedule.Amount,
					Units:           0, // Will be calculated using NAV
					NAV:             0, // Will be fetched
					FolioNumber:     "",
					Notes:           "Auto-generated from SIP schedule",
				})
			}
		}
	}

	return transactions, nil
}

// checkActualTransaction checks if an actual transaction exists for a SIP installment
func checkActualTransaction(db *sql.DB, clientID, fundID int, expectedDate time.Time) (bool, *models.Transaction) {
	// Check ±3 days tolerance for weekends/holidays
	for i := -3; i <= 3; i++ {
		checkDate := expectedDate.AddDate(0, 0, i)

		query := `
			SELECT id, client_id, fund_id, transaction_type, transaction_date, amount, nav, units, folio_number, notes, created_at
			FROM transactions
			WHERE client_id = $1 AND fund_id = $2 AND transaction_date = $3 AND transaction_type = 'SIP'
		`

		var txn models.Transaction
		err := db.QueryRow(query, clientID, fundID, checkDate).Scan(
			&txn.ID, &txn.ClientID, &txn.FundID, &txn.TransactionType,
			&txn.TransactionDate, &txn.Amount, &txn.NAV, &txn.Units,
			&txn.FolioNumber, &txn.Notes, &txn.CreatedAt,
		)

		if err == nil {
			return true, &txn
		}
	}

	return false, nil
}

// GetAllSIPInstallments gets all expected vs actual SIP installments for reporting
func GetAllSIPInstallments(db *sql.DB) ([]GeneratedSIPTransaction, error) {
	schedules, err := models.GetActiveSIPSchedules(db)
	if err != nil {
		return nil, err
	}

	var allInstallments []GeneratedSIPTransaction

	for _, schedule := range schedules {
		installments := GenerateExpectedInstallments(&schedule)

		for _, expectedDate := range installments {
			hasActual, actualTxn := checkActualTransaction(db, schedule.ClientID, schedule.FundID, expectedDate)

			genTxn := GeneratedSIPTransaction{
				SIPScheduleID: schedule.ID,
				ClientID:      schedule.ClientID,
				FundID:        schedule.FundID,
				Amount:        schedule.Amount,
				ExpectedDate:  expectedDate,
				IsMissed:      !hasActual && expectedDate.Before(time.Now().AddDate(0, 0, -7)), // Missed if >7 days ago
			}

			if hasActual {
				genTxn.TransactionID = &actualTxn.ID
				genTxn.ActualDate = &actualTxn.TransactionDate
			}

			allInstallments = append(allInstallments, genTxn)
		}
	}

	return allInstallments, nil
}
