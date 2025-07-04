# Hashing Strategies

Comprehensive guide to hashing data for storage, indexing, and security purposes in database systems.

## 🎯 Overview

Hashing is essential for:
- **Security** - Password storage, API keys, sensitive data
- **Performance** - Fast lookups, efficient indexing
- **Integrity** - Data verification, checksums
- **Uniqueness** - Avoiding duplicates, unique constraints

## 🔐 Password Hashing

### Best Practices

```sql
-- PostgreSQL with pgcrypto extension
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- User table with hashed passwords
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Secure password storage
INSERT INTO users (email, password_hash, salt)
VALUES (
    'user@example.com',
    crypt('user_password', gen_salt('bf', 12)), -- bcrypt with cost 12
    gen_salt('bf', 12)
);

-- Password verification
SELECT 
    id,
    email,
    (password_hash = crypt('user_password', password_hash)) AS password_valid
FROM users 
WHERE email = 'user@example.com';
```

### Hash Algorithm Selection

```sql
-- Different algorithms for different use cases
CREATE TABLE auth_methods (
    method_name VARCHAR(50),
    use_case TEXT,
    example_query TEXT
);

INSERT INTO auth_methods VALUES
('bcrypt', 'Password hashing', 'crypt(password, gen_salt(''bf'', 12))'),
('scrypt', 'High-security passwords', 'crypt(password, gen_salt(''scrypt''))'),
('argon2', 'Modern password hashing', 'Custom implementation needed'),
('PBKDF2', 'Legacy compatibility', 'Custom implementation needed');
```

## 🔍 Data Integrity Hashing

### Checksum Generation

```sql
-- Content integrity verification
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL,
    content BYTEA NOT NULL,
    md5_hash CHAR(32) GENERATED ALWAYS AS (md5(content)) STORED,
    sha256_hash CHAR(64) GENERATED ALWAYS AS (encode(sha256(content), 'hex')) STORED,
    size_bytes BIGINT GENERATED ALWAYS AS (length(content)) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Verify document integrity
SELECT 
    filename,
    (md5_hash = md5(content)) AS md5_valid,
    (sha256_hash = encode(sha256(content), 'hex')) AS sha256_valid
FROM documents
WHERE id = 'document-uuid';
```

### Hash-Based Deduplication

```sql
-- Efficient file deduplication
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_filename VARCHAR(255) NOT NULL,
    content_hash CHAR(64) NOT NULL, -- SHA-256
    file_size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint on content hash for deduplication
    UNIQUE(content_hash)
);

-- Reference table for file usage
CREATE TABLE file_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID REFERENCES files(id),
    entity_type VARCHAR(50) NOT NULL, -- 'user_avatar', 'document', etc.
    entity_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🚀 Performance Hashing

### Long String Indexing

```sql
-- Problem: Long URLs are expensive to index
CREATE TABLE urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_url TEXT NOT NULL,
    -- Hash for efficient indexing and uniqueness
    url_hash CHAR(64) GENERATED ALWAYS AS (encode(sha256(original_url::bytea), 'hex')) STORED,
    click_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Index on hash instead of full URL
    UNIQUE(url_hash)
);

-- Efficient lookup by hash
SELECT original_url, click_count
FROM urls
WHERE url_hash = encode(sha256('https://example.com/very/long/url'::bytea), 'hex');
```

### Hash-Based Partitioning

```sql
-- Distribute data across partitions using hash
CREATE TABLE user_activity (
    user_id UUID NOT NULL,
    activity_type VARCHAR(50) NOT NULL,
    activity_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Hash for consistent partitioning
    partition_hash INTEGER GENERATED ALWAYS AS (abs(hashtext(user_id::text)) % 16) STORED
) PARTITION BY HASH (partition_hash);

-- Create partitions
CREATE TABLE user_activity_0 PARTITION OF user_activity FOR VALUES WITH (MODULUS 16, REMAINDER 0);
CREATE TABLE user_activity_1 PARTITION OF user_activity FOR VALUES WITH (MODULUS 16, REMAINDER 1);
-- ... create remaining partitions
```

## 🔧 Database-Specific Implementations

### MySQL Hashing

```sql
-- MySQL hash functions
CREATE TABLE mysql_hashes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    data TEXT NOT NULL,
    md5_hash CHAR(32) GENERATED ALWAYS AS (MD5(data)) STORED,
    sha1_hash CHAR(40) GENERATED ALWAYS AS (SHA1(data)) STORED,
    sha2_hash CHAR(64) GENERATED ALWAYS AS (SHA2(data, 256)) STORED,
    crc32_hash INT GENERATED ALWAYS AS (CRC32(data)) STORED
);

-- Unique constraint on hash to prevent duplicates
ALTER TABLE mysql_hashes ADD UNIQUE KEY unique_content (sha2_hash);

-- Insert with automatic hash generation
INSERT INTO mysql_hashes (data) VALUES ('Sample content');
```

### PostgreSQL Hashing

```sql
-- PostgreSQL crypto functions
CREATE TABLE postgres_hashes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data TEXT NOT NULL,
    md5_hash CHAR(32) GENERATED ALWAYS AS (md5(data)) STORED,
    sha256_hash CHAR(64) GENERATED ALWAYS AS (encode(digest(data, 'sha256'), 'hex')) STORED,
    blake2b_hash CHAR(128) GENERATED ALWAYS AS (encode(digest(data, 'blake2b'), 'hex')) STORED,
    -- Hash for integer lookups
    hash_int INTEGER GENERATED ALWAYS AS (hashtext(data)) STORED
);

-- Create index on hash for fast lookups
CREATE INDEX idx_postgres_hashes_sha256 ON postgres_hashes (sha256_hash);
```

## 🛡️ Security Considerations

### Salt Management

```sql
-- Proper salt storage and usage
CREATE TABLE secure_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    hash_algorithm VARCHAR(20) NOT NULL DEFAULT 'bcrypt',
    hash_iterations INTEGER NOT NULL DEFAULT 12,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to create secure password hash
CREATE OR REPLACE FUNCTION create_password_hash(password TEXT, algorithm TEXT DEFAULT 'bcrypt')
RETURNS TABLE(hash TEXT, salt TEXT) AS $$
BEGIN
    CASE algorithm
        WHEN 'bcrypt' THEN
            salt := gen_salt('bf', 12);
            hash := crypt(password, salt);
        WHEN 'scrypt' THEN
            salt := gen_salt('scrypt');
            hash := crypt(password, salt);
        ELSE
            RAISE EXCEPTION 'Unsupported hash algorithm: %', algorithm;
    END CASE;
    
    RETURN QUERY SELECT hash, salt;
END;
$$ LANGUAGE plpgsql;
```

### Hash Timing Attack Prevention

```sql
-- Constant-time comparison function
CREATE OR REPLACE FUNCTION secure_compare(a TEXT, b TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    result BOOLEAN := TRUE;
    i INTEGER;
BEGIN
    -- Ensure both strings are same length to prevent timing attacks
    IF length(a) != length(b) THEN
        RETURN FALSE;
    END IF;
    
    -- Compare each character
    FOR i IN 1..length(a) LOOP
        IF substring(a, i, 1) != substring(b, i, 1) THEN
            result := FALSE;
        END IF;
    END LOOP;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Performance Optimization

### Hash Index Types

```sql
-- Different index strategies for hashes
CREATE TABLE hash_performance (
    id UUID PRIMARY KEY,
    content TEXT NOT NULL,
    md5_hash CHAR(32) NOT NULL,
    sha256_hash CHAR(64) NOT NULL
);

-- B-tree index for exact matches (default)
CREATE INDEX idx_btree_md5 ON hash_performance (md5_hash);

-- Hash index for exact equality (PostgreSQL)
CREATE INDEX idx_hash_md5 ON hash_performance USING HASH (md5_hash);

-- Partial index for active records
CREATE INDEX idx_active_hashes ON hash_performance (sha256_hash) 
WHERE created_at > CURRENT_DATE - INTERVAL '30 days';
```

### Hash Distribution Analysis

```sql
-- Analyze hash distribution quality
WITH hash_stats AS (
    SELECT 
        substring(md5_hash, 1, 2) as hash_prefix,
        COUNT(*) as count
    FROM hash_performance
    GROUP BY substring(md5_hash, 1, 2)
)
SELECT 
    hash_prefix,
    count,
    count * 100.0 / SUM(count) OVER () as percentage
FROM hash_stats
ORDER BY count DESC;
```

## 🎯 Best Practices

### Hash Selection Guidelines

1. **Password Storage**
   - Use bcrypt, scrypt, or Argon2
   - Never use MD5 or SHA-1 for passwords
   - Include proper salt generation

2. **Data Integrity**
   - Use SHA-256 or SHA-3 for file checksums
   - Include file size for additional verification
   - Store hashes as generated columns when possible

3. **Performance Optimization**
   - Use hash indexes for exact matches
   - Consider hash-based partitioning for large tables
   - Monitor hash distribution quality

4. **Security**
   - Always use salt for password hashing
   - Implement constant-time comparison
   - Regularly update hashing algorithms

### Common Anti-Patterns

```sql
-- ❌ Bad: Storing plain text passwords
CREATE TABLE bad_users (
    id INT PRIMARY KEY,
    password TEXT -- Never do this!
);

-- ❌ Bad: Using MD5 for passwords
CREATE TABLE bad_auth (
    id INT PRIMARY KEY,
    password_md5 CHAR(32) -- MD5 is not secure for passwords
);

-- ❌ Bad: No salt for password hashing
CREATE TABLE bad_security (
    id INT PRIMARY KEY,
    password_hash TEXT -- Without salt, vulnerable to rainbow tables
);

-- ✅ Good: Secure password storage
CREATE TABLE good_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    password_hash TEXT NOT NULL, -- bcrypt/scrypt hash
    salt TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🔄 Migration Strategies

### Upgrading Hash Algorithms

```sql
-- Gradual migration from old to new hash algorithm
ALTER TABLE users 
ADD COLUMN new_password_hash TEXT,
ADD COLUMN hash_algorithm VARCHAR(20) DEFAULT 'md5';

-- During login, upgrade hash if using old algorithm
CREATE OR REPLACE FUNCTION upgrade_password_hash(
    user_id UUID,
    plain_password TEXT,
    current_hash TEXT
) RETURNS BOOLEAN AS $$
BEGIN
    -- Verify current password
    IF current_hash != md5(plain_password) THEN
        RETURN FALSE;
    END IF;
    
    -- Update to new secure hash
    UPDATE users 
    SET 
        new_password_hash = crypt(plain_password, gen_salt('bf', 12)),
        hash_algorithm = 'bcrypt'
    WHERE id = user_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

This comprehensive guide covers all aspects of hashing in database systems, from basic concepts to advanced security implementations.
