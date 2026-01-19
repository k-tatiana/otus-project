-- Add index for fast lookups by customer_id
CREATE INDEX IF NOT EXISTS idx_loyalty_customer_id ON loyalty(customer_id);

-- Add index for the join with loyalty_levels
CREATE INDEX IF NOT EXISTS idx_loyalty_level_id ON loyalty(level_id);

-- Add index for loyalty_levels name lookups
CREATE INDEX IF NOT EXISTS idx_loyalty_levels_name ON loyalty_levels(name);
