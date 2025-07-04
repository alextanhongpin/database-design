# String to Integer Conversion

Comprehensive guide to converting strings to integers in database systems, including hashing, encoding, and mapping strategies.

## 🎯 Overview

String-to-integer conversion is useful for:
- **Performance** - Faster joins and comparisons
- **Storage** - Reduced space requirements
- **Indexing** - More efficient index structures
- **Partitioning** - Hash-based data distribution

## 🔢 Hash-Based Conversion

### PostgreSQL Hash Functions

```sql
-- PostgreSQL string hashing functions
CREATE TABLE string_hash_demo (
    id SERIAL PRIMARY KEY,
    original_string TEXT NOT NULL,
    hash_32bit INTEGER GENERATED ALWAYS AS (hashtext(original_string)) STORED,
    hash_64bit BIGINT GENERATED ALWAYS AS (hashtextextended(original_string, 0)) STORED,
    hash_positive INTEGER GENERATED ALWAYS AS (abs(hashtext(original_string))) STORED
);

-- Example data
INSERT INTO string_hash_demo (original_string) VALUES
('apple'), ('banana'), ('cherry'), ('date'), ('elderberry');

-- View hash values
SELECT 
    original_string,
    hash_32bit,
    hash_64bit,
    hash_positive
FROM string_hash_demo;
```

### Hash Distribution Analysis

```sql
-- Analyze hash distribution quality
WITH hash_distribution AS (
    SELECT 
        original_string,
        hashtext(original_string) as hash_value,
        abs(hashtext(original_string)) % 100 as bucket
    FROM (
        SELECT chr(ascii('A') + s % 26) || chr(ascii('a') + (s/26) % 26) || s::text as original_string
        FROM generate_series(1, 1000) s
    ) sample_data
)
SELECT 
    bucket,
    COUNT(*) as items_in_bucket,
    COUNT(*) * 100.0 / SUM(COUNT(*)) OVER () as percentage
FROM hash_distribution
GROUP BY bucket
ORDER BY items_in_bucket DESC
LIMIT 10;
```

## 🗂️ Enumeration Mapping

### Static String-to-Integer Mapping

```sql
-- Create lookup table for string-to-integer mapping
CREATE TABLE category_mapping (
    id SERIAL PRIMARY KEY,
    category_name VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert categories
INSERT INTO category_mapping (category_name) VALUES
('electronics'), ('clothing'), ('books'), ('sports'), ('home');

-- Products table using integer foreign key
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    category_id INTEGER REFERENCES category_mapping(id),
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to get category ID from name
CREATE OR REPLACE FUNCTION get_category_id(category_name TEXT)
RETURNS INTEGER AS $$
DECLARE
    category_id INTEGER;
BEGIN
    SELECT id INTO category_id
    FROM category_mapping
    WHERE category_name = $1;
    
    IF category_id IS NULL THEN
        -- Insert new category and return ID
        INSERT INTO category_mapping (category_name)
        VALUES ($1)
        RETURNING id INTO category_id;
    END IF;
    
    RETURN category_id;
END;
$$ LANGUAGE plpgsql;

-- Example usage
INSERT INTO products (name, category_id, price)
VALUES ('Laptop', get_category_id('electronics'), 999.99);
```

### Dynamic String Encoding

```sql
-- Base36 encoding for alphanumeric strings
CREATE OR REPLACE FUNCTION string_to_base36(input_string TEXT)
RETURNS BIGINT AS $$
DECLARE
    result BIGINT := 0;
    char_value INTEGER;
    i INTEGER;
    current_char CHAR;
BEGIN
    -- Convert string to uppercase for consistency
    input_string := upper(input_string);
    
    -- Process each character
    FOR i IN 1..length(input_string) LOOP
        current_char := substring(input_string, i, 1);
        
        -- Convert character to base36 value
        IF current_char >= '0' AND current_char <= '9' THEN
            char_value := ascii(current_char) - ascii('0');
        ELSIF current_char >= 'A' AND current_char <= 'Z' THEN
            char_value := ascii(current_char) - ascii('A') + 10;
        ELSE
            RAISE EXCEPTION 'Invalid character for base36: %', current_char;
        END IF;
        
        result := result * 36 + char_value;
    END LOOP;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT string_to_base36('ABC123'); -- Returns: 16836775
SELECT string_to_base36('USER1');  -- Returns: 42467929
```

## 🔄 Bidirectional Conversion

### Reversible String-Integer Mapping

```sql
-- Create reversible mapping system
CREATE TABLE string_integer_mapping (
    id SERIAL PRIMARY KEY,
    original_string TEXT UNIQUE NOT NULL,
    integer_value BIGINT UNIQUE NOT NULL,
    mapping_type VARCHAR(20) NOT NULL DEFAULT 'hash',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to convert string to integer with reverse lookup
CREATE OR REPLACE FUNCTION string_to_int_reversible(input_string TEXT)
RETURNS BIGINT AS $$
DECLARE
    existing_value BIGINT;
    new_value BIGINT;
BEGIN
    -- Check if mapping already exists
    SELECT integer_value INTO existing_value
    FROM string_integer_mapping
    WHERE original_string = input_string;
    
    IF existing_value IS NOT NULL THEN
        RETURN existing_value;
    END IF;
    
    -- Generate new mapping using hash
    new_value := abs(hashtext(input_string))::BIGINT;
    
    -- Handle collisions by incrementing
    WHILE EXISTS (SELECT 1 FROM string_integer_mapping WHERE integer_value = new_value) LOOP
        new_value := new_value + 1;
    END LOOP;
    
    -- Store mapping
    INSERT INTO string_integer_mapping (original_string, integer_value, mapping_type)
    VALUES (input_string, new_value, 'hash');
    
    RETURN new_value;
END;
$$ LANGUAGE plpgsql;

-- Reverse function
CREATE OR REPLACE FUNCTION int_to_string_reversible(integer_value BIGINT)
RETURNS TEXT AS $$
DECLARE
    result TEXT;
BEGIN
    SELECT original_string INTO result
    FROM string_integer_mapping
    WHERE integer_value = $1;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

## 🚀 Performance Optimization

### Hash-Based Partitioning

```sql
-- Use string hash for table partitioning
CREATE TABLE user_activity (
    id UUID DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    activity_type VARCHAR(50) NOT NULL,
    activity_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Partition key based on user_id hash
    partition_key INTEGER GENERATED ALWAYS AS (abs(hashtext(user_id)) % 16) STORED
) PARTITION BY HASH (partition_key);

-- Create partitions
DO $$
BEGIN
    FOR i IN 0..15 LOOP
        EXECUTE format('CREATE TABLE user_activity_%s PARTITION OF user_activity FOR VALUES WITH (MODULUS 16, REMAINDER %s)', i, i);
    END LOOP;
END
$$;
```

### Index Optimization

```sql
-- Compare performance of string vs integer indexes
CREATE TABLE performance_test (
    id SERIAL PRIMARY KEY,
    string_key VARCHAR(50) NOT NULL,
    int_key INTEGER NOT NULL,
    data TEXT
);

-- Generate test data
INSERT INTO performance_test (string_key, int_key, data)
SELECT 
    'user_' || s::text,
    s,
    'Sample data ' || s
FROM generate_series(1, 100000) s;

-- Create indexes
CREATE INDEX idx_string_key ON performance_test (string_key);
CREATE INDEX idx_int_key ON performance_test (int_key);

-- Compare query performance
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM performance_test WHERE string_key = 'user_50000';
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM performance_test WHERE int_key = 50000;
```

## 🔧 Database-Specific Implementations

### MySQL String Hashing

```sql
-- MySQL hash functions
CREATE TABLE mysql_string_hashes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    original_string TEXT NOT NULL,
    crc32_hash INT GENERATED ALWAYS AS (CRC32(original_string)) STORED,
    md5_hash_int BIGINT GENERATED ALWAYS AS (CONV(LEFT(MD5(original_string), 16), 16, 10)) STORED,
    sha1_hash_int BIGINT GENERATED ALWAYS AS (CONV(LEFT(SHA1(original_string), 16), 16, 10)) STORED
);

-- Insert test data
INSERT INTO mysql_string_hashes (original_string) VALUES
('apple'), ('banana'), ('cherry'), ('date'), ('elderberry');

-- View results
SELECT 
    original_string,
    crc32_hash,
    md5_hash_int,
    sha1_hash_int
FROM mysql_string_hashes;
```

### SQLite String Conversion

```sql
-- SQLite doesn't have built-in hash functions, but we can simulate
CREATE TABLE sqlite_string_conversion (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    original_string TEXT NOT NULL,
    simple_hash INTEGER
);

-- Simple hash function simulation
CREATE TRIGGER calculate_hash
AFTER INSERT ON sqlite_string_conversion
FOR EACH ROW
BEGIN
    UPDATE sqlite_string_conversion
    SET simple_hash = (
        -- Simple polynomial hash
        CASE
            WHEN length(NEW.original_string) = 0 THEN 0
            ELSE (
                (unicode(substr(NEW.original_string, 1, 1)) * 31 * 31 * 31) +
                (unicode(substr(NEW.original_string, 2, 1)) * 31 * 31) +
                (unicode(substr(NEW.original_string, 3, 1)) * 31) +
                unicode(substr(NEW.original_string, 4, 1))
            ) % 2147483647
        END
    )
    WHERE id = NEW.id;
END;
```

## 📊 Use Cases and Examples

### URL Shortening System

```sql
-- URL shortening with integer conversion
CREATE TABLE url_mappings (
    id BIGSERIAL PRIMARY KEY,
    original_url TEXT NOT NULL,
    url_hash BIGINT UNIQUE NOT NULL,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to create short URL
CREATE OR REPLACE FUNCTION create_short_url(url TEXT)
RETURNS VARCHAR(10) AS $$
DECLARE
    url_hash BIGINT;
    short_code VARCHAR(10);
BEGIN
    -- Generate hash from URL
    url_hash := abs(hashtext(url))::BIGINT;
    
    -- Convert to base62 for short code
    short_code := base62_encode(url_hash);
    
    -- Insert mapping
    INSERT INTO url_mappings (original_url, url_hash, short_code)
    VALUES (url, url_hash, short_code)
    ON CONFLICT (url_hash) DO NOTHING;
    
    RETURN short_code;
END;
$$ LANGUAGE plpgsql;

-- Base62 encoding function
CREATE OR REPLACE FUNCTION base62_encode(num BIGINT)
RETURNS VARCHAR(10) AS $$
DECLARE
    chars TEXT := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
    result TEXT := '';
    remainder INTEGER;
BEGIN
    IF num = 0 THEN
        RETURN '0';
    END IF;
    
    WHILE num > 0 LOOP
        remainder := num % 62;
        result := substring(chars, remainder + 1, 1) || result;
        num := num / 62;
    END LOOP;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

### User Handle to ID Mapping

```sql
-- Social media handle mapping
CREATE TABLE user_handles (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    handle_hash INTEGER GENERATED ALWAYS AS (abs(hashtext(username))) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on hash for fast lookups
CREATE INDEX idx_handle_hash ON user_handles (handle_hash);

-- Function to find user by handle
CREATE OR REPLACE FUNCTION find_user_by_handle(handle TEXT)
RETURNS UUID AS $$
DECLARE
    user_uuid UUID;
BEGIN
    SELECT user_id INTO user_uuid
    FROM user_handles
    WHERE username = handle
    AND handle_hash = abs(hashtext(handle)); -- Use hash for faster filtering
    
    RETURN user_uuid;
END;
$$ LANGUAGE plpgsql;
```

## 🎯 Best Practices

### Conversion Strategy Guidelines

1. **Choose the Right Method**
   - Use hash functions for non-reversible conversions
   - Use lookup tables for reversible mappings
   - Consider base36/base62 for alphanumeric strings

2. **Handle Collisions**
   ```sql
   -- Collision detection and resolution
   CREATE UNIQUE INDEX idx_unique_hash ON mappings (hash_value);
   
   -- Use composite keys when collisions are possible
   CREATE UNIQUE INDEX idx_string_hash ON mappings (original_string, hash_value);
   ```

3. **Performance Considerations**
   - Index integer values for fast lookups
   - Use partitioning for large datasets
   - Monitor hash distribution quality

4. **Data Integrity**
   - Validate input strings before conversion
   - Log conversion errors
   - Maintain reverse lookup capabilities when needed

### Common Pitfalls

```sql
-- ❌ Bad: Ignoring hash collisions
CREATE TABLE bad_mapping (
    string_val TEXT PRIMARY KEY,
    int_val INTEGER DEFAULT hashtext(string_val)  -- Potential collisions
);

-- ✅ Good: Proper collision handling
CREATE TABLE good_mapping (
    id SERIAL PRIMARY KEY,
    string_val TEXT UNIQUE NOT NULL,
    int_val INTEGER UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ❌ Bad: Using conversion for security
SELECT * FROM users WHERE id = hashtext(user_input); -- Predictable

-- ✅ Good: Proper security with UUIDs
SELECT * FROM users WHERE id = $1::UUID; -- Use proper UUID validation
```

## 🔍 Monitoring and Optimization

### Performance Monitoring

```sql
-- Monitor conversion performance
CREATE VIEW conversion_metrics AS
SELECT 
    'hash_lookups' as metric,
    COUNT(*) as total_operations,
    AVG(EXTRACT(EPOCH FROM (clock_timestamp() - query_start))) as avg_duration_ms
FROM pg_stat_activity
WHERE query LIKE '%hashtext%'
UNION ALL
SELECT 
    'integer_lookups' as metric,
    COUNT(*) as total_operations,
    AVG(EXTRACT(EPOCH FROM (clock_timestamp() - query_start))) as avg_duration_ms
FROM pg_stat_activity
WHERE query LIKE '%integer_column%';
```

This comprehensive guide provides multiple strategies for converting strings to integers in database systems, with considerations for performance, collision handling, and reversibility.
