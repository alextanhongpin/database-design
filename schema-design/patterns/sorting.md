# Sorting & Ordering Patterns

Efficient sorting is crucial for database performance and user experience. This guide covers various sorting strategies, including UUID sorting, composite ordering, and performance optimization techniques.

## 🎯 Core Sorting Challenges

### 1. UUID Sorting Performance Issues

**Problem**: UUIDs (v4) are random and don't sort chronologically
**Impact**: Poor index performance, cache inefficiency, random I/O patterns

```sql
-- ❌ Problematic: Random UUID v4 sorting
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Random UUIDs
    title TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- This query will be slow for large datasets
SELECT * FROM posts ORDER BY id LIMIT 10;

-- ❌ Even worse: sorting by random UUID with other criteria
SELECT * FROM posts ORDER BY id, created_at DESC LIMIT 20;
```

### 2. Solutions for UUID Sorting

#### Option A: Separate Sort Column

```sql
-- ✅ Better: UUID primary key + dedicated sort column
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sort_order BIGINT GENERATED ALWAYS AS IDENTITY,
    title TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Efficient sorting using numeric column
CREATE INDEX idx_posts_sort_order ON posts(sort_order);
SELECT * FROM posts ORDER BY sort_order DESC LIMIT 10;

-- Can still use UUID for public APIs
SELECT id, title FROM posts WHERE id = 'specific-uuid';
```

#### Option B: Time-Based UUIDs (v7/v8)

```sql
-- ✅ Best: Use UUID v7 for time-based sorting
-- Note: Implementation depends on your database and extensions

-- PostgreSQL with uuid-ossp extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Custom UUID v7 function (simplified example)
CREATE OR REPLACE FUNCTION uuid_v7() RETURNS UUID AS $$
DECLARE
    timestamp_ms BIGINT;
    random_bytes BYTEA;
BEGIN
    -- Get current timestamp in milliseconds
    timestamp_ms := EXTRACT(EPOCH FROM NOW()) * 1000;
    
    -- Generate random bytes for the rest
    random_bytes := gen_random_bytes(10);
    
    -- Combine timestamp + random (simplified - actual v7 is more complex)
    RETURN encode(
        int8send(timestamp_ms) || random_bytes, 
        'hex'
    )::UUID;
END;
$$ LANGUAGE plpgsql;

-- Table using sortable UUIDs
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_v7(),
    event_type TEXT NOT NULL,
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Now sorting by ID is chronological and efficient
SELECT * FROM events ORDER BY id DESC LIMIT 10;
```

## 🏗️ Advanced Sorting Patterns

### 1. Multi-Column Sorting Strategies

```sql
-- Efficient composite sorting with proper indexing
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    rating DECIMAL(3,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Strategic composite indexes for common sorting patterns
-- Most selective column first
CREATE INDEX idx_products_category_price ON products(category_id, price DESC);
CREATE INDEX idx_products_category_rating ON products(category_id, rating DESC, created_at DESC);
CREATE INDEX idx_products_created ON products(created_at DESC);

-- Efficient queries using composite indexes
-- ✅ Uses idx_products_category_price
SELECT * FROM products 
WHERE category_id = 'electronics-uuid'
ORDER BY price DESC
LIMIT 20;

-- ✅ Uses idx_products_category_rating  
SELECT * FROM products
WHERE category_id = 'electronics-uuid'
ORDER BY rating DESC, created_at DESC
LIMIT 20;
```

### 2. Custom Sorting Logic

```sql
-- Custom sort orders with CASE statements
SELECT 
    id,
    name,
    status,
    priority
FROM tasks
ORDER BY 
    -- Custom priority ordering
    CASE status
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        ELSE 5
    END,
    -- Then by due date
    due_date ASC NULLS LAST,
    -- Finally by creation time
    created_at DESC;

-- Using computed sort columns for complex logic
ALTER TABLE tasks ADD COLUMN sort_weight INTEGER GENERATED ALWAYS AS (
    CASE status
        WHEN 'critical' THEN 1000
        WHEN 'high' THEN 800
        WHEN 'medium' THEN 600
        WHEN 'low' THEN 400
        ELSE 200
    END +
    CASE 
        WHEN due_date < NOW() THEN 100 -- Overdue bonus
        WHEN due_date < NOW() + INTERVAL '1 day' THEN 50 -- Due soon bonus
        ELSE 0
    END
) STORED;

CREATE INDEX idx_tasks_sort_weight ON tasks(sort_weight DESC, created_at DESC);

-- Now sorting is simple and fast
SELECT * FROM tasks ORDER BY sort_weight DESC, created_at DESC;
```

### 3. Pagination-Friendly Sorting

```sql
-- Cursor-based pagination with stable sorting
CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    published_at TIMESTAMP NOT NULL,
    view_count INTEGER DEFAULT 0,
    sort_id BIGINT GENERATED ALWAYS AS IDENTITY
);

-- Stable cursor pagination using multiple sort criteria
CREATE INDEX idx_articles_published_sort ON articles(published_at DESC, sort_id DESC);

-- First page
SELECT id, title, published_at, sort_id
FROM articles
ORDER BY published_at DESC, sort_id DESC
LIMIT 20;

-- Next page using cursor (last_published_at, last_sort_id from previous page)
SELECT id, title, published_at, sort_id
FROM articles
WHERE (published_at, sort_id) < ('2024-01-15 10:30:00', 12345)
ORDER BY published_at DESC, sort_id DESC
LIMIT 20;
```

## 🎭 Domain-Specific Sorting

### 1. User-Defined Sort Orders

```sql
-- User-customizable sort preferences
CREATE TABLE user_sort_preferences (
    user_id UUID NOT NULL,
    entity_type TEXT NOT NULL, -- 'posts', 'products', etc.
    sort_fields JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (user_id, entity_type),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Example sort preference: [{"field": "price", "direction": "asc"}, {"field": "rating", "direction": "desc"}]

-- Function to build dynamic ORDER BY
CREATE OR REPLACE FUNCTION build_sort_clause(
    p_entity_type TEXT,
    p_user_id UUID DEFAULT NULL
) RETURNS TEXT AS $$
DECLARE
    sort_config JSONB;
    sort_clause TEXT := '';
    field_config JSONB;
BEGIN
    -- Get user preferences or use defaults
    SELECT COALESCE(usp.sort_fields, '[{"field": "created_at", "direction": "desc"}]')
    INTO sort_config
    FROM user_sort_preferences usp
    WHERE usp.user_id = p_user_id AND usp.entity_type = p_entity_type;
    
    -- Build ORDER BY clause
    FOR field_config IN SELECT * FROM jsonb_array_elements(sort_config)
    LOOP
        IF sort_clause != '' THEN
            sort_clause := sort_clause || ', ';
        END IF;
        
        sort_clause := sort_clause || 
            (field_config->>'field') || ' ' || 
            UPPER(field_config->>'direction');
    END LOOP;
    
    RETURN sort_clause;
END;
$$ LANGUAGE plpgsql;
```

### 2. Relevance-Based Sorting

```sql
-- Full-text search with relevance sorting
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    category_id UUID,
    view_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Full-text search vector
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(content, '')), 'B')
    ) STORED
);

CREATE INDEX idx_documents_search ON documents USING gin(search_vector);

-- Search with relevance + popularity sorting
CREATE OR REPLACE FUNCTION search_documents(
    p_query TEXT,
    p_limit INTEGER DEFAULT 20
) RETURNS TABLE(
    id UUID,
    title TEXT,
    relevance_score REAL,
    popularity_score REAL,
    combined_score REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        d.id,
        d.title,
        ts_rank(d.search_vector, plainto_tsquery('english', p_query)) as relevance_score,
        log(greatest(d.view_count, 1))::REAL as popularity_score,
        (
            ts_rank(d.search_vector, plainto_tsquery('english', p_query)) * 0.7 +
            (log(greatest(d.view_count, 1)) / 10.0) * 0.3
        )::REAL as combined_score
    FROM documents d
    WHERE d.search_vector @@ plainto_tsquery('english', p_query)
    ORDER BY combined_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

## 🚀 Performance Optimization

### 1. Index Strategy for Sorting

```sql
-- Covering indexes for sort + select operations
CREATE INDEX idx_products_category_covering ON products(
    category_id, 
    price DESC, 
    created_at DESC
) INCLUDE (name, description, image_url);

-- Partial indexes for filtered sorts
CREATE INDEX idx_active_products_price ON products(price DESC) 
WHERE status = 'active';

CREATE INDEX idx_featured_products_sort ON products(featured_priority DESC, created_at DESC)
WHERE is_featured = true;

-- Expression indexes for computed sorts
CREATE INDEX idx_products_discount_rate ON products(
    ((original_price - current_price) / original_price) DESC
) WHERE original_price > current_price;
```

### 2. Materialized Views for Complex Sorting

```sql
-- Pre-computed popular content view
CREATE MATERIALIZED VIEW popular_content AS
SELECT 
    id,
    title,
    content_type,
    view_count,
    like_count,
    comment_count,
    (
        view_count * 0.4 +
        like_count * 0.4 +
        comment_count * 0.2 +
        EXTRACT(EPOCH FROM (NOW() - created_at)) / 86400 * -0.1 -- Recency factor
    ) as popularity_score,
    created_at
FROM content
WHERE status = 'published'
ORDER BY popularity_score DESC;

CREATE UNIQUE INDEX idx_popular_content_score ON popular_content(popularity_score DESC, id);

-- Refresh strategy
CREATE OR REPLACE FUNCTION refresh_popular_content()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY popular_content;
END;
$$ LANGUAGE plpgsql;

-- Schedule refresh (example with pg_cron)
-- SELECT cron.schedule('refresh-popular-content', '*/15 * * * *', 'SELECT refresh_popular_content();');
```

### 3. Sorting Large Datasets

```sql
-- Efficient sorting with partitioning
CREATE TABLE large_events (
    id UUID DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    data JSONB
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE large_events_2024_01 PARTITION OF large_events
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE large_events_2024_02 PARTITION OF large_events
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Indexes on partitions
CREATE INDEX idx_large_events_2024_01_created ON large_events_2024_01(created_at DESC);
CREATE INDEX idx_large_events_2024_02_created ON large_events_2024_02(created_at DESC);

-- Queries automatically use partition pruning
SELECT * FROM large_events 
WHERE created_at >= '2024-01-15' 
ORDER BY created_at DESC 
LIMIT 100;
```

## 📊 Monitoring Sort Performance

### 1. Query Analysis

```sql
-- Identify slow sorting queries
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows
FROM pg_stat_statements
WHERE query LIKE '%ORDER BY%'
ORDER BY mean_time DESC
LIMIT 10;

-- Check index usage for sorting
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE indexname LIKE '%sort%' OR indexname LIKE '%order%'
ORDER BY idx_scan DESC;
```

## ⚠️ Common Anti-Patterns

### 1. Sorting Without Indexes
```sql
-- ❌ No supporting index
SELECT * FROM large_table ORDER BY random_column LIMIT 10;

-- ✅ Add appropriate index
CREATE INDEX idx_large_table_random_column ON large_table(random_column);
```

### 2. Complex ORDER BY Expressions
```sql
-- ❌ Complex expression in ORDER BY
SELECT * FROM products 
ORDER BY (price * discount_rate) + (rating * 10) DESC;

-- ✅ Use computed column
ALTER TABLE products ADD COLUMN sort_score DECIMAL(10,2) GENERATED ALWAYS AS (
    (price * discount_rate) + (rating * 10)
) STORED;

CREATE INDEX idx_products_sort_score ON products(sort_score DESC);
```

### 3. Inconsistent Sort Orders
```sql
-- ❌ Inconsistent tie-breaking
SELECT * FROM posts ORDER BY created_at DESC LIMIT 20 OFFSET 40;
-- ^ Same created_at values might appear in different pages

-- ✅ Stable sort with unique tie-breaker
SELECT * FROM posts ORDER BY created_at DESC, id DESC LIMIT 20 OFFSET 40;
```

## 🎯 Best Practices

1. **Use Time-Based UUIDs** - UUID v7/v8 for chronological sorting
2. **Add Sort Columns** - Dedicated numeric columns for complex sorting
3. **Strategic Indexing** - Cover common sort patterns with composite indexes
4. **Stable Sorting** - Always include a unique column for tie-breaking
5. **Limit Deep Pagination** - Use cursor-based pagination for large datasets
6. **Monitor Performance** - Track slow sorting queries and index usage
7. **Consider Materialized Views** - For complex, frequently-used sort orders
8. **Partition Large Tables** - Improve sort performance on massive datasets
9. **Cache Sort Results** - Store pre-computed sort orders when possible
10. **Test with Real Data** - Performance characteristics change with data size

## 📊 Performance Comparison

| Sorting Method | Performance | Scalability | Complexity | Use Case |
|----------------|-------------|-------------|------------|----------|
| **Sequential ID** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐ | Internal ordering |
| **UUID v7** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | Time-based public IDs |
| **UUID + Sort Column** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | UUID APIs with sorting |
| **Timestamp Sorting** | ⭐⭐⭐ | ⭐⭐⭐ | ⭐ | Time-based data |
| **Computed Sort Scores** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | Complex ranking |
| **Random UUID v4** | ⭐ | ⭐ | ⭐ | Should be avoided |

## 🔗 References

- [UUID v7 Specification](https://datatracker.ietf.org/doc/draft-peabody-dispatch-new-uuid-format/)
- [PostgreSQL Indexes and ORDER BY](https://www.postgresql.org/docs/current/indexes-ordering.html)
- [Efficient Pagination Strategies](https://use-the-index-luke.com/sql/partial-results/fetch-next-page)
- [UUID Performance Analysis](https://www.2ndquadrant.com/en/blog/sequential-uuid-generators/)
