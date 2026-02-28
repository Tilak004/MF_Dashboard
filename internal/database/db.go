package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// Initialize establishes database connection and runs migrations
func Initialize() error {
	// Prefer DATABASE_URL (set automatically by Render when a Postgres service is linked)
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Fallback to individual vars (local development)
		sslMode := os.Getenv("DB_SSLMODE")
		if sslMode == "" {
			sslMode = "disable"
		}
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbUser := os.Getenv("DB_USER")
		dbName := os.Getenv("DB_NAME")
		log.Printf("Connecting to database: host=%s port=%s user=%s dbname=%s sslmode=%s", dbHost, dbPort, dbUser, dbName, sslMode)
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, os.Getenv("DB_PASSWORD"), dbName, sslMode,
		)
	} else {
		log.Println("Connecting to database via DATABASE_URL")
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	log.Println("Database connection established successfully")

	// Verify database
	var currentDB string
	DB.QueryRow("SELECT current_database()").Scan(&currentDB)
	log.Printf("Connected to database: %s", currentDB)

	// Check if users table has any records
	var userCount int
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	log.Printf("Users table has %d records", userCount)

	// Run migrations
	if err = runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations executes the SQL migration file
func runMigrations() error {
	migrationFile := "migrations/001_initial_schema.sql"

	content, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = DB.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
