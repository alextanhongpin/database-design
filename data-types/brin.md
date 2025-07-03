# PostgreSQL BRIN Indexes

Block Range Indexes (BRIN) are specialized indexes designed for very large tables where data has natural ordering, providing massive storage savings with acceptable performance trade-offs.

## What are BRIN Indexes?

BRIN (Block Range Index) stores summary information about ranges of table pages instead of indexing every row. They work best with data that has natural correlation with physical storage order.

### How BRIN Works

```sql
-- BRIN index stores min/max values for each page range
-- Instead of: Row 1 -> Value A, Row 2 -> Value B, Row 3 -> Value C...
-- BRIN stores: Pages 1-100 -> Min: A, Max: Z

CREATE TABLE time_series_data (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    sensor_id INT NOT NULL,
    value DECIMAL(10,4) NOT NULL
);

-- BRIN index on timestamp (naturally ordered data)
CREATE INDEX idx_time_series_brin_timestamp 
ON time_series_data USING BRIN (timestamp);
```

## Use Cases for BRIN Indexes

### 1. Time-Series Data
Perfect for naturally ordered temporal data.

```sql
-- Log table with chronological inserts
CREATE TABLE application_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(10) NOT NULL,
    message TEXT,
    user_id INT
);

-- BRIN index ideal for time-based queries
CREATE INDEX idx_logs_brin_created_at 
ON application_logs USING BRIN (created_at);

-- Efficient queries on time ranges
SELECT * FROM application_logs 
WHERE created_at BETWEEN '2024-07-01' AND '2024-07-02';
```

### 2. Sequential Numeric Data
When numeric columns correlate with insertion order.

```sql
-- Order system with sequential order IDs
CREATE TABLE orders (
    order_id BIGSERIAL PRIMARY KEY,
    order_number BIGINT NOT NULL, -- Sequential numbering
    customer_id INT NOT NULL,
    total_amount DECIMAL(12,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- BRIN on order_number (sequential, correlates with storage)
CREATE INDEX idx_orders_brin_number 
ON orders USING BRIN (order_number);

-- Efficient range queries
SELECT * FROM orders 
WHERE order_number BETWEEN 1000000 AND 1100000;
```

### 3. Geographic Data with Spatial Clustering
When geographic data is clustered by region.

```sql
-- Location data clustered by geographic region
CREATE TABLE weather_stations (
    id SERIAL PRIMARY KEY,
    station_name VARCHAR(100),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    elevation INT,
    country_code CHAR(2)
);

-- If data is inserted region by region
CREATE INDEX idx_weather_brin_lat 
ON weather_stations USING BRIN (latitude);

CREATE INDEX idx_weather_brin_lon 
ON weather_stations USING BRIN (longitude);
```

### 4. Append-Only Tables
Tables where data is only inserted, never updated.

```sql
-- Audit table with append-only pattern
CREATE TABLE audit_trail (
    id BIGSERIAL PRIMARY KEY,
    table_name VARCHAR(100),
    operation VARCHAR(10), -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    user_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- BRIN perfect for append-only audit data
CREATE INDEX idx_audit_brin_created_at 
ON audit_trail USING BRIN (created_at);
```

## Advantages of BRIN Indexes

### 1. Massive Storage Savings
```sql
-- Comparison: 100 million row table
-- B-tree index: ~2.2 GB
-- BRIN index: ~150 KB (99.99% smaller!)

-- Check index sizes
SELECT 
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes 
WHERE tablename = 'time_series_data'
ORDER BY pg_relation_size(indexrelid) DESC;
```

### 2. Fast Index Creation and Maintenance
```sql
-- BRIN index creation is very fast
CREATE INDEX CONCURRENTLY idx_large_table_brin_timestamp 
ON large_table USING BRIN (timestamp);

-- Maintenance is minimal since only summary data is stored
```

### 3. Minimal Impact on INSERT Performance
```sql
-- INSERTs barely affected by BRIN indexes
-- vs B-tree indexes which can significantly slow down inserts
INSERT INTO time_series_data (timestamp, sensor_id, value)
SELECT 
    CURRENT_TIMESTAMP + (i || ' seconds')::INTERVAL,
    (random() * 1000)::INT,
    random() * 100
FROM generate_series(1, 1000000) i;
```

### 4. Automatic Summarization
```sql
-- BRIN automatically summarizes new page ranges
-- But you can also trigger manual summarization
SELECT brin_summarize_new_values('idx_time_series_brin_timestamp');

-- Check summarization status
SELECT * FROM brin_page_items(get_raw_page('idx_time_series_brin_timestamp', 1), 'idx_time_series_brin_timestamp');
```

## When NOT to Use BRIN

### 1. Random Data Distribution
```sql
-- BAD: UUIDs have no correlation with storage order
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    price DECIMAL(10,2)
);

-- BRIN will be ineffective here
-- CREATE INDEX idx_products_brin_id ON products USING BRIN (id); -- Don't do this!

-- Use B-tree instead
CREATE INDEX idx_products_btree_id ON products (id);
```

### 2. High Selectivity Queries
```sql
-- BRIN is poor for highly selective queries
-- This query will scan many pages with BRIN
SELECT * FROM time_series_data WHERE sensor_id = 12345;

-- Use B-tree for high-selectivity columns
CREATE INDEX idx_time_series_sensor_id ON time_series_data (sensor_id);
```

### 3. Frequently Updated Data
```sql
-- Updates can degrade BRIN effectiveness
-- If timestamp values are frequently updated, BRIN ranges become less effective
UPDATE time_series_data 
SET timestamp = '2023-01-01 12:00:00' 
WHERE id = 50000; -- This hurts BRIN efficiency
```

## BRIN Configuration and Tuning

### 1. Pages Per Range Configuration
```sql
-- Default pages_per_range is 128
-- Smaller values = more precise ranges, larger index
-- Larger values = less precise ranges, smaller index

CREATE INDEX idx_logs_brin_small_range 
ON application_logs USING BRIN (created_at) 
WITH (pages_per_range = 64); -- More precise

CREATE INDEX idx_logs_brin_large_range 
ON application_logs USING BRIN (created_at) 
WITH (pages_per_range = 256); -- Less precise, smaller index
```

### 2. Multiple Column BRIN Indexes
```sql
-- Multi-column BRIN for correlated columns
CREATE INDEX idx_orders_brin_multi 
ON orders USING BRIN (order_number, created_at);

-- Effective when both columns have natural ordering
SELECT * FROM orders 
WHERE order_number > 1000000 
AND created_at > '2024-01-01';
```

### 3. Monitoring BRIN Effectiveness
```sql
-- Check how well BRIN is working
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM time_series_data 
WHERE timestamp BETWEEN '2024-07-01' AND '2024-07-02';

-- Look for:
-- - "Bitmap Heap Scan" with "Recheck Cond"
-- - Pages read vs pages available
-- - Execution time vs expected improvement
```

## BRIN Maintenance

### 1. Manual Summarization
```sql
-- Summarize new ranges after bulk inserts
SELECT brin_summarize_new_values('idx_time_series_brin_timestamp');

-- Check which ranges need summarization
SELECT * FROM brin_page_items(get_raw_page('idx_time_series_brin_timestamp', 1), 'idx_time_series_brin_timestamp');
```

### 2. Automatic Summarization
```sql
-- Enable autosummarize for automatic maintenance
CREATE INDEX idx_logs_auto_brin 
ON application_logs USING BRIN (created_at) 
WITH (autosummarize = on);

-- Monitor autosummarize in pg_stat_progress_create_index
```

### 3. Rebuilding BRIN Indexes
```sql
-- Rebuild if data correlation changes
REINDEX INDEX idx_time_series_brin_timestamp;

-- Or drop and recreate with different parameters
DROP INDEX idx_time_series_brin_timestamp;
CREATE INDEX idx_time_series_brin_timestamp 
ON time_series_data USING BRIN (timestamp) 
WITH (pages_per_range = 128);
```

## Performance Comparison

### BRIN vs B-tree Trade-offs

| Aspect | BRIN | B-tree |
|--------|------|--------|
| **Index Size** | Tiny (KB-MB) | Large (GB for big tables) |
| **Creation Speed** | Very Fast | Slow for large tables |
| **INSERT Impact** | Minimal | Can be significant |
| **Range Query Performance** | Good (correlated data) | Excellent |
| **Point Query Performance** | Poor | Excellent |
| **Memory Usage** | Very Low | High |
| **Maintenance Overhead** | Very Low | Moderate to High |

### Example Performance Test
```sql
-- Create test data
INSERT INTO time_series_data (timestamp, sensor_id, value)
SELECT 
    '2024-01-01'::TIMESTAMP + (i || ' seconds')::INTERVAL,
    (i % 1000) + 1,
    random() * 100
FROM generate_series(1, 10000000) i; -- 10M rows

-- Test range query performance
-- BRIN query (should be efficient)
EXPLAIN (ANALYZE, BUFFERS)
SELECT COUNT(*) FROM time_series_data 
WHERE timestamp BETWEEN '2024-01-01' AND '2024-01-02';

-- Check pages read and execution time
```

## Best Practices

1. **Verify Data Correlation** - Ensure your data has natural ordering
2. **Monitor Query Patterns** - BRIN works best for range queries
3. **Test Performance** - Compare BRIN vs B-tree for your use case
4. **Use Appropriate pages_per_range** - Balance precision vs size
5. **Enable Autosummarize** - For append-heavy workloads
6. **Combine with Other Indexes** - Use B-tree for point queries, BRIN for ranges
7. **Regular Monitoring** - Watch for performance degradation

## Common Pitfalls

- **Using on Random Data** - BRIN requires correlation with storage order
- **Expecting Point Query Performance** - BRIN is for range queries
- **Ignoring Data Updates** - Updates can hurt BRIN effectiveness
- **Wrong pages_per_range** - Too small wastes space, too large loses precision
- **Not Monitoring Summarization** - Unsummarized ranges hurt performance

## Related Topics

- [Index Design](../performance/indexing.md) - General indexing strategies
- [Time-Series Patterns](../specialized/time-series.md) - Temporal data optimization
- [Large Table Optimization](../performance/large-tables.md) - Handling big data
- [PostgreSQL Performance](../performance/postgres-optimization.md) - PostgreSQL-specific tuning

## External References

- [PostgreSQL BRIN Indexes: Big Data Performance](https://www.crunchydata.com/blog/postgresql-brin-indexes-big-data-performance-with-minimal-storage)
- [Avoiding BRIN Index Pitfalls](https://www.crunchydata.com/blog/avoiding-the-pitfalls-of-brin-indexes-in-postgres)
- [BRIN Index Benefits](https://www.percona.com/blog/2019/07/16/brin-index-for-postgresql-dont-forget-the-benefits/)
- [PostgreSQL BRIN Documentation](https://www.postgresql.org/docs/current/brin.html)
