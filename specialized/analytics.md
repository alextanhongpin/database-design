# Database Analytics Patterns

Best practices and common patterns for building robust analytics systems on database platforms.

## 🎯 Overview

Analytics systems require careful attention to:
- **Data Quality** - Ensuring accurate and consistent data
- **Query Performance** - Optimizing for analytical workloads
- **Maintainability** - Creating sustainable analytics infrastructure
- **Scalability** - Handling growing data volumes

## ⚠️ Common Analytics Mistakes

### 1. Incorrect Count After Filtering

```sql
-- ❌ Wrong: Count affected by joins
SELECT 
    u.department,
    COUNT(u.id) as user_count  -- This will be wrong if users have multiple orders
FROM users u
JOIN orders o ON u.id = o.user_id
WHERE o.status = 'completed'
GROUP BY u.department;

-- ✅ Correct: Use DISTINCT or subquery
SELECT 
    u.department,
    COUNT(DISTINCT u.id) as user_count
FROM users u
JOIN orders o ON u.id = o.user_id
WHERE o.status = 'completed'
GROUP BY u.department;

-- ✅ Alternative: Subquery approach
SELECT 
    department,
    COUNT(*) as user_count
FROM users 
WHERE id IN (
    SELECT DISTINCT user_id 
    FROM orders 
    WHERE status = 'completed'
)
GROUP BY department;
```

### 2. Forgetting Soft Delete Filters

```sql
-- ❌ Wrong: Including deleted records
SELECT 
    DATE(created_at) as date,
    COUNT(*) as daily_signups
FROM users
GROUP BY DATE(created_at);

-- ✅ Correct: Filter out soft-deleted records
SELECT 
    DATE(created_at) as date,
    COUNT(*) as daily_signups
FROM users
WHERE deleted_at IS NULL
GROUP BY DATE(created_at);

-- ✅ Better: Create a view for active users
CREATE VIEW active_users AS
SELECT * FROM users WHERE deleted_at IS NULL;

-- Then use the view
SELECT 
    DATE(created_at) as date,
    COUNT(*) as daily_signups
FROM active_users
GROUP BY DATE(created_at);
```

### 3. Wrong Join Types

```sql
-- ❌ Wrong: INNER JOIN excludes users without orders
SELECT 
    u.department,
    COUNT(o.id) as total_orders,
    AVG(o.amount) as avg_order_value
FROM users u
JOIN orders o ON u.id = o.user_id  -- Missing users with no orders
GROUP BY u.department;

-- ✅ Correct: LEFT JOIN includes all users
SELECT 
    u.department,
    COUNT(o.id) as total_orders,
    AVG(o.amount) as avg_order_value,
    COUNT(DISTINCT u.id) as total_users,
    COUNT(DISTINCT CASE WHEN o.id IS NOT NULL THEN u.id END) as users_with_orders
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.deleted_at IS NULL
GROUP BY u.department;
```

### 4. Incorrect Aggregation

```sql
-- ❌ Wrong: Double counting due to multiple joins
SELECT 
    u.id,
    COUNT(o.id) as order_count,
    COUNT(r.id) as review_count  -- Wrong if user has multiple orders and reviews
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
LEFT JOIN reviews r ON u.id = r.user_id
GROUP BY u.id;

-- ✅ Correct: Separate aggregations
WITH user_orders AS (
    SELECT user_id, COUNT(*) as order_count
    FROM orders
    GROUP BY user_id
),
user_reviews AS (
    SELECT user_id, COUNT(*) as review_count
    FROM reviews
    GROUP BY user_id
)
SELECT 
    u.id,
    COALESCE(uo.order_count, 0) as order_count,
    COALESCE(ur.review_count, 0) as review_count
FROM users u
LEFT JOIN user_orders uo ON u.id = uo.user_id
LEFT JOIN user_reviews ur ON u.id = ur.user_id
WHERE u.deleted_at IS NULL;
```

## 🔧 Reducing Complex SQL in Applications

### 1. Create Views for Common Queries

```sql
-- Complex user analytics view
CREATE VIEW user_analytics AS
SELECT 
    u.id,
    u.email,
    u.created_at,
    u.department,
    COUNT(DISTINCT o.id) as total_orders,
    SUM(o.amount) as total_spent,
    AVG(o.amount) as avg_order_value,
    COUNT(DISTINCT DATE(o.created_at)) as days_with_orders,
    MAX(o.created_at) as last_order_date,
    COUNT(DISTINCT r.id) as total_reviews,
    AVG(r.rating) as avg_rating_given
FROM users u
LEFT JOIN orders o ON u.id = o.user_id AND o.status = 'completed'
LEFT JOIN reviews r ON u.id = r.user_id
WHERE u.deleted_at IS NULL
GROUP BY u.id, u.email, u.created_at, u.department;

-- Simple application queries
SELECT * FROM user_analytics WHERE total_spent > 1000;
SELECT department, AVG(total_spent) FROM user_analytics GROUP BY department;
```

### 2. Create Functions for Complex Logic

```sql
-- Function for customer lifetime value calculation
CREATE OR REPLACE FUNCTION calculate_customer_ltv(customer_id INTEGER, months INTEGER DEFAULT 12)
RETURNS DECIMAL(10,2) AS $$
DECLARE
    ltv DECIMAL(10,2);
BEGIN
    SELECT 
        COALESCE(
            SUM(amount) * 12.0 / GREATEST(
                EXTRACT(MONTH FROM AGE(MAX(created_at), MIN(created_at))), 1
            ), 0
        ) INTO ltv
    FROM orders 
    WHERE user_id = customer_id 
    AND status = 'completed'
    AND created_at >= CURRENT_DATE - (months || ' months')::INTERVAL;
    
    RETURN ltv;
END;
$$ LANGUAGE plpgsql;

-- Usage in applications
SELECT 
    id,
    email,
    calculate_customer_ltv(id) as lifetime_value
FROM users 
WHERE created_at >= CURRENT_DATE - INTERVAL '1 year';
```

### 3. Use CTEs for Readability

```sql
-- Complex cohort analysis made readable
WITH user_cohorts AS (
    SELECT 
        user_id,
        DATE_TRUNC('month', created_at) as cohort_month
    FROM users
    WHERE deleted_at IS NULL
),
monthly_activity AS (
    SELECT 
        o.user_id,
        DATE_TRUNC('month', o.created_at) as activity_month,
        SUM(o.amount) as monthly_revenue
    FROM orders o
    WHERE o.status = 'completed'
    GROUP BY o.user_id, DATE_TRUNC('month', o.created_at)
),
cohort_analysis AS (
    SELECT 
        uc.cohort_month,
        ma.activity_month,
        EXTRACT(EPOCH FROM (ma.activity_month - uc.cohort_month)) / 2592000 as period_number,
        COUNT(DISTINCT ma.user_id) as active_users,
        SUM(ma.monthly_revenue) as cohort_revenue
    FROM user_cohorts uc
    JOIN monthly_activity ma ON uc.user_id = ma.user_id
    WHERE ma.activity_month >= uc.cohort_month
    GROUP BY uc.cohort_month, ma.activity_month
)
SELECT 
    cohort_month,
    period_number,
    active_users,
    cohort_revenue,
    LAG(active_users) OVER (
        PARTITION BY cohort_month 
        ORDER BY period_number
    ) as prev_period_users,
    ROUND(
        active_users * 100.0 / FIRST_VALUE(active_users) OVER (
            PARTITION BY cohort_month 
            ORDER BY period_number
        ), 2
    ) as retention_rate
FROM cohort_analysis
ORDER BY cohort_month, period_number;
```

### 4. Lateral Joins for Dynamic Analysis

```sql
-- Use lateral joins for complex calculations
SELECT 
    u.id,
    u.email,
    u.created_at,
    recent_orders.order_count,
    recent_orders.total_spent,
    user_trends.spending_trend
FROM users u
LEFT JOIN LATERAL (
    SELECT 
        COUNT(*) as order_count,
        SUM(amount) as total_spent
    FROM orders o
    WHERE o.user_id = u.id 
    AND o.created_at >= CURRENT_DATE - INTERVAL '30 days'
    AND o.status = 'completed'
) recent_orders ON true
LEFT JOIN LATERAL (
    SELECT 
        CASE 
            WHEN COUNT(*) < 2 THEN 'Insufficient Data'
            WHEN REGR_SLOPE(amount, EXTRACT(EPOCH FROM created_at)) > 0 THEN 'Increasing'
            WHEN REGR_SLOPE(amount, EXTRACT(EPOCH FROM created_at)) < 0 THEN 'Decreasing'
            ELSE 'Stable'
        END as spending_trend
    FROM orders o
    WHERE o.user_id = u.id 
    AND o.created_at >= CURRENT_DATE - INTERVAL '180 days'
    AND o.status = 'completed'
) user_trends ON true
WHERE u.deleted_at IS NULL;
```

## 📊 Useful Analytics Patterns

### 1. Revenue Analytics

```sql
-- Comprehensive revenue dashboard
CREATE VIEW revenue_analytics AS
WITH daily_revenue AS (
    SELECT 
        DATE(created_at) as date,
        SUM(amount) as revenue,
        COUNT(*) as order_count,
        COUNT(DISTINCT user_id) as unique_customers
    FROM orders 
    WHERE status = 'completed'
    GROUP BY DATE(created_at)
),
revenue_trends AS (
    SELECT 
        *,
        LAG(revenue) OVER (ORDER BY date) as prev_day_revenue,
        AVG(revenue) OVER (
            ORDER BY date 
            ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
        ) as revenue_7day_avg,
        AVG(revenue) OVER (
            ORDER BY date 
            ROWS BETWEEN 29 PRECEDING AND CURRENT ROW
        ) as revenue_30day_avg
    FROM daily_revenue
)
SELECT 
    date,
    revenue,
    order_count,
    unique_customers,
    ROUND(revenue / order_count, 2) as avg_order_value,
    revenue_7day_avg,
    revenue_30day_avg,
    ROUND(
        (revenue - prev_day_revenue) * 100.0 / NULLIF(prev_day_revenue, 0), 2
    ) as day_over_day_pct
FROM revenue_trends
ORDER BY date DESC;
```

### 2. Customer Segmentation

```sql
-- RFM (Recency, Frequency, Monetary) analysis
CREATE VIEW customer_rfm AS
WITH customer_metrics AS (
    SELECT 
        user_id,
        MAX(created_at) as last_order_date,
        COUNT(*) as frequency,
        SUM(amount) as monetary_value,
        CURRENT_DATE - MAX(DATE(created_at)) as recency_days
    FROM orders
    WHERE status = 'completed'
    GROUP BY user_id
),
rfm_scores AS (
    SELECT 
        *,
        NTILE(5) OVER (ORDER BY recency_days) as recency_score,
        NTILE(5) OVER (ORDER BY frequency DESC) as frequency_score,
        NTILE(5) OVER (ORDER BY monetary_value DESC) as monetary_score
    FROM customer_metrics
)
SELECT 
    user_id,
    recency_days,
    frequency,
    monetary_value,
    recency_score,
    frequency_score,
    monetary_score,
    CASE 
        WHEN recency_score >= 4 AND frequency_score >= 4 AND monetary_score >= 4 THEN 'Champions'
        WHEN recency_score >= 3 AND frequency_score >= 3 AND monetary_score >= 3 THEN 'Loyal Customers'
        WHEN recency_score >= 3 AND frequency_score <= 2 THEN 'Potential Loyalists'
        WHEN recency_score <= 2 AND frequency_score >= 3 THEN 'At Risk'
        WHEN recency_score <= 2 AND frequency_score <= 2 AND monetary_score >= 3 THEN 'Cannot Lose Them'
        WHEN recency_score <= 2 AND frequency_score <= 2 THEN 'Hibernating'
        ELSE 'Others'
    END as customer_segment
FROM rfm_scores;
```

### 3. Product Performance

```sql
-- Product analytics with inventory insights
CREATE VIEW product_analytics AS
WITH product_sales AS (
    SELECT 
        p.id as product_id,
        p.name,
        p.category,
        p.price,
        COUNT(oi.id) as units_sold,
        SUM(oi.quantity * oi.price) as total_revenue,
        COUNT(DISTINCT o.user_id) as unique_buyers,
        AVG(r.rating) as avg_rating,
        COUNT(r.id) as review_count
    FROM products p
    LEFT JOIN order_items oi ON p.id = oi.product_id
    LEFT JOIN orders o ON oi.order_id = o.id AND o.status = 'completed'
    LEFT JOIN reviews r ON p.id = r.product_id
    WHERE p.deleted_at IS NULL
    GROUP BY p.id, p.name, p.category, p.price
)
SELECT 
    *,
    CASE 
        WHEN units_sold = 0 THEN 'No Sales'
        WHEN units_sold <= 10 THEN 'Low Performer'
        WHEN units_sold <= 50 THEN 'Medium Performer'
        ELSE 'High Performer'
    END as performance_category,
    ROUND(total_revenue / NULLIF(units_sold, 0), 2) as revenue_per_unit
FROM product_sales
ORDER BY total_revenue DESC;
```

## 🚀 Performance Optimization

### 1. Materialized Views for Heavy Analytics

```sql
-- Create materialized view for expensive analytics
CREATE MATERIALIZED VIEW monthly_user_metrics AS
SELECT 
    DATE_TRUNC('month', u.created_at) as month,
    u.department,
    COUNT(*) as new_users,
    COUNT(CASE WHEN o.user_id IS NOT NULL THEN u.id END) as users_with_orders,
    SUM(COALESCE(user_totals.total_spent, 0)) as total_revenue,
    AVG(COALESCE(user_totals.total_spent, 0)) as avg_revenue_per_user
FROM users u
LEFT JOIN LATERAL (
    SELECT SUM(amount) as total_spent
    FROM orders 
    WHERE user_id = u.id AND status = 'completed'
) user_totals ON true
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.deleted_at IS NULL
GROUP BY DATE_TRUNC('month', u.created_at), u.department;

-- Create index for fast queries
CREATE INDEX ON monthly_user_metrics (month, department);

-- Refresh schedule (can be automated)
REFRESH MATERIALIZED VIEW monthly_user_metrics;
```

### 2. Aggregation Tables

```sql
-- Pre-computed daily aggregations
CREATE TABLE daily_analytics (
    date DATE PRIMARY KEY,
    new_users INTEGER DEFAULT 0,
    total_orders INTEGER DEFAULT 0,
    total_revenue DECIMAL(15,2) DEFAULT 0,
    unique_customers INTEGER DEFAULT 0,
    avg_order_value DECIMAL(10,2) DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to update daily analytics
CREATE OR REPLACE FUNCTION refresh_daily_analytics(target_date DATE DEFAULT CURRENT_DATE)
RETURNS VOID AS $$
BEGIN
    INSERT INTO daily_analytics (
        date, new_users, total_orders, total_revenue, 
        unique_customers, avg_order_value
    )
    WITH daily_metrics AS (
        SELECT 
            target_date as date,
            (SELECT COUNT(*) FROM users WHERE DATE(created_at) = target_date AND deleted_at IS NULL) as new_users,
            (SELECT COUNT(*) FROM orders WHERE DATE(created_at) = target_date AND status = 'completed') as total_orders,
            (SELECT COALESCE(SUM(amount), 0) FROM orders WHERE DATE(created_at) = target_date AND status = 'completed') as total_revenue,
            (SELECT COUNT(DISTINCT user_id) FROM orders WHERE DATE(created_at) = target_date AND status = 'completed') as unique_customers
    )
    SELECT 
        date,
        new_users,
        total_orders,
        total_revenue,
        unique_customers,
        CASE WHEN total_orders > 0 THEN total_revenue / total_orders ELSE 0 END as avg_order_value
    FROM daily_metrics
    ON CONFLICT (date) DO UPDATE SET
        new_users = EXCLUDED.new_users,
        total_orders = EXCLUDED.total_orders,
        total_revenue = EXCLUDED.total_revenue,
        unique_customers = EXCLUDED.unique_customers,
        avg_order_value = EXCLUDED.avg_order_value,
        updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;
```

## 🎯 Best Practices

### Data Quality Checklist

1. **Always filter soft-deleted records**
2. **Use appropriate join types (LEFT vs INNER)**
3. **Handle NULL values explicitly**
4. **Validate date ranges and time zones**
5. **Use DISTINCT when dealing with potential duplicates**
6. **Test with realistic data volumes**

### Query Optimization

1. **Create covering indexes for analytics queries**
2. **Use window functions instead of self-joins**
3. **Leverage CTEs for complex logic**
4. **Consider materialized views for expensive calculations**
5. **Monitor query performance regularly**

### Code Organization

1. **Create reusable views for common metrics**
2. **Document complex analytical logic**
3. **Use consistent naming conventions**
4. **Version control analytics queries**
5. **Implement automated testing for critical metrics**

## 🔗 Related Resources

- **[Useful Analytics Recipes](http://www.silota.com/docs/recipes/)** - SQL patterns for analytics
- **[Window Functions](../query-patterns/window-functions.md)** - Advanced analytical functions
- **[Performance Optimization](../performance/README.md)** - Query optimization techniques
- **[Time Series Patterns](temporal.md)** - Time-based analytics

Building robust analytics requires attention to data quality, query performance, and maintainable code structure. These patterns provide a solid foundation for database-driven analytics systems.
