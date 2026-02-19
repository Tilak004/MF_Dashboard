package models

import (
	"database/sql"
	"time"
)

type Fund struct {
	ID         int       `json:"id"`
	SchemeCode string    `json:"scheme_code"`
	SchemeName string    `json:"scheme_name"`
	FundHouse  string    `json:"fund_house"`
	Category   string    `json:"category"`   // Equity, Debt, Hybrid
	RiskLevel  string    `json:"risk_level"` // High, Medium, Low
	CreatedAt  time.Time `json:"created_at"`
}

// CreateFund inserts a new fund into the database
func CreateFund(db *sql.DB, fund *Fund) error {
	query := `
		INSERT INTO funds (scheme_code, scheme_name, fund_house, category, risk_level)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return db.QueryRow(query, fund.SchemeCode, fund.SchemeName, fund.FundHouse, fund.Category, fund.RiskLevel).
		Scan(&fund.ID, &fund.CreatedAt)
}

// GetFundByID retrieves a fund by ID
func GetFundByID(db *sql.DB, id int) (*Fund, error) {
	fund := &Fund{}
	query := `
		SELECT id, scheme_code, scheme_name, fund_house, category, risk_level, created_at
		FROM funds
		WHERE id = $1
	`
	err := db.QueryRow(query, id).Scan(
		&fund.ID, &fund.SchemeCode, &fund.SchemeName, &fund.FundHouse,
		&fund.Category, &fund.RiskLevel, &fund.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return fund, nil
}

// GetFundBySchemeCode retrieves a fund by scheme code
func GetFundBySchemeCode(db *sql.DB, schemeCode string) (*Fund, error) {
	fund := &Fund{}
	query := `
		SELECT id, scheme_code, scheme_name, fund_house, category, risk_level, created_at
		FROM funds
		WHERE scheme_code = $1
	`
	err := db.QueryRow(query, schemeCode).Scan(
		&fund.ID, &fund.SchemeCode, &fund.SchemeName, &fund.FundHouse,
		&fund.Category, &fund.RiskLevel, &fund.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return fund, nil
}

// GetAllFunds retrieves all funds
func GetAllFunds(db *sql.DB) ([]Fund, error) {
	query := `
		SELECT id, scheme_code, scheme_name, fund_house, category, risk_level, created_at
		FROM funds
		ORDER BY scheme_name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funds []Fund
	for rows.Next() {
		var fund Fund
		err := rows.Scan(
			&fund.ID, &fund.SchemeCode, &fund.SchemeName, &fund.FundHouse,
			&fund.Category, &fund.RiskLevel, &fund.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		funds = append(funds, fund)
	}
	return funds, nil
}

// GetActiveFunds retrieves all funds that have at least one transaction
func GetActiveFunds(db *sql.DB) ([]Fund, error) {
	query := `
		SELECT DISTINCT f.id, f.scheme_code, f.scheme_name, f.fund_house, f.category, f.risk_level, f.created_at
		FROM funds f
		INNER JOIN transactions t ON f.id = t.fund_id
		ORDER BY f.scheme_name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funds []Fund
	for rows.Next() {
		var fund Fund
		err := rows.Scan(
			&fund.ID, &fund.SchemeCode, &fund.SchemeName, &fund.FundHouse,
			&fund.Category, &fund.RiskLevel, &fund.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		funds = append(funds, fund)
	}
	return funds, nil
}
