# String Data Types: Complete Guide

String handling in databases involves choosing appropriate types, managing encoding, optimizing storage, and implementing proper validation. This guide covers comprehensive string data type strategies.

## Table of Contents
- [String Type Selection](#string-type-selection)
- [Character Encoding](#character-encoding)
- [Text Processing and Validation](#text-processing-and-validation)
- [Search and Pattern Matching](#search-and-pattern-matching)
- [Performance Optimization](#performance-optimization)
- [Internationalization](#internationalization)
- [Best Practices](#best-practices)

## String Type Selection

### PostgreSQL String Types

```sql
-- Variable length with limit
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    
    -- Short strings with known limits
    email VARCHAR(255) NOT NULL,
    username VARCHAR(50) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    
    -- Medium text
    bio TEXT, -- No length limit, efficient for any size
    
    -- Large text content
    description TEXT,
    notes TEXT,
    
    -- Fixed length (rare use cases)
    country_code CHAR(2), -- ISO country codes
    currency_code CHAR(3), -- ISO currency codes
    
    -- Character data with specific constraints
    phone_number VARCHAR(20) CHECK (phone_number ~ '^\+?[\d\s\-\(\)]+$'),
    postal_code VARCHAR(10)
);

-- Performance comparison: VARCHAR vs TEXT
-- In PostgreSQL, VARCHAR and TEXT are essentially the same internally
-- TEXT is often preferred for its simplicity
```

### MySQL String Types

```sql
-- MySQL has more string type distinctions
CREATE TABLE content (
    id INT AUTO_INCREMENT PRIMARY KEY,
    
    -- Fixed length - right-padded with spaces
    code CHAR(10),
    
    -- Variable length string (0-255 characters)
    title VARCHAR(255) NOT NULL,
    
    -- Small text (up to 255 characters)
    summary TINYTEXT,
    
    -- Medium text (up to 65,535 characters)
    description TEXT,
    
    -- Large text (up to 16,777,215 characters)
    content MEDIUMTEXT,
    
    -- Very large text (up to 4,294,967,295 characters)
    full_content LONGTEXT,
    
    -- Binary strings
    image_data BLOB,
    file_data LONGBLOB
);

-- Storage requirements matter more in MySQL
-- Choose appropriate size to optimize storage
```

### String Length Constraints

```sql
-- Practical length constraints based on use cases
CREATE TABLE practical_strings (
    id SERIAL PRIMARY KEY,
    
    -- Short identifiers
    username VARCHAR(30) CHECK (LENGTH(username) >= 3),
    email VARCHAR(320), -- RFC 5321 maximum email length
    
    -- Names (accommodate international names)
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    display_name VARCHAR(200),
    
    -- Descriptions
    short_description VARCHAR(500),   -- Tweet-like
    medium_description VARCHAR(2000), -- Short article summary
    
    -- URLs and paths
    website_url VARCHAR(2048), -- Maximum practical URL length
    file_path VARCHAR(4096),   -- File system path
    
    -- Addresses
    street_address VARCHAR(200),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    
    -- Content
    title VARCHAR(300),        -- Article/product titles
    slug VARCHAR(100),         -- URL-friendly identifiers
    meta_description VARCHAR(160), -- SEO meta descriptions
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Character Encoding

### UTF-8 Configuration

```sql
-- PostgreSQL: UTF-8 is default and recommended
CREATE DATABASE app_db 
WITH ENCODING 'UTF8' 
LC_COLLATE = 'en_US.UTF-8' 
LC_CTYPE = 'en_US.UTF-8';

-- MySQL: Ensure UTF-8 encoding
CREATE DATABASE app_db 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;

-- Table with explicit encoding
CREATE TABLE multilingual_content (
    id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    language_code CHAR(2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Collation Examples

```sql
-- Case-sensitive vs case-insensitive comparisons
SELECT 'Hello' = 'hello' COLLATE utf8mb4_bin;        -- 0 (false)
SELECT 'Hello' = 'hello' COLLATE utf8mb4_unicode_ci; -- 1 (true)

-- Accent-sensitive comparisons
SELECT 'café' = 'cafe' COLLATE utf8mb4_unicode_ci;   -- 0 (false)
SELECT 'café' = 'cafe' COLLATE utf8mb4_general_ci;   -- 1 (true)

-- Natural sorting
CREATE TABLE items (
    name VARCHAR(100) COLLATE utf8mb4_unicode_ci
);

INSERT INTO items VALUES ('item1'), ('item10'), ('item2'), ('item20');

-- Natural order vs lexicographic order
SELECT name FROM items ORDER BY name; -- item1, item10, item2, item20
SELECT name FROM items ORDER BY CAST(REGEXP_SUBSTR(name, '[0-9]+') AS UNSIGNED), name;
-- item1, item2, item10, item20
```

## Text Processing and Validation

### String Cleaning and Normalization

```sql
-- Text normalization functions
CREATE OR REPLACE FUNCTION normalize_text(input_text TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Trim whitespace, normalize to single spaces, remove control characters
    RETURN REGEXP_REPLACE(
        REGEXP_REPLACE(
            TRIM(input_text), 
            '\s+', ' ', 'g'  -- Multiple spaces to single space
        ), 
        '[[:cntrl:]]', '', 'g'  -- Remove control characters
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Email normalization
CREATE OR REPLACE FUNCTION normalize_email(email TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(TRIM(email));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Phone number normalization
CREATE OR REPLACE FUNCTION normalize_phone(phone TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Remove all non-numeric characters except '+'
    RETURN REGEXP_REPLACE(phone, '[^+0-9]', '', 'g');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Usage in triggers
CREATE OR REPLACE FUNCTION clean_user_data()
RETURNS TRIGGER AS $$
BEGIN
    NEW.email := normalize_email(NEW.email);
    NEW.first_name := normalize_text(NEW.first_name);
    NEW.last_name := normalize_text(NEW.last_name);
    NEW.phone_number := normalize_phone(NEW.phone_number);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER clean_users_trigger
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION clean_user_data();
```

### Input Validation

```sql
-- Comprehensive validation constraints
CREATE TABLE validated_strings (
    id SERIAL PRIMARY KEY,
    
    -- Email validation (basic regex)
    email TEXT CHECK (
        email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'
    ),
    
    -- Username validation (alphanumeric, underscore, hyphen)
    username TEXT CHECK (
        username ~ '^[a-zA-Z0-9_-]{3,30}$'
    ),
    
    -- Password strength (example - use proper hashing in practice)
    password_hash TEXT CHECK (
        LENGTH(password_hash) >= 60  -- bcrypt hash length
    ),
    
    -- URL validation (basic)
    website_url TEXT CHECK (
        website_url ~* '^https?://[^\s/$.?#].[^\s]*$'
    ),
    
    -- Phone number (international format)
    phone_number TEXT CHECK (
        phone_number ~ '^\+?[1-9]\d{1,14}$'
    ),
    
    -- Postal codes (various formats)
    postal_code TEXT CHECK (
        postal_code ~ '^[A-Z0-9\s-]{3,10}$'
    ),
    
    -- Content validation
    title TEXT CHECK (
        LENGTH(TRIM(title)) BETWEEN 1 AND 300
    ),
    
    -- Slug validation (URL-friendly)
    slug TEXT CHECK (
        slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'
    )
);
```

### Advanced String Validation

```sql
-- Custom validation functions
CREATE OR REPLACE FUNCTION is_valid_isbn(isbn TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    clean_isbn TEXT;
    check_digit INTEGER;
    calculated_check INTEGER;
BEGIN
    -- Remove hyphens and spaces
    clean_isbn := REGEXP_REPLACE(isbn, '[^0-9X]', '', 'g');
    
    -- ISBN-10 validation
    IF LENGTH(clean_isbn) = 10 THEN
        -- Calculate check digit
        calculated_check := 0;
        FOR i IN 1..9 LOOP
            calculated_check := calculated_check + 
                (11 - i) * SUBSTR(clean_isbn, i, 1)::INTEGER;
        END LOOP;
        calculated_check := 11 - (calculated_check % 11);
        
        IF calculated_check = 11 THEN calculated_check := 0; END IF;
        IF calculated_check = 10 THEN 
            RETURN SUBSTR(clean_isbn, 10, 1) = 'X';
        ELSE
            RETURN calculated_check = SUBSTR(clean_isbn, 10, 1)::INTEGER;
        END IF;
    END IF;
    
    -- ISBN-13 validation would go here...
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Credit card validation (Luhn algorithm)
CREATE OR REPLACE FUNCTION is_valid_credit_card(card_number TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    clean_number TEXT;
    sum_val INTEGER := 0;
    digit INTEGER;
    doubled INTEGER;
BEGIN
    clean_number := REGEXP_REPLACE(card_number, '[^0-9]', '', 'g');
    
    IF LENGTH(clean_number) NOT BETWEEN 13 AND 19 THEN
        RETURN FALSE;
    END IF;
    
    -- Luhn algorithm
    FOR i IN 1..LENGTH(clean_number) LOOP
        digit := SUBSTR(clean_number, LENGTH(clean_number) - i + 1, 1)::INTEGER;
        
        IF i % 2 = 0 THEN
            doubled := digit * 2;
            IF doubled > 9 THEN
                doubled := doubled - 9;
            END IF;
            sum_val := sum_val + doubled;
        ELSE
            sum_val := sum_val + digit;
        END IF;
    END LOOP;
    
    RETURN sum_val % 10 = 0;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

## Search and Pattern Matching

### Full-Text Search

```sql
-- PostgreSQL full-text search
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    author TEXT NOT NULL,
    
    -- Add search vector column
    search_vector TSVECTOR,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create search vector
UPDATE articles SET search_vector = 
    setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(content, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(author, '')), 'C');

-- Create GIN index for fast searching
CREATE INDEX idx_articles_search ON articles USING GIN(search_vector);

-- Search queries
SELECT id, title, ts_rank(search_vector, query) as rank
FROM articles, plainto_tsquery('english', 'database design') query
WHERE search_vector @@ query
ORDER BY rank DESC;

-- Highlight search results
SELECT id, title, 
    ts_headline('english', content, plainto_tsquery('english', 'database')) as snippet
FROM articles
WHERE search_vector @@ plainto_tsquery('english', 'database');
```

### Pattern Matching Examples

```sql
-- Regular expression examples
SELECT 
    -- Email validation
    'user@example.com' ~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$' as valid_email,
    
    -- Phone number extraction
    REGEXP_REPLACE('Call me at (555) 123-4567', '[^\d]', '', 'g') as clean_phone,
    
    -- Extract domain from email
    REGEXP_REPLACE('user@example.com', '^[^@]+@(.+)$', '\1') as domain,
    
    -- Validate strong password
    'MyP@ssw0rd123' ~ '^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$' as strong_password;

-- Text length validation with patterns
SELECT text FROM reviews 
WHERE text ~ '.{10,}'; -- At least 10 characters

SELECT text FROM reviews 
WHERE text ~ '[a-zA-Z0-9]{10,}'; -- At least 10 alphanumeric characters

SELECT text, REGEXP_MATCHES(text, '\w{10,}', 'g') as long_words
FROM reviews;
```

### String Similarity and Fuzzy Matching

```sql
-- PostgreSQL: Install pg_trgm extension for similarity
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Similarity searches
SELECT name, SIMILARITY(name, 'John Smith') as sim
FROM users
WHERE SIMILARITY(name, 'John Smith') > 0.3
ORDER BY sim DESC;

-- Trigram index for fast similarity searches
CREATE INDEX idx_users_name_trgm ON users USING GIN(name gin_trgm_ops);

-- Fuzzy matching examples
SELECT 
    'Hello' <-> 'Helo' as distance1,  -- Edit distance
    'Hello' % 'Helo' as similar1,     -- Similarity boolean
    LEVENSHTEIN('Hello', 'Helo') as edit_distance;

-- Sound-based matching (Soundex)
SELECT 
    SOUNDEX('Smith') = SOUNDEX('Smyth') as sounds_similar,
    DIFFERENCE('Smith', 'Smyth') as soundex_difference; -- 0-4 scale
```

## Performance Optimization

### Indexing Strategies

```sql
-- B-tree indexes for exact matches and ranges
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_last_name ON users(last_name);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Partial indexes for filtered queries
CREATE INDEX idx_active_users_email ON users(email) 
WHERE status = 'active';

-- Composite indexes for multi-column searches
CREATE INDEX idx_users_name_email ON users(last_name, first_name, email);

-- Hash indexes for exact equality (PostgreSQL)
CREATE INDEX idx_users_email_hash ON users USING HASH(email);

-- Expression indexes for computed values
CREATE INDEX idx_users_lower_email ON users(LOWER(email));
CREATE INDEX idx_users_full_name ON users((first_name || ' ' || last_name));

-- Text pattern indexes
CREATE INDEX idx_users_email_pattern ON users(email text_pattern_ops);
-- Supports LIKE 'prefix%' queries efficiently
```

### Query Optimization

```sql
-- Efficient string searches
-- ✅ Good: Uses index
SELECT * FROM users WHERE email = 'user@example.com';

-- ✅ Good: Prefix search with pattern index
SELECT * FROM users WHERE email LIKE 'admin%';

-- ❌ Bad: Leading wildcard can't use index
SELECT * FROM users WHERE email LIKE '%@gmail.com';

-- ✅ Better: Use functional index or full-text search
CREATE INDEX idx_users_email_suffix ON users(REVERSE(email));
SELECT * FROM users WHERE REVERSE(email) LIKE REVERSE('%@gmail.com');

-- Optimized case-insensitive searches
-- ❌ Slow: Function call on every row
SELECT * FROM users WHERE UPPER(email) = UPPER('User@Example.com');

-- ✅ Fast: Use functional index
CREATE INDEX idx_users_email_lower ON users(LOWER(email));
SELECT * FROM users WHERE LOWER(email) = LOWER('User@Example.com');
```

### Storage Optimization

```sql
-- Choose appropriate string lengths
-- ❌ Wasteful: Fixed length for variable data
CREATE TABLE inefficient (
    username CHAR(100),  -- Wastes space for short usernames
    email CHAR(320)      -- Most emails are much shorter
);

-- ✅ Efficient: Variable length
CREATE TABLE efficient (
    username VARCHAR(50),   -- Reasonable maximum
    email VARCHAR(320)      -- Only uses space needed
);

-- Compression for large text fields
-- PostgreSQL: TOAST automatically compresses large values
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,  -- Automatically compressed if large
    metadata JSONB -- Also compressed
);

-- MySQL: Enable compression
CREATE TABLE large_content (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content LONGTEXT
) ENGINE=InnoDB ROW_FORMAT=COMPRESSED;
```

## Internationalization

### Unicode Handling

```sql
-- Proper Unicode support
CREATE TABLE international_content (
    id SERIAL PRIMARY KEY,
    
    -- Support all Unicode characters
    title TEXT,
    content TEXT,
    language_code CHAR(2),
    
    -- Store original and normalized versions
    title_original TEXT,
    title_normalized TEXT GENERATED ALWAYS AS (
        NORMALIZE(LOWER(title), NFC)
    ) STORED,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Language-specific collations
CREATE TABLE multilingual_products (
    id SERIAL PRIMARY KEY,
    name_en TEXT COLLATE "en_US.UTF-8",
    name_es TEXT COLLATE "es_ES.UTF-8",
    name_fr TEXT COLLATE "fr_FR.UTF-8",
    name_de TEXT COLLATE "de_DE.UTF-8",
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Locale-Aware Sorting

```sql
-- Sorting by different locales
SELECT name FROM products ORDER BY name COLLATE "en_US.UTF-8";
SELECT name FROM products ORDER BY name COLLATE "de_DE.UTF-8";

-- Case-insensitive, accent-insensitive sorting
SELECT name FROM products 
ORDER BY UNACCENT(LOWER(name)) COLLATE "C";

-- Natural language sorting for mixed text/numbers
CREATE OR REPLACE FUNCTION natural_sort_key(text)
RETURNS TEXT AS $$
    SELECT STRING_AGG(
        CASE 
            WHEN part ~ '^\d+$' THEN LPAD(part, 10, '0')
            ELSE part
        END, 
        ''
    )
    FROM REGEXP_SPLIT_TO_TABLE($1, '(\d+)', 'g') AS part;
$$ LANGUAGE SQL IMMUTABLE;

-- Usage: SELECT name FROM items ORDER BY natural_sort_key(name);
```

## Best Practices

### 1. Choose Appropriate Types

```sql
-- ✅ Good: Specific constraints
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(320) NOT NULL UNIQUE,  -- RFC maximum
    username VARCHAR(30) NOT NULL UNIQUE CHECK (LENGTH(username) >= 3),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    bio TEXT,  -- No artificial limit on biography
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended'))
);

-- ❌ Bad: Generic or inappropriate sizes
CREATE TABLE bad_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255),    -- Arbitrary limit
    username TEXT,         -- Too flexible
    name VARCHAR(50),      -- Too restrictive for international names
    bio VARCHAR(1000)      -- Arbitrary limit
);
```

### 2. Implement Proper Validation

```sql
-- Multi-layer validation
CREATE TABLE contact_info (
    id SERIAL PRIMARY KEY,
    
    -- Database-level validation
    email TEXT NOT NULL CHECK (
        email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'
    ),
    
    -- Domain validation using lookup table
    email_domain TEXT GENERATED ALWAYS AS (
        SUBSTRING(email FROM '@(.+)$')
    ) STORED,
    
    -- Normalization
    email_normalized TEXT GENERATED ALWAYS AS (
        LOWER(TRIM(email))
    ) STORED,
    
    UNIQUE(email_normalized)
);

-- Separate domain whitelist/blacklist table
CREATE TABLE email_domain_policy (
    domain TEXT PRIMARY KEY,
    policy TEXT CHECK (policy IN ('allowed', 'blocked', 'suspicious')),
    reason TEXT
);
```

### 3. Performance Monitoring

```sql
-- Monitor string column usage
SELECT 
    schemaname,
    tablename,
    attname as column_name,
    n_distinct,
    avg_width,
    correlation
FROM pg_stats 
WHERE tablename IN ('users', 'products', 'articles')
  AND attname LIKE '%name%' OR attname LIKE '%email%' OR attname LIKE '%text%';

-- Find inefficient string operations
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM users WHERE UPPER(email) LIKE '%GMAIL%';
```

### 4. Security Considerations

```sql
-- Prevent SQL injection in dynamic queries
CREATE OR REPLACE FUNCTION safe_user_search(search_term TEXT)
RETURNS TABLE(id INTEGER, username TEXT, email TEXT) AS $$
BEGIN
    -- Always use parameterized queries
    RETURN QUERY
    SELECT u.id, u.username, u.email
    FROM users u
    WHERE u.username ILIKE '%' || search_term || '%'
       OR u.email ILIKE '%' || search_term || '%';
END;
$$ LANGUAGE plpgsql;

-- Input sanitization
CREATE OR REPLACE FUNCTION sanitize_user_input(input TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Remove potentially dangerous characters
    RETURN REGEXP_REPLACE(
        TRIM(input),
        '[<>&"''\\]',  -- Remove HTML/SQL injection characters
        '',
        'g'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

## Conclusion

Effective string handling in databases requires:

1. **Type Selection**: Choose appropriate VARCHAR/TEXT types with reasonable constraints
2. **Encoding**: Use UTF-8 consistently across your stack
3. **Validation**: Implement multi-layer validation (database + application)
4. **Indexing**: Create appropriate indexes for search patterns
5. **Normalization**: Clean and normalize data at input time
6. **Performance**: Monitor and optimize string operations
7. **Security**: Sanitize inputs and use parameterized queries
8. **Internationalization**: Support Unicode and locale-specific requirements

The key is balancing flexibility with constraints, performance with functionality, and simplicity with robust data integrity.
