package utils

import (
	"errors"
	"math"
	"time"
)

// Cashflow represents a single cash flow with date and amount
type Cashflow struct {
	Date   time.Time
	Amount float64
}

// CalculateXIRR calculates the Extended Internal Rate of Return using Newton-Raphson method
// Returns annualized return as a percentage (e.g., 12.5 for 12.5%)
func CalculateXIRR(cashflows []Cashflow) (float64, error) {
	if len(cashflows) < 2 {
		return 0, errors.New("at least 2 cashflows required for XIRR calculation")
	}

	// Check if there's at least one positive and one negative cashflow
	hasPositive, hasNegative := false, false
	for _, cf := range cashflows {
		if cf.Amount > 0 {
			hasPositive = true
		}
		if cf.Amount < 0 {
			hasNegative = true
		}
	}
	if !hasPositive || !hasNegative {
		return 0, errors.New("cashflows must contain both positive and negative values")
	}

	// Use first date as the base date
	baseDate := cashflows[0].Date

	// Initial guess: 10% annual return
	guess := 0.1
	maxIterations := 100
	tolerance := 0.0001

	for i := 0; i < maxIterations; i++ {
		npv := calculateNPV(cashflows, baseDate, guess)
		dnpv := calculateDerivativeNPV(cashflows, baseDate, guess)

		if math.Abs(dnpv) < 1e-10 {
			return 0, errors.New("derivative too small, cannot converge")
		}

		newGuess := guess - (npv / dnpv)

		// Check for convergence
		if math.Abs(newGuess-guess) < tolerance {
			return newGuess * 100, nil // Convert to percentage
		}

		guess = newGuess

		// Prevent guess from going too extreme
		if guess < -0.99 {
			guess = -0.99
		} else if guess > 10 {
			guess = 10
		}
	}

	return 0, errors.New("XIRR calculation did not converge")
}

// calculateNPV calculates Net Present Value given cashflows and rate
func calculateNPV(cashflows []Cashflow, baseDate time.Time, rate float64) float64 {
	npv := 0.0
	for _, cf := range cashflows {
		days := cf.Date.Sub(baseDate).Hours() / 24
		years := days / 365.25
		npv += cf.Amount / math.Pow(1+rate, years)
	}
	return npv
}

// calculateDerivativeNPV calculates the derivative of NPV with respect to rate
func calculateDerivativeNPV(cashflows []Cashflow, baseDate time.Time, rate float64) float64 {
	dnpv := 0.0
	for _, cf := range cashflows {
		days := cf.Date.Sub(baseDate).Hours() / 24
		years := days / 365.25
		dnpv -= (cf.Amount * years) / math.Pow(1+rate, years+1)
	}
	return dnpv
}

// SimplifiedXIRR calculates approximate XIRR for quick estimates
// This is useful when you need a rough calculation without full accuracy
func SimplifiedXIRR(totalInvested, currentValue float64, startDate, endDate time.Time) float64 {
	if totalInvested <= 0 {
		return 0
	}

	days := endDate.Sub(startDate).Hours() / 24
	if days <= 0 {
		return 0
	}

	years := days / 365.25
	absoluteReturn := ((currentValue - totalInvested) / totalInvested)
	annualizedReturn := (math.Pow(1+absoluteReturn, 1/years) - 1) * 100

	return annualizedReturn
}
