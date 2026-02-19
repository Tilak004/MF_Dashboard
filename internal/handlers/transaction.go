package handlers

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/models"
	"client-dashboard/internal/services"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// CreateTransactionHandler creates a new transaction
func CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var txn models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If units or NAV not provided, calculate them
	if txn.Units == 0 || txn.NAV == 0 {
		fund, err := models.GetFundByID(database.DB, txn.FundID)
		if err != nil {
			http.Error(w, "Fund not found", http.StatusNotFound)
			return
		}

		units, nav, err := services.CalculateUnitsForTransaction(
			database.DB,
			txn.FundID,
			fund.SchemeCode,
			txn.Amount,
			txn.TransactionDate,
		)
		if err != nil {
			http.Error(w, "Failed to calculate units: "+err.Error(), http.StatusInternalServerError)
			return
		}

		txn.Units = units
		txn.NAV = nav
	}

	if err := models.CreateTransaction(database.DB, &txn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(txn)
}

// GetClientTransactionsHandler returns all transactions for a client
func GetClientTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	transactions, err := models.GetTransactionsByClient(database.DB, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

// UpdateTransactionHandler updates an existing transaction
func UpdateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	var txn models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	txn.ID = id
	if err := models.UpdateTransaction(database.DB, &txn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

// DeleteTransactionHandler deletes a transaction
func DeleteTransactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	if err := models.DeleteTransaction(database.DB, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetClientPortfolioHandler returns portfolio details for a client
func GetClientPortfolioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	portfolio, err := services.CalculateClientPortfolio(database.DB, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}
