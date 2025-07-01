# Pagination Patterns: Complete Guide

Pagination is essential for handling large datasets efficiently. This guide covers different pagination strategies, their trade-offs, and real-world implementation patterns.

## Table of Contents
- [Pagination Strategies](#pagination-strategies)
- [Cursor-Based Pagination](#cursor-based-pagination)
- [Offset-Based Pagination](#offset-based-pagination)
- [Hybrid Approaches](#hybrid-approaches)
- [Performance Considerations](#performance-considerations)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)

## Pagination Strategies

### 1. Offset-Based Pagination (LIMIT/OFFSET)

The traditional approach using LIMIT and OFFSET:

```sql
-- Basic offset pagination
SELECT * FROM products 
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 20 OFFSET 40; -- Page 3 (assuming 20 items per page)

-- With total count (expensive for large datasets)
SELECT COUNT(*) FROM products WHERE status = 'active';
```

**Pros:**
- Simple to implement
- Allows jumping to specific pages
- Familiar to users and developers

**Cons:**
- Performance degrades with large offsets
- Inconsistent results during data changes
- Expensive count queries for large tables

### 2. Cursor-Based Pagination (Keyset Pagination)

Uses a cursor (unique identifier) to determine the next set of results:

```sql
-- Cursor-based pagination using ID
SELECT * FROM products 
WHERE status = 'active' 
AND id > 1000 -- cursor value
ORDER BY id ASC
LIMIT 20;

-- Multi-column cursor for complex sorting
SELECT * FROM products 
WHERE status = 'active' 
AND (created_at, id) > ('2023-01-01 10:00:00', 1000)
ORDER BY created_at DESC, id DESC
LIMIT 20;
```

**Pros:**
- Consistent performance regardless of dataset size
- Stable results during data changes
- No "page drift" issues

**Cons:**
- Cannot jump to specific pages
- More complex to implement
- Requires sortable, unique cursor field

### 3. Seek-Based Pagination

Similar to cursor-based but uses meaningful business values:

```sql
-- Pagination using meaningful values (product name)
SELECT * FROM products 
WHERE status = 'active' 
AND name > 'iPhone 13' -- last seen product name
ORDER BY name ASC
LIMIT 20;
```

## Cursor-Based Pagination

### Simple ID-Based Cursor

```sql
-- Products table with auto-increment ID
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category_id INTEGER,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for efficient cursor queries
CREATE INDEX idx_products_id_status ON products(id, status);

-- First page (no cursor)
SELECT id, name, price, created_at
FROM products 
WHERE status = 'active'
ORDER BY id ASC
LIMIT 21; -- Fetch one extra to check if there are more results

-- Subsequent pages (with cursor)
SELECT id, name, price, created_at
FROM products 
WHERE status = 'active' 
AND id > 1020 -- cursor from last result of previous page
ORDER BY id ASC
LIMIT 21;
```

### Timestamp-Based Cursor

```sql
-- Pagination by creation time
SELECT id, name, price, created_at
FROM products 
WHERE status = 'active' 
AND created_at < '2023-12-01 10:00:00'::timestamptz -- cursor
ORDER BY created_at DESC, id DESC -- Include ID for uniqueness
LIMIT 20;

-- Handle duplicate timestamps with composite cursor
SELECT id, name, price, created_at
FROM products 
WHERE status = 'active' 
AND (created_at, id) < ('2023-12-01 10:00:00'::timestamptz, 1050)
ORDER BY created_at DESC, id DESC
LIMIT 20;
```

### Multi-Column Cursor

```sql
-- Complex sorting with multiple columns
SELECT id, name, price, rating, created_at
FROM products 
WHERE status = 'active' 
AND (rating, created_at, id) < (4.5, '2023-12-01 10:00:00'::timestamptz, 1050)
ORDER BY rating DESC, created_at DESC, id DESC
LIMIT 20;

-- Function to encode/decode cursor values
CREATE OR REPLACE FUNCTION encode_cursor(rating DECIMAL, created_at TIMESTAMPTZ, id INTEGER)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(
        convert_to(rating::text || '|' || created_at::text || '|' || id::text, 'UTF8'),
        'base64'
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION decode_cursor(cursor_text TEXT)
RETURNS TABLE(rating DECIMAL, created_at TIMESTAMPTZ, id INTEGER) AS $$
DECLARE
    decoded_text TEXT;
    parts TEXT[];
BEGIN
    decoded_text := convert_from(decode(cursor_text, 'base64'), 'UTF8');
    parts := string_to_array(decoded_text, '|');
    
    RETURN QUERY SELECT 
        parts[1]::DECIMAL,
        parts[2]::TIMESTAMPTZ,
        parts[3]::INTEGER;
END;
$$ LANGUAGE plpgsql;
```

## Offset-Based Pagination

### Basic Implementation

```sql
-- Standard offset pagination
CREATE OR REPLACE FUNCTION get_products_page(
    page_number INTEGER DEFAULT 1,
    page_size INTEGER DEFAULT 20,
    filter_status TEXT DEFAULT 'active'
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    price DECIMAL,
    created_at TIMESTAMPTZ,
    total_count BIGINT
) AS $$
DECLARE
    offset_value INTEGER;
BEGIN
    offset_value := (page_number - 1) * page_size;
    
    RETURN QUERY
    WITH paginated_products AS (
        SELECT p.id, p.name, p.price, p.created_at
        FROM products p
        WHERE p.status = filter_status
        ORDER BY p.created_at DESC
        LIMIT page_size OFFSET offset_value
    ),
    total_count_query AS (
        SELECT COUNT(*) as total
        FROM products p
        WHERE p.status = filter_status
    )
    SELECT 
        pp.id, pp.name, pp.price, pp.created_at,
        tc.total
    FROM paginated_products pp
    CROSS JOIN total_count_query tc;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT * FROM get_products_page(page_number := 1, page_size := 20);
```

### Optimized Count Queries

```sql
-- Approximate count for large tables
CREATE OR REPLACE FUNCTION get_approximate_count(table_name TEXT)
RETURNS BIGINT AS $$
DECLARE
    count_estimate BIGINT;
BEGIN
    EXECUTE format('
        SELECT n_tup_ins - n_tup_del 
        FROM pg_stat_user_tables 
        WHERE relname = %L', table_name)
    INTO count_estimate;
    
    RETURN COALESCE(count_estimate, 0);
END;
$$ LANGUAGE plpgsql;

-- Use approximate count for better performance
SELECT 
    p.id, p.name, p.price,
    get_approximate_count('products') as approximate_total
FROM products p
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 20 OFFSET 40;
```

## Hybrid Approaches

### Window Function Pagination

```sql
-- Using window functions for pagination with additional metadata
WITH paginated_data AS (
    SELECT 
        id, name, price, created_at,
        ROW_NUMBER() OVER (ORDER BY created_at DESC) as row_num,
        COUNT(*) OVER () as total_count
    FROM products
    WHERE status = 'active'
)
SELECT 
    id, name, price, created_at,
    row_num,
    total_count,
    CASE 
        WHEN row_num <= 20 THEN 'first_page'
        WHEN row_num > total_count - 20 THEN 'last_page'
        ELSE 'middle_page'
    END as page_position
FROM paginated_data
WHERE row_num BETWEEN 21 AND 40; -- Page 2 (items 21-40)
```

### Cursor with Page Numbers

```sql
-- Hybrid approach: cursor-based with page approximation
CREATE TABLE pagination_bookmarks (
    id SERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    page_number INTEGER NOT NULL,
    cursor_value TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(table_name, page_number)
);

-- Store bookmarks for frequently accessed pages
INSERT INTO pagination_bookmarks (table_name, page_number, cursor_value)
VALUES 
    ('products', 1, '0'),
    ('products', 10, '1000'),
    ('products', 20, '2000'),
    ('products', 50, '5000')
ON CONFLICT (table_name, page_number) 
DO UPDATE SET cursor_value = EXCLUDED.cursor_value;

-- Function to get page using bookmarks
CREATE OR REPLACE FUNCTION get_products_with_bookmark(target_page INTEGER)
RETURNS TABLE(id INTEGER, name TEXT, price DECIMAL, created_at TIMESTAMPTZ) AS $$
DECLARE
    nearest_bookmark RECORD;
    cursor_start INTEGER;
BEGIN
    -- Find the nearest bookmark
    SELECT page_number, cursor_value::INTEGER as cursor_val
    INTO nearest_bookmark
    FROM pagination_bookmarks
    WHERE table_name = 'products' AND page_number <= target_page
    ORDER BY page_number DESC
    LIMIT 1;
    
    cursor_start := COALESCE(nearest_bookmark.cursor_val, 0);
    
    RETURN QUERY
    SELECT p.id, p.name, p.price, p.created_at
    FROM products p
    WHERE p.status = 'active' 
    AND p.id > cursor_start
    ORDER BY p.id ASC
    LIMIT 20 OFFSET ((target_page - COALESCE(nearest_bookmark.page_number, 1)) * 20);
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Indexing Strategy

```sql
-- Indexes for different pagination patterns

-- For ID-based cursor pagination
CREATE INDEX idx_products_cursor_id ON products(id, status) 
WHERE status = 'active';

-- For timestamp-based cursor pagination
CREATE INDEX idx_products_cursor_time ON products(created_at DESC, id DESC, status) 
WHERE status = 'active';

-- For multi-column sorting
CREATE INDEX idx_products_complex_sort ON products(rating DESC, created_at DESC, id DESC)
WHERE status = 'active';

-- For offset pagination (less efficient for large offsets)
CREATE INDEX idx_products_offset ON products(created_at DESC, status)
WHERE status = 'active';

-- Covering index to avoid table lookups
CREATE INDEX idx_products_covering ON products(created_at DESC, id DESC) 
INCLUDE (name, price, category_id)
WHERE status = 'active';
```

### Query Optimization

```sql
-- Analyze query performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM products 
WHERE status = 'active' 
AND id > 1000
ORDER BY id ASC
LIMIT 20;

-- Use appropriate data types for cursors
-- BIGINT for large tables, INTEGER for smaller ones
-- TIMESTAMPTZ for time-based cursors
-- Composite types for complex cursors

-- Avoid expensive operations in ORDER BY
-- Bad: ORDER BY (expensive_function(column))
-- Good: Pre-compute values and order by stored column
```

## Real-World Examples

### Social Media Feed Pagination

```sql
-- Social media posts with engagement-based sorting
CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    author_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    likes_count INTEGER DEFAULT 0,
    comments_count INTEGER DEFAULT 0,
    shares_count INTEGER DEFAULT 0,
    engagement_score DECIMAL GENERATED ALWAYS AS (
        likes_count * 1.0 + comments_count * 2.0 + shares_count * 3.0
    ) STORED
);

-- Index for engagement-based pagination
CREATE INDEX idx_posts_engagement ON posts(engagement_score DESC, created_at DESC, id DESC);

-- Pagination function for social feed
CREATE OR REPLACE FUNCTION get_social_feed(
    user_id BIGINT,
    cursor_engagement DECIMAL DEFAULT NULL,
    cursor_created_at TIMESTAMPTZ DEFAULT NULL,
    cursor_id BIGINT DEFAULT NULL,
    page_size INTEGER DEFAULT 20
)
RETURNS TABLE(
    id BIGINT,
    author_id BIGINT,
    content TEXT,
    created_at TIMESTAMPTZ,
    engagement_score DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT p.id, p.author_id, p.content, p.created_at, p.engagement_score
    FROM posts p
    INNER JOIN user_follows uf ON uf.followed_id = p.author_id
    WHERE uf.follower_id = user_id
    AND (
        cursor_engagement IS NULL OR
        (p.engagement_score, p.created_at, p.id) < (cursor_engagement, cursor_created_at, cursor_id)
    )
    ORDER BY p.engagement_score DESC, p.created_at DESC, p.id DESC
    LIMIT page_size + 1; -- +1 to check if there are more results
END;
$$ LANGUAGE plpgsql;
```

### E-commerce Product Catalog

```sql
-- Product catalog with filtering and sorting
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category_id INTEGER,
    brand_id INTEGER,
    rating DECIMAL(3,2) DEFAULT 0,
    review_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    status TEXT DEFAULT 'active'
);

-- Pagination with dynamic filtering
CREATE OR REPLACE FUNCTION get_products_catalog(
    category_ids INTEGER[] DEFAULT NULL,
    brand_ids INTEGER[] DEFAULT NULL,
    min_price DECIMAL DEFAULT NULL,
    max_price DECIMAL DEFAULT NULL,
    min_rating DECIMAL DEFAULT NULL,
    sort_by TEXT DEFAULT 'created_at',
    sort_order TEXT DEFAULT 'desc',
    cursor_value TEXT DEFAULT NULL,
    page_size INTEGER DEFAULT 20
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    price DECIMAL,
    rating DECIMAL,
    created_at TIMESTAMPTZ,
    next_cursor TEXT
) AS $$
DECLARE
    sql_query TEXT;
    cursor_conditions TEXT;
    order_clause TEXT;
BEGIN
    -- Build dynamic WHERE clause
    sql_query := 'SELECT p.id, p.name, p.price, p.rating, p.created_at FROM products p WHERE p.status = ''active''';
    
    IF category_ids IS NOT NULL THEN
        sql_query := sql_query || ' AND p.category_id = ANY($1)';
    END IF;
    
    IF brand_ids IS NOT NULL THEN
        sql_query := sql_query || ' AND p.brand_id = ANY($2)';
    END IF;
    
    IF min_price IS NOT NULL THEN
        sql_query := sql_query || ' AND p.price >= $3';
    END IF;
    
    IF max_price IS NOT NULL THEN
        sql_query := sql_query || ' AND p.price <= $4';
    END IF;
    
    IF min_rating IS NOT NULL THEN
        sql_query := sql_query || ' AND p.rating >= $5';
    END IF;
    
    -- Build cursor conditions based on sort
    IF cursor_value IS NOT NULL THEN
        CASE sort_by
            WHEN 'price' THEN
                cursor_conditions := format(' AND (p.price, p.id) %s (decode_cursor_price($6))', 
                    CASE WHEN sort_order = 'desc' THEN '<' ELSE '>' END);
            WHEN 'rating' THEN
                cursor_conditions := format(' AND (p.rating, p.id) %s (decode_cursor_rating($6))', 
                    CASE WHEN sort_order = 'desc' THEN '<' ELSE '>' END);
            ELSE
                cursor_conditions := format(' AND (p.created_at, p.id) %s (decode_cursor_time($6))', 
                    CASE WHEN sort_order = 'desc' THEN '<' ELSE '>' END);
        END CASE;
        
        sql_query := sql_query || cursor_conditions;
    END IF;
    
    -- Add ORDER BY clause
    order_clause := format(' ORDER BY p.%s %s, p.id %s LIMIT %s', 
        sort_by, upper(sort_order), upper(sort_order), page_size + 1);
    sql_query := sql_query || order_clause;
    
    -- Execute dynamic query
    RETURN QUERY EXECUTE sql_query USING category_ids, brand_ids, min_price, max_price, min_rating, cursor_value;
END;
$$ LANGUAGE plpgsql;
```

### Analytics Dashboard Pagination

```sql
-- Time-series data pagination for analytics
CREATE TABLE analytics_events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    user_id BIGINT,
    session_id TEXT,
    properties JSONB,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    date_partition DATE GENERATED ALWAYS AS (timestamp::DATE) STORED
);

-- Partition by date for better performance
CREATE INDEX idx_analytics_time_partition ON analytics_events(date_partition, timestamp DESC, id DESC);

-- Pagination function for analytics data
CREATE OR REPLACE FUNCTION get_analytics_events(
    start_date DATE,
    end_date DATE,
    event_types TEXT[] DEFAULT NULL,
    cursor_timestamp TIMESTAMPTZ DEFAULT NULL,
    cursor_id BIGINT DEFAULT NULL,
    page_size INTEGER DEFAULT 100
)
RETURNS TABLE(
    id BIGINT,
    event_type TEXT,
    user_id BIGINT,
    properties JSONB,
    timestamp TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT ae.id, ae.event_type, ae.user_id, ae.properties, ae.timestamp
    FROM analytics_events ae
    WHERE ae.date_partition BETWEEN start_date AND end_date
    AND (event_types IS NULL OR ae.event_type = ANY(event_types))
    AND (
        cursor_timestamp IS NULL OR
        (ae.timestamp, ae.id) < (cursor_timestamp, cursor_id)
    )
    ORDER BY ae.timestamp DESC, ae.id DESC
    LIMIT page_size + 1;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Choose the Right Pagination Strategy

```sql
-- Decision matrix for pagination strategy selection

-- Use OFFSET pagination when:
-- - Small to medium datasets (< 100K records)
-- - Need to jump to specific pages
-- - Simple sorting requirements
-- - Acceptable performance for your use case

-- Use CURSOR pagination when:
-- - Large datasets (> 100K records)  
-- - High-frequency access patterns
-- - Real-time data updates
-- - Performance is critical

-- Use HYBRID approach when:
-- - Need both performance and page jumping
-- - Can pre-compute page bookmarks
-- - Have predictable access patterns
```

### 2. Implement Cursor Encoding

```sql
-- Encode cursors to hide implementation details
CREATE OR REPLACE FUNCTION encode_product_cursor(
    created_at TIMESTAMPTZ,
    id INTEGER
)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(
        convert_to(
            json_build_object(
                'created_at', extract(epoch from created_at),
                'id', id
            )::text,
            'UTF8'
        ),
        'base64'
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION decode_product_cursor(cursor_text TEXT)
RETURNS TABLE(created_at TIMESTAMPTZ, id INTEGER) AS $$
DECLARE
    cursor_data JSONB;
BEGIN
    cursor_data := convert_from(decode(cursor_text, 'base64'), 'UTF8')::jsonb;
    
    RETURN QUERY SELECT 
        to_timestamp((cursor_data->>'created_at')::DOUBLE PRECISION),
        (cursor_data->>'id')::INTEGER;
END;
$$ LANGUAGE plpgsql;
```

### 3. Handle Edge Cases

```sql
-- Handle empty results
CREATE OR REPLACE FUNCTION safe_paginated_query(
    cursor_value TEXT DEFAULT NULL,
    page_size INTEGER DEFAULT 20
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    has_next BOOLEAN,
    next_cursor TEXT
) AS $$
DECLARE
    results RECORD;
    result_count INTEGER;
BEGIN
    -- Get one extra record to check if there are more
    FOR results IN
        SELECT p.id, p.name
        FROM products p
        WHERE (cursor_value IS NULL OR p.id > decode_cursor(cursor_value))
        ORDER BY p.id ASC
        LIMIT page_size + 1
    LOOP
        result_count := result_count + 1;
        
        -- Only return up to page_size records
        IF result_count <= page_size THEN
            RETURN QUERY SELECT 
                results.id, 
                results.name,
                FALSE, -- has_next will be updated later
                NULL::TEXT; -- next_cursor will be updated later
        END IF;
    END LOOP;
    
    -- Update the last record with pagination metadata
    IF result_count > 0 THEN
        UPDATE pg_temp.result_set 
        SET 
            has_next = (result_count > page_size),
            next_cursor = CASE 
                WHEN result_count > page_size 
                THEN encode_cursor(last_id)
                ELSE NULL
            END;
    END IF;
    
    RETURN;
END;
$$ LANGUAGE plpgsql;
```

### 4. Monitor and Optimize

```sql
-- Create monitoring views for pagination performance
CREATE VIEW pagination_performance AS
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    ROUND(
        (seq_tup_read + idx_tup_fetch)::DECIMAL / 
        NULLIF(seq_scan + idx_scan, 0), 2
    ) as avg_rows_per_scan
FROM pg_stat_user_tables
WHERE tablename IN ('products', 'posts', 'analytics_events')
ORDER BY seq_tup_read DESC;

-- Monitor slow pagination queries
SELECT 
    query,
    calls,
    mean_exec_time,
    total_exec_time,
    rows
FROM pg_stat_statements
WHERE query LIKE '%LIMIT%'
AND mean_exec_time > 100 -- Queries taking more than 100ms
ORDER BY mean_exec_time DESC;
```

## Conclusion

Choosing the right pagination strategy depends on your specific use case:

- **Offset-based pagination** is simple but doesn't scale well
- **Cursor-based pagination** provides consistent performance but limits navigation
- **Hybrid approaches** can provide the best of both worlds with additional complexity

Key considerations:
- Dataset size and growth rate
- Query frequency and access patterns  
- User experience requirements (page jumping vs. infinite scroll)
- Performance requirements and constraints

Always measure performance with realistic data volumes and access patterns to make informed decisions about pagination strategies.
