package handlers

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/models"
	"client-dashboard/internal/services"
	"client-dashboard/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

// DashboardSummary represents the main dashboard summary
type DashboardSummary struct {
	TotalAUM         float64            `json:"total_aum"`
	TotalClients     int                `json:"total_clients"`
	TotalInvested    float64            `json:"total_invested"`
	OverallReturns   float64            `json:"overall_returns"`
	AssetAllocation  map[string]float64 `json:"asset_allocation"`
	ClientsWithAUM   []ClientAUM        `json:"clients_with_aum"`
	TopPerformers    TopPerformers      `json:"top_performers"`
	MissedSIPs       int                `json:"missed_sips"`
}

type ClientAUM struct {
	ClientID       int     `json:"client_id"`
	ClientName     string  `json:"client_name"`
	CurrentValue   float64 `json:"current_value"`
	TotalInvested  float64 `json:"total_invested"`
	AbsoluteReturn float64 `json:"absolute_return"`
	XIRR           float64 `json:"xirr"`
}

type TopPerformers struct {
	TopClients []ClientAUM         `json:"top_clients"`
	TopFunds   []services.FundHolding `json:"top_funds"`
}

// GetDashboardSummaryHandler returns the main dashboard summary
func GetDashboardSummaryHandler(w http.ResponseWriter, r *http.Request) {
	utils.LogInfo("Dashboard", "Loading dashboard summary")

	clients, err := models.GetAllClients(database.DB)
	if err != nil {
		utils.LogError("Dashboard", fmt.Errorf("failed to get clients: %w", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.LogInfo("Dashboard", fmt.Sprintf("Calculating portfolio for %d clients", len(clients)))

	var totalAUM, totalInvested float64
	var clientsWithAUM []ClientAUM
	var allHoldings []services.FundHolding
	categoryValues := make(map[string]float64)

	// Calculate portfolio for each client
	for _, client := range clients {
		portfolio, err := services.CalculateClientPortfolio(database.DB, client.ID)
		if err != nil {
			utils.LogError("Dashboard", fmt.Errorf("portfolio calc failed for client %d (%s): %w", client.ID, client.Name, err))
			continue
		}
		if portfolio.CurrentValue == 0 {
			utils.LogInfo("Dashboard", fmt.Sprintf("Skipping client %s (ID: %d) - no current value", client.Name, client.ID))
			continue
		}

		totalAUM += portfolio.CurrentValue
		totalInvested += portfolio.TotalInvested

		clientsWithAUM = append(clientsWithAUM, ClientAUM{
			ClientID:       client.ID,
			ClientName:     client.Name,
			CurrentValue:   portfolio.CurrentValue,
			TotalInvested:  portfolio.TotalInvested,
			AbsoluteReturn: portfolio.AbsoluteReturn,
			XIRR:           portfolio.OverallXIRR,
		})

		// Collect holdings for category allocation
		for _, holding := range portfolio.Holdings {
			categoryValues[holding.Category] += holding.CurrentValue
			allHoldings = append(allHoldings, holding)
		}
	}

	// Calculate asset allocation percentages
	assetAllocation := make(map[string]float64)
	if totalAUM > 0 {
		for category, value := range categoryValues {
			assetAllocation[category] = (value / totalAUM) * 100
		}
	}

	// Overall returns percentage
	overallReturns := 0.0
	if totalInvested > 0 {
		overallReturns = ((totalAUM - totalInvested) / totalInvested) * 100
	}

	// Sort clients by XIRR for top performers
	sort.Slice(clientsWithAUM, func(i, j int) bool {
		return clientsWithAUM[i].XIRR > clientsWithAUM[j].XIRR
	})

	topClients := clientsWithAUM
	if len(topClients) > 10 {
		topClients = topClients[:10]
	}

	// Sort funds by XIRR for top performers
	sort.Slice(allHoldings, func(i, j int) bool {
		return allHoldings[i].XIRR > allHoldings[j].XIRR
	})

	topFunds := allHoldings
	if len(topFunds) > 10 {
		topFunds = topFunds[:10]
	}

	// Check for missed SIPs
	missedSIPs, err := services.CheckSIPCompliance(database.DB)
	if err != nil {
		utils.LogError("Dashboard", fmt.Errorf("SIP compliance check failed: %w", err))
	}

	utils.LogInfo("Dashboard", fmt.Sprintf("Summary ready: AUM=%.0f, clients=%d, missed SIPs=%d", totalAUM, len(clients), len(missedSIPs)))

	summary := DashboardSummary{
		TotalAUM:        totalAUM,
		TotalClients:    len(clients),
		TotalInvested:   totalInvested,
		OverallReturns:  overallReturns,
		AssetAllocation: assetAllocation,
		ClientsWithAUM:  clientsWithAUM,
		TopPerformers: TopPerformers{
			TopClients: topClients,
			TopFunds:   topFunds,
		},
		MissedSIPs: len(missedSIPs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetSIPAlertsHandler returns upcoming SIP installments
func GetSIPAlertsHandler(w http.ResponseWriter, r *http.Request) {
	upcomingSIPs, err := services.GetUpcomingSIPs(database.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upcomingSIPs)
}

// RefreshNAVHandler manually triggers NAV refresh (fetches last 30 days)
func RefreshNAVHandler(w http.ResponseWriter, r *http.Request) {
	utils.LogInfo("RefreshNAV", "Manual NAV refresh triggered")

	funds, err := models.GetAllFunds(database.DB)
	if err != nil {
		utils.LogError("RefreshNAV", fmt.Errorf("failed to get funds: %w", err))
		http.Error(w, "Failed to get funds: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(funds) == 0 {
		utils.LogInfo("RefreshNAV", "No funds found in database")
		http.Error(w, "No funds found. Import data first.", http.StatusBadRequest)
		return
	}

	utils.LogInfo("RefreshNAV", fmt.Sprintf("Fetching NAV for %d funds", len(funds)))

	// Fetch recent NAV (last 30 days) for each fund
	successCount := 0
	failedFunds := []string{}
	for _, fund := range funds {
		utils.LogInfo("RefreshNAV", fmt.Sprintf("Fetching NAV for fund: %s (scheme: %s)", fund.SchemeName, fund.SchemeCode))
		if err := services.FetchRecentNAV(database.DB, fund.ID, fund.SchemeCode); err != nil {
			utils.LogError("RefreshNAV", fmt.Errorf("failed for fund %s (%s): %w", fund.SchemeName, fund.SchemeCode, err))
			failedFunds = append(failedFunds, fund.SchemeName)
			continue
		}
		successCount++
	}

	if successCount == 0 {
		utils.LogError("RefreshNAV", fmt.Errorf("all %d NAV fetches failed", len(funds)))
		http.Error(w, "All NAV fetches failed", http.StatusInternalServerError)
		return
	}

	utils.LogInfo("RefreshNAV", fmt.Sprintf("NAV refresh complete: %d/%d succeeded, %d failed", successCount, len(funds), len(failedFunds)))

	response := map[string]interface{}{
		"success":       true,
		"fetched_count": successCount,
		"total_count":   len(funds),
		"message":       "Fetched last 30 days of NAV data",
	}

	if len(failedFunds) > 0 {
		response["failed_funds"] = failedFunds
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
