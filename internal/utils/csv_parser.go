package utils

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// TransactionCSVRow represents a single row from the CSV file
type TransactionCSVRow struct {
	ClientName      string
	ClientPAN       string
	ClientEmail     string
	ClientPhone     string
	FundName        string
	SchemeCode      string
	FundHouse       string
	Category        string
	TransactionType string
	TransactionDate time.Time
	Amount          float64
	FolioNumber     string
	Notes           string
}

// ParseTransactionCSV parses a CSV file containing transaction data
// Expected columns: Client Name, PAN, Email, Phone, Fund Name, Scheme Code, Fund House, Category, Transaction Type, Date, Amount, Folio, Notes
func ParseTransactionCSV(reader io.Reader) ([]TransactionCSVRow, []string, error) {
	csvReader := csv.NewReader(reader)

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header
	expectedCols := []string{"Client Name", "PAN", "Email", "Phone", "Fund Name", "Scheme Code", "Fund House", "Category", "Transaction Type", "Date", "Amount", "Folio", "Notes"}
	if len(header) < len(expectedCols) {
		return nil, nil, fmt.Errorf("invalid CSV format: expected %d columns, got %d", len(expectedCols), len(header))
	}

	var rows []TransactionCSVRow
	var errors []string
	lineNum := 1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("Line %d: %s", lineNum+1, err.Error()))
			lineNum++
			continue
		}
		lineNum++

		if len(record) < 13 {
			errors = append(errors, fmt.Sprintf("Line %d: insufficient columns (expected 13)", lineNum))
			continue
		}

		// Parse transaction date
		date, err := parseDate(strings.TrimSpace(record[9]))
		if err != nil {
			errors = append(errors, fmt.Sprintf("Line %d: invalid date format '%s' (expected DD-MM-YYYY or DD/MM/YYYY)", lineNum, record[9]))
			continue
		}

		// Parse amount
		amountStr := strings.TrimSpace(record[10])
		amountStr = strings.ReplaceAll(amountStr, ",", "") // Remove commas
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Line %d: invalid amount '%s'", lineNum, record[10]))
			continue
		}

		// Validate transaction type
		txnType := strings.ToUpper(strings.TrimSpace(record[8]))
		validTypes := map[string]bool{
			"LUMPSUM":    true,
			"SIP":        true,
			"REDEMPTION": true,
			"SWITCH_IN":  true,
			"SWITCH_OUT": true,
		}
		if !validTypes[txnType] {
			errors = append(errors, fmt.Sprintf("Line %d: invalid transaction type '%s' (must be LUMPSUM, SIP, REDEMPTION, SWITCH_IN, or SWITCH_OUT)", lineNum, record[8]))
			continue
		}

		// Validate category
		category := strings.TrimSpace(record[7])
		validCategories := map[string]bool{
			"Equity":  true,
			"Debt":    true,
			"Hybrid":  true,
		}
		if category != "" && !validCategories[category] {
			errors = append(errors, fmt.Sprintf("Line %d: invalid category '%s' (must be Equity, Debt, or Hybrid)", lineNum, category))
			continue
		}

		row := TransactionCSVRow{
			ClientName:      strings.TrimSpace(record[0]),
			ClientPAN:       strings.ToUpper(strings.TrimSpace(record[1])),
			ClientEmail:     strings.TrimSpace(record[2]),
			ClientPhone:     strings.TrimSpace(record[3]),
			FundName:        strings.TrimSpace(record[4]),
			SchemeCode:      strings.TrimSpace(record[5]),
			FundHouse:       strings.TrimSpace(record[6]),
			Category:        category,
			TransactionType: txnType,
			TransactionDate: date,
			Amount:          amount,
			FolioNumber:     strings.TrimSpace(record[11]),
			Notes:           strings.TrimSpace(record[12]),
		}

		// Validate required fields
		if row.ClientName == "" {
			errors = append(errors, fmt.Sprintf("Line %d: client name is required", lineNum))
			continue
		}
		if row.FundName == "" && row.SchemeCode == "" {
			errors = append(errors, fmt.Sprintf("Line %d: either fund name or scheme code is required", lineNum))
			continue
		}

		rows = append(rows, row)
	}

	return rows, errors, nil
}

// parseDate attempts to parse date in multiple formats
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"02-01-2006",
		"02/01/2006",
		"2006-01-02",
		"02-Jan-2006",
		"02 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("unsupported date format")
}

// GenerateSampleCSV returns a sample CSV content for users to download
func GenerateSampleCSV() string {
	return `Client Name,PAN,Email,Phone,Fund Name,Scheme Code,Fund House,Category,Transaction Type,Date,Amount,Folio,Notes
Rajesh Kumar,ABCDE1234F,rajesh@example.com,9876543210,HDFC Equity Fund,120503,HDFC,Equity,LUMPSUM,01-01-2024,50000,ABC123,Initial investment
Priya Sharma,FGHIJ5678K,priya@example.com,9876543211,ICICI Prudential Debt Fund,120259,ICICI Prudential,Debt,SIP,05-01-2024,10000,DEF456,Monthly SIP
Amit Patel,LMNOP9012Q,amit@example.com,9876543212,SBI Hybrid Fund,119598,SBI,Hybrid,LUMPSUM,10-01-2024,100000,GHI789,Lumpsum purchase`
}
