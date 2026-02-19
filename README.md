# Mutual Fund Distributor Dashboard

A comprehensive web application for mutual fund distributors to manage client portfolios, track AUM, monitor SIP compliance, and analyze investment performance.

## Features

- **Client Management**: Track all client details with PAN, email, and phone
- **Transaction Management**: Record lumpsum, SIP, redemptions, and fund switches
- **Automatic NAV Fetching**: Daily NAV updates from mfapi.in
- **Portfolio Calculations**: Automatic unit calculation, current value, and XIRR
- **Dashboard Analytics**:
  - Total AUM across all clients
  - Asset allocation (Equity/Debt/Hybrid)
  - Client-wise portfolio breakdown
  - Top performing clients and funds
- **SIP Compliance Monitoring**: Track missed SIP installments with email alerts
- **Bulk Import**: CSV upload for quick transaction entry
- **Risk Assessment**: Portfolio categorization by asset class

## Tech Stack

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 15+
- **Frontend**: HTML5, CSS3, JavaScript, Chart.js
- **External API**: mfapi.in for NAV data
- **Cron Jobs**: Automated daily NAV fetch and SIP monitoring

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 15 or higher
- SMTP server credentials (for email alerts)

## Installation

### 1. Clone and Setup

```bash
cd "Client dashboard"
```

### 2. Install Dependencies

Dependencies are already initialized in `go.mod`. To download them:

```bash
go mod download
```

### 3. Setup PostgreSQL Database

Create a PostgreSQL database:

```sql
CREATE DATABASE mf_dashboard;
```

### 4. Configure Environment Variables

Copy `.env.example` to `.env` and update with your credentials:

```bash
cp .env.example .env
```

Edit `.env`:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=mf_dashboard

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
ALERT_EMAIL=your-alert-email@gmail.com

PORT=8080
SESSION_SECRET=your-random-secret-key
```

**Note**: For Gmail, you need to create an [App Password](https://support.google.com/accounts/answer/185833).

### 5. Run Database Migrations

Migrations run automatically when you start the server. The migration file is located at:
`migrations/001_initial_schema.sql`

### 6. Create Initial User

Before starting the server, you need to create an initial user. Connect to PostgreSQL:

```bash
psql -U postgres -d mf_dashboard
```

Run this SQL command (replace 'admin' and 'password' with your desired credentials):

```sql
INSERT INTO users (username, password_hash, email)
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin@example.com');
```

**Default Password**: `password` (You should change this immediately after first login)

Or, to create a user with a custom password, use this Go snippet:

```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "your_password_here"
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
```

Then insert the hash into the database.

### 7. Build and Run

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## Usage

### 1. Login

Navigate to `http://localhost:8080/login` and login with your credentials.

### 2. Add Clients

Go to **Clients** page and add your clients with their details (Name, PAN, Email, Phone).

### 3. Import Transactions

The easiest way to add transactions is via CSV import:

1. Go to **Import** page
2. Download the sample CSV template
3. Fill in your transaction data following the format
4. Upload the CSV file

**CSV Format:**
```
Client Name, PAN, Email, Phone, Fund Name, Scheme Code, Fund House, Category, Transaction Type, Date, Amount, Folio, Notes
```

**Example Row:**
```
Rajesh Kumar,ABCDE1234F,rajesh@example.com,9876543210,HDFC Equity Fund,120503,HDFC,Equity,LUMPSUM,01-01-2024,50000,ABC123,Initial investment
```

### 4. View Dashboard

The dashboard shows:
- Total AUM and overall returns
- Asset allocation pie chart
- Client-wise portfolio breakdown
- Top performing clients
- SIP compliance alerts

### 5. Refresh NAV

Click "Refresh NAV" button on the dashboard to manually fetch latest NAV for all funds.

### 6. Monitor SIP Compliance

The system automatically checks for missed SIP installments daily at 9 AM and sends email alerts.

## Finding Mutual Fund Scheme Codes

To get scheme codes for mutual funds, visit [https://www.mfapi.in/](https://www.mfapi.in/)

Example scheme codes:
- HDFC Equity Fund: `120503`
- ICICI Prudential Balanced Advantage Fund: `120259`
- SBI Bluechip Fund: `119598`

## Automated Jobs

The application runs two automated cron jobs:

1. **NAV Fetch**: Daily at 8 PM IST - Fetches latest NAV for all active funds
2. **SIP Compliance**: Daily at 9 AM IST - Checks for missed SIP installments and sends email alerts

## API Endpoints

### Authentication
- `POST /api/login` - Login
- `POST /api/logout` - Logout

### Dashboard
- `GET /api/dashboard/summary` - Dashboard summary data
- `GET /api/dashboard/sip-alerts` - Missed SIP installments
- `POST /api/dashboard/refresh-nav` - Manual NAV refresh

### Clients
- `GET /api/clients` - Get all clients
- `POST /api/clients` - Create new client
- `GET /api/clients/{id}` - Get client by ID
- `PUT /api/clients/{id}` - Update client
- `DELETE /api/clients/{id}` - Delete client

### Transactions
- `POST /api/transactions` - Create transaction
- `GET /api/clients/{id}/transactions` - Get client transactions
- `GET /api/clients/{id}/portfolio` - Get client portfolio
- `PUT /api/transactions/{id}` - Update transaction
- `DELETE /api/transactions/{id}` - Delete transaction

### Import
- `POST /api/import/transactions` - Upload CSV file
- `GET /api/import/sample-csv` - Download sample CSV

## Database Schema

The application uses 6 main tables:

1. **users** - Authentication
2. **clients** - Client information
3. **funds** - Mutual fund schemes
4. **transactions** - All transactions (lumpsum, SIP, redemptions, switches)
5. **nav_history** - Historical NAV data
6. **sip_schedules** - Scheduled SIP information

## XIRR Calculation

The application uses the Newton-Raphson method to calculate Extended Internal Rate of Return (XIRR), which gives accurate annualized returns considering the timing of all cashflows.

## Troubleshooting

### Database Connection Issues

Ensure PostgreSQL is running and credentials in `.env` are correct:

```bash
# Check if PostgreSQL is running
# Windows:
services.msc  # Look for PostgreSQL service

# Test connection
psql -U postgres -d mf_dashboard
```

### Email Alerts Not Working

1. Verify SMTP credentials in `.env`
2. For Gmail, ensure you're using an App Password, not your regular password
3. Check if "Less secure app access" is enabled (if not using App Password)

### NAV Fetch Failing

- Check internet connectivity
- Verify scheme codes are correct
- The mfapi.in API is free and doesn't require authentication

### Port Already in Use

If port 8080 is occupied, change it in `.env`:

```env
PORT=3000
```

## Production Deployment

For production deployment:

1. Use a reverse proxy (Nginx/Apache)
2. Enable HTTPS
3. Use environment variables instead of `.env` file
4. Set up database backups
5. Configure proper logging
6. Use systemd or similar for process management
7. Consider using a session store (Redis) instead of in-memory sessions

## Future Enhancements

- Commission tracking
- Client portal (clients can login to view their own portfolio)
- WhatsApp alerts for SIP compliance
- Advanced reporting and analytics
- Mobile app
- Multi-user support with roles
- Goal-based planning
- Tax reporting (Capital gains)

## License

This project is built for internal use by mutual fund distributors.

## Support

For issues or questions, please contact your development team.
