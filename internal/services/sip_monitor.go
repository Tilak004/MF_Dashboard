package services

import (
	"client-dashboard/internal/models"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// MissedSIP represents a missed SIP installment
type MissedSIP struct {
	ClientID     int       `json:"client_id"`
	ClientName   string    `json:"client_name"`
	ClientEmail  string    `json:"client_email"`
	FundID       int       `json:"fund_id"`
	FundName     string    `json:"fund_name"`
	ExpectedDate time.Time `json:"expected_date"`
	Amount       float64   `json:"amount"`
	DaysMissed   int       `json:"days_missed"`
}

// CheckSIPCompliance checks all active SIP schedules for missed installments
func CheckSIPCompliance(db *sql.DB) ([]MissedSIP, error) {
	// Get all active SIP schedules
	schedules, err := models.GetActiveSIPSchedules(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get SIP schedules: %w", err)
	}

	var missedSIPs []MissedSIP
	today := time.Now()

	for _, schedule := range schedules {
		// Generate expected payment dates
		expectedDates := generateExpectedDates(schedule.StartDate, schedule.EndDate, schedule.Frequency, schedule.DayOfMonth, today)

		// Check each expected date
		for _, expectedDate := range expectedDates {
			// Only check dates that have passed (with 3-day grace period)
			if time.Since(expectedDate) < 3*24*time.Hour {
				continue
			}

			// Check if transaction exists within ±3 days of expected date
			hasTransaction, err := hasTransactionNearDate(db, schedule.ClientID, schedule.FundID, expectedDate, 3)
			if err != nil {
				log.Printf("Error checking transaction for SIP schedule %d: %v", schedule.ID, err)
				continue
			}

			if !hasTransaction {
				// Get client details
				client, err := models.GetClientByID(db, schedule.ClientID)
				if err != nil {
					continue
				}

				// Get fund details
				fund, err := models.GetFundByID(db, schedule.FundID)
				if err != nil {
					continue
				}

				daysMissed := int(time.Since(expectedDate).Hours() / 24)

				missedSIPs = append(missedSIPs, MissedSIP{
					ClientID:     client.ID,
					ClientName:   client.Name,
					ClientEmail:  client.Email,
					FundID:       fund.ID,
					FundName:     fund.SchemeName,
					ExpectedDate: expectedDate,
					Amount:       schedule.Amount,
					DaysMissed:   daysMissed,
				})
			}
		}
	}

	return missedSIPs, nil
}

// UpcomingSIP represents an upcoming SIP installment
type UpcomingSIP struct {
	ClientID     int       `json:"client_id"`
	ClientName   string    `json:"client_name"`
	FundID       int       `json:"fund_id"`
	FundName     string    `json:"fund_name"`
	NextSIPDate  time.Time `json:"next_sip_date"`
	Amount       float64   `json:"amount"`
	DaysUntil    int       `json:"days_until"`
}

// GetUpcomingSIPs returns all upcoming SIP installments (future dates only)
func GetUpcomingSIPs(db *sql.DB) ([]UpcomingSIP, error) {
	// Get all active SIP schedules
	schedules, err := models.GetActiveSIPSchedules(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get SIP schedules: %w", err)
	}

	var upcomingSIPs []UpcomingSIP
	today := time.Now()

	for _, schedule := range schedules {
		// Calculate the next SIP date after today
		nextSIPDate := calculateNextSIPDate(schedule.StartDate, schedule.Frequency, schedule.DayOfMonth, today)

		// Skip if end date is specified and next SIP is after end date
		if schedule.EndDate != nil && nextSIPDate.After(*schedule.EndDate) {
			continue
		}

		// Get client details
		client, err := models.GetClientByID(db, schedule.ClientID)
		if err != nil {
			continue
		}

		// Get fund details
		fund, err := models.GetFundByID(db, schedule.FundID)
		if err != nil {
			continue
		}

		daysUntil := int(time.Until(nextSIPDate).Hours() / 24)

		upcomingSIPs = append(upcomingSIPs, UpcomingSIP{
			ClientID:    client.ID,
			ClientName:  client.Name,
			FundID:      fund.ID,
			FundName:    fund.SchemeName,
			NextSIPDate: nextSIPDate,
			Amount:      schedule.Amount,
			DaysUntil:   daysUntil,
		})
	}

	return upcomingSIPs, nil
}

// calculateNextSIPDate calculates the next SIP installment date after today
func calculateNextSIPDate(startDate time.Time, frequency string, dayOfMonth int, today time.Time) time.Time {
	current := normalizeToDay(startDate, dayOfMonth)

	// Move forward until we find a date after today
	for !current.After(today) {
		if frequency == "QUARTERLY" {
			current = current.AddDate(0, 3, 0)
		} else { // Default to MONTHLY
			current = current.AddDate(0, 1, 0)
		}
		current = normalizeToDay(current, dayOfMonth)
	}

	return current
}

// generateExpectedDates generates all expected SIP payment dates
func generateExpectedDates(startDate time.Time, endDate *time.Time, frequency string, dayOfMonth int, today time.Time) []time.Time {
	var dates []time.Time

	currentDate := normalizeToDay(startDate, dayOfMonth)

	// If end date is not specified, use current date + 1 month as upper limit
	maxDate := today.AddDate(0, 1, 0)
	if endDate != nil && endDate.Before(maxDate) {
		maxDate = *endDate
	}

	for currentDate.Before(maxDate) {
		// Only include dates that are after start date and not in the future
		if currentDate.After(startDate) && currentDate.Before(today) {
			dates = append(dates, currentDate)
		}

		// Move to next period based on frequency
		if frequency == "QUARTERLY" {
			currentDate = currentDate.AddDate(0, 3, 0)
		} else { // Default to MONTHLY
			currentDate = currentDate.AddDate(0, 1, 0)
		}

		// Normalize to the correct day of month
		currentDate = normalizeToDay(currentDate, dayOfMonth)
	}

	return dates
}

// normalizeToDay adjusts a date to a specific day of the month
func normalizeToDay(date time.Time, dayOfMonth int) time.Time {
	year, month, _ := date.Date()

	// Get the last day of the month
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, date.Location())
	lastDay := nextMonth.AddDate(0, 0, -1).Day()

	// If requested day is greater than last day of month, use last day
	if dayOfMonth > lastDay {
		dayOfMonth = lastDay
	}

	return time.Date(year, month, dayOfMonth, 0, 0, 0, 0, date.Location())
}

// hasTransactionNearDate checks if a SIP transaction exists within +/- days of the expected date
func hasTransactionNearDate(db *sql.DB, clientID, fundID int, expectedDate time.Time, toleranceDays int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM transactions
		WHERE client_id = $1
		  AND fund_id = $2
		  AND transaction_type = 'SIP'
		  AND transaction_date >= $3
		  AND transaction_date <= $4
	`

	startDate := expectedDate.AddDate(0, 0, -toleranceDays)
	endDate := expectedDate.AddDate(0, 0, toleranceDays)

	var count int
	err := db.QueryRow(query, clientID, fundID, startDate, endDate).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// RunSIPComplianceCheck runs the compliance check and sends email alerts
func RunSIPComplianceCheck(db *sql.DB) error {
	log.Println("Starting SIP compliance check...")

	missedSIPs, err := CheckSIPCompliance(db)
	if err != nil {
		return fmt.Errorf("SIP compliance check failed: %w", err)
	}

	if len(missedSIPs) == 0 {
		log.Println("No missed SIP installments found")
		return nil
	}

	log.Printf("Found %d missed SIP installments", len(missedSIPs))

	// Send email alert
	if err := SendMissedSIPAlert(missedSIPs); err != nil {
		log.Printf("Failed to send email alert: %v", err)
		// Don't return error, compliance check was successful
	}

	return nil
}
