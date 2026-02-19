package models

import (
	"database/sql"
	"time"
)

type Client struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Email     string       `json:"email"`
	Phone     string       `json:"phone"`
	PAN       string       `json:"pan"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// CreateClient inserts a new client into the database
func CreateClient(db *sql.DB, client *Client) error {
	query := `
		INSERT INTO clients (name, email, phone, pan)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(query, client.Name, client.Email, client.Phone, client.PAN).
		Scan(&client.ID, &client.CreatedAt, &client.UpdatedAt)
}

// GetClientByID retrieves a client by ID
func GetClientByID(db *sql.DB, id int) (*Client, error) {
	client := &Client{}
	query := `
		SELECT id, name, email, phone, pan, created_at, updated_at
		FROM clients
		WHERE id = $1
	`
	err := db.QueryRow(query, id).Scan(
		&client.ID, &client.Name, &client.Email, &client.Phone,
		&client.PAN, &client.CreatedAt, &client.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetAllClients retrieves all clients
func GetAllClients(db *sql.DB) ([]Client, error) {
	query := `
		SELECT id, name, email, phone, pan, created_at, updated_at
		FROM clients
		ORDER BY name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var client Client
		err := rows.Scan(
			&client.ID, &client.Name, &client.Email, &client.Phone,
			&client.PAN, &client.CreatedAt, &client.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, nil
}

// UpdateClient updates an existing client
func UpdateClient(db *sql.DB, client *Client) error {
	query := `
		UPDATE clients
		SET name = $1, email = $2, phone = $3, pan = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING updated_at
	`
	return db.QueryRow(query, client.Name, client.Email, client.Phone, client.PAN, client.ID).
		Scan(&client.UpdatedAt)
}

// DeleteClient deletes a client by ID
func DeleteClient(db *sql.DB, id int) error {
	query := `DELETE FROM clients WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}
