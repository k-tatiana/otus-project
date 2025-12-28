-- 2_loyalty_system.up.sql
-- Create tables for loyalty system

-- Customers table
CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL
);

-- Loyalty levels table with predefined values
CREATE TABLE IF NOT EXISTS loyalty_levels (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    percent INTEGER NOT NULL
);

-- Loyalty table (current loyalty status for each customer)
CREATE TABLE IF NOT EXISTS loyalty (
    customer_id INTEGER PRIMARY KEY REFERENCES customers(id) ON DELETE SET NULL,
    balance INTEGER NOT NULL DEFAULT 0,
    level_id INTEGER NOT NULL REFERENCES loyalty_levels(id),
    CONSTRAINT balance_non_negative CHECK (balance >= 0)
);


-- Balance reasons table
CREATE TABLE IF NOT EXISTS balance_reasons (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

-- Balance history table
CREATE TABLE IF NOT EXISTS balance_history (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE SET NULL,
    points INTEGER NOT NULL,
    reason_id INTEGER NOT NULL REFERENCES balance_reasons(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expire_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days')
);

