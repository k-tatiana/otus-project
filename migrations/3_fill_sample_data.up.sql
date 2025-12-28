-- 3_fill_sample_data.up.sql
-- Insert sample data for loyalty system

INSERT INTO loyalty_levels (id, name, percent) VALUES
    (1, 'Bronze', 1),
    (2, 'Silver', 2),
    (3, 'Gold', 5),
    (4, 'Platinum', 10);

-- Insert sample customers
INSERT INTO customers (first_name, last_name) VALUES
    ('John', 'Doe'),
    ('Jane', 'Smith'),
    ('Alice', 'Johnson');


-- Insert predefined balance reasons
INSERT INTO balance_reasons (id, name) VALUES
    (1, 'marketing activity'),
    (2, 'order'),
    (3, 'using points'),
    (4, 'birthday'),
    (5, 'expire points');