package handlers

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/models"
	"client-dashboard/internal/services"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// SIPScheduleRequest represents the request body for creating/updating SIP schedules
type SIPScheduleRequest struct {
	ClientID   int     `json:"client_id"`
	FundID     int     `json:"fund_id"`
	Amount     float64 `json:"amount"`
	StartDate  string  `json:"start_date"`  // Format: DD/MM/YYYY
	EndDate    string  `json:"end_date,omitempty"`
	Frequency  string  `json:"frequency"`   // MONTHLY, QUARTERLY
	DayOfMonth int     `json:"day_of_month"`
	IsActive   bool    `json:"is_active"`
}

// CreateSIPScheduleHandler creates a new SIP schedule
func CreateSIPScheduleHandler(w http.ResponseWriter, r *http.Request) {
	var req SIPScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse start date (DD/MM/YYYY format)
	startDate, err := time.Parse("02/01/2006", req.StartDate)
	if err != nil {
		http.Error(w, "Invalid start date format. Use DD/MM/YYYY", http.StatusBadRequest)
		return
	}

	// Parse end date if provided
	var endDate *time.Time
	if req.EndDate != "" {
		parsed, err := time.Parse("02/01/2006", req.EndDate)
		if err != nil {
			http.Error(w, "Invalid end date format. Use DD/MM/YYYY", http.StatusBadRequest)
			return
		}
		endDate = &parsed
	}

	// Create SIP schedule
	sip := &models.SIPSchedule{
		ClientID:   req.ClientID,
		FundID:     req.FundID,
		Amount:     req.Amount,
		StartDate:  startDate,
		EndDate:    endDate,
		Frequency:  req.Frequency,
		DayOfMonth: req.DayOfMonth,
		IsActive:   req.IsActive,
	}

	if err := models.CreateSIPSchedule(database.DB, sip); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sip)
}

// GetSIPSchedulesByClientHandler retrieves all SIP schedules for a client
func GetSIPSchedulesByClientHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID, err := strconv.Atoi(vars["clientId"])
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	sips, err := models.GetSIPSchedulesByClient(database.DB, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sips)
}

// GetAllSIPSchedulesHandler retrieves all active SIP schedules
func GetAllSIPSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	sips, err := models.GetActiveSIPSchedules(database.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sips)
}

// UpdateSIPScheduleHandler updates an existing SIP schedule
func UpdateSIPScheduleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sipID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid SIP ID", http.StatusBadRequest)
		return
	}

	var req SIPScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse end date if provided
	var endDate *time.Time
	if req.EndDate != "" {
		parsed, err := time.Parse("02/01/2006", req.EndDate)
		if err != nil {
			http.Error(w, "Invalid end date format. Use DD/MM/YYYY", http.StatusBadRequest)
			return
		}
		endDate = &parsed
	}

	sip := &models.SIPSchedule{
		ID:       sipID,
		Amount:   req.Amount,
		EndDate:  endDate,
		IsActive: req.IsActive,
	}

	if err := models.UpdateSIPSchedule(database.DB, sip); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeactivateSIPScheduleHandler deactivates a SIP schedule
func DeactivateSIPScheduleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sipID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid SIP ID", http.StatusBadRequest)
		return
	}

	if err := models.DeactivateSIPSchedule(database.DB, sipID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetSIPInstallmentsHandler returns all expected vs actual SIP installments
func GetSIPInstallmentsHandler(w http.ResponseWriter, r *http.Request) {
	installments, err := services.GetAllSIPInstallments(database.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installments)
}
