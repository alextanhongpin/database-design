# Bitemporal Data Modeling

Bitemporal data modeling involves tracking two different time dimensions: **valid time** (when the data is true in the real world) and **transaction time** (when the data was stored in the database). This pattern is essential for financial systems, legal compliance, and any domain requiring complete historical accuracy.

## Core Concepts

### Valid Time vs Transaction Time

- **Valid Time**: When the fact was true in the real world
- **Transaction Time**: When the fact was recorded in the database

```sql
-- Basic bitemporal table structure
CREATE TABLE employee_salary_bitemporal (
    id SERIAL PRIMARY KEY,
    employee_id UUID NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    
    -- Valid time dimension
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ NOT NULL DEFAULT 'infinity',
    
    -- Transaction time dimension
    transaction_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    transaction_to TIMESTAMPTZ NOT NULL DEFAULT 'infinity',
    
    -- Ensure no overlapping periods for same employee
    EXCLUDE USING gist (
        employee_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&,
        tstzrange(transaction_from, transaction_to, '[)') WITH &&
    )
);
```

## Implementation Patterns

### Basic Bitemporal Insert

```sql
-- Insert a new salary record
INSERT INTO employee_salary_bitemporal (
    employee_id, salary, valid_from, valid_to
) VALUES (
    'emp-123', 75000, '2024-01-01', '2024-12-31'
);
```

### Correcting Historical Data

When you need to correct historical data:

```sql
-- Scenario: Correct a salary that was recorded incorrectly
-- Original record: Employee had salary of 70000 from Jan 1, but we recorded 75000

-- 1. Close the incorrect transaction
UPDATE employee_salary_bitemporal 
SET transaction_to = NOW()
WHERE employee_id = 'emp-123' 
  AND valid_from = '2024-01-01'
  AND transaction_to = 'infinity';

-- 2. Insert the corrected record
INSERT INTO employee_salary_bitemporal (
    employee_id, salary, valid_from, valid_to
) VALUES (
    'emp-123', 70000, '2024-01-01', '2024-12-31'
);
```

### Backdating Changes

When you learn about changes that happened in the past:

```sql
-- Scenario: Employee got a raise on March 1, but we're recording it on March 15

-- 1. Close the current record's valid time
UPDATE employee_salary_bitemporal 
SET valid_to = '2024-03-01'
WHERE employee_id = 'emp-123' 
  AND valid_to = 'infinity'
  AND transaction_to = 'infinity';

-- 2. Insert the new salary record
INSERT INTO employee_salary_bitemporal (
    employee_id, salary, valid_from, valid_to
) VALUES (
    'emp-123', 80000, '2024-03-01', 'infinity'
);
```

## Query Patterns

### Current State Query

Get the current state as of now:

```sql
-- What is the current salary?
SELECT employee_id, salary
FROM employee_salary_bitemporal
WHERE employee_id = 'emp-123'
  AND valid_from <= NOW()
  AND valid_to > NOW()
  AND transaction_from <= NOW()
  AND transaction_to > NOW();
```

### Point-in-Time Query

Get the state as it was known at a specific point in time:

```sql
-- What did we think the salary was on February 1?
SELECT employee_id, salary
FROM employee_salary_bitemporal
WHERE employee_id = 'emp-123'
  AND valid_from <= '2024-02-01'
  AND valid_to > '2024-02-01'
  AND transaction_from <= '2024-02-01'
  AND transaction_to > '2024-02-01';
```

### Historical Truth Query

Get what the salary actually was (ignoring when we learned about it):

```sql
-- What was the actual salary on February 1?
SELECT employee_id, salary
FROM employee_salary_bitemporal
WHERE employee_id = 'emp-123'
  AND valid_from <= '2024-02-01'
  AND valid_to > '2024-02-01'
  AND transaction_to = 'infinity';  -- Current version of truth
```

### Audit Trail Query

See all changes and when they were made:

```sql
-- Show complete audit trail for an employee
SELECT 
    employee_id,
    salary,
    valid_from,
    valid_to,
    transaction_from,
    transaction_to,
    CASE 
        WHEN transaction_to = 'infinity' THEN 'Current'
        ELSE 'Superseded'
    END AS status
FROM employee_salary_bitemporal
WHERE employee_id = 'emp-123'
ORDER BY transaction_from, valid_from;
```

## Advanced Patterns

### Temporal Joins

Join bitemporal tables while respecting both time dimensions:

```sql
-- Join employee and department data for a specific point in time
SELECT 
    e.employee_id,
    e.salary,
    d.department_name
FROM employee_salary_bitemporal e
JOIN department_employee_bitemporal de ON e.employee_id = de.employee_id
JOIN departments d ON de.department_id = d.id
WHERE e.valid_from <= '2024-02-01'
  AND e.valid_to > '2024-02-01'
  AND e.transaction_to = 'infinity'
  AND de.valid_from <= '2024-02-01'
  AND de.valid_to > '2024-02-01'
  AND de.transaction_to = 'infinity';
```

### Slowly Changing Dimensions

Handle Type 2 SCDs with bitemporal patterns:

```sql
CREATE TABLE customer_bitemporal (
    id SERIAL PRIMARY KEY,
    customer_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    address TEXT,
    phone VARCHAR(20),
    
    -- Bitemporal columns
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ NOT NULL DEFAULT 'infinity',
    transaction_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    transaction_to TIMESTAMPTZ NOT NULL DEFAULT 'infinity',
    
    -- Track change reasons
    change_reason VARCHAR(100),
    
    EXCLUDE USING gist (
        customer_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&,
        tstzrange(transaction_from, transaction_to, '[)') WITH &&
    )
);
```

## Trigger-Based Implementation

Automate bitemporal operations with triggers:

```sql
-- Function to handle bitemporal updates
CREATE OR REPLACE FUNCTION handle_bitemporal_update()
RETURNS TRIGGER AS $$
BEGIN
    -- Close the old record
    UPDATE employee_salary_bitemporal 
    SET transaction_to = NOW()
    WHERE id = OLD.id;
    
    -- Insert new record with updated data
    INSERT INTO employee_salary_bitemporal (
        employee_id, salary, valid_from, valid_to
    ) VALUES (
        NEW.employee_id, NEW.salary, NEW.valid_from, NEW.valid_to
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
CREATE TRIGGER bitemporal_update_trigger
    INSTEAD OF UPDATE ON employee_salary_bitemporal
    FOR EACH ROW
    EXECUTE FUNCTION handle_bitemporal_update();
```

## Use Cases

### Financial Systems

- **Trade Records**: Track when trades occurred vs when they were recorded
- **Account Balances**: Maintain historical accuracy with late-arriving transactions
- **Risk Calculations**: Recompute historical risk with updated market data

### Legal and Compliance

- **Contract Terms**: Track when terms were effective vs when they were discovered
- **Regulatory Reporting**: Maintain audit trails for compliance
- **Legal Obligations**: Handle backdated legal decisions

### Healthcare

- **Patient Records**: Track when conditions occurred vs when they were diagnosed
- **Treatment History**: Handle corrections to medical records
- **Insurance Claims**: Manage claims processing with late arrivals

## Performance Considerations

### Indexing Strategy

```sql
-- Indexes for efficient bitemporal queries
CREATE INDEX idx_employee_salary_bitemporal_current
ON employee_salary_bitemporal (employee_id, valid_from, valid_to)
WHERE transaction_to = 'infinity';

CREATE INDEX idx_employee_salary_bitemporal_valid_time
ON employee_salary_bitemporal USING gist (
    employee_id, tstzrange(valid_from, valid_to, '[)')
);

CREATE INDEX idx_employee_salary_bitemporal_transaction_time
ON employee_salary_bitemporal USING gist (
    employee_id, tstzrange(transaction_from, transaction_to, '[)')
);
```

### Partitioning

```sql
-- Partition by transaction time for better performance
CREATE TABLE employee_salary_bitemporal_2024 PARTITION OF employee_salary_bitemporal
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

CREATE TABLE employee_salary_bitemporal_2025 PARTITION OF employee_salary_bitemporal
FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
```

## Best Practices

1. **Use appropriate data types** - TIMESTAMPTZ for precision, 'infinity' for open intervals
2. **Implement proper constraints** - Prevent temporal overlaps with exclusion constraints
3. **Index strategically** - Create indexes for your specific query patterns
4. **Consider partitioning** - For large tables, partition by transaction time
5. **Automate with triggers** - Use triggers to maintain bitemporal invariants
6. **Document time semantics** - Clearly define what each time dimension means
7. **Handle null values carefully** - Use 'infinity' instead of null for open intervals
8. **Test edge cases** - Validate behavior with concurrent updates and time zone changes

## Common Pitfalls

- Confusing valid time with transaction time
- Not handling time zone issues properly
- Poor indexing leading to slow queries
- Allowing temporal overlaps
- Not considering the impact of corrections on downstream systems
- Inadequate testing of concurrent operations

## References

- [PostgreSQL Temporal Features](https://www.postgresql.eu/events/pgdayparis2019/sessions/session/2291/slides/171/pgdayparis_2019_msedivy_bitemporality.pdf)
- [Temporal Data & PostgreSQL](https://www.postgresql.org/docs/current/rangetypes.html)
- [Bitemporal Data Modeling Patterns](https://en.wikipedia.org/wiki/Temporal_database)
