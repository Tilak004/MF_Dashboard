package models

import (
	"database/sql"
	"time"
)

type NAVHistory struct {
	ID        int       `json:"id"`
	FundID    int       `json:"fund_id"`
	NAVDate   time.Time `json:"nav_date"`
	NAVValue  float64   `json:"nav_value"`
	FetchedAt time.Time `json:"fetched_at"`
}

// CreateNAVHistory inserts a new NAV record
func CreateNAVHistory(db *sql.DB, nav *NAVHistory) error {
	query := `
		INSERT INTO nav_history (fund_id, nav_date, nav_value)
		VALUES ($1, $2, $3)
		ON CONFLICT (fund_id, nav_date) DO UPDATE
		SET nav_value = EXCLUDED.nav_value, fetched_at = CURRENT_TIMESTAMP
		RETURNING id, fetched_at
	`
	return db.QueryRow(query, nav.FundID, nav.NAVDate, nav.NAVValue).
		Scan(&nav.ID, &nav.FetchedAt)
}

// GetLatestNAV retrieves the latest NAV for a fund
func GetLatestNAV(db *sql.DB, fundID int) (*NAVHistory, error) {
	nav := &NAVHistory{}
	query := `
		SELECT id, fund_id, nav_date, nav_value, fetched_at
		FROM nav_history
		WHERE fund_id = $1
		ORDER BY nav_date DESC
		LIMIT 1
	`
	err := db.QueryRow(query, fundID).Scan(
		&nav.ID, &nav.FundID, &nav.NAVDate, &nav.NAVValue, &nav.FetchedAt,
	)
	if err != nil {
		return nil, err
	}
	return nav, nil
}

// GetNAVByDate retrieves NAV for a fund on a specific date
func GetNAVByDate(db *sql.DB, fundID int, date time.Time) (*NAVHistory, error) {
	nav := &NAVHistory{}
	query := `
		SELECT id, fund_id, nav_date, nav_value, fetched_at
		FROM nav_history
		WHERE fund_id = $1 AND nav_date = $2
	`
	err := db.QueryRow(query, fundID, date).Scan(
		&nav.ID, &nav.FundID, &nav.NAVDate, &nav.NAVValue, &nav.FetchedAt,
	)
	if err != nil {
		return nil, err
	}
	return nav, nil
}

// GetNAVHistory retrieves NAV history for a fund
func GetNAVHistory(db *sql.DB, fundID int, limit int) ([]NAVHistory, error) {
	query := `
		SELECT id, fund_id, nav_date, nav_value, fetched_at
		FROM nav_history
		WHERE fund_id = $1
		ORDER BY nav_date DESC
		LIMIT $2
	`
	rows, err := db.Query(query, fundID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var navs []NAVHistory
	for rows.Next() {
		var nav NAVHistory
		err := rows.Scan(
			&nav.ID, &nav.FundID, &nav.NAVDate, &nav.NAVValue, &nav.FetchedAt,
		)
		if err != nil {
			return nil, err
		}
		navs = append(navs, nav)
	}
	return navs, nil
}
