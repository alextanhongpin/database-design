# Basic Analytics Patterns

Fundamental analytics patterns and techniques for database-driven insights and reporting.

## 🎯 Overview

Basic analytics patterns provide the foundation for:
- **Time Series Analysis** - Tracking changes over time
- **Statistical Functions** - Understanding data distributions
- **Ranking and Percentiles** - Comparative analysis
- **Trend Analysis** - Identifying patterns and growth

## 📊 Time Series Patterns

### Basic Time Series

```sql
-- Simple time series aggregation
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Daily event counts
SELECT 
    DATE(created_at) as date,
    COUNT(*) as event_count
FROM events 
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date;

-- Hourly patterns
SELECT 
    EXTRACT(HOUR FROM created_at) as hour,
    COUNT(*) as event_count,
    AVG(COUNT(*)) OVER () as avg_hourly_count
FROM events 
WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY EXTRACT(HOUR FROM created_at)
ORDER BY hour;
```

### Running Totals and Moving Averages

```sql
-- Running total of daily signups
SELECT 
    DATE(created_at) as date,
    COUNT(*) as daily_signups,
    SUM(COUNT(*)) OVER (ORDER BY DATE(created_at)) as cumulative_signups
FROM users 
GROUP BY DATE(created_at)
ORDER BY date;

-- 7-day moving average
SELECT 
    DATE(created_at) as date,
    COUNT(*) as daily_count,
    AVG(COUNT(*)) OVER (
        ORDER BY DATE(created_at) 
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ) as moving_avg_7day
FROM events 
GROUP BY DATE(created_at)
ORDER BY date;
```

## 📈 NTILE and Percentile Analysis

### NTILE for Quartiles and Deciles

```sql
-- Divide users into quartiles by activity level
WITH user_activity AS (
    SELECT 
        user_id,
        COUNT(*) as activity_count
    FROM events 
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
    GROUP BY user_id
)
SELECT 
    user_id,
    activity_count,
    NTILE(4) OVER (ORDER BY activity_count) as quartile,
    NTILE(10) OVER (ORDER BY activity_count) as decile
FROM user_activity
ORDER BY activity_count DESC;

-- Analyze quartile characteristics
WITH user_quartiles AS (
    SELECT 
        user_id,
        COUNT(*) as activity_count,
        NTILE(4) OVER (ORDER BY COUNT(*)) as quartile
    FROM events 
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
    GROUP BY user_id
)
SELECT 
    quartile,
    COUNT(*) as user_count,
    MIN(activity_count) as min_activity,
    MAX(activity_count) as max_activity,
    AVG(activity_count) as avg_activity,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY activity_count) as median_activity
FROM user_quartiles
GROUP BY quartile
ORDER BY quartile;
```

### Percentile Analysis

```sql
-- Revenue percentiles
WITH daily_revenue AS (
    SELECT 
        DATE(created_at) as date,
        SUM(amount) as revenue
    FROM orders 
    WHERE created_at >= CURRENT_DATE - INTERVAL '90 days'
    GROUP BY DATE(created_at)
)
SELECT 
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY revenue) as p25,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY revenue) as median,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY revenue) as p75,
    PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY revenue) as p90,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY revenue) as p95
FROM daily_revenue;

-- Identify outlier days
WITH daily_revenue AS (
    SELECT 
        DATE(created_at) as date,
        SUM(amount) as revenue
    FROM orders 
    WHERE created_at >= CURRENT_DATE - INTERVAL '90 days'
    GROUP BY DATE(created_at)
),
revenue_stats AS (
    SELECT 
        *,
        PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY revenue) OVER () as q1,
        PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY revenue) OVER () as q3
    FROM daily_revenue
)
SELECT 
    date,
    revenue,
    CASE 
        WHEN revenue < (q1 - 1.5 * (q3 - q1)) THEN 'Low Outlier'
        WHEN revenue > (q3 + 1.5 * (q3 - q1)) THEN 'High Outlier'
        ELSE 'Normal'
    END as outlier_status
FROM revenue_stats
ORDER BY date;
```

## 🔄 Growth and Trend Analysis

### Period-over-Period Comparison

```sql
-- Month-over-month growth
WITH monthly_metrics AS (
    SELECT 
        DATE_TRUNC('month', created_at) as month,
        COUNT(*) as user_count,
        SUM(COUNT(*)) OVER (ORDER BY DATE_TRUNC('month', created_at)) as cumulative_users
    FROM users 
    GROUP BY DATE_TRUNC('month', created_at)
)
SELECT 
    month,
    user_count,
    cumulative_users,
    LAG(user_count) OVER (ORDER BY month) as prev_month_count,
    ROUND(
        (user_count - LAG(user_count) OVER (ORDER BY month)) * 100.0 / 
        NULLIF(LAG(user_count) OVER (ORDER BY month), 0), 2
    ) as month_over_month_pct
FROM monthly_metrics
ORDER BY month;
```

### Cohort Analysis Basics

```sql
-- Simple cohort analysis - user retention
WITH user_cohorts AS (
    SELECT 
        user_id,
        DATE_TRUNC('month', created_at) as cohort_month
    FROM users
),
user_activities AS (
    SELECT 
        e.user_id,
        uc.cohort_month,
        DATE_TRUNC('month', e.created_at) as activity_month,
        EXTRACT(EPOCH FROM (DATE_TRUNC('month', e.created_at) - uc.cohort_month)) / 2592000 as period_number
    FROM events e
    JOIN user_cohorts uc ON e.user_id = uc.user_id
)
SELECT 
    cohort_month,
    period_number,
    COUNT(DISTINCT user_id) as active_users
FROM user_activities
WHERE period_number >= 0
GROUP BY cohort_month, period_number
ORDER BY cohort_month, period_number;
```

## 📊 Distribution Analysis

### Frequency Distributions

```sql
-- Order value distribution
WITH order_buckets AS (
    SELECT 
        amount,
        CASE 
            WHEN amount < 25 THEN '0-25'
            WHEN amount < 50 THEN '25-50'
            WHEN amount < 100 THEN '50-100'
            WHEN amount < 200 THEN '100-200'
            ELSE '200+'
        END as amount_bucket
    FROM orders
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
)
SELECT 
    amount_bucket,
    COUNT(*) as order_count,
    COUNT(*) * 100.0 / SUM(COUNT(*)) OVER () as percentage
FROM order_buckets
GROUP BY amount_bucket
ORDER BY MIN(amount);

-- User activity distribution
SELECT 
    activity_level,
    COUNT(*) as user_count,
    COUNT(*) * 100.0 / SUM(COUNT(*)) OVER () as percentage
FROM (
    SELECT 
        user_id,
        CASE 
            WHEN COUNT(*) = 0 THEN 'Inactive'
            WHEN COUNT(*) <= 5 THEN 'Low'
            WHEN COUNT(*) <= 20 THEN 'Medium'
            WHEN COUNT(*) <= 50 THEN 'High'
            ELSE 'Very High'
        END as activity_level
    FROM events 
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
    GROUP BY user_id
) user_activity_levels
GROUP BY activity_level;
```

## 🎯 Performance Optimization

### Efficient Analytics Queries

```sql
-- Pre-aggregated tables for common analytics
CREATE TABLE daily_metrics (
    date DATE PRIMARY KEY,
    new_users INTEGER DEFAULT 0,
    total_orders INTEGER DEFAULT 0,
    total_revenue DECIMAL(15,2) DEFAULT 0,
    active_users INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to maintain daily metrics
CREATE OR REPLACE FUNCTION update_daily_metrics()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO daily_metrics (date, new_users)
    VALUES (DATE(NEW.created_at), 1)
    ON CONFLICT (date) DO UPDATE SET
        new_users = daily_metrics.new_users + 1,
        updated_at = CURRENT_TIMESTAMP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_daily_metrics_trigger
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_daily_metrics();
```

### Index Strategies for Analytics

```sql
-- Indexes for time-series queries
CREATE INDEX idx_events_created_at ON events (created_at);
CREATE INDEX idx_events_user_created ON events (user_id, created_at);
CREATE INDEX idx_events_type_created ON events (event_type, created_at);

-- Covering indexes for common analytics queries
CREATE INDEX idx_orders_analytics_covering 
ON orders (created_at, customer_id, amount, status);

-- Partial indexes for active data
CREATE INDEX idx_recent_events 
ON events (user_id, created_at) 
WHERE created_at >= CURRENT_DATE - INTERVAL '90 days';
```

## 🔧 Common Analytics Functions

### Window Functions for Analytics

```sql
-- Ranking and comparative analysis
SELECT 
    user_id,
    total_spent,
    RANK() OVER (ORDER BY total_spent DESC) as spending_rank,
    DENSE_RANK() OVER (ORDER BY total_spent DESC) as dense_rank,
    ROW_NUMBER() OVER (ORDER BY total_spent DESC) as row_num,
    PERCENT_RANK() OVER (ORDER BY total_spent) as percentile_rank
FROM (
    SELECT 
        customer_id as user_id,
        SUM(amount) as total_spent
    FROM orders 
    WHERE created_at >= CURRENT_DATE - INTERVAL '365 days'
    GROUP BY customer_id
) user_spending
ORDER BY total_spent DESC;

-- Lead and lag for trend analysis
SELECT 
    date,
    daily_revenue,
    LAG(daily_revenue, 1) OVER (ORDER BY date) as prev_day_revenue,
    LEAD(daily_revenue, 1) OVER (ORDER BY date) as next_day_revenue,
    daily_revenue - LAG(daily_revenue, 1) OVER (ORDER BY date) as day_over_day_change
FROM (
    SELECT 
        DATE(created_at) as date,
        SUM(amount) as daily_revenue
    FROM orders
    GROUP BY DATE(created_at)
) daily_totals
ORDER BY date;
```

## 🎯 Best Practices

### Data Quality for Analytics

1. **Handle Missing Data**
   ```sql
   -- Fill gaps in time series
   WITH date_range AS (
       SELECT generate_series(
           CURRENT_DATE - INTERVAL '30 days',
           CURRENT_DATE,
           INTERVAL '1 day'
       )::date as date
   )
   SELECT 
       dr.date,
       COALESCE(dm.new_users, 0) as new_users,
       COALESCE(dm.total_revenue, 0) as total_revenue
   FROM date_range dr
   LEFT JOIN daily_metrics dm ON dr.date = dm.date
   ORDER BY dr.date;
   ```

2. **Validate Data Consistency**
   ```sql
   -- Check for data anomalies
   SELECT 
       date,
       new_users,
       CASE 
           WHEN new_users > (AVG(new_users) OVER () + 3 * STDDEV(new_users) OVER ()) 
           THEN 'Potential Outlier High'
           WHEN new_users < (AVG(new_users) OVER () - 3 * STDDEV(new_users) OVER ()) 
           THEN 'Potential Outlier Low'
           ELSE 'Normal'
       END as anomaly_flag
   FROM daily_metrics
   WHERE date >= CURRENT_DATE - INTERVAL '90 days'
   ORDER BY date;
   ```

3. **Performance Monitoring**
   ```sql
   -- Monitor query performance
   SELECT 
       query,
       calls,
       total_time,
       mean_time,
       rows
   FROM pg_stat_statements
   WHERE query LIKE '%analytics%' OR query LIKE '%GROUP BY%'
   ORDER BY total_time DESC;
   ```

These basic analytics patterns provide the foundation for more advanced analytics and reporting systems, enabling data-driven decision making and business insights.
