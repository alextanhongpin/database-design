# Materialized Views

Materialized views are a powerful database feature for improving query performance by storing pre-computed results of complex queries. They are particularly useful for heavy analytical workloads and reporting systems.

## Table of Contents

1. [Basic Concepts](#basic-concepts)
2. [Creating Materialized Views](#creating-materialized-views)
3. [Refresh Strategies](#refresh-strategies)
4. [Use Cases](#use-cases)
5. [Performance Optimization](#performance-optimization)
6. [Best Practices](#best-practices)
7. [Maintenance](#maintenance)
8. [Anti-Patterns](#anti-patterns)

## Basic Concepts

### What are Materialized Views?

A materialized view is a database object that stores the result of a query physically on disk, unlike regular views which are virtual and compute results on-demand.

**Benefits:**
- Faster query performance for complex calculations
- Reduced CPU usage for repeated queries
- Better performance for analytical workloads
- Can be indexed for additional performance gains

**Trade-offs:**
- Additional storage space required
- Data can become stale without proper refresh strategy
- Maintenance overhead for keeping data current

## Creating Materialized Views

### Basic Materialized View

```sql
-- PostgreSQL syntax
CREATE MATERIALIZED VIEW sales_summary AS
SELECT 
    DATE_TRUNC('month', order_date) as month,
    product_category,
    COUNT(*) as order_count,
    SUM(amount) as total_revenue,
    AVG(amount) as avg_order_value,
    COUNT(DISTINCT customer_id) as unique_customers
FROM orders
WHERE order_status = 'completed'
GROUP BY DATE_TRUNC('month', order_date), product_category;

-- Create index for better query performance
CREATE INDEX idx_sales_summary_month_category 
ON sales_summary (month, product_category);
```

### Complex Analytical View

```sql
CREATE MATERIALIZED VIEW customer_analytics AS
WITH customer_metrics AS (
    SELECT 
        customer_id,
        COUNT(*) as total_orders,
        SUM(amount) as lifetime_value,
        AVG(amount) as avg_order_value,
        MIN(order_date) as first_order_date,
        MAX(order_date) as last_order_date,
        COUNT(DISTINCT product_category) as categories_purchased
    FROM orders
    WHERE order_status = 'completed'
    GROUP BY customer_id
),
customer_segments AS (
    SELECT 
        customer_id,
        total_orders,
        lifetime_value,
        avg_order_value,
        first_order_date,
        last_order_date,
        categories_purchased,
        CASE 
            WHEN lifetime_value > 5000 AND total_orders > 20 THEN 'VIP'
            WHEN lifetime_value > 2000 AND total_orders > 10 THEN 'Premium'
            WHEN lifetime_value > 500 AND total_orders > 5 THEN 'Regular'
            ELSE 'New'
        END as customer_segment,
        CASE 
            WHEN last_order_date > CURRENT_DATE - INTERVAL '30 days' THEN 'Active'
            WHEN last_order_date > CURRENT_DATE - INTERVAL '90 days' THEN 'At Risk'
            ELSE 'Inactive'
        END as activity_status
    FROM customer_metrics
)
SELECT 
    cs.*,
    c.name,
    c.email,
    c.created_at as customer_since
FROM customer_segments cs
JOIN customers c ON cs.customer_id = c.id;

-- Indexes for common query patterns
CREATE INDEX idx_customer_analytics_segment ON customer_analytics (customer_segment);
CREATE INDEX idx_customer_analytics_status ON customer_analytics (activity_status);
CREATE INDEX idx_customer_analytics_value ON customer_analytics (lifetime_value);
```

### Time-Series Aggregation

```sql
CREATE MATERIALIZED VIEW hourly_metrics AS
SELECT 
    DATE_TRUNC('hour', created_at) as hour,
    COUNT(*) as total_events,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(CASE WHEN event_type = 'purchase' THEN 1 END) as purchases,
    COUNT(CASE WHEN event_type = 'view' THEN 1 END) as page_views,
    AVG(CASE WHEN event_type = 'purchase' THEN amount END) as avg_purchase_amount,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount) as median_amount
FROM events
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY DATE_TRUNC('hour', created_at);

CREATE INDEX idx_hourly_metrics_hour ON hourly_metrics (hour);
```

## Refresh Strategies

### Manual Refresh

```sql
-- Complete refresh (rebuilds entire view)
REFRESH MATERIALIZED VIEW sales_summary;

-- Concurrent refresh (allows queries during refresh - PostgreSQL)
REFRESH MATERIALIZED VIEW CONCURRENTLY sales_summary;
```

### Automated Refresh with Cron Jobs

```sql
-- PostgreSQL: Using pg_cron extension
SELECT cron.schedule('refresh-sales-summary', '0 1 * * *', 'REFRESH MATERIALIZED VIEW sales_summary;');

-- Or create a stored procedure
CREATE OR REPLACE FUNCTION refresh_analytics_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY sales_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY customer_analytics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY hourly_metrics;
    
    -- Log refresh completion
    INSERT INTO view_refresh_log (view_name, refreshed_at)
    VALUES 
        ('sales_summary', NOW()),
        ('customer_analytics', NOW()),
        ('hourly_metrics', NOW());
END;
$$ LANGUAGE plpgsql;
```

### Incremental Refresh Pattern

```sql
-- Create a helper table to track last refresh
CREATE TABLE materialized_view_refresh_log (
    view_name VARCHAR(100) PRIMARY KEY,
    last_refresh_at TIMESTAMP DEFAULT NOW()
);

-- Incremental refresh function
CREATE OR REPLACE FUNCTION refresh_sales_summary_incremental()
RETURNS void AS $$
DECLARE
    last_refresh TIMESTAMP;
BEGIN
    -- Get last refresh time
    SELECT last_refresh_at INTO last_refresh
    FROM materialized_view_refresh_log
    WHERE view_name = 'sales_summary';
    
    -- If no previous refresh, set to beginning of current month
    IF last_refresh IS NULL THEN
        last_refresh := DATE_TRUNC('month', NOW());
    END IF;
    
    -- Delete data that might have changed
    DELETE FROM sales_summary_temp
    WHERE month >= DATE_TRUNC('month', last_refresh);
    
    -- Insert new/updated data
    INSERT INTO sales_summary_temp
    SELECT 
        DATE_TRUNC('month', order_date) as month,
        product_category,
        COUNT(*) as order_count,
        SUM(amount) as total_revenue,
        AVG(amount) as avg_order_value,
        COUNT(DISTINCT customer_id) as unique_customers
    FROM orders
    WHERE order_status = 'completed'
        AND order_date >= last_refresh
    GROUP BY DATE_TRUNC('month', order_date), product_category;
    
    -- Swap tables atomically
    BEGIN
        ALTER TABLE sales_summary RENAME TO sales_summary_old;
        ALTER TABLE sales_summary_temp RENAME TO sales_summary;
        DROP TABLE sales_summary_old;
    END;
    
    -- Update refresh log
    INSERT INTO materialized_view_refresh_log (view_name, last_refresh_at)
    VALUES ('sales_summary', NOW())
    ON CONFLICT (view_name) 
    DO UPDATE SET last_refresh_at = NOW();
END;
$$ LANGUAGE plpgsql;
```

## Use Cases

### Business Intelligence Dashboards

```sql
-- Executive dashboard metrics
CREATE MATERIALIZED VIEW executive_dashboard AS
SELECT 
    'monthly_revenue' as metric_name,
    TO_CHAR(DATE_TRUNC('month', order_date), 'YYYY-MM') as period,
    SUM(amount) as value
FROM orders
WHERE order_status = 'completed'
GROUP BY DATE_TRUNC('month', order_date)

UNION ALL

SELECT 
    'monthly_orders' as metric_name,
    TO_CHAR(DATE_TRUNC('month', order_date), 'YYYY-MM') as period,
    COUNT(*) as value
FROM orders
WHERE order_status = 'completed'
GROUP BY DATE_TRUNC('month', order_date)

UNION ALL

SELECT 
    'monthly_customers' as metric_name,
    TO_CHAR(DATE_TRUNC('month', first_order_date), 'YYYY-MM') as period,
    COUNT(*) as value
FROM (
    SELECT customer_id, MIN(order_date) as first_order_date
    FROM orders
    WHERE order_status = 'completed'
    GROUP BY customer_id
) first_orders
GROUP BY DATE_TRUNC('month', first_order_date);
```

### Product Performance Analysis

```sql
CREATE MATERIALIZED VIEW product_performance AS
WITH product_metrics AS (
    SELECT 
        p.id as product_id,
        p.name as product_name,
        p.category,
        COUNT(oi.id) as total_sales,
        SUM(oi.quantity) as units_sold,
        SUM(oi.price * oi.quantity) as revenue,
        AVG(oi.price) as avg_selling_price,
        COUNT(DISTINCT o.customer_id) as unique_buyers,
        MIN(o.order_date) as first_sale_date,
        MAX(o.order_date) as last_sale_date
    FROM products p
    LEFT JOIN order_items oi ON p.id = oi.product_id
    LEFT JOIN orders o ON oi.order_id = o.id
    WHERE o.order_status = 'completed'
    GROUP BY p.id, p.name, p.category
),
ranked_products AS (
    SELECT 
        *,
        ROW_NUMBER() OVER (PARTITION BY category ORDER BY revenue DESC) as category_rank,
        ROW_NUMBER() OVER (ORDER BY revenue DESC) as overall_rank,
        CASE 
            WHEN revenue > 10000 THEN 'High Performer'
            WHEN revenue > 5000 THEN 'Medium Performer'
            WHEN revenue > 1000 THEN 'Low Performer'
            ELSE 'Underperformer'
        END as performance_category
    FROM product_metrics
)
SELECT * FROM ranked_products;
```

## Performance Optimization

### Indexing Strategies

```sql
-- Index commonly filtered columns
CREATE INDEX idx_sales_summary_month ON sales_summary (month);
CREATE INDEX idx_sales_summary_category ON sales_summary (product_category);

-- Composite indexes for multi-column filters
CREATE INDEX idx_sales_summary_month_category ON sales_summary (month, product_category);

-- Covering indexes to avoid table lookups
CREATE INDEX idx_customer_analytics_covering 
ON customer_analytics (customer_segment, activity_status) 
INCLUDE (lifetime_value, total_orders);
```

### Partitioning Large Materialized Views

```sql
-- Partition by month for time-series data
CREATE TABLE sales_summary_partitioned (
    month DATE,
    product_category VARCHAR(50),
    order_count INTEGER,
    total_revenue DECIMAL(15,2),
    avg_order_value DECIMAL(10,2),
    unique_customers INTEGER
) PARTITION BY RANGE (month);

-- Create partitions
CREATE TABLE sales_summary_2024_01 PARTITION OF sales_summary_partitioned
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE sales_summary_2024_02 PARTITION OF sales_summary_partitioned
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

## Best Practices

### 1. Choose Appropriate Refresh Frequency

```sql
-- High-frequency data (every hour)
SELECT cron.schedule('refresh-realtime-metrics', '0 * * * *', 
    'REFRESH MATERIALIZED VIEW CONCURRENTLY realtime_metrics;');

-- Daily business reports (early morning)
SELECT cron.schedule('refresh-daily-reports', '0 2 * * *', 
    'REFRESH MATERIALIZED VIEW daily_business_report;');

-- Weekly/monthly analysis (weekends)
SELECT cron.schedule('refresh-monthly-analysis', '0 3 * * 0', 
    'REFRESH MATERIALIZED VIEW monthly_customer_analysis;');
```

### 2. Monitor View Freshness

```sql
-- Create freshness monitoring
CREATE VIEW materialized_view_status AS
SELECT 
    schemaname,
    matviewname,
    matviewowner,
    tablespace,
    hasindexes,
    ispopulated,
    definition
FROM pg_matviews;

-- Query to check last refresh times
CREATE VIEW view_freshness AS
SELECT 
    mv.matviewname,
    CASE 
        WHEN mv.ispopulated THEN 'Populated'
        ELSE 'Not Populated'
    END as status,
    pg_stat_get_last_analyze_time(c.oid) as last_refreshed
FROM pg_matviews mv
JOIN pg_class c ON c.relname = mv.matviewname;
```

### 3. Handle Refresh Failures

```sql
-- Robust refresh procedure with error handling
CREATE OR REPLACE FUNCTION safe_refresh_materialized_view(view_name TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    refresh_start TIMESTAMP;
    refresh_end TIMESTAMP;
    success BOOLEAN := FALSE;
BEGIN
    refresh_start := NOW();
    
    BEGIN
        EXECUTE format('REFRESH MATERIALIZED VIEW CONCURRENTLY %I', view_name);
        success := TRUE;
        refresh_end := NOW();
        
        -- Log successful refresh
        INSERT INTO materialized_view_refresh_log 
        (view_name, refresh_started_at, refresh_completed_at, success, error_message)
        VALUES (view_name, refresh_start, refresh_end, TRUE, NULL);
        
    EXCEPTION WHEN OTHERS THEN
        refresh_end := NOW();
        
        -- Log failed refresh
        INSERT INTO materialized_view_refresh_log 
        (view_name, refresh_started_at, refresh_completed_at, success, error_message)
        VALUES (view_name, refresh_start, refresh_end, FALSE, SQLERRM);
        
        -- Optionally notify administrators
        -- PERFORM pg_notify('mv_refresh_failed', view_name || ': ' || SQLERRM);
    END;
    
    RETURN success;
END;
$$ LANGUAGE plpgsql;
```

## Maintenance

### Regular Maintenance Tasks

```sql
-- Analyze materialized views for optimal query plans
ANALYZE sales_summary;
ANALYZE customer_analytics;

-- Reindex if necessary
REINDEX INDEX idx_sales_summary_month_category;

-- Check view sizes and growth
SELECT 
    schemaname,
    matviewname,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||matviewname)) as size
FROM pg_matviews
ORDER BY pg_total_relation_size(schemaname||'.'||matviewname) DESC;
```

### Cleanup Old Data

```sql
-- Remove old partitions automatically
CREATE OR REPLACE FUNCTION cleanup_old_partitions()
RETURNS void AS $$
DECLARE
    old_partition TEXT;
BEGIN
    -- Drop partitions older than 2 years
    FOR old_partition IN
        SELECT tablename
        FROM pg_tables
        WHERE tablename LIKE 'sales_summary_%'
          AND tablename < 'sales_summary_' || TO_CHAR(NOW() - INTERVAL '2 years', 'YYYY_MM')
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I', old_partition);
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

## Anti-Patterns

### Common Mistakes to Avoid

```sql
-- ❌ Bad: Over-refreshing fast-changing data
-- Don't refresh every minute for data that changes constantly
-- SELECT cron.schedule('bad-refresh', '* * * * *', 'REFRESH MATERIALIZED VIEW user_sessions;');

-- ✅ Good: Use appropriate refresh frequency
SELECT cron.schedule('good-refresh', '*/15 * * * *', 'REFRESH MATERIALIZED VIEW user_session_summary;');

-- ❌ Bad: No indexes on materialized views
CREATE MATERIALIZED VIEW slow_view AS
SELECT customer_id, SUM(amount) as total
FROM orders GROUP BY customer_id;
-- Missing: CREATE INDEX idx_slow_view_customer ON slow_view (customer_id);

-- ❌ Bad: Materialized views without proper monitoring
-- Always monitor refresh status and data freshness

-- ❌ Bad: Overly complex materialized views
-- Don't put everything in one massive view
-- Break complex logic into multiple simpler views
```

### When NOT to Use Materialized Views

1. **Rapidly Changing Data**: When source data changes frequently and you need real-time results
2. **Simple Queries**: When the underlying query is already fast
3. **Low Query Frequency**: When the view is rarely queried
4. **Storage Constraints**: When storage space is limited
5. **Complex Dependencies**: When maintaining refresh order becomes too complex

## Related Patterns

- [View Patterns](view.md)
- [Query Optimization](query.md)
- [Indexing Strategies](../performance/README.md)
- [Time-Series Patterns](../temporal/README.md)