# Test Data Generation (Faker)

Comprehensive guide to generating realistic test data for database development, testing, and performance analysis.

## 🎯 Overview

Test data generation is crucial for:
- **Development** - Realistic data for building features
- **Testing** - Comprehensive test scenarios
- **Performance Testing** - Large datasets for optimization
- **Demo Environments** - Realistic demonstrations
- **Data Migration Testing** - Validating migration scripts

## 📊 Basic Data Generation

### Simple Random Data

```sql
-- Generate basic test data with PostgreSQL
INSERT INTO users (name, email, created_at)
SELECT 
    'User ' || s,
    'user' || s || '@example.com',
    CURRENT_TIMESTAMP - (random() * INTERVAL '365 days')
FROM generate_series(1, 10000) AS s;

-- Generate random strings with controlled length
SELECT 
    left(md5(random()::text), 10) as short_string,
    left(md5(random()::text), round(random() * 10 + 5)::int) as variable_length
FROM generate_series(1, 100);

-- Generate random numbers within ranges
SELECT 
    floor(random() * 100)::int as random_0_to_99,
    floor(random() * 50 + 10)::int as random_10_to_59,
    random() * 1000 as random_decimal
FROM generate_series(1, 100);
```

### Reproducible Data Generation

```sql
-- Ensure reproducibility with seed
SELECT setseed(0.42);

-- Generate reproducible data
WITH seed AS (
    SELECT setseed(0.5)
)
SELECT random()
FROM generate_series(1, 10), seed;

-- Alternative approach
SELECT setseed(0.5), random() FROM generate_series(1, 10);
```

### Using Recursive CTEs

```sql
-- Simulate generate_series with recursive CTE
WITH RECURSIVE counter(n) AS (
    SELECT 1 as n
    UNION ALL
    SELECT n + 1
    FROM counter
    WHERE n < 10000
)
INSERT INTO sample_data (id, value)
SELECT n, md5(n::text)
FROM counter;
```

## 🏗️ Complex Data Patterns

### Hierarchical Data Generation

```sql
-- Generate organizational hierarchy
WITH RECURSIVE org_hierarchy AS (
    -- Root level
    SELECT 
        1 as id,
        'CEO' as title,
        'John Doe' as name,
        NULL as parent_id,
        1 as level,
        '001' as path
    
    UNION ALL
    
    -- Generate child levels
    SELECT 
        h.id * 10 + s as id,
        CASE h.level
            WHEN 1 THEN 'VP'
            WHEN 2 THEN 'Director'
            WHEN 3 THEN 'Manager'
            ELSE 'Employee'
        END as title,
        'Employee ' || (h.id * 10 + s) as name,
        h.id as parent_id,
        h.level + 1 as level,
        h.path || '.' || lpad(s::text, 3, '0') as path
    FROM org_hierarchy h
    CROSS JOIN generate_series(1, CASE h.level WHEN 1 THEN 3 WHEN 2 THEN 4 WHEN 3 THEN 5 ELSE 0 END) AS s
    WHERE h.level < 4
)
INSERT INTO employees (id, title, name, parent_id, level, path)
SELECT id, title, name, parent_id, level, path
FROM org_hierarchy;
```

### Weighted Random Data

```sql
-- Generate data with realistic distributions
CREATE OR REPLACE FUNCTION weighted_random_status()
RETURNS TEXT AS $$
DECLARE
    rand_val FLOAT := random();
BEGIN
    CASE 
        WHEN rand_val < 0.7 THEN RETURN 'active';     -- 70% active
        WHEN rand_val < 0.9 THEN RETURN 'inactive';   -- 20% inactive
        ELSE RETURN 'suspended';                       -- 10% suspended
    END CASE;
END;
$$ LANGUAGE plpgsql;

-- Generate realistic user data
INSERT INTO users (name, email, status, created_at)
SELECT 
    'User ' || s,
    'user' || s || '@' || 
    CASE floor(random() * 4)
        WHEN 0 THEN 'gmail.com'
        WHEN 1 THEN 'yahoo.com'
        WHEN 2 THEN 'outlook.com'
        ELSE 'company.com'
    END,
    weighted_random_status(),
    CURRENT_TIMESTAMP - (random() * INTERVAL '2 years')
FROM generate_series(1, 10000) AS s;
```

## 🔗 Foreign Key Relationships

### Simple Relationships

```sql
-- Step 1: Generate users data
INSERT INTO users (name) 
SELECT 'user_' || s FROM generate_series(1, 1000) s;

-- Step 2: Generate foreign key based on existing user IDs
INSERT INTO todos (user_id, title) 
SELECT u.id, 'todo_' || s
FROM users u CROSS JOIN generate_series(1, 1000) s;
```

### Complex Relationships

```sql
-- Generate data with proper foreign key relationships
WITH user_data AS (
    INSERT INTO users (name, email, created_at)
    SELECT 
        'User ' || s,
        'user' || s || '@example.com',
        CURRENT_TIMESTAMP - (random() * INTERVAL '365 days')
    FROM generate_series(1, 1000) AS s
    RETURNING id, created_at
),
project_data AS (
    INSERT INTO projects (name, owner_id, created_at)
    SELECT 
        'Project ' || row_number() OVER (),
        u.id,
        u.created_at + (random() * INTERVAL '30 days')
    FROM user_data u
    CROSS JOIN generate_series(1, 3) -- Each user gets 3 projects
    RETURNING id, owner_id, created_at
)
INSERT INTO tasks (title, project_id, assigned_to, created_at)
SELECT 
    'Task ' || row_number() OVER (),
    p.id,
    p.owner_id,
    p.created_at + (random() * INTERVAL '60 days')
FROM project_data p
CROSS JOIN generate_series(1, 5); -- Each project gets 5 tasks
```

## 🎭 Realistic Data Patterns

### Names and Demographics

```sql
-- Generate realistic names
CREATE TABLE first_names (name TEXT);
CREATE TABLE last_names (name TEXT);

INSERT INTO first_names VALUES 
('John'), ('Jane'), ('Michael'), ('Sarah'), ('David'), ('Emily'),
('Robert'), ('Lisa'), ('James'), ('Maria'), ('William'), ('Jennifer'),
('Richard'), ('Patricia'), ('Charles'), ('Linda'), ('Joseph'), ('Elizabeth'),
('Thomas'), ('Barbara'), ('Christopher'), ('Susan'), ('Daniel'), ('Jessica'),
('Paul'), ('Margaret'), ('Mark'), ('Dorothy'), ('Donald'), ('Lisa');

INSERT INTO last_names VALUES 
('Smith'), ('Johnson'), ('Williams'), ('Brown'), ('Jones'), ('Garcia'),
('Miller'), ('Davis'), ('Rodriguez'), ('Martinez'), ('Hernandez'), ('Lopez'),
('Gonzalez'), ('Wilson'), ('Anderson'), ('Thomas'), ('Taylor'), ('Moore'),
('Jackson'), ('Martin'), ('Lee'), ('Perez'), ('Thompson'), ('White'),
('Harris'), ('Sanchez'), ('Clark'), ('Ramirez'), ('Lewis'), ('Robinson');

-- Generate users with realistic names
INSERT INTO users (first_name, last_name, email, birth_date)
SELECT 
    fn.name as first_name,
    ln.name as last_name,
    lower(fn.name) || '.' || lower(ln.name) || s || '@example.com' as email,
    CURRENT_DATE - (random() * 365 * 50 + 18 * 365)::int as birth_date
FROM first_names fn
CROSS JOIN last_names ln
CROSS JOIN generate_series(1, 3) s
ORDER BY random()
LIMIT 10000;
```

### Time-Series Data

```sql
-- Generate realistic time-series data
INSERT INTO metrics (metric_name, value, timestamp)
SELECT 
    'cpu_usage' as metric_name,
    -- Generate realistic CPU usage (0-100% with some correlation)
    GREATEST(0, LEAST(100, 
        50 + -- Base level
        20 * sin(extract(epoch from ts) / 3600) + -- Hourly pattern
        10 * sin(extract(epoch from ts) / 86400) + -- Daily pattern
        5 * (random() - 0.5) * 2 -- Random noise
    )) as value,
    ts as timestamp
FROM generate_series(
    CURRENT_TIMESTAMP - INTERVAL '30 days',
    CURRENT_TIMESTAMP,
    INTERVAL '1 minute'
) AS ts;
```

## 🔧 Performance Optimization

### Batch Generation

```sql
-- Generate large datasets efficiently
CREATE OR REPLACE FUNCTION generate_large_dataset(table_size INTEGER)
RETURNS VOID AS $$
DECLARE
    batch_size INTEGER := 10000;
    current_batch INTEGER := 0;
BEGIN
    WHILE current_batch < table_size LOOP
        INSERT INTO large_table (data, category, created_at)
        SELECT 
            md5(random()::text) as data,
            'Category ' || (floor(random() * 10) + 1) as category,
            CURRENT_TIMESTAMP - (random() * INTERVAL '365 days')
        FROM generate_series(1, LEAST(batch_size, table_size - current_batch));
        
        current_batch := current_batch + batch_size;
        
        -- Progress indicator
        IF current_batch % 100000 = 0 THEN
            RAISE NOTICE 'Generated % rows', current_batch;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT generate_large_dataset(1000000);
```

### Optimized Sample Data

```sql
-- Generate sample data with hash for indexing
CREATE TABLE sample_data AS
SELECT 
    s,
    md5(random()::text) as md5,
    hashtext(md5(random()::text)) as hash
FROM generate_series(1, 100000) s;

-- Create indexes on generated data
CREATE INDEX idx_sample_data_hash ON sample_data(hash);
CREATE INDEX idx_sample_data_md5 ON sample_data(md5);
```

## 🛠️ Database-Specific Techniques

### MySQL Implementation

```sql
-- MySQL-specific test data generation
DELIMITER $$

CREATE FUNCTION rand_string(length INT) RETURNS VARCHAR(255)
READS SQL DATA
DETERMINISTIC
BEGIN
    DECLARE chars VARCHAR(62) DEFAULT 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    DECLARE result VARCHAR(255) DEFAULT '';
    DECLARE i INT DEFAULT 0;
    
    WHILE i < length DO
        SET result = CONCAT(result, SUBSTRING(chars, FLOOR(1 + RAND() * 62), 1));
        SET i = i + 1;
    END WHILE;
    
    RETURN result;
END$$

DELIMITER ;

-- Generate test data
INSERT INTO users (name, email, created_at)
SELECT 
    CONCAT('User ', n),
    CONCAT('user', n, '@example.com'),
    DATE_SUB(CURDATE(), INTERVAL FLOOR(RAND() * 365) DAY)
FROM (
    SELECT @row := @row + 1 as n
    FROM information_schema.tables t1
    CROSS JOIN information_schema.tables t2
    CROSS JOIN (SELECT @row := 0) r
) numbers
WHERE n <= 10000;
```

### SQLite Implementation

```sql
-- SQLite test data generation
WITH RECURSIVE counter(n) AS (
    SELECT 1
    UNION ALL
    SELECT n + 1 FROM counter WHERE n < 10000
)
INSERT INTO users (name, email, created_at)
SELECT 
    'User ' || n,
    'user' || n || '@example.com',
    datetime('now', '-' || (abs(random()) % 365) || ' days')
FROM counter;
```

## 📊 Data Validation

### Consistency Checks

```sql
-- Validate generated data consistency
CREATE OR REPLACE FUNCTION validate_test_data()
RETURNS TABLE(check_name TEXT, result BOOLEAN, details TEXT) AS $$
BEGIN
    -- Check referential integrity
    RETURN QUERY
    SELECT 
        'Orphaned Tasks' as check_name,
        COUNT(*) = 0 as result,
        'Found ' || COUNT(*) || ' tasks without valid projects' as details
    FROM tasks t
    LEFT JOIN projects p ON t.project_id = p.id
    WHERE p.id IS NULL;
    
    -- Check data distribution
    RETURN QUERY
    SELECT 
        'User Creation Distribution' as check_name,
        (MAX(created_at) - MIN(created_at)) > INTERVAL '300 days' as result,
        'Date range: ' || MIN(created_at) || ' to ' || MAX(created_at) as details
    FROM users;
END;
$$ LANGUAGE plpgsql;

-- Run validation
SELECT * FROM validate_test_data();
```

### Data Cleanup

```sql
-- Clean up test data
CREATE OR REPLACE FUNCTION cleanup_test_data()
RETURNS VOID AS $$
BEGIN
    -- Remove test data (be careful with this!)
    DELETE FROM tasks WHERE title LIKE 'Task %';
    DELETE FROM projects WHERE name LIKE 'Project %';
    DELETE FROM users WHERE email LIKE 'user%@example.com';
    
    -- Reset sequences
    ALTER SEQUENCE users_id_seq RESTART WITH 1;
    ALTER SEQUENCE projects_id_seq RESTART WITH 1;
    ALTER SEQUENCE tasks_id_seq RESTART WITH 1;
END;
$$ LANGUAGE plpgsql;
```

## 🎯 Best Practices

### Generation Strategy

1. **Start Small** - Generate small datasets first, then scale up
2. **Maintain Relationships** - Ensure foreign keys are valid
3. **Realistic Distributions** - Use weighted random values
4. **Time Correlation** - Make timestamps logically consistent
5. **Performance Consideration** - Generate data in batches

### Edge Cases

```sql
-- Generate edge cases for testing
INSERT INTO edge_case_data (value, category)
SELECT 
    CASE 
        WHEN n % 100 = 0 THEN NULL -- 1% null values
        WHEN n % 50 = 0 THEN '' -- 2% empty strings
        WHEN n % 25 = 0 THEN REPEAT('x', 1000) -- 4% very long strings
        ELSE 'Normal value ' || n
    END as value,
    CASE 
        WHEN n % 10 = 0 THEN 'boundary'
        WHEN n % 5 = 0 THEN 'edge'
        ELSE 'normal'
    END as category
FROM generate_series(1, 10000) n;
```

## 🔗 Related Resources

### External Tools
- **Faker.js** - JavaScript library for generating fake data
- **Python Faker** - Python library for realistic test data
- **Mockaroo** - Online test data generator
- **DBT** - Data transformation tool for test data

### Database Tools
- **pgbench** - PostgreSQL benchmarking tool
- **sysbench** - Multi-threaded benchmark tool
- **PostgREST** - Generate REST APIs for testing

This comprehensive approach to test data generation ensures you have realistic, consistent, and useful data for development, testing, and performance optimization.
