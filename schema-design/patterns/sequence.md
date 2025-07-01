# Sequence & Auto-Increment Patterns

Sequences provide a reliable way to generate unique, sequential numbers for database records. This guide covers PostgreSQL sequences, MySQL auto-increment, and advanced sequential ID patterns.

## 🎯 Understanding Sequences

### What are Sequences?
- **Database Objects** that generate unique numeric values
- **Thread-Safe** - Multiple connections can safely request values
- **Persistent** - Values survive database restarts
- **Configurable** - Control increment, start value, min/max bounds

### Common Use Cases
- **Primary Keys** - Unique record identifiers
- **Order Numbers** - Sequential business identifiers  
- **Invoice Numbers** - Gapless sequential numbering
- **Version Control** - Document/record versioning
- **Batch Processing** - Sequential job IDs

## 🏗️ PostgreSQL Sequences

### 1. Basic Sequence Creation

```sql
-- Create a simple sequence
CREATE SEQUENCE user_id_seq
    START WITH 1
    INCREMENT BY 1
    MINVALUE 1
    MAXVALUE 9223372036854775807
    CACHE 1;

-- Using sequences in tables
CREATE TABLE users (
    id BIGINT PRIMARY KEY DEFAULT nextval('user_id_seq'),
    username TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Alternative: SERIAL type (creates sequence automatically)
CREATE TABLE products (
    id SERIAL PRIMARY KEY,  -- Creates product_id_seq automatically
    name TEXT NOT NULL,
    price DECIMAL(10,2)
);

-- Modern approach: GENERATED ALWAYS AS IDENTITY (PostgreSQL 10+)
CREATE TABLE orders (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 2. Advanced Sequence Configuration

```sql
-- Custom increment sequences
CREATE SEQUENCE even_numbers_seq
    START WITH 2
    INCREMENT BY 2
    MINVALUE 2
    MAXVALUE 1000000
    CYCLE;  -- Restart from MINVALUE when MAXVALUE reached

-- Descending sequences
CREATE SEQUENCE countdown_seq
    START WITH 1000
    INCREMENT BY -1
    MINVALUE 1
    MAXVALUE 1000;

-- Large cache for high-throughput applications
CREATE SEQUENCE high_volume_seq
    START WITH 1
    INCREMENT BY 1
    CACHE 100;  -- Cache 100 values per connection

-- Year-based sequences
CREATE SEQUENCE invoice_2024_seq
    START WITH 240001  -- Year prefix + counter
    INCREMENT BY 1
    MAXVALUE 249999;
```

### 3. Managing Sequences

```sql
-- View all sequences in current database
SELECT 
    schemaname,
    sequencename,
    start_value,
    min_value,
    max_value,
    increment_by,
    cycle,
    cache_size,
    last_value
FROM pg_sequences;

-- Get current value without advancing
SELECT last_value FROM user_id_seq;

-- Get next value (advances sequence)
SELECT nextval('user_id_seq');

-- Set current value
SELECT setval('user_id_seq', 1000);

-- Reset sequence to start value
ALTER SEQUENCE user_id_seq RESTART;

-- Modify sequence properties
ALTER SEQUENCE user_id_seq 
    INCREMENT BY 2
    CACHE 50
    NO CYCLE;

-- Drop sequence (be careful with dependencies)
DROP SEQUENCE IF EXISTS user_id_seq CASCADE;
```

## 🔢 MySQL Auto-Increment

### 1. Basic Auto-Increment

```sql
-- MySQL auto-increment column
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Custom starting value
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) AUTO_INCREMENT = 1000;

-- Multiple auto-increment columns (MyISAM only)
CREATE TABLE composite_auto (
    category_id INT NOT NULL,
    item_id INT AUTO_INCREMENT,
    name VARCHAR(100),
    PRIMARY KEY (category_id, item_id)
) ENGINE = MyISAM;
```

### 2. Managing MySQL Auto-Increment

```sql
-- Check current auto-increment value
SELECT AUTO_INCREMENT 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'your_database' 
AND TABLE_NAME = 'users';

-- Reset auto-increment value
ALTER TABLE users AUTO_INCREMENT = 1;

-- Set specific auto-increment value
ALTER TABLE users AUTO_INCREMENT = 5000;

-- Get last inserted ID (session-specific)
SELECT LAST_INSERT_ID();

-- Find gaps in auto-increment sequence
SELECT 
    id + 1 AS missing_start,
    next_id - 1 AS missing_end
FROM (
    SELECT 
        id,
        LEAD(id) OVER (ORDER BY id) AS next_id
    FROM users
) AS gaps
WHERE next_id - id > 1;
```

## 🎭 Advanced Sequential Patterns

### 1. Gapless Sequential Numbers

```sql
-- Gapless sequence function (PostgreSQL)
CREATE OR REPLACE FUNCTION get_next_invoice_number()
RETURNS INTEGER AS $$
DECLARE
    next_num INTEGER;
BEGIN
    -- Lock table to prevent concurrent access
    LOCK TABLE invoice_counters IN EXCLUSIVE MODE;
    
    -- Get and increment counter
    UPDATE invoice_counters 
    SET current_value = current_value + 1
    WHERE counter_name = 'invoice'
    RETURNING current_value INTO next_num;
    
    -- Initialize if not exists
    IF next_num IS NULL THEN
        INSERT INTO invoice_counters (counter_name, current_value)
        VALUES ('invoice', 1);
        next_num := 1;
    END IF;
    
    RETURN next_num;
END;
$$ LANGUAGE plpgsql;

-- Supporting table for gapless counters
CREATE TABLE invoice_counters (
    counter_name TEXT PRIMARY KEY,
    current_value INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Usage
INSERT INTO invoices (invoice_number, customer_id, amount)
VALUES (get_next_invoice_number(), 123, 999.99);
```

### 2. Formatted Sequential IDs

```sql
-- Generate formatted order numbers: ORD-2024-000001
CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS TEXT AS $$
DECLARE
    year_part TEXT;
    sequence_part TEXT;
    next_seq INTEGER;
BEGIN
    year_part := EXTRACT(YEAR FROM NOW())::TEXT;
    
    -- Get next sequence for current year
    INSERT INTO yearly_sequences (year, sequence_name, current_value)
    VALUES (EXTRACT(YEAR FROM NOW()), 'order', 1)
    ON CONFLICT (year, sequence_name)
    DO UPDATE SET 
        current_value = yearly_sequences.current_value + 1,
        updated_at = NOW()
    RETURNING current_value INTO next_seq;
    
    -- Format with leading zeros
    sequence_part := LPAD(next_seq::TEXT, 6, '0');
    
    RETURN 'ORD-' || year_part || '-' || sequence_part;
END;
$$ LANGUAGE plpgsql;

-- Supporting table for yearly sequences
CREATE TABLE yearly_sequences (
    year INTEGER NOT NULL,
    sequence_name TEXT NOT NULL,
    current_value INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (year, sequence_name)
);
```

### 3. Multi-Tenant Sequences

```sql
-- Tenant-specific sequences
CREATE TABLE tenant_sequences (
    tenant_id UUID NOT NULL,
    sequence_name TEXT NOT NULL,
    current_value BIGINT NOT NULL DEFAULT 0,
    prefix TEXT,
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (tenant_id, sequence_name)
);

-- Function to get next tenant-specific sequence value
CREATE OR REPLACE FUNCTION get_next_tenant_sequence(
    p_tenant_id UUID,
    p_sequence_name TEXT,
    p_prefix TEXT DEFAULT NULL
) RETURNS TEXT AS $$
DECLARE
    next_val BIGINT;
    formatted_id TEXT;
BEGIN
    -- Upsert sequence value
    INSERT INTO tenant_sequences (tenant_id, sequence_name, current_value, prefix)
    VALUES (p_tenant_id, p_sequence_name, 1, p_prefix)
    ON CONFLICT (tenant_id, sequence_name)
    DO UPDATE SET 
        current_value = tenant_sequences.current_value + 1,
        updated_at = NOW()
    RETURNING current_value INTO next_val;
    
    -- Format with prefix if provided
    IF p_prefix IS NOT NULL THEN
        formatted_id := p_prefix || LPAD(next_val::TEXT, 8, '0');
    ELSE
        formatted_id := next_val::TEXT;
    END IF;
    
    RETURN formatted_id;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT get_next_tenant_sequence(
    '123e4567-e89b-12d3-a456-426614174000'::UUID, 
    'invoice', 
    'INV-'
); -- Returns: INV-00000001
```

## 🔄 Sequence Synchronization & Recovery

### 1. Fixing Sequence After Manual Inserts

```sql
-- Fix sequence after manual ID insertions
CREATE OR REPLACE FUNCTION fix_sequence_value(
    sequence_name TEXT,
    table_name TEXT,
    column_name TEXT
) RETURNS VOID AS $$
DECLARE
    max_id BIGINT;
    sql_query TEXT;
BEGIN
    -- Get maximum existing ID
    sql_query := format('SELECT COALESCE(MAX(%I), 0) FROM %I', 
                       column_name, table_name);
    EXECUTE sql_query INTO max_id;
    
    -- Set sequence to max + 1
    PERFORM setval(sequence_name, max_id + 1, false);
    
    RAISE INFO 'Sequence % reset to %', sequence_name, max_id + 1;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT fix_sequence_value('users_id_seq', 'users', 'id');
```

### 2. Sequence Monitoring & Alerts

```sql
-- View to monitor sequence usage
CREATE VIEW sequence_monitoring AS
SELECT 
    s.sequencename,
    s.last_value,
    s.max_value,
    ROUND(
        (s.last_value::FLOAT / s.max_value::FLOAT) * 100, 2
    ) AS percentage_used,
    CASE 
        WHEN s.last_value::FLOAT / s.max_value::FLOAT > 0.9 
        THEN 'CRITICAL'
        WHEN s.last_value::FLOAT / s.max_value::FLOAT > 0.8 
        THEN 'WARNING'
        ELSE 'OK'
    END AS status,
    s.increment_by,
    s.cycle
FROM pg_sequences s
WHERE s.schemaname = 'public'
ORDER BY percentage_used DESC;

-- Function to alert on sequence exhaustion
CREATE OR REPLACE FUNCTION check_sequence_limits()
RETURNS TABLE(
    sequence_name TEXT,
    current_value BIGINT,
    max_value BIGINT,
    remaining_values BIGINT,
    days_until_exhaustion NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH sequence_stats AS (
        SELECT 
            sequencename,
            last_value,
            max_value,
            increment_by,
            -- Estimate daily usage based on recent activity
            COALESCE(
                (SELECT COUNT(*) FROM pg_stat_activity) * 100, -- Rough estimate
                1000
            ) AS estimated_daily_usage
        FROM pg_sequences
        WHERE schemaname = 'public'
    )
    SELECT 
        s.sequencename::TEXT,
        s.last_value,
        s.max_value,
        (s.max_value - s.last_value) AS remaining,
        CASE 
            WHEN s.estimated_daily_usage > 0 
            THEN ROUND((s.max_value - s.last_value)::NUMERIC / s.estimated_daily_usage, 2)
            ELSE NULL 
        END AS days_left
    FROM sequence_stats s
    WHERE (s.last_value::FLOAT / s.max_value::FLOAT) > 0.8
    ORDER BY (s.last_value::FLOAT / s.max_value::FLOAT) DESC;
END;
$$ LANGUAGE plpgsql;
```

## ⚡ Performance Optimization

### 1. Sequence Caching

```sql
-- High-throughput sequence with large cache
CREATE SEQUENCE high_volume_seq
    INCREMENT BY 1
    CACHE 1000;  -- Cache 1000 values per connection

-- Monitor cache effectiveness
SELECT 
    schemaname,
    sequencename,
    cache_size,
    calls,
    calls / GREATEST(cache_size, 1) AS cache_misses_estimate
FROM pg_sequences s
LEFT JOIN pg_stat_user_sequences st ON s.sequencename = st.relname
WHERE s.schemaname = 'public';
```

### 2. Bulk Sequence Allocation

```sql
-- Allocate sequence ranges for bulk operations
CREATE OR REPLACE FUNCTION allocate_sequence_range(
    seq_name TEXT,
    range_size INTEGER
) RETURNS TABLE(start_id BIGINT, end_id BIGINT) AS $$
DECLARE
    start_val BIGINT;
    end_val BIGINT;
BEGIN
    -- Allocate range by advancing sequence
    SELECT nextval(seq_name) INTO start_val;
    SELECT setval(seq_name, currval(seq_name) + range_size - 1) INTO end_val;
    
    RETURN QUERY SELECT start_val, end_val;
END;
$$ LANGUAGE plpgsql;

-- Usage for bulk inserts
DO $$
DECLARE
    id_range RECORD;
    current_id BIGINT;
BEGIN
    -- Allocate 1000 IDs
    SELECT * FROM allocate_sequence_range('users_id_seq', 1000) INTO id_range;
    
    current_id := id_range.start_id;
    
    -- Use allocated range for bulk insert
    FOR i IN 1..1000 LOOP
        INSERT INTO users (id, username, email) 
        VALUES (current_id, 'user' || current_id, 'user' || current_id || '@example.com');
        current_id := current_id + 1;
    END LOOP;
END $$;
```

## ⚠️ Common Pitfalls

### 1. Sequence Gaps
```sql
-- ❌ Sequences can have gaps due to:
-- - Rolled back transactions
-- - Connection failures
-- - Multiple concurrent users

-- ✅ Use gapless sequences only when truly needed
-- Consider if gaps actually matter for your use case
```

### 2. Cache vs. Consistency Trade-offs
```sql
-- ❌ Large cache = better performance but larger gaps
CREATE SEQUENCE fast_seq CACHE 1000;

-- ❌ Small cache = fewer gaps but worse performance  
CREATE SEQUENCE slow_seq CACHE 1;

-- ✅ Balance based on your requirements
CREATE SEQUENCE balanced_seq CACHE 50;
```

### 3. Cross-Database Sequence Issues
```sql
-- ❌ Sequences are database-specific
-- Moving data between databases can cause conflicts

-- ✅ Plan for data migration and sequence synchronization
-- Consider UUIDs for cross-database portability
```

## 🎯 Best Practices

1. **Choose Appropriate Data Types** - Use BIGINT for high-volume sequences
2. **Set Reasonable Cache Sizes** - Balance performance vs. gap tolerance
3. **Monitor Sequence Usage** - Alert before exhaustion
4. **Plan for Rollbacks** - Expect gaps in sequences
5. **Use Identity Columns** - Prefer `GENERATED ALWAYS AS IDENTITY` over SERIAL
6. **Document Business Logic** - Explain gapless vs. gapped requirements
7. **Consider UUIDs** - For distributed systems or cross-database needs
8. **Regular Maintenance** - Check and fix sequence values after manual operations
9. **Performance Testing** - Verify sequence performance under load
10. **Backup Considerations** - Ensure sequence values are properly backed up

## 📊 Sequence Pattern Comparison

| Pattern | Use Case | Performance | Complexity | Gap Tolerance |
|---------|----------|-------------|------------|---------------|
| **PostgreSQL SERIAL** | Legacy primary keys | ⭐⭐⭐⭐ | ⭐⭐ | High |
| **IDENTITY Column** | Modern primary keys | ⭐⭐⭐⭐⭐ | ⭐ | High |
| **MySQL AUTO_INCREMENT** | MySQL primary keys | ⭐⭐⭐⭐ | ⭐ | High |
| **Gapless Sequences** | Invoice numbers | ⭐⭐ | ⭐⭐⭐⭐ | None |
| **Formatted IDs** | Business identifiers | ⭐⭐⭐ | ⭐⭐⭐ | Low |
| **UUID** | Distributed systems | ⭐⭐⭐ | ⭐ | N/A |

## 🔗 References

- [PostgreSQL Sequences](https://www.postgresql.org/docs/current/sql-createsequence.html)
- [PostgreSQL Identity Columns](https://www.postgresql.org/docs/current/sql-createtable.html#SQL-CREATETABLE-PARMS-GENERATED-IDENTITY)
- [MySQL AUTO_INCREMENT](https://dev.mysql.com/doc/refman/8.0/en/example-auto-increment.html)
- [Sequence Performance Tips](https://wiki.postgresql.org/wiki/Performance_Optimization)


