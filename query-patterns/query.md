
# SQL Query Patterns & Techniques

Essential SQL query patterns for efficient data retrieval, manipulation, and analysis. This guide covers fundamental techniques used in production database applications.

## 🎯 Core Query Patterns

### 1. Tuple Matching

**Use case**: Matching multiple column combinations efficiently

```sql
-- Multiple condition matching with tuples
SELECT * FROM orders 
WHERE (status, priority) IN (
    ('delivered', 'high'), 
    ('shipped', 'urgent'),
    ('processing', 'critical')
);

-- Equivalent to (but more efficient than):
SELECT * FROM orders 
WHERE (status = 'delivered' AND priority = 'high')
   OR (status = 'shipped' AND priority = 'urgent')
   OR (status = 'processing' AND priority = 'critical');

-- Complex tuple matching with subqueries
SELECT p.*, c.name as category_name
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE (p.status, p.availability) IN (
    SELECT 'active', 'in_stock'
    UNION ALL
    SELECT 'featured', 'pre_order'
    UNION ALL
    SELECT 'clearance', 'limited'
);
```

### 2. Conditional Aggregation

```sql
-- Count different conditions in single query
SELECT 
    category_id,
    COUNT(*) as total_products,
    COUNT(*) FILTER (WHERE status = 'active') as active_products,
    COUNT(*) FILTER (WHERE price > 100) as expensive_products,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_count,
    AVG(CASE WHEN status = 'active' THEN price END) as avg_active_price
FROM products
GROUP BY category_id;

-- Conditional sums and calculations
SELECT 
    user_id,
    SUM(amount) as total_spent,
    SUM(amount) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days') as last_30_days,
    SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END) as completed_amount,
    COUNT(CASE WHEN status = 'refunded' THEN 1 END) as refund_count
FROM orders
GROUP BY user_id;
```

### 3. Complex JOIN Patterns

```sql
-- Self-join for hierarchical data
SELECT 
    c1.id,
    c1.name,
    c1.parent_id,
    c2.name as parent_name,
    c3.name as grandparent_name
FROM categories c1
LEFT JOIN categories c2 ON c1.parent_id = c2.id
LEFT JOIN categories c3 ON c2.parent_id = c3.id;

-- Multiple table joins with aggregations
SELECT 
    u.id,
    u.username,
    COUNT(DISTINCT o.id) as total_orders,
    COUNT(DISTINCT p.id) as unique_products_ordered,
    SUM(oi.quantity * oi.unit_price) as total_spent,
    MAX(o.created_at) as last_order_date
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
LEFT JOIN order_items oi ON o.id = oi.order_id
LEFT JOIN products p ON oi.product_id = p.id
WHERE u.status = 'active'
GROUP BY u.id, u.username
HAVING COUNT(DISTINCT o.id) > 0;
```

## 🚀 Advanced Query Techniques

### 1. Window Functions for Analytics

```sql
-- Running totals and rankings
SELECT 
    order_id,
    user_id,
    amount,
    created_at,
    SUM(amount) OVER (
        PARTITION BY user_id 
        ORDER BY created_at 
        ROWS UNBOUNDED PRECEDING
    ) as running_total,
    ROW_NUMBER() OVER (
        PARTITION BY user_id 
        ORDER BY amount DESC
    ) as order_rank_by_amount,
    LAG(amount, 1) OVER (
        PARTITION BY user_id 
        ORDER BY created_at
    ) as previous_order_amount
FROM orders
WHERE created_at >= '2024-01-01';

-- Percentile calculations
SELECT 
    product_id,
    price,
    PERCENT_RANK() OVER (ORDER BY price) as price_percentile,
    NTILE(10) OVER (ORDER BY price) as price_decile,
    CUME_DIST() OVER (ORDER BY price) as cumulative_distribution
FROM products
WHERE status = 'active';
```

### 2. Common Table Expressions (CTEs)

```sql
-- Recursive CTE for hierarchical data
WITH RECURSIVE category_tree AS (
    -- Base case: root categories
    SELECT 
        id, 
        name, 
        parent_id, 
        0 as level,
        ARRAY[id] as path,
        name as full_path
    FROM categories 
    WHERE parent_id IS NULL
    
    UNION ALL
    
    -- Recursive case: child categories
    SELECT 
        c.id,
        c.name,
        c.parent_id,
        ct.level + 1,
        ct.path || c.id,
        ct.full_path || ' > ' || c.name
    FROM categories c
    JOIN category_tree ct ON c.parent_id = ct.id
    WHERE ct.level < 10 -- Prevent infinite recursion
)
SELECT 
    id,
    name,
    level,
    full_path,
    array_length(path, 1) as depth
FROM category_tree
ORDER BY path;

-- Multiple CTEs for complex calculations
WITH monthly_sales AS (
    SELECT 
        DATE_TRUNC('month', created_at) as month,
        SUM(amount) as total_sales,
        COUNT(*) as order_count
    FROM orders
    WHERE created_at >= '2023-01-01'
    GROUP BY DATE_TRUNC('month', created_at)
),
sales_growth AS (
    SELECT 
        month,
        total_sales,
        order_count,
        LAG(total_sales) OVER (ORDER BY month) as prev_month_sales,
        (total_sales - LAG(total_sales) OVER (ORDER BY month)) / 
        NULLIF(LAG(total_sales) OVER (ORDER BY month), 0) * 100 as growth_percentage
    FROM monthly_sales
)
SELECT 
    month,
    total_sales,
    order_count,
    ROUND(growth_percentage, 2) as growth_pct,
    CASE 
        WHEN growth_percentage > 10 THEN 'High Growth'
        WHEN growth_percentage > 0 THEN 'Growth'
        WHEN growth_percentage > -10 THEN 'Decline'
        ELSE 'Significant Decline'
    END as growth_category
FROM sales_growth
ORDER BY month;
```

### 3. Advanced Filtering Techniques

```sql
-- Dynamic filtering with multiple optional conditions
SELECT * FROM products p
WHERE 1=1
    AND ($1::TEXT IS NULL OR p.category_id = $1::UUID)
    AND ($2::TEXT IS NULL OR p.price >= $2::DECIMAL)
    AND ($3::TEXT IS NULL OR p.price <= $3::DECIMAL)
    AND ($4::TEXT IS NULL OR p.name ILIKE '%' || $4 || '%')
    AND ($5::TEXT IS NULL OR p.status = ANY(string_to_array($5, ',')));

-- Existence-based filtering
SELECT DISTINCT p.*
FROM products p
WHERE EXISTS (
    SELECT 1 FROM order_items oi
    JOIN orders o ON oi.order_id = o.id
    WHERE oi.product_id = p.id
    AND o.created_at >= NOW() - INTERVAL '30 days'
)
AND NOT EXISTS (
    SELECT 1 FROM product_reviews pr
    WHERE pr.product_id = p.id
    AND pr.rating < 3
);

-- Array and JSON filtering
SELECT * FROM users
WHERE preferences->>'theme' = 'dark'
    AND CAST(preferences->>'notifications_enabled' AS BOOLEAN) = true
    AND tags && ARRAY['premium', 'verified']  -- Array overlap
    AND 'admin' = ANY(roles);                 -- Array membership
```

## 🎭 Specialized Query Patterns

### 1. Pivot and Unpivot Operations

```sql
-- Pivot: Transform rows to columns
SELECT 
    product_id,
    SUM(CASE WHEN EXTRACT(MONTH FROM created_at) = 1 THEN amount ELSE 0 END) as jan_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM created_at) = 2 THEN amount ELSE 0 END) as feb_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM created_at) = 3 THEN amount ELSE 0 END) as mar_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM created_at) = 4 THEN amount ELSE 0 END) as apr_sales
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
WHERE EXTRACT(YEAR FROM o.created_at) = 2024
GROUP BY product_id;

-- PostgreSQL CROSSTAB (requires tablefunc extension)
SELECT * FROM crosstab(
    'SELECT product_id, EXTRACT(MONTH FROM created_at), SUM(amount)
     FROM order_items oi JOIN orders o ON oi.order_id = o.id
     WHERE EXTRACT(YEAR FROM o.created_at) = 2024
     GROUP BY product_id, EXTRACT(MONTH FROM created_at)
     ORDER BY 1, 2',
    'VALUES (1), (2), (3), (4), (5), (6), (7), (8), (9), (10), (11), (12)'
) AS ct(product_id UUID, jan DECIMAL, feb DECIMAL, mar DECIMAL, apr DECIMAL, 
         may DECIMAL, jun DECIMAL, jul DECIMAL, aug DECIMAL, 
         sep DECIMAL, oct DECIMAL, nov DECIMAL, dec DECIMAL);
```

### 2. Time-Series Queries

```sql
-- Generate time series with gaps filled
WITH date_series AS (
    SELECT generate_series(
        '2024-01-01'::DATE,
        '2024-12-31'::DATE,
        '1 day'::INTERVAL
    )::DATE as date
),
daily_sales AS (
    SELECT 
        DATE(created_at) as sale_date,
        SUM(amount) as daily_total,
        COUNT(*) as order_count
    FROM orders
    WHERE created_at >= '2024-01-01'
    GROUP BY DATE(created_at)
)
SELECT 
    ds.date,
    COALESCE(daily_total, 0) as sales,
    COALESCE(order_count, 0) as orders,
    SUM(COALESCE(daily_total, 0)) OVER (
        ORDER BY ds.date 
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ) as rolling_7day_sales
FROM date_series ds
LEFT JOIN daily_sales dsales ON ds.date = dsales.sale_date
ORDER BY ds.date;

-- Moving averages and trends
SELECT 
    DATE_TRUNC('week', created_at) as week,
    COUNT(*) as weekly_orders,
    AVG(COUNT(*)) OVER (
        ORDER BY DATE_TRUNC('week', created_at)
        ROWS BETWEEN 3 PRECEDING AND CURRENT ROW
    ) as moving_4week_avg,
    (COUNT(*) - LAG(COUNT(*), 1) OVER (ORDER BY DATE_TRUNC('week', created_at))) /
    NULLIF(LAG(COUNT(*), 1) OVER (ORDER BY DATE_TRUNC('week', created_at)), 0) * 100 as week_over_week_growth
FROM orders
WHERE created_at >= NOW() - INTERVAL '1 year'
GROUP BY DATE_TRUNC('week', created_at)
ORDER BY week;
```

### 3. Text and Pattern Matching

```sql
-- Full-text search with ranking
SELECT 
    id,
    title,
    content,
    ts_rank(to_tsvector('english', title || ' ' || content), query) as rank
FROM articles,
     plainto_tsquery('english', 'database design patterns') as query
WHERE to_tsvector('english', title || ' ' || content) @@ query
ORDER BY rank DESC;

-- Pattern matching and extraction
SELECT 
    id,
    email,
    -- Extract domain from email
    regexp_replace(email, '^[^@]+@', '') as domain,
    -- Validate email format
    email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$' as is_valid_email,
    -- Extract numbers from text
    regexp_replace(phone, '[^0-9]', '', 'g') as clean_phone
FROM users
WHERE email IS NOT NULL;

-- Fuzzy matching with similarity
SELECT 
    id,
    name,
    similarity(name, 'John Smith') as similarity_score
FROM users
WHERE similarity(name, 'John Smith') > 0.3
ORDER BY similarity_score DESC;
```

## ⚡ Performance Optimization

### 1. Index-Friendly Queries

```sql
-- Use leading columns of composite indexes
-- Given index: (user_id, created_at, status)

-- ✅ Good: Uses index efficiently
SELECT * FROM orders 
WHERE user_id = '123' 
AND created_at >= '2024-01-01'
ORDER BY created_at DESC;

-- ❌ Poor: Doesn't use index efficiently  
SELECT * FROM orders 
WHERE created_at >= '2024-01-01'
AND status = 'completed';

-- ✅ Good: Covers all needed columns
CREATE INDEX idx_orders_covering ON orders(user_id, status, created_at) 
INCLUDE (amount, shipping_address);
```

### 2. Query Optimization Techniques

```sql
-- Use LIMIT for large result sets
SELECT * FROM products p
JOIN product_reviews pr ON p.id = pr.product_id
WHERE p.category_id = '123'
ORDER BY pr.rating DESC, p.created_at DESC
LIMIT 20;

-- Avoid N+1 queries with proper JOINs
-- ❌ Poor: Causes N+1 queries in application
SELECT id, name FROM categories;
-- Then for each category: SELECT * FROM products WHERE category_id = ?

-- ✅ Good: Single query with aggregation
SELECT 
    c.id,
    c.name,
    COUNT(p.id) as product_count,
    AVG(p.price) as avg_price
FROM categories c
LEFT JOIN products p ON c.id = p.category_id
WHERE p.status = 'active' OR p.id IS NULL
GROUP BY c.id, c.name;
```

## 🔍 Query Analysis & Debugging

### 1. Execution Plan Analysis

```sql
-- Analyze query performance
EXPLAIN (ANALYZE, BUFFERS, COSTS, VERBOSE) 
SELECT p.*, c.name as category_name
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.price > 100
ORDER BY p.created_at DESC
LIMIT 10;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
```

### 2. Query Statistics

```sql
-- Monitor slow queries (requires pg_stat_statements)
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    stddev_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements
WHERE mean_time > 100  -- Queries slower than 100ms
ORDER BY mean_time DESC
LIMIT 10;
```

## ⚠️ Common Anti-Patterns

### 1. Inefficient Queries
```sql
-- ❌ SELECT * instead of specific columns
SELECT * FROM large_table WHERE condition;

-- ✅ Select only needed columns
SELECT id, name, created_at FROM large_table WHERE condition;

-- ❌ Function calls in WHERE clause
SELECT * FROM orders WHERE DATE(created_at) = '2024-01-01';

-- ✅ Range comparison
SELECT * FROM orders WHERE created_at >= '2024-01-01' AND created_at < '2024-01-02';
```

### 2. Poor JOIN Performance
```sql
-- ❌ Cartesian product risk
SELECT * FROM table1, table2 WHERE table1.id > 100;

-- ✅ Explicit JOINs with proper conditions
SELECT * FROM table1 t1
JOIN table2 t2 ON t1.id = t2.table1_id
WHERE t1.id > 100;
```

## 🎯 Best Practices

1. **Use Specific Columns** - Avoid SELECT * in production queries
2. **Leverage Indexes** - Design queries to use existing indexes efficiently
3. **Limit Result Sets** - Always use appropriate LIMIT clauses
4. **Use CTEs for Readability** - Break complex queries into readable parts
5. **Avoid Functions in WHERE** - Use sargable predicates when possible
6. **Parameterize Queries** - Prevent SQL injection and enable plan reuse
7. **Monitor Performance** - Regularly check query execution plans
8. **Use Window Functions** - Replace complex self-joins when possible
9. **Aggregate Efficiently** - Group by the most selective columns first
10. **Test with Real Data** - Query performance varies with data size and distribution

## 🔗 References

- [PostgreSQL Query Planning](https://www.postgresql.org/docs/current/planner-optimizer.html)
- [SQL Window Functions](https://www.postgresql.org/docs/current/tutorial-window.html)
- [Common Table Expressions](https://www.postgresql.org/docs/current/queries-with.html)
- [PostgreSQL Performance Tips](https://wiki.postgresql.org/wiki/Performance_Optimization)
