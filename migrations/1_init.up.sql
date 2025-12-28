-- 1_init.up.sql
-- Create a sample table for demonstration purposes

CREATE TABLE IF NOT EXISTS items (
    id serial PRIMARY KEY,
    name varchar(255) NOT NULL
);
