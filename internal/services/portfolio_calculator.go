package services

import (
	"client-dashboard/internal/models"
	"client-dashboard/internal/utils"
	"database/sql"
	"fmt"
	"time"
)

// FundHolding represents a client's holding in a specific fund
type FundHolding struct {
	FundID         int     `json:"fund_id"`
	FundName       string  `json:"fund_name"`
	SchemeCode     string  `json:"scheme_code"`
	Category       string  `json:"category"`
	TotalUnits     float64 `json:"total_units"`
	TotalInvested  float64 `json:"total_invested"`
	CurrentNAV     float64 `json:"current_nav"`
	CurrentValue   float64 `json:"current_value"`
	AbsoluteReturn float64 `json:"absolute_return"`
	XIRR           float64 `json:"xirr"`
}

// ClientPortfolio represents a client's complete portfolio
type ClientPortfolio struct {
	ClientID       int           `json:"client_id"`
	ClientName     string        `json:"client_name"`
	Holdings       []FundHolding `json:"holdings"`
	TotalInvested  float64       `json:"total_invested"`
	CurrentValue   float64       `json:"current_value"`
	AbsoluteReturn float64       `json:"absolute_return"`
	OverallXIRR    float64       `json:"overall_xirr"`
}

// CalculateFundHolding calculates holdings for a client in a specific fund
func CalculateFundHolding(db *sql.DB, clientID, fundID int) (*FundHolding, error) {
	// Get fund details
	fund, err := models.GetFundByID(db, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fund: %w", err)
	}

	// Get all actual transactions for this client and fund
	actualTransactions, err := models.GetTransactionsByClientAndFund(db, clientID, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Get SIP-generated transactions for this client
	allSIPTxns, _ := GetSIPGeneratedTransactions(db, clientID)
	var sipTransactions []models.Transaction
	for _, txn := range allSIPTxns {
		if txn.FundID == fundID {
			sipTransactions = append(sipTransactions, txn)
		}
	}

	// Merge actual and SIP transactions
	transactions := append(actualTransactions, sipTransactions...)

	if len(transactions) == 0 {
		return nil, fmt.Errorf("no transactions found")
	}

	// Calculate total units and invested amount
	var totalUnits float64
	var totalInvested float64

	for _, txn := range transactions {
		units := txn.Units

		// If units are 0 (SIP-generated transaction), calculate them now
		if units == 0 && (txn.TransactionType == "SIP" || txn.TransactionType == "LUMPSUM") {
			calculatedUnits, _, err := CalculateUnitsForTransaction(db, txn.FundID, fund.SchemeCode, txn.Amount, txn.TransactionDate)
			if err == nil {
				units = calculatedUnits
			}
		}

		switch txn.TransactionType {
		case "LUMPSUM", "SIP", "SWITCH_IN":
			totalUnits += units
			totalInvested += txn.Amount
		case "REDEMPTION", "SWITCH_OUT":
			totalUnits -= units
			totalInvested -= txn.Amount
		}
	}

	// If no units remaining, return empty holding
	if totalUnits <= 0.001 {
		return &FundHolding{
			FundID:         fund.ID,
			FundName:       fund.SchemeName,
			SchemeCode:     fund.SchemeCode,
			Category:       fund.Category,
			TotalUnits:     0,
			TotalInvested:  totalInvested,
			CurrentValue:   0,
			AbsoluteReturn: -totalInvested,
			XIRR:           0,
		}, nil
	}

	// Get latest NAV
	latestNAV, err := models.GetLatestNAV(db, fundID)
	if err != nil {
		return nil, fmt.Errorf("no NAV data available: %w", err)
	}

	utils.LogInfo("Portfolio", fmt.Sprintf("Fund %d: using NAV %.4f from %s for current valuation (%.3f units)",
		fundID, latestNAV.NAVValue, latestNAV.NAVDate.Format("02-Jan-2006"), totalUnits))

	currentValue := totalUnits * latestNAV.NAVValue
	absoluteReturn := currentValue - totalInvested

	// Calculate XIRR
	var cashflows []utils.Cashflow
	for _, txn := range transactions {
		amount := txn.Amount
		if txn.TransactionType == "LUMPSUM" || txn.TransactionType == "SIP" || txn.TransactionType == "SWITCH_IN" {
			amount = -amount // Investments are negative cashflows
		}
		cashflows = append(cashflows, utils.Cashflow{
			Date:   txn.TransactionDate,
			Amount: amount,
		})
	}

	// Add current value as final positive cashflow
	cashflows = append(cashflows, utils.Cashflow{
		Date:   time.Now(),
		Amount: currentValue,
	})

	xirr, err := utils.CalculateXIRR(cashflows)
	if err != nil {
		// If XIRR calculation fails, use simplified method
		firstTxn := transactions[0]
		xirr = utils.SimplifiedXIRR(totalInvested, currentValue, firstTxn.TransactionDate, time.Now())
	}

	return &FundHolding{
		FundID:         fund.ID,
		FundName:       fund.SchemeName,
		SchemeCode:     fund.SchemeCode,
		Category:       fund.Category,
		TotalUnits:     totalUnits,
		TotalInvested:  totalInvested,
		CurrentNAV:     latestNAV.NAVValue,
		CurrentValue:   currentValue,
		AbsoluteReturn: absoluteReturn,
		XIRR:           xirr,
	}, nil
}

// CalculateClientPortfolio calculates complete portfolio for a client
func CalculateClientPortfolio(db *sql.DB, clientID int) (*ClientPortfolio, error) {
	// Get client details
	client, err := models.GetClientByID(db, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	// Get all actual transactions for client
	actualTransactions, err := models.GetTransactionsByClient(db, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Get SIP-generated transactions (virtual transactions from SIP schedules)
	sipTransactions, err := GetSIPGeneratedTransactions(db, clientID)
	if err != nil {
		// Log error but continue
		sipTransactions = []models.Transaction{}
	}

	// Merge actual and SIP transactions
	transactions := append(actualTransactions, sipTransactions...)

	if len(transactions) == 0 {
		return &ClientPortfolio{
			ClientID:   client.ID,
			ClientName: client.Name,
			Holdings:   []FundHolding{},
		}, nil
	}

	// Get unique fund IDs
	fundIDsMap := make(map[int]bool)
	for _, txn := range transactions {
		fundIDsMap[txn.FundID] = true
	}

	// Calculate holdings for each fund
	var holdings []FundHolding
	var totalInvested, currentValue float64
	var allCashflows []utils.Cashflow

	for fundID := range fundIDsMap {
		holding, err := CalculateFundHolding(db, clientID, fundID)
		if err != nil {
			continue // Skip funds with errors
		}

		if holding.TotalUnits > 0.001 { // Only include active holdings
			holdings = append(holdings, *holding)
			totalInvested += holding.TotalInvested
			currentValue += holding.CurrentValue
		}

		// Collect cashflows for overall XIRR
		fundTxns, _ := models.GetTransactionsByClientAndFund(db, clientID, fundID)
		for _, txn := range fundTxns {
			amount := txn.Amount
			if txn.TransactionType == "LUMPSUM" || txn.TransactionType == "SIP" || txn.TransactionType == "SWITCH_IN" {
				amount = -amount
			}
			allCashflows = append(allCashflows, utils.Cashflow{
				Date:   txn.TransactionDate,
				Amount: amount,
			})
		}
	}

	// Calculate overall XIRR
	allCashflows = append(allCashflows, utils.Cashflow{
		Date:   time.Now(),
		Amount: currentValue,
	})

	overallXIRR, err := utils.CalculateXIRR(allCashflows)
	if err != nil && len(transactions) > 0 {
		// Use simplified method if XIRR fails
		firstTxn := transactions[len(transactions)-1] // Oldest transaction (ordered DESC)
		overallXIRR = utils.SimplifiedXIRR(totalInvested, currentValue, firstTxn.TransactionDate, time.Now())
	}

	return &ClientPortfolio{
		ClientID:       client.ID,
		ClientName:     client.Name,
		Holdings:       holdings,
		TotalInvested:  totalInvested,
		CurrentValue:   currentValue,
		AbsoluteReturn: currentValue - totalInvested,
		OverallXIRR:    overallXIRR,
	}, nil
}

// CalculateUnitsForTransaction calculates units for a transaction based on NAV
func CalculateUnitsForTransaction(db *sql.DB, fundID int, schemeCode string, amount float64, date time.Time) (float64, float64, error) {
	// Get NAV for the transaction date
	nav, err := GetOrFetchNAV(db, fundID, schemeCode, date)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get NAV: %w", err)
	}

	units := amount / nav
	return units, nav, nil
}
