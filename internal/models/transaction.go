package models

import (
	"database/sql"
	"time"
)

type Transaction struct {
	ID              int       `json:"id"`
	ClientID        int       `json:"client_id"`
	FundID          int       `json:"fund_id"`
	TransactionType string    `json:"transaction_type"` // LUMPSUM, SIP, REDEMPTION, SWITCH_IN, SWITCH_OUT
	TransactionDate time.Time `json:"transaction_date"`
	Amount          float64   `json:"amount"`
	NAV             float64   `json:"nav"`
	Units           float64   `json:"units"`
	FolioNumber     string    `json:"folio_number"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateTransaction inserts a new transaction
func CreateTransaction(db *sql.DB, txn *Transaction) error {
	query := `
		INSERT INTO transactions (client_id, fund_id, transaction_type, transaction_date, amount, nav, units, folio_number, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	return db.QueryRow(query,
		txn.ClientID, txn.FundID, txn.TransactionType, txn.TransactionDate,
		txn.Amount, txn.NAV, txn.Units, txn.FolioNumber, txn.Notes,
	).Scan(&txn.ID, &txn.CreatedAt)
}

// GetTransactionByID retrieves a transaction by ID
func GetTransactionByID(db *sql.DB, id int) (*Transaction, error) {
	txn := &Transaction{}
	query := `
		SELECT id, client_id, fund_id, transaction_type, transaction_date, amount, nav, units, folio_number, notes, created_at
		FROM transactions
		WHERE id = $1
	`
	err := db.QueryRow(query, id).Scan(
		&txn.ID, &txn.ClientID, &txn.FundID, &txn.TransactionType,
		&txn.TransactionDate, &txn.Amount, &txn.NAV, &txn.Units,
		&txn.FolioNumber, &txn.Notes, &txn.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return txn, nil
}

// GetTransactionsByClient retrieves all transactions for a client
func GetTransactionsByClient(db *sql.DB, clientID int) ([]Transaction, error) {
	query := `
		SELECT id, client_id, fund_id, transaction_type, transaction_date, amount, nav, units, folio_number, notes, created_at
		FROM transactions
		WHERE client_id = $1
		ORDER BY transaction_date DESC
	`
	rows, err := db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var txn Transaction
		err := rows.Scan(
			&txn.ID, &txn.ClientID, &txn.FundID, &txn.TransactionType,
			&txn.TransactionDate, &txn.Amount, &txn.NAV, &txn.Units,
			&txn.FolioNumber, &txn.Notes, &txn.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}
	return transactions, nil
}

// GetTransactionsByClientAndFund retrieves transactions for a specific client and fund
func GetTransactionsByClientAndFund(db *sql.DB, clientID, fundID int) ([]Transaction, error) {
	query := `
		SELECT id, client_id, fund_id, transaction_type, transaction_date, amount, nav, units, folio_number, notes, created_at
		FROM transactions
		WHERE client_id = $1 AND fund_id = $2
		ORDER BY transaction_date ASC
	`
	rows, err := db.Query(query, clientID, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var txn Transaction
		err := rows.Scan(
			&txn.ID, &txn.ClientID, &txn.FundID, &txn.TransactionType,
			&txn.TransactionDate, &txn.Amount, &txn.NAV, &txn.Units,
			&txn.FolioNumber, &txn.Notes, &txn.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}
	return transactions, nil
}

// UpdateTransaction updates an existing transaction
func UpdateTransaction(db *sql.DB, txn *Transaction) error {
	query := `
		UPDATE transactions
		SET client_id = $1, fund_id = $2, transaction_type = $3, transaction_date = $4,
		    amount = $5, nav = $6, units = $7, folio_number = $8, notes = $9
		WHERE id = $10
	`
	_, err := db.Exec(query,
		txn.ClientID, txn.FundID, txn.TransactionType, txn.TransactionDate,
		txn.Amount, txn.NAV, txn.Units, txn.FolioNumber, txn.Notes, txn.ID,
	)
	return err
}

// DeleteTransaction deletes a transaction by ID
func DeleteTransaction(db *sql.DB, id int) error {
	query := `DELETE FROM transactions WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}
