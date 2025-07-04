# MD5 Hashing in Database Design

Using MD5 hashing for data integrity, uniqueness constraints, and performance optimization in database applications.

## ⚠️ Important Security Notice

**MD5 is not secure for cryptographic purposes** and should not be used for:
- Password storage
- Security tokens
- Cryptographic signatures
- Any security-sensitive applications

Use MD5 only for:
- Data integrity checks (non-security)
- Unique constraint optimization
- Performance improvements
- Deduplication (when security isn't a concern)

## 🎯 Use Cases for MD5

### Data Integrity Checking

```sql
-- PostgreSQL: Store MD5 hash for data integrity
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    content_md5 CHAR(32) GENERATED ALWAYS AS (md5(content)) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Verify data integrity
SELECT 
    filename,
    (content_md5 = md5(content)) as integrity_check
FROM documents
WHERE id = 123;
```

### Unique Constraint Optimization

```sql
-- Problem: Long strings are expensive to index
CREATE TABLE urls (
    id SERIAL PRIMARY KEY,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Solution: Use MD5 hash for uniqueness
CREATE TABLE urls_optimized (
    id SERIAL PRIMARY KEY,
    original_url TEXT NOT NULL,
    url_hash CHAR(32) GENERATED ALWAYS AS (md5(original_url)) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Index on hash instead of full URL
    UNIQUE(url_hash)
);

-- Benefits:
-- 1. Fixed 32-character index vs variable length URL
-- 2. Faster uniqueness checks
-- 3. Reduced storage for index
```

### MySQL Binary Storage

```sql
-- MySQL: Store MD5 as binary for space efficiency
CREATE TABLE content_hashes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    content LONGTEXT NOT NULL,
    -- Store as binary(16) instead of char(32)
    content_hash BINARY(16) GENERATED ALWAYS AS (UNHEX(MD5(content))) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY idx_content_hash (content_hash)
);

-- Insert and query
INSERT INTO content_hashes (content) VALUES ('Sample content');

-- Query using MD5
SELECT * FROM content_hashes 
WHERE content_hash = UNHEX(MD5('Sample content'));
```

## 🚀 Performance Optimization

### Index Efficiency

```sql
-- Compare index performance: full text vs MD5 hash
-- Large table with URLs
CREATE TABLE url_performance_test (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    url_md5 CHAR(32) NOT NULL
);

-- Insert test data
INSERT INTO url_performance_test (url, url_md5)
SELECT 
    'https://example.com/page/' || s || '/item/' || s,
    md5('https://example.com/page/' || s || '/item/' || s)
FROM generate_series(1, 1000000) s;

-- Create indexes
CREATE INDEX idx_url_full ON url_performance_test (url);
CREATE INDEX idx_url_md5 ON url_performance_test (url_md5);

-- Compare query performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM url_performance_test 
WHERE url = 'https://example.com/page/50000/item/50000';

EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM url_performance_test 
WHERE url_md5 = md5('https://example.com/page/50000/item/50000');
```

### Deduplication

```sql
-- Use MD5 for efficient deduplication
CREATE TABLE file_storage (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    content BYTEA NOT NULL,
    content_md5 CHAR(32) GENERATED ALWAYS AS (md5(content)) STORED,
    file_size INTEGER GENERATED ALWAYS AS (length(content)) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint prevents duplicate content
    UNIQUE(content_md5)
);

-- Reference table for file usage
CREATE TABLE file_references (
    id SERIAL PRIMARY KEY,
    file_id INTEGER REFERENCES file_storage(id),
    entity_type VARCHAR(50) NOT NULL,
    entity_id INTEGER NOT NULL,
    reference_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert with automatic deduplication
INSERT INTO file_storage (filename, content)
VALUES ('document.txt', 'File content here')
ON CONFLICT (content_md5) DO NOTHING;
```

## 🔍 MySQL Specific Optimizations

### Prefix Index Limitations

```sql
-- MySQL problem: Prefix indexes on long strings
CREATE TABLE articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content LONGTEXT NOT NULL,
    -- MySQL unique index only compares first 767 bytes by default
    -- This could lead to false duplicates
    UNIQUE KEY idx_title (title(100))  -- Only first 100 characters
);

-- Solution: Use MD5 hash for full uniqueness
CREATE TABLE articles_optimized (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content LONGTEXT NOT NULL,
    title_hash CHAR(32) GENERATED ALWAYS AS (MD5(title)) STORED,
    content_hash CHAR(32) GENERATED ALWAYS AS (MD5(content)) STORED,
    
    -- Full uniqueness guaranteed
    UNIQUE KEY idx_title_hash (title_hash),
    UNIQUE KEY idx_content_hash (content_hash)
);
```

### Binary Storage Benefits

```sql
-- Storage comparison: CHAR(32) vs BINARY(16)
CREATE TABLE hash_comparison (
    id INT AUTO_INCREMENT PRIMARY KEY,
    data TEXT NOT NULL,
    -- 32 bytes storage
    hash_char CHAR(32) GENERATED ALWAYS AS (MD5(data)) STORED,
    -- 16 bytes storage (50% reduction)
    hash_binary BINARY(16) GENERATED ALWAYS AS (UNHEX(MD5(data))) STORED
);

-- Query using binary hash
SELECT * FROM hash_comparison 
WHERE hash_binary = UNHEX(MD5('search value'));

-- Convert binary back to hex for display
SELECT 
    data,
    HEX(hash_binary) as hash_hex
FROM hash_comparison;
```

## 📊 Collision Handling

### Understanding MD5 Collisions

```sql
-- MD5 collision detection (extremely rare but possible)
CREATE TABLE collision_detection (
    id SERIAL PRIMARY KEY,
    original_data TEXT NOT NULL,
    md5_hash CHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to detect and handle collisions
CREATE OR REPLACE FUNCTION safe_md5_insert(data TEXT)
RETURNS INTEGER AS $$
DECLARE
    hash_value CHAR(32);
    existing_data TEXT;
    result_id INTEGER;
BEGIN
    hash_value := md5(data);
    
    -- Check for existing hash
    SELECT original_data INTO existing_data
    FROM collision_detection
    WHERE md5_hash = hash_value;
    
    IF existing_data IS NOT NULL THEN
        -- Collision detected
        IF existing_data = data THEN
            -- Same data, return existing ID
            SELECT id INTO result_id
            FROM collision_detection
            WHERE md5_hash = hash_value;
            RETURN result_id;
        ELSE
            -- True collision! Log and handle
            RAISE EXCEPTION 'MD5 collision detected: % vs %', existing_data, data;
        END IF;
    END IF;
    
    -- Insert new record
    INSERT INTO collision_detection (original_data, md5_hash)
    VALUES (data, hash_value)
    RETURNING id INTO result_id;
    
    RETURN result_id;
END;
$$ LANGUAGE plpgsql;
```

## 🛡️ Security Considerations

### When NOT to Use MD5

```sql
-- ❌ NEVER use MD5 for passwords
CREATE TABLE users_bad (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password_md5 CHAR(32) NOT NULL  -- NEVER DO THIS!
);

-- ✅ Use proper password hashing
CREATE TABLE users_good (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password_hash TEXT NOT NULL  -- Use bcrypt, scrypt, or argon2
);
```

### Safe MD5 Usage

```sql
-- ✅ Safe: Data integrity checking
CREATE TABLE backup_verification (
    id SERIAL PRIMARY KEY,
    backup_filename VARCHAR(255) NOT NULL,
    backup_size BIGINT NOT NULL,
    backup_md5 CHAR(32) NOT NULL,  -- For integrity, not security
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ✅ Safe: Deduplication
CREATE TABLE image_cache (
    id SERIAL PRIMARY KEY,
    image_data BYTEA NOT NULL,
    image_md5 CHAR(32) GENERATED ALWAYS AS (md5(image_data)) STORED,
    UNIQUE(image_md5)
);
```

## 🎯 Best Practices

### Implementation Guidelines

1. **Use for Performance, Not Security**
   - Index optimization
   - Deduplication
   - Data integrity checks

2. **Handle Collisions Gracefully**
   - Always verify original data matches
   - Log collision attempts
   - Have fallback strategies

3. **Storage Optimization**
   - Use BINARY(16) in MySQL for space efficiency
   - Use generated columns for automatic calculation
   - Index the hash, not the original data

4. **Documentation**
   - Clearly document why MD5 is used
   - Document collision handling strategy
   - Include security disclaimers

### Common Patterns

```sql
-- Pattern 1: Unique constraint with fallback
CREATE TABLE content_dedup (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    content_md5 CHAR(32) GENERATED ALWAYS AS (md5(content)) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(content_md5)
);

-- Pattern 2: Composite uniqueness
CREATE TABLE user_uploads (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content BYTEA NOT NULL,
    content_md5 CHAR(32) GENERATED ALWAYS AS (md5(content)) STORED,
    
    -- Unique per user, allowing same file across users
    UNIQUE(user_id, content_md5)
);

-- Pattern 3: Performance optimization
CREATE TABLE search_cache (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    query_md5 CHAR(32) GENERATED ALWAYS AS (md5(query_text)) STORED,
    results JSONB NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    
    INDEX(query_md5, expires_at)
);
```

## 📈 Performance Monitoring

### Hash Distribution Analysis

```sql
-- Analyze hash distribution quality
WITH hash_stats AS (
    SELECT 
        substring(content_md5, 1, 2) as hash_prefix,
        COUNT(*) as count
    FROM content_dedup
    GROUP BY substring(content_md5, 1, 2)
)
SELECT 
    hash_prefix,
    count,
    count * 100.0 / SUM(count) OVER () as percentage
FROM hash_stats
ORDER BY count DESC;

-- Monitor for potential collisions
SELECT 
    content_md5,
    COUNT(*) as collision_count,
    array_agg(DISTINCT content) as different_contents
FROM content_dedup
GROUP BY content_md5
HAVING COUNT(*) > 1;
```

## 🔄 Migration from MD5

### Upgrading to Stronger Hashes

```sql
-- Migrate from MD5 to SHA-256 for better collision resistance
ALTER TABLE content_hashes 
ADD COLUMN content_sha256 CHAR(64);

-- Populate new column
UPDATE content_hashes 
SET content_sha256 = encode(digest(content, 'sha256'), 'hex');

-- Create new index
CREATE INDEX idx_content_sha256 ON content_hashes(content_sha256);

-- Gradually migrate applications to use SHA-256
-- Then drop MD5 column
-- ALTER TABLE content_hashes DROP COLUMN content_md5;
```

MD5 can be a useful tool for specific database optimization scenarios, but it should never be used for security purposes. Always consider the trade-offs between performance benefits and potential collision risks.
