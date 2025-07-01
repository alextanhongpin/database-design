# Database Performance Optimization: Complete Guide

This guide covers essential techniques for monitoring, diagnosing, and optimizing database performance. Focus is on PostgreSQL with MySQL examples where relevant.

## Table of Contents
- [Performance Monitoring](#performance-monitoring)
- [Index Optimization](#index-optimization)
- [Query Optimization](#query-optimization)
- [Cache Management](#cache-management)
- [Real-World Optimization](#real-world-optimization)
- [Performance Tools](#performance-tools)

## Performance Monitoring

### PostgreSQL Key Metrics

#### 1. Cache Hit Ratios

```sql
-- Overall cache hit ratio (should be > 95%)
SELECT 
    'Buffer Cache' as cache_type,
    round(
        (sum(heap_blks_hit) * 100.0) / 
        nullif(sum(heap_blks_hit) + sum(heap_blks_read), 0), 
        2
    ) as hit_ratio_percent
FROM pg_statio_user_tables

UNION ALL

SELECT 
    'Index Cache' as cache_type,
    round(
        (sum(idx_blks_hit) * 100.0) / 
        nullif(sum(idx_blks_hit) + sum(idx_blks_read), 0), 
        2
    ) as hit_ratio_percent
FROM pg_statio_user_indexes;

-- Detailed table-level cache performance
SELECT 
    schemaname,
    relname as table_name,
    heap_blks_read + heap_blks_hit as total_reads,
    round(
        (heap_blks_hit * 100.0) / 
        nullif(heap_blks_hit + heap_blks_read, 0), 
        2
    ) as cache_hit_ratio,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||relname)) as table_size
FROM pg_statio_user_tables
WHERE heap_blks_read + heap_blks_hit > 0
ORDER BY total_reads DESC
LIMIT 20;
```

#### 2. Index Usage Analysis

```sql
-- Index usage efficiency
SELECT 
    schemaname,
    relname as table_name,
    indexrelname as index_name,
    idx_scan as index_scans,
    seq_scan as sequential_scans,
    n_live_tup as estimated_rows,
    round(
        (idx_scan * 100.0) / 
        nullif(idx_scan + seq_scan, 0), 
        2
    ) as index_usage_percent,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes pui
JOIN pg_stat_user_tables put ON pui.relid = put.relid
WHERE idx_scan + seq_scan > 0
ORDER BY n_live_tup DESC;

-- Identify tables with poor index usage
SELECT 
    schemaname,
    relname as table_name,
    seq_scan,
    seq_tup_read,
    seq_tup_read / seq_scan as avg_seq_read,
    idx_scan,
    n_live_tup as estimated_rows
FROM pg_stat_user_tables
WHERE seq_scan > 0 
AND seq_tup_read / seq_scan > 10000  -- Tables doing large sequential scans
ORDER BY seq_tup_read DESC;
```

#### 3. Unused and Inefficient Indexes

```sql
-- Find unused indexes (candidates for removal)
SELECT 
    schemaname || '.' || relname as table,
    indexrelname as index_name,
    pg_size_pretty(pg_relation_size(i.indexrelid)) as index_size,
    idx_scan as times_used,
    idx_tup_read as rows_read,
    idx_tup_fetch as rows_fetched
FROM pg_stat_user_indexes ui
JOIN pg_index i ON ui.indexrelid = i.indexrelid
WHERE idx_scan < 10  -- Used less than 10 times
AND pg_relation_size(i.indexrelid) > 1024 * 1024  -- Larger than 1MB
AND NOT i.indisunique  -- Not unique indexes (those are needed for constraints)
ORDER BY pg_relation_size(i.indexrelid) DESC;

-- Find duplicate indexes
SELECT 
    array_agg(indexname) as duplicate_indexes,
    tablename,
    array_agg(indexdef) as definitions
FROM pg_indexes
GROUP BY tablename, replace(indexdef, indexname, '')
HAVING count(*) > 1;
```

### MySQL Performance Monitoring

```sql
-- MySQL global status for performance analysis
SHOW GLOBAL STATUS WHERE Variable_name IN (
    'Select_full_join',
    'Select_full_range_join', 
    'Select_range',
    'Select_scan',
    'Sort_merge_passes',
    'Sort_range',
    'Sort_rows',
    'Sort_scan',
    'Handler_read_rnd_next'
);

-- Query cache hit ratio (if enabled)
SHOW GLOBAL STATUS WHERE Variable_name IN (
    'Qcache_hits',
    'Qcache_inserts',
    'Qcache_not_cached'
);

-- InnoDB buffer pool efficiency
SHOW GLOBAL STATUS WHERE Variable_name IN (
    'Innodb_buffer_pool_read_requests',
    'Innodb_buffer_pool_reads'
);
```

## Index Optimization

### Index Strategy Guidelines

```sql
-- 1. Single column indexes for frequently filtered columns
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);

-- 2. Composite indexes for multi-column queries
-- Order matters: most selective column first
CREATE INDEX idx_orders_status_created ON orders(status, created_at);
CREATE INDEX idx_orders_customer_status ON orders(customer_id, status);

-- 3. Covering indexes to avoid table lookups
CREATE INDEX idx_orders_covering ON orders(customer_id, status) 
INCLUDE (total_amount, created_at);

-- 4. Partial indexes for filtered queries
CREATE INDEX idx_orders_pending ON orders(created_at) 
WHERE status = 'pending';

CREATE INDEX idx_active_users ON users(last_login) 
WHERE is_active = true;

-- 5. Functional indexes for computed values
CREATE INDEX idx_users_email_lower ON users(lower(email));
CREATE INDEX idx_orders_year ON orders(extract(year from created_at));

-- 6. Text search indexes
CREATE INDEX idx_products_search ON products 
USING GIN (to_tsvector('english', name || ' ' || description));
```

### Index Maintenance

```sql
-- Monitor index bloat
CREATE OR REPLACE VIEW index_bloat AS
SELECT
    schemaname,
    tablename,
    indexname,
    real_size,
    extra_size,
    extra_ratio,
    fill_factor,
    bloat_size,
    bloat_ratio
FROM (
    SELECT
        schemaname,
        tablename,
        indexname,
        pg_size_pretty(real_size::bigint) as real_size,
        pg_size_pretty(extra_size::bigint) as extra_size,
        round(extra_ratio::numeric, 2) as extra_ratio,
        fill_factor,
        pg_size_pretty(bloat_size::bigint) as bloat_size,
        round(bloat_ratio::numeric, 2) as bloat_ratio
    FROM (
        SELECT
            schemaname,
            tablename,
            indexname,
            bs*(relpages)::bigint AS real_size,
            bs*(relpages-est_pages)::bigint AS extra_size,
            100 * (relpages-est_pages)::float / relpages AS extra_ratio,
            fillfactor,
            bs*(relpages-est_pages_ff) AS bloat_size,
            100 * (relpages-est_pages_ff)::float / relpages AS bloat_ratio
        FROM (
            SELECT
                schemaname, tablename, indexname, attname, itemsize, relpages, fillfactor, bs,
                ceil((reltuples*itemsize+nullhighmark*attsize)/(bs-24)) AS est_pages,
                ceil((reltuples*itemsize+nullhighmark*attsize)/(bs*fillfactor/100-24)) AS est_pages_ff
            FROM (
                -- Simplified bloat calculation query
                SELECT 
                    schemaname,
                    tablename,
                    indexname,
                    current_setting('block_size')::numeric AS bs,
                    fillfactor,
                    relpages,
                    reltuples,
                    1 as itemsize,
                    0 as nullhighmark,
                    1 as attsize,
                    '' as attname
                FROM pg_stat_user_indexes
                JOIN pg_class ON pg_class.oid = indexrelid
                JOIN pg_index ON pg_index.indexrelid = pg_class.oid
            ) sub1
        ) sub2
    ) sub3
) sub4
WHERE bloat_ratio > 20;  -- Only show indexes with > 20% bloat

-- Reindex maintenance script
CREATE OR REPLACE FUNCTION reindex_bloated_indexes()
RETURNS TEXT AS $$
DECLARE
    index_record RECORD;
    reindex_commands TEXT := '';
BEGIN
    FOR index_record IN 
        SELECT schemaname, indexname 
        FROM index_bloat 
        WHERE bloat_ratio > 30
    LOOP
        reindex_commands := reindex_commands || 
            format('REINDEX INDEX CONCURRENTLY %I.%I;%s', 
                   index_record.schemaname, 
                   index_record.indexname, 
                   chr(10));
    END LOOP;
    
    RETURN reindex_commands;
END;
$$ LANGUAGE plpgsql;

-- Generate reindex commands
SELECT reindex_bloated_indexes();
```

## Query Optimization

### Query Analysis Techniques

```sql
-- 1. Use EXPLAIN ANALYZE for actual performance data
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) 
SELECT o.id, o.total_amount, c.name as customer_name
FROM orders o
JOIN customers c ON c.id = o.customer_id
WHERE o.status = 'pending'
AND o.created_at > NOW() - INTERVAL '7 days';

-- 2. Identify slow queries using pg_stat_statements
SELECT 
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- Queries averaging > 100ms
ORDER BY mean_exec_time DESC
LIMIT 20;

-- 3. Find queries causing most I/O
SELECT 
    query,
    calls,
    shared_blks_read,
    shared_blks_written,
    shared_blks_dirtied,
    temp_blks_read,
    temp_blks_written
FROM pg_stat_statements
WHERE shared_blks_read > 1000
ORDER BY shared_blks_read DESC
LIMIT 10;
```

### Common Query Optimization Patterns

```sql
-- 1. Optimize EXISTS vs IN
-- Good: Using EXISTS for large subqueries
SELECT c.id, c.name
FROM customers c
WHERE EXISTS (
    SELECT 1 FROM orders o 
    WHERE o.customer_id = c.id 
    AND o.status = 'pending'
);

-- Avoid: IN with large subqueries can be slower
SELECT c.id, c.name
FROM customers c
WHERE c.id IN (
    SELECT o.customer_id FROM orders o 
    WHERE o.status = 'pending'
);

-- 2. Optimize aggregations with proper indexing
-- Ensure indexes support GROUP BY columns
SELECT 
    customer_id,
    COUNT(*) as order_count,
    SUM(total_amount) as total_spent
FROM orders
WHERE created_at >= '2024-01-01'
GROUP BY customer_id
HAVING COUNT(*) > 10;

-- Supporting index
CREATE INDEX idx_orders_customer_date ON orders(customer_id, created_at);

-- 3. Optimize LIMIT with OFFSET alternatives
-- Avoid: OFFSET becomes slower with large offsets
SELECT * FROM products ORDER BY id LIMIT 20 OFFSET 100000;

-- Better: Use cursor-based pagination
SELECT * FROM products 
WHERE id > 100000  -- last seen ID
ORDER BY id 
LIMIT 20;
```

### Query Rewriting Techniques

```sql
-- 1. Replace correlated subqueries with JOINs
-- Slow: Correlated subquery
SELECT p.id, p.name,
    (SELECT AVG(rating) FROM reviews r WHERE r.product_id = p.id) as avg_rating
FROM products p;

-- Fast: LEFT JOIN with aggregation
SELECT p.id, p.name, r.avg_rating
FROM products p
LEFT JOIN (
    SELECT product_id, AVG(rating) as avg_rating
    FROM reviews
    GROUP BY product_id
) r ON r.product_id = p.id;

-- 2. Use window functions for ranking
-- Instead of multiple subqueries
SELECT 
    customer_id,
    order_date,
    total_amount,
    ROW_NUMBER() OVER (PARTITION BY customer_id ORDER BY order_date DESC) as order_rank,
    SUM(total_amount) OVER (PARTITION BY customer_id) as customer_total
FROM orders;

-- 3. Optimize OR conditions with UNION
-- Slow: OR conditions can't use indexes efficiently
SELECT * FROM products 
WHERE category = 'electronics' OR name ILIKE '%phone%';

-- Fast: UNION with separate index-friendly queries
SELECT * FROM products WHERE category = 'electronics'
UNION
SELECT * FROM products WHERE name ILIKE '%phone%';
```

## Cache Management

### PostgreSQL Memory Configuration

```sql
-- Check current memory settings
SHOW shared_buffers;
SHOW work_mem;
SHOW maintenance_work_mem;
SHOW effective_cache_size;

-- Recommended settings for different workloads
-- For analytical workloads (large work_mem)
-- work_mem = 256MB - 1GB per connection
-- For OLTP workloads (smaller work_mem, more connections)
-- work_mem = 4MB - 32MB per connection

-- Monitor memory usage by query type
SELECT 
    query,
    temp_blks_read,
    temp_blks_written,
    work_mem_kb
FROM pg_stat_statements pss
JOIN (
    SELECT 
        current_setting('work_mem') as work_mem_kb
) cfg ON true
WHERE temp_blks_read > 0 OR temp_blks_written > 0
ORDER BY temp_blks_read + temp_blks_written DESC;
```

### Connection and Buffer Management

```sql
-- Monitor connection patterns
SELECT 
    state,
    COUNT(*) as connection_count,
    AVG(EXTRACT(EPOCH FROM (now() - query_start))) as avg_query_duration
FROM pg_stat_activity
WHERE state IS NOT NULL
GROUP BY state;

-- Check buffer usage patterns
SELECT 
    c.relname,
    pg_size_pretty(count(*) * 8192) as buffered,
    round(100.0 * count(*) / (
        SELECT setting FROM pg_settings WHERE name='shared_buffers'
    )::integer, 1) as buffer_percent,
    round(100.0 * count(*) * 8192 / pg_relation_size(c.oid), 1) as percent_of_relation
FROM pg_buffercache b
INNER JOIN pg_class c ON b.relfilenode = pg_relation_filenode(c.oid)
WHERE b.reldatabase IN (0, (SELECT oid FROM pg_database WHERE datname = current_database()))
GROUP BY c.oid, c.relname
ORDER BY 2 DESC
LIMIT 20;
```

## Real-World Optimization

### E-commerce Query Optimization

```sql
-- Scenario: Product catalog with filtering and sorting
-- Typical query pattern
SELECT 
    p.id,
    p.name,
    p.price,
    c.name as category_name,
    AVG(r.rating) as avg_rating,
    COUNT(r.id) as review_count
FROM products p
JOIN categories c ON c.id = p.category_id
LEFT JOIN reviews r ON r.product_id = p.id
WHERE p.status = 'active'
AND c.parent_id = 5  -- Electronics category
AND p.price BETWEEN 100 AND 500
GROUP BY p.id, p.name, p.price, c.name
HAVING AVG(r.rating) >= 4.0
ORDER BY avg_rating DESC, review_count DESC
LIMIT 20;

-- Optimized index strategy
CREATE INDEX idx_products_active_category_price ON products(status, category_id, price) 
WHERE status = 'active';

CREATE INDEX idx_reviews_product_rating ON reviews(product_id, rating);

CREATE INDEX idx_categories_parent ON categories(parent_id) 
INCLUDE (id, name);

-- Alternative: Pre-computed aggregates for better performance
CREATE MATERIALIZED VIEW product_stats AS
SELECT 
    p.id as product_id,
    p.name,
    p.price,
    p.category_id,
    c.name as category_name,
    c.parent_id,
    COALESCE(r.avg_rating, 0) as avg_rating,
    COALESCE(r.review_count, 0) as review_count
FROM products p
JOIN categories c ON c.id = p.category_id
LEFT JOIN (
    SELECT 
        product_id,
        AVG(rating) as avg_rating,
        COUNT(*) as review_count
    FROM reviews
    GROUP BY product_id
) r ON r.product_id = p.id
WHERE p.status = 'active';

CREATE INDEX idx_product_stats_category_price ON product_stats(parent_id, price);
CREATE INDEX idx_product_stats_rating ON product_stats(avg_rating DESC, review_count DESC);

-- Refresh materialized view periodically
-- SELECT cron.schedule('refresh-product-stats', '*/5 * * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY product_stats;');
```

### Analytics Query Optimization

```sql
-- Scenario: Time-series analytics dashboard
-- Original slow query
SELECT 
    DATE_TRUNC('day', created_at) as day,
    COUNT(*) as order_count,
    SUM(total_amount) as daily_revenue,
    AVG(total_amount) as avg_order_value
FROM orders
WHERE created_at >= NOW() - INTERVAL '90 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY day;

-- Optimization 1: Proper indexing
CREATE INDEX idx_orders_created_date ON orders(DATE_TRUNC('day', created_at));

-- Optimization 2: Pre-aggregated daily stats table
CREATE TABLE daily_order_stats (
    date DATE PRIMARY KEY,
    order_count INTEGER NOT NULL,
    total_revenue DECIMAL(15,2) NOT NULL,
    avg_order_value DECIMAL(10,2) NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to update daily stats
CREATE OR REPLACE FUNCTION update_daily_stats(target_date DATE DEFAULT CURRENT_DATE)
RETURNS VOID AS $$
BEGIN
    INSERT INTO daily_order_stats (date, order_count, total_revenue, avg_order_value)
    SELECT 
        target_date,
        COUNT(*),
        SUM(total_amount),
        AVG(total_amount)
    FROM orders
    WHERE DATE_TRUNC('day', created_at) = target_date
    ON CONFLICT (date) 
    DO UPDATE SET
        order_count = EXCLUDED.order_count,
        total_revenue = EXCLUDED.total_revenue,
        avg_order_value = EXCLUDED.avg_order_value,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Fast dashboard query using pre-aggregated data
SELECT * FROM daily_order_stats
WHERE date >= CURRENT_DATE - INTERVAL '90 days'
ORDER BY date;
```

## Performance Tools

### Monitoring Queries

```sql
-- Create comprehensive performance monitoring view
CREATE VIEW performance_overview AS
SELECT 
    'Cache Hit Ratio' as metric,
    round(
        (sum(heap_blks_hit) * 100.0) / 
        nullif(sum(heap_blks_hit) + sum(heap_blks_read), 0), 
        2
    )::TEXT || '%' as value
FROM pg_statio_user_tables
UNION ALL
SELECT 
    'Index Usage',
    round(
        (sum(idx_scan) * 100.0) / 
        nullif(sum(idx_scan) + sum(seq_scan), 0), 
        2
    )::TEXT || '%'
FROM pg_stat_user_tables
UNION ALL
SELECT 
    'Active Connections',
    COUNT(*)::TEXT
FROM pg_stat_activity 
WHERE state = 'active'
UNION ALL
SELECT 
    'Database Size',
    pg_size_pretty(pg_database_size(current_database()))
UNION ALL
SELECT 
    'Largest Table',
    (SELECT relname FROM pg_stat_user_tables ORDER BY n_live_tup DESC LIMIT 1);

-- Query performance monitoring
SELECT 
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time,
    rows,
    shared_blks_hit,
    shared_blks_read
FROM pg_stat_statements
WHERE calls > 100
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### Automated Performance Alerts

```sql
-- Function to check for performance issues
CREATE OR REPLACE FUNCTION check_performance_issues()
RETURNS TABLE(
    issue_type TEXT,
    severity TEXT,
    description TEXT,
    recommendation TEXT
) AS $$
BEGIN
    -- Check cache hit ratio
    RETURN QUERY
    SELECT 
        'Cache Performance'::TEXT,
        CASE WHEN hit_ratio < 95 THEN 'HIGH' ELSE 'LOW' END,
        'Cache hit ratio is ' || hit_ratio::TEXT || '%',
        'Consider increasing shared_buffers or optimizing queries'
    FROM (
        SELECT round(
            (sum(heap_blks_hit) * 100.0) / 
            nullif(sum(heap_blks_hit) + sum(heap_blks_read), 0), 
            2
        ) as hit_ratio
        FROM pg_statio_user_tables
    ) cache_stats
    WHERE hit_ratio < 98;
    
    -- Check for tables with poor index usage
    RETURN QUERY
    SELECT 
        'Index Usage'::TEXT,
        'MEDIUM'::TEXT,
        'Table ' || relname || ' has ' || index_usage::TEXT || '% index usage',
        'Consider adding indexes for frequently queried columns'
    FROM (
        SELECT 
            relname,
            round(
                (idx_scan * 100.0) / 
                nullif(idx_scan + seq_scan, 0), 
                2
            ) as index_usage
        FROM pg_stat_user_tables
        WHERE seq_scan + idx_scan > 1000
    ) index_stats
    WHERE index_usage < 90;
    
    -- Check for large unused indexes
    RETURN QUERY
    SELECT 
        'Storage Efficiency'::TEXT,
        'MEDIUM'::TEXT,
        'Index ' || indexrelname || ' (' || pg_size_pretty(pg_relation_size(indexrelid)) || ') is rarely used',
        'Consider dropping unused index to save storage and improve write performance'
    FROM pg_stat_user_indexes
    WHERE idx_scan < 10 
    AND pg_relation_size(indexrelid) > 10 * 1024 * 1024;  -- > 10MB
END;
$$ LANGUAGE plpgsql;

-- Run performance check
SELECT * FROM check_performance_issues();
```

## Conclusion

Database performance optimization is an ongoing process that requires:

**Monitoring Foundation:**
- Establish baseline metrics for cache hit ratios, index usage, and query performance
- Implement automated monitoring for performance degradation
- Regular analysis of slow query logs and pg_stat_statements

**Index Strategy:**
- Create indexes based on actual query patterns, not assumptions
- Use composite indexes for multi-column filters
- Implement partial indexes for filtered queries
- Regular maintenance to handle index bloat

**Query Optimization:**
- Use EXPLAIN ANALYZE to understand query execution
- Rewrite correlated subqueries as JOINs when possible
- Consider materialized views for complex aggregations
- Implement proper pagination strategies

**Ongoing Maintenance:**
- Monitor and reindex bloated indexes
- Update table statistics regularly with ANALYZE
- Review and optimize configuration parameters
- Plan for data growth and adjust strategies accordingly

Remember: measure before optimizing, and always verify that optimizations provide real-world benefits for your specific workload patterns.
