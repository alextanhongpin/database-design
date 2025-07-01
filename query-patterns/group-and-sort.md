# Group and Sort Patterns

Grouping data and selecting specific rows within each group is a fundamental database pattern. This guide covers various approaches to solve "top-N per group", "latest per group", and related problems efficiently.

## 🎯 Core Group-and-Sort Problems

### The Challenge
- **Latest per group** - Most recent order per customer
- **Top N per group** - Best 3 products per category  
- **Unique per group** - One contact per company
- **Ranked results** - Employees ranked by salary within department

### Common Use Cases
- **Financial data** - Latest exchange rates, stock prices
- **E-commerce** - Featured products per category
- **User activity** - Most recent login per user
- **Content management** - Latest posts per author
- **Analytics** - Top performers per region

## 🏗️ Core Techniques

### 1. Window Functions (Recommended)

```sql
-- Latest exchange rate per currency pair
WITH ranked_rates AS (
    SELECT 
        source_currency_id,
        target_currency_id,
        rate,
        effective_date,
        created_at,
        ROW_NUMBER() OVER (
            PARTITION BY source_currency_id, target_currency_id 
            ORDER BY effective_date DESC, created_at DESC
        ) as rn
    FROM fx_rates
    WHERE is_active = true
)
SELECT 
    source_currency_id,
    target_currency_id,
    rate,
    effective_date
FROM ranked_rates 
WHERE rn = 1;

-- Top 3 products per category by rating
WITH top_products AS (
    SELECT 
        category_id,
        product_id,
        name,
        rating,
        price,
        ROW_NUMBER() OVER (
            PARTITION BY category_id 
            ORDER BY rating DESC, review_count DESC
        ) as rank
    FROM products
    WHERE status = 'active'
)
SELECT * FROM top_products WHERE rank <= 3;
```

### 2. Lateral Joins (PostgreSQL)

```sql
-- Most recent order per customer using LATERAL
SELECT 
    c.id as customer_id,
    c.name as customer_name,
    recent_order.id as order_id,
    recent_order.total,
    recent_order.created_at
FROM customers c
CROSS JOIN LATERAL (
    SELECT id, total, created_at
    FROM orders o
    WHERE o.customer_id = c.id
    ORDER BY created_at DESC
    LIMIT 1
) recent_order;

-- Top 2 employees per department by salary
SELECT 
    d.name as department_name,
    top_employees.employee_name,
    top_employees.salary
FROM departments d
CROSS JOIN LATERAL (
    SELECT 
        e.name as employee_name,
        e.salary
    FROM employees e
    WHERE e.department_id = d.id
    ORDER BY salary DESC
    LIMIT 2
) top_employees;
```

### 3. Correlated Subqueries

```sql
-- Latest stock price per symbol
SELECT DISTINCT ON (symbol) 
    symbol,
    price,
    volume,
    timestamp
FROM stock_prices
ORDER BY symbol, timestamp DESC;

-- Alternative with correlated subquery
SELECT sp1.*
FROM stock_prices sp1
WHERE sp1.timestamp = (
    SELECT MAX(sp2.timestamp)
    FROM stock_prices sp2
    WHERE sp2.symbol = sp1.symbol
);
```

## 🚀 Advanced Grouping Patterns

### 1. Multi-Level Grouping

```sql
-- Best performing sales rep per region per quarter
WITH quarterly_performance AS (
    SELECT 
        region_id,
        sales_rep_id,
        DATE_TRUNC('quarter', sale_date) as quarter,
        SUM(amount) as total_sales,
        COUNT(*) as sale_count
    FROM sales
    WHERE sale_date >= '2024-01-01'
    GROUP BY region_id, sales_rep_id, DATE_TRUNC('quarter', sale_date)
),
ranked_performance AS (
    SELECT 
        *,
        ROW_NUMBER() OVER (
            PARTITION BY region_id, quarter 
            ORDER BY total_sales DESC
        ) as rank
    FROM quarterly_performance
)
SELECT 
    r.name as region_name,
    qp.quarter,
    sr.name as sales_rep_name,
    qp.total_sales,
    qp.sale_count
FROM ranked_performance qp
JOIN regions r ON qp.region_id = r.id
JOIN sales_reps sr ON qp.sales_rep_id = sr.id
WHERE qp.rank = 1
ORDER BY qp.quarter DESC, qp.total_sales DESC;
```

### 2. Conditional Grouping

```sql
-- Latest active price per product, fallback to inactive if no active
WITH price_priority AS (
    SELECT 
        product_id,
        price,
        is_active,
        created_at,
        ROW_NUMBER() OVER (
            PARTITION BY product_id, is_active 
            ORDER BY created_at DESC
        ) as rn_within_status,
        ROW_NUMBER() OVER (
            PARTITION BY product_id 
            ORDER BY 
                is_active DESC,  -- Active prices first
                created_at DESC
        ) as rn_overall
    FROM product_prices
)
SELECT 
    product_id,
    price,
    is_active,
    created_at,
    CASE 
        WHEN is_active THEN 'current'
        ELSE 'fallback'
    END as price_type
FROM price_priority
WHERE rn_overall = 1;
```

### 3. Time-Window Grouping

```sql
-- Most active user per day over the last 30 days
WITH daily_activity AS (
    SELECT 
        DATE(created_at) as activity_date,
        user_id,
        COUNT(*) as activity_count
    FROM user_activities
    WHERE created_at >= NOW() - INTERVAL '30 days'
    GROUP BY DATE(created_at), user_id
),
daily_leaders AS (
    SELECT 
        activity_date,
        user_id,
        activity_count,
        ROW_NUMBER() OVER (
            PARTITION BY activity_date 
            ORDER BY activity_count DESC, user_id
        ) as rank
    FROM daily_activity
)
SELECT 
    dl.activity_date,
    u.username,
    dl.activity_count
FROM daily_leaders dl
JOIN users u ON dl.user_id = u.id
WHERE dl.rank = 1
ORDER BY dl.activity_date DESC;
```

## 🎭 Specialized Grouping Scenarios

### 1. Handling Ties

```sql
-- All products tied for highest rating per category
WITH category_max_ratings AS (
    SELECT 
        category_id,
        MAX(rating) as max_rating
    FROM products
    GROUP BY category_id
)
SELECT 
    p.category_id,
    c.name as category_name,
    p.name as product_name,
    p.rating
FROM products p
JOIN category_max_ratings cmr ON (
    p.category_id = cmr.category_id 
    AND p.rating = cmr.max_rating
)
JOIN categories c ON p.category_id = c.id
ORDER BY p.category_id, p.name;

-- Using RANK() to include ties
WITH ranked_products AS (
    SELECT 
        category_id,
        name,
        rating,
        RANK() OVER (
            PARTITION BY category_id 
            ORDER BY rating DESC
        ) as rank
    FROM products
)
SELECT * FROM ranked_products WHERE rank = 1;
```

### 2. Percentage-Based Selection

```sql
-- Top 10% of customers by spending per region
WITH customer_spending AS (
    SELECT 
        c.id,
        c.name,
        c.region_id,
        SUM(o.total) as total_spent,
        PERCENT_RANK() OVER (
            PARTITION BY c.region_id 
            ORDER BY SUM(o.total) DESC
        ) as spending_percentile
    FROM customers c
    JOIN orders o ON c.id = o.customer_id
    WHERE o.created_at >= NOW() - INTERVAL '1 year'
    GROUP BY c.id, c.name, c.region_id
)
SELECT 
    r.name as region_name,
    cs.name as customer_name,
    cs.total_spent,
    ROUND(cs.spending_percentile * 100, 2) as percentile
FROM customer_spending cs
JOIN regions r ON cs.region_id = r.id
WHERE cs.spending_percentile >= 0.9  -- Top 10%
ORDER BY cs.region_id, cs.total_spent DESC;
```

### 3. Rolling Aggregations

```sql
-- 7-day rolling average of daily sales per product
WITH daily_sales AS (
    SELECT 
        product_id,
        DATE(created_at) as sale_date,
        SUM(quantity * unit_price) as daily_total
    FROM order_items
    WHERE created_at >= NOW() - INTERVAL '30 days'
    GROUP BY product_id, DATE(created_at)
),
rolling_averages AS (
    SELECT 
        product_id,
        sale_date,
        daily_total,
        AVG(daily_total) OVER (
            PARTITION BY product_id 
            ORDER BY sale_date 
            ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
        ) as rolling_7day_avg
    FROM daily_sales
)
SELECT 
    p.name as product_name,
    ra.sale_date,
    ra.daily_total,
    ROUND(ra.rolling_7day_avg, 2) as avg_7day
FROM rolling_averages ra
JOIN products p ON ra.product_id = p.id
WHERE ra.sale_date >= NOW() - INTERVAL '7 days'
ORDER BY ra.product_id, ra.sale_date DESC;
```

## ⚡ Performance Optimization

### 1. Strategic Indexing

```sql
-- Indexes for group-and-sort queries
-- For latest per group queries
CREATE INDEX idx_fx_rates_currency_date ON fx_rates(
    source_currency_id, 
    target_currency_id, 
    effective_date DESC, 
    created_at DESC
);

-- For top-N per category queries
CREATE INDEX idx_products_category_rating ON products(
    category_id, 
    rating DESC, 
    review_count DESC
) WHERE status = 'active';

-- Covering index to avoid table lookups
CREATE INDEX idx_orders_customer_covering ON orders(
    customer_id, 
    created_at DESC
) INCLUDE (id, total, status);
```

### 2. Materialized Views for Complex Groupings

```sql
-- Pre-computed latest exchange rates
CREATE MATERIALIZED VIEW latest_fx_rates AS
WITH ranked_rates AS (
    SELECT 
        source_currency_id,
        target_currency_id,
        rate,
        effective_date,
        ROW_NUMBER() OVER (
            PARTITION BY source_currency_id, target_currency_id 
            ORDER BY effective_date DESC
        ) as rn
    FROM fx_rates
    WHERE is_active = true
)
SELECT 
    source_currency_id,
    target_currency_id,
    rate,
    effective_date
FROM ranked_rates 
WHERE rn = 1;

CREATE UNIQUE INDEX idx_latest_fx_rates_currencies 
ON latest_fx_rates(source_currency_id, target_currency_id);

-- Refresh function
CREATE OR REPLACE FUNCTION refresh_latest_fx_rates()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY latest_fx_rates;
END;
$$ LANGUAGE plpgsql;
```

### 3. Partitioning for Large Datasets

```sql
-- Partition large tables for better group-and-sort performance
CREATE TABLE sales_data (
    id UUID DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    sale_date DATE NOT NULL,
    region_id UUID NOT NULL
) PARTITION BY RANGE (sale_date);

-- Create monthly partitions
CREATE TABLE sales_data_2024_01 PARTITION OF sales_data
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE sales_data_2024_02 PARTITION OF sales_data
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Indexes on partitions
CREATE INDEX idx_sales_2024_01_region_amount 
ON sales_data_2024_01(region_id, amount DESC);
```

## 🔍 Monitoring & Troubleshooting

### 1. Query Performance Analysis

```sql
-- Find expensive group-and-sort queries
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows
FROM pg_stat_statements
WHERE query LIKE '%PARTITION BY%' 
   OR query LIKE '%ROW_NUMBER()%'
   OR query LIKE '%GROUP BY%'
ORDER BY mean_time DESC
LIMIT 10;

-- Analyze specific query execution
EXPLAIN (ANALYZE, BUFFERS) 
WITH ranked_products AS (
    SELECT 
        category_id,
        name,
        rating,
        ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY rating DESC) as rn
    FROM products
)
SELECT * FROM ranked_products WHERE rn <= 3;
```

### 2. Index Usage Monitoring

```sql
-- Check if group-and-sort indexes are being used
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE indexname LIKE '%partition%' 
   OR indexname LIKE '%group%'
   OR indexname LIKE '%rank%'
ORDER BY idx_scan DESC;
```

## ⚠️ Common Anti-Patterns

### 1. Inefficient Self-Joins
```sql
-- ❌ Inefficient self-join approach
SELECT fx1.*
FROM fx_rates fx1
WHERE fx1.effective_date = (
    SELECT MAX(fx2.effective_date)
    FROM fx_rates fx2
    WHERE fx2.source_currency_id = fx1.source_currency_id
    AND fx2.target_currency_id = fx1.target_currency_id
);

-- ✅ Use window functions instead
WITH ranked AS (
    SELECT *, 
           ROW_NUMBER() OVER (PARTITION BY source_currency_id, target_currency_id ORDER BY effective_date DESC) as rn
    FROM fx_rates
)
SELECT * FROM ranked WHERE rn = 1;
```

### 2. Missing ORDER BY in Window Functions
```sql
-- ❌ Non-deterministic results
SELECT *, ROW_NUMBER() OVER (PARTITION BY category_id) as rn
FROM products;

-- ✅ Always specify ORDER BY
SELECT *, ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY created_at DESC) as rn
FROM products;
```

### 3. Overusing DISTINCT ON
```sql
-- ❌ Can be inefficient for complex cases
SELECT DISTINCT ON (customer_id) *
FROM orders
ORDER BY customer_id, created_at DESC;

-- ✅ Window functions often perform better
WITH ranked_orders AS (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY customer_id ORDER BY created_at DESC) as rn
    FROM orders
)
SELECT * FROM ranked_orders WHERE rn = 1;
```

## 🎯 Best Practices

1. **Use Window Functions** - Generally more efficient than self-joins
2. **Index Partition Columns** - Always index the PARTITION BY columns
3. **Include ORDER BY** - Always specify ordering in window functions
4. **Consider DISTINCT ON** - For simple latest-per-group in PostgreSQL
5. **Use Lateral Joins** - For top-N per group with complex logic
6. **Materialize Complex Views** - Cache expensive group-and-sort results
7. **Partition Large Tables** - Improve performance on massive datasets
8. **Monitor Query Plans** - Verify efficient execution
9. **Limit Result Sets** - Use appropriate WHERE clauses
10. **Test with Real Data** - Performance varies with data distribution

## 📊 Performance Comparison

| Technique | Performance | Complexity | Flexibility | Best For |
|-----------|-------------|------------|-------------|----------|
| **Window Functions** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | General purpose |
| **DISTINCT ON** | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | Simple latest per group |
| **Lateral Joins** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Complex top-N logic |
| **Correlated Subqueries** | ⭐⭐ | ⭐⭐ | ⭐⭐⭐ | Legacy systems |
| **Self Joins** | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ | Should be avoided |

## 🔗 References

- [PostgreSQL Window Functions](https://www.postgresql.org/docs/current/tutorial-window.html)
- [Lateral Joins in PostgreSQL](https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-LATERAL)
- [SQL Window Functions Explained](https://www.windowfunctions.com/)
- [Efficient Top-N Queries](https://use-the-index-luke.com/sql/partial-results/top-n-queries)
