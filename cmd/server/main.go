package main

import (
	"client-dashboard/internal/database"
	"client-dashboard/internal/handlers"
	"client-dashboard/internal/services"
	"client-dashboard/internal/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize logger
	if err := utils.InitLogger(); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer utils.CloseLogger()

	utils.LogInfo("Startup", "Application starting...")

	// Initialize database
	if err := database.Initialize(); err != nil {
		utils.LogError("Startup", err)
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	utils.LogInfo("Startup", "Database initialized successfully")

	// Start cron jobs
	startCronJobs()

	// Create router
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Public routes
	r.HandleFunc("/api/login", handlers.LoginHandler).Methods("POST")
	r.HandleFunc("/api/logout", handlers.LogoutHandler).Methods("POST")

	// Protected API routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(authMiddleware)

	// Dashboard endpoints
	api.HandleFunc("/dashboard/summary", handlers.GetDashboardSummaryHandler).Methods("GET")
	api.HandleFunc("/dashboard/sip-alerts", handlers.GetSIPAlertsHandler).Methods("GET")
	api.HandleFunc("/dashboard/refresh-nav", handlers.RefreshNAVHandler).Methods("POST")

	// Market data endpoints
	api.HandleFunc("/market/overview", handlers.GetMarketOverviewHandler).Methods("GET")

	// Client endpoints
	api.HandleFunc("/clients", handlers.GetAllClientsHandler).Methods("GET")
	api.HandleFunc("/clients", handlers.CreateClientHandler).Methods("POST")
	api.HandleFunc("/clients/{id}", handlers.GetClientByIDHandler).Methods("GET")
	api.HandleFunc("/clients/{id}", handlers.UpdateClientHandler).Methods("PUT")
	api.HandleFunc("/clients/{id}", handlers.DeleteClientHandler).Methods("DELETE")

	// Transaction endpoints
	api.HandleFunc("/transactions", handlers.CreateTransactionHandler).Methods("POST")
	api.HandleFunc("/transactions/{id}", handlers.UpdateTransactionHandler).Methods("PUT")
	api.HandleFunc("/transactions/{id}", handlers.DeleteTransactionHandler).Methods("DELETE")
	api.HandleFunc("/clients/{id}/transactions", handlers.GetClientTransactionsHandler).Methods("GET")
	api.HandleFunc("/clients/{id}/portfolio", handlers.GetClientPortfolioHandler).Methods("GET")

	// Import endpoints
	api.HandleFunc("/import/transactions", handlers.ImportTransactionsHandler).Methods("POST")
	api.HandleFunc("/import/sip-schedules", handlers.ImportSIPSchedulesHandler).Methods("POST")
	api.HandleFunc("/import/sample-csv", handlers.GetSampleCSVHandler).Methods("GET")

	// SIP Schedule endpoints
	api.HandleFunc("/sip-schedules", handlers.GetAllSIPSchedulesHandler).Methods("GET")
	api.HandleFunc("/sip-schedules", handlers.CreateSIPScheduleHandler).Methods("POST")
	api.HandleFunc("/sip-schedules/{id}", handlers.UpdateSIPScheduleHandler).Methods("PUT")
	api.HandleFunc("/sip-schedules/{id}/deactivate", handlers.DeactivateSIPScheduleHandler).Methods("POST")
	api.HandleFunc("/clients/{clientId}/sip-schedules", handlers.GetSIPSchedulesByClientHandler).Methods("GET")
	api.HandleFunc("/sip-installments", handlers.GetSIPInstallmentsHandler).Methods("GET")

	// Frontend routes (serve HTML templates)
	r.HandleFunc("/", serveTemplate("login.html"))
	r.HandleFunc("/login", serveTemplate("login.html"))
	r.Handle("/dashboard", authMiddleware(http.HandlerFunc(serveTemplate("dashboard.html"))))
	r.Handle("/clients", authMiddleware(http.HandlerFunc(serveTemplate("clients.html"))))
	r.Handle("/client-portfolio", authMiddleware(http.HandlerFunc(serveTemplate("client_portfolio.html"))))
	r.Handle("/transactions", authMiddleware(http.HandlerFunc(serveTemplate("transactions.html"))))
	r.Handle("/sip-schedules", authMiddleware(http.HandlerFunc(serveTemplate("sip_schedules.html"))))
	r.Handle("/import", authMiddleware(http.HandlerFunc(serveTemplate("import.html"))))

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Wrap router with logging middleware
	loggedRouter := loggingMiddleware(r)

	utils.LogInfo("Startup", fmt.Sprintf("Server starting on port %s...", port))
	utils.LogInfo("Startup", fmt.Sprintf("Dashboard: http://localhost:%s/dashboard", port))
	log.Fatal(http.ListenAndServe(":"+port, loggedRouter))
}

// startCronJobs initializes and starts scheduled jobs
func startCronJobs() {
	c := cron.New()

	// Daily SIP compliance check at 9 AM IST (03:30 UTC)
	// NAV fetch is now manual via the "Refresh NAV" button on the dashboard
	c.AddFunc("0 9 * * *", func() {
		utils.LogInfo("CronJob", "Starting scheduled SIP compliance check...")
		if err := services.RunSIPComplianceCheck(database.DB); err != nil {
			utils.LogError("CronJob", fmt.Errorf("SIP check failed: %w", err))
		} else {
			utils.LogInfo("CronJob", "SIP compliance check completed successfully")
		}
	})

	c.Start()
	utils.LogInfo("Startup", "Cron jobs initialized (SIP compliance only; NAV fetch is manual)")
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log the request
		duration := time.Since(start)
		utils.LogRequest(r.Method, r.URL.Path, r.RemoteAddr, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// authMiddleware wraps handlers that require authentication (for mux.Use)
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check session (using handlers package session check)
		if !handlers.IsSessionValid(cookie.Value) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// serveTemplate returns a handler that serves an HTML template (no-cache to always pick up edits)
// serveTemplate reads the HTML file fresh on every request — bypasses browser 304/ETag caching
func serveTemplate(templateName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		templatePath := fmt.Sprintf("web/templates/%s", templateName)
		content, err := os.ReadFile(templatePath)
		if err != nil {
			utils.LogError("Template", fmt.Errorf("failed to read template %s: %w", templateName, err))
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Write(content)
	}
}
