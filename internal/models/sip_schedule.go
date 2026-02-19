package models

import (
	"database/sql"
	"time"
)

type SIPSchedule struct {
	ID         int       `json:"id"`
	ClientID   int       `json:"client_id"`
	FundID     int       `json:"fund_id"`
	Amount     float64   `json:"amount"`
	StartDate  time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Frequency  string    `json:"frequency"`   // MONTHLY, QUARTERLY
	DayOfMonth int       `json:"day_of_month"` // 1-31
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateSIPSchedule inserts a new SIP schedule
func CreateSIPSchedule(db *sql.DB, sip *SIPSchedule) error {
	query := `
		INSERT INTO sip_schedules (client_id, fund_id, amount, start_date, end_date, frequency, day_of_month, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return db.QueryRow(query,
		sip.ClientID, sip.FundID, sip.Amount, sip.StartDate, sip.EndDate,
		sip.Frequency, sip.DayOfMonth, sip.IsActive,
	).Scan(&sip.ID, &sip.CreatedAt)
}

// GetSIPScheduleByID retrieves a SIP schedule by ID
func GetSIPScheduleByID(db *sql.DB, id int) (*SIPSchedule, error) {
	sip := &SIPSchedule{}
	query := `
		SELECT id, client_id, fund_id, amount, start_date, end_date, frequency, day_of_month, is_active, created_at
		FROM sip_schedules
		WHERE id = $1
	`
	err := db.QueryRow(query, id).Scan(
		&sip.ID, &sip.ClientID, &sip.FundID, &sip.Amount, &sip.StartDate,
		&sip.EndDate, &sip.Frequency, &sip.DayOfMonth, &sip.IsActive, &sip.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sip, nil
}

// GetActiveSIPSchedules retrieves all active SIP schedules
func GetActiveSIPSchedules(db *sql.DB) ([]SIPSchedule, error) {
	query := `
		SELECT id, client_id, fund_id, amount, start_date, end_date, frequency, day_of_month, is_active, created_at
		FROM sip_schedules
		WHERE is_active = true
		ORDER BY client_id, fund_id
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sips []SIPSchedule
	for rows.Next() {
		var sip SIPSchedule
		err := rows.Scan(
			&sip.ID, &sip.ClientID, &sip.FundID, &sip.Amount, &sip.StartDate,
			&sip.EndDate, &sip.Frequency, &sip.DayOfMonth, &sip.IsActive, &sip.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		sips = append(sips, sip)
	}
	return sips, nil
}

// GetSIPSchedulesByClient retrieves all SIP schedules for a client
func GetSIPSchedulesByClient(db *sql.DB, clientID int) ([]SIPSchedule, error) {
	query := `
		SELECT id, client_id, fund_id, amount, start_date, end_date, frequency, day_of_month, is_active, created_at
		FROM sip_schedules
		WHERE client_id = $1
		ORDER BY is_active DESC, start_date DESC
	`
	rows, err := db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sips []SIPSchedule
	for rows.Next() {
		var sip SIPSchedule
		err := rows.Scan(
			&sip.ID, &sip.ClientID, &sip.FundID, &sip.Amount, &sip.StartDate,
			&sip.EndDate, &sip.Frequency, &sip.DayOfMonth, &sip.IsActive, &sip.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		sips = append(sips, sip)
	}
	return sips, nil
}

// UpdateSIPSchedule updates an existing SIP schedule
func UpdateSIPSchedule(db *sql.DB, sip *SIPSchedule) error {
	query := `
		UPDATE sip_schedules
		SET amount = $1, end_date = $2, is_active = $3
		WHERE id = $4
	`
	_, err := db.Exec(query, sip.Amount, sip.EndDate, sip.IsActive, sip.ID)
	return err
}

// DeactivateSIPSchedule sets is_active to false
func DeactivateSIPSchedule(db *sql.DB, id int) error {
	query := `UPDATE sip_schedules SET is_active = false WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}
