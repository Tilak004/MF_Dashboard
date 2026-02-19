package models

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password in JSON
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateUser inserts a new user with hashed password
func CreateUser(db *sql.DB, username, password, email string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Email:        email,
	}

	query := `
		INSERT INTO users (username, password_hash, email)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err = db.QueryRow(query, user.Username, user.PasswordHash, user.Email).
		Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, username, password_hash, email, created_at
		FROM users
		WHERE username = $1
	`
	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ValidatePassword checks if the provided password matches the hash
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
