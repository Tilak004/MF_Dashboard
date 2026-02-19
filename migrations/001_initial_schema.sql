-- Mutual Fund Dashboard - Initial Database Schema

-- Users table for authentication
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Clients table
CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(20),
    pan VARCHAR(10) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Funds table
CREATE TABLE IF NOT EXISTS funds (
    id SERIAL PRIMARY KEY,
    scheme_code VARCHAR(50) UNIQUE NOT NULL,
    scheme_name VARCHAR(500) NOT NULL,
    fund_house VARCHAR(255),
    category VARCHAR(50) NOT NULL,  -- 'Equity', 'Debt', 'Hybrid'
    risk_level VARCHAR(20),         -- 'High', 'Medium', 'Low'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id) ON DELETE CASCADE,
    fund_id INTEGER REFERENCES funds(id),
    transaction_type VARCHAR(20) NOT NULL,  -- 'LUMPSUM', 'SIP', 'REDEMPTION', 'SWITCH_IN', 'SWITCH_OUT'
    transaction_date DATE NOT NULL,
    amount DECIMAL(15, 2) NOT NULL,
    nav DECIMAL(10, 4),
    units DECIMAL(15, 4),
    folio_number VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- NAV History table
CREATE TABLE IF NOT EXISTS nav_history (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id),
    nav_date DATE NOT NULL,
    nav_value DECIMAL(10, 4) NOT NULL,
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(fund_id, nav_date)
);

-- SIP Schedules table
CREATE TABLE IF NOT EXISTS sip_schedules (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id) ON DELETE CASCADE,
    fund_id INTEGER REFERENCES funds(id),
    amount DECIMAL(15, 2) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    frequency VARCHAR(20) DEFAULT 'MONTHLY',  -- 'MONTHLY', 'QUARTERLY'
    day_of_month INTEGER,                     -- 1-31
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_transactions_client_id ON transactions(client_id);
CREATE INDEX IF NOT EXISTS idx_transactions_fund_id ON transactions(fund_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_nav_history_fund_date ON nav_history(fund_id, nav_date);
CREATE INDEX IF NOT EXISTS idx_sip_schedules_client ON sip_schedules(client_id);
CREATE INDEX IF NOT EXISTS idx_sip_schedules_active ON sip_schedules(is_active);
