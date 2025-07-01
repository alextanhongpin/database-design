# SQL CASE Statement Patterns

Advanced patterns and best practices for using CASE statements effectively in SQL queries.

## Table of Contents

1. [Basic CASE Patterns](#basic-case-patterns)
2. [Conditional Aggregation](#conditional-aggregation)
3. [Data Transformation](#data-transformation)
4. [Sorting and Ordering](#sorting-and-ordering)
5. [Pivot Operations](#pivot-operations)
6. [Complex Conditional Logic](#complex-conditional-logic)
7. [Performance Considerations](#performance-considerations)
8. [Anti-Patterns](#anti-patterns)

## Basic CASE Patterns

### Simple CASE Expression

```sql
-- Basic boolean case
SELECT 
    id,
    name,
    CASE WHEN active THEN 'Active' ELSE 'Inactive' END as status
FROM users;

-- Multiple conditions
SELECT 
    id,
    name,
    CASE 
        WHEN age < 18 THEN 'Minor'
        WHEN age BETWEEN 18 AND 64 THEN 'Adult'
        WHEN age >= 65 THEN 'Senior'
        ELSE 'Unknown'
    END as age_group
FROM users;
```

### Range-Based Classification

```sql
-- User level classification based on points
WITH users_with_level AS (
    SELECT *, 
        CASE 
            WHEN points < 2 THEN 'Freshie'
            WHEN points < 10 THEN 'Rookie'
            WHEN points < 40 THEN 'Wonderkid'
            WHEN points < 125 THEN 'Prodigy'
            WHEN points < 250 THEN 'Genius'
            WHEN points < 750 THEN 'Master'
            WHEN points < 1500 THEN 'Grand Master'
            WHEN points < 3000 THEN 'Wizard'
            WHEN points < 6000 THEN 'God of Wisdom'
            WHEN points >= 6000 THEN 'Unicorn'
        END as level_name
    FROM user_profiles
),
levels AS (
    SELECT * 
    FROM (VALUES 
        ('Freshie', 1, 0), 
        ('Rookie', 2, 2),
        ('Wonderkid', 3, 10),
        ('Prodigy', 4, 40),
        ('Genius', 5, 125),
        ('Master', 6, 250),
        ('Grand Master', 7, 750),
        ('Wizard', 8, 1500),
        ('God of Wisdom', 9, 3000),
        ('Unicorn', 10, 6000)
    ) AS levels (name, level, min_points)
)
SELECT 
    COUNT(*) as total_users,
    levels.name as level_name, 
    (ARRAY_AGG(levels.level))[1] as level_rank,
    (ARRAY_AGG(levels.min_points))[1] as level_min_point,
    AVG(points) as level_avg_point,
    MAX(points) as level_max_point,
    MIN(points) as level_min_point
FROM users_with_level
JOIN levels ON (levels.name = users_with_level.level_name)
GROUP BY levels.name
ORDER BY level_rank DESC;
```

### Searched CASE vs Simple CASE

```sql
-- Simple CASE (matches exact values)
SELECT 
    name,
    CASE status
        WHEN 'A' THEN 'Active'
        WHEN 'I' THEN 'Inactive'
        WHEN 'P' THEN 'Pending'
        ELSE 'Unknown'
    END as status_description
FROM users;

-- Searched CASE (evaluates conditions)
SELECT 
    name,
    CASE 
        WHEN status = 'A' AND last_login > NOW() - INTERVAL '30 days' THEN 'Recently Active'
        WHEN status = 'A' THEN 'Active'
        WHEN status = 'I' THEN 'Inactive'
        ELSE 'Unknown'
    END as detailed_status
FROM users;
```

## Conditional Aggregation

### Cross-Tabulation with CASE

```sql
-- Count by categories
SELECT 
    department,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as active_count,
    COUNT(CASE WHEN status = 'inactive' THEN 1 END) as inactive_count,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_count,
    COUNT(*) as total_count
FROM employees
GROUP BY department;

-- Sum with conditions
SELECT 
    customer_id,
    SUM(CASE WHEN order_status = 'completed' THEN amount ELSE 0 END) as completed_revenue,
    SUM(CASE WHEN order_status = 'pending' THEN amount ELSE 0 END) as pending_revenue,
    AVG(CASE WHEN order_status = 'completed' THEN amount END) as avg_completed_order
FROM orders
GROUP BY customer_id;
```

### Age Group Analysis

```sql
SELECT 
    department,
    AVG(CASE WHEN age < 30 THEN salary END) as avg_salary_under_30,
    AVG(CASE WHEN age BETWEEN 30 AND 50 THEN salary END) as avg_salary_30_50,
    AVG(CASE WHEN age > 50 THEN salary END) as avg_salary_over_50,
    COUNT(CASE WHEN age < 30 THEN 1 END) as count_under_30,
    COUNT(CASE WHEN age BETWEEN 30 AND 50 THEN 1 END) as count_30_50,
    COUNT(CASE WHEN age > 50 THEN 1 END) as count_over_50
FROM employees
GROUP BY department;
```

## Data Transformation

### NULL Handling and Default Values

```sql
-- Replace NULL values
SELECT 
    id,
    CASE 
        WHEN name IS NULL OR name = '' THEN 'Unknown User'
        ELSE name
    END as display_name,
    CASE 
        WHEN email IS NULL THEN 'no-email@example.com'
        ELSE email
    END as contact_email
FROM users;

-- Complex NULL handling
SELECT 
    id,
    CASE 
        WHEN first_name IS NOT NULL AND last_name IS NOT NULL 
            THEN CONCAT(first_name, ' ', last_name)
        WHEN first_name IS NOT NULL 
            THEN first_name
        WHEN last_name IS NOT NULL 
            THEN last_name
        ELSE 'Anonymous'
    END as full_name
FROM users;
```

### Data Classification

```sql
-- Risk scoring
SELECT 
    customer_id,
    name,
    CASE 
        WHEN credit_score >= 750 THEN 'Low Risk'
        WHEN credit_score >= 650 THEN 'Medium Risk'
        WHEN credit_score >= 550 THEN 'High Risk'
        WHEN credit_score IS NULL THEN 'No Score'
        ELSE 'Very High Risk'
    END as risk_category,
    CASE 
        WHEN total_orders > 100 AND avg_order_value > 500 THEN 'VIP'
        WHEN total_orders > 50 THEN 'Premium'
        WHEN total_orders > 10 THEN 'Regular'
        ELSE 'New'
    END as customer_tier
FROM customers;
```

## Sorting and Ordering

### Custom Sort Orders

```sql
-- Priority-based sorting
SELECT * FROM tasks
ORDER BY 
    CASE priority
        WHEN 'urgent' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        ELSE 5
    END,
    created_at DESC;

-- Complex multi-field sorting
SELECT * FROM posts
ORDER BY 
    CASE WHEN is_pinned THEN 0 ELSE 1 END,
    CASE WHEN is_featured THEN 0 ELSE 1 END,
    CASE 
        WHEN status = 'published' THEN 1
        WHEN status = 'draft' THEN 2
        ELSE 3
    END,
    created_at DESC;
```

### Dynamic Sorting

```sql
-- Conditional column sorting (PostgreSQL)
SELECT *
FROM users
ORDER BY 
    CASE 
        WHEN @sort_by = 'name' THEN name
        WHEN @sort_by = 'email' THEN email
        ELSE created_at::text
    END;
```

## Pivot Operations

### Monthly Sales Pivot

```sql
SELECT 
    product_category,
    SUM(CASE WHEN EXTRACT(MONTH FROM order_date) = 1 THEN amount ELSE 0 END) as jan_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM order_date) = 2 THEN amount ELSE 0 END) as feb_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM order_date) = 3 THEN amount ELSE 0 END) as mar_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM order_date) = 4 THEN amount ELSE 0 END) as apr_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM order_date) = 5 THEN amount ELSE 0 END) as may_sales,
    SUM(CASE WHEN EXTRACT(MONTH FROM order_date) = 6 THEN amount ELSE 0 END) as jun_sales,
    SUM(amount) as total_sales
FROM orders
WHERE EXTRACT(YEAR FROM order_date) = 2024
GROUP BY product_category;
```

### Performance Metrics Pivot

```sql
SELECT 
    employee_id,
    name,
    AVG(CASE WHEN review_type = 'technical' THEN score END) as technical_avg,
    AVG(CASE WHEN review_type = 'communication' THEN score END) as communication_avg,
    AVG(CASE WHEN review_type = 'leadership' THEN score END) as leadership_avg,
    COUNT(CASE WHEN score >= 4 THEN 1 END) as high_scores,
    COUNT(CASE WHEN score < 3 THEN 1 END) as low_scores
FROM employee_reviews er
JOIN employees e ON er.employee_id = e.id
GROUP BY employee_id, name;
```

## Complex Conditional Logic

### Nested CASE Statements

```sql
SELECT 
    customer_id,
    order_total,
    CASE 
        WHEN customer_tier = 'VIP' THEN
            CASE 
                WHEN order_total > 1000 THEN order_total * 0.8  -- 20% discount
                WHEN order_total > 500 THEN order_total * 0.85   -- 15% discount
                ELSE order_total * 0.9                           -- 10% discount
            END
        WHEN customer_tier = 'Premium' THEN
            CASE 
                WHEN order_total > 500 THEN order_total * 0.9    -- 10% discount
                ELSE order_total * 0.95                          -- 5% discount
            END
        ELSE order_total
    END as discounted_total
FROM orders o
JOIN customers c ON o.customer_id = c.id;
```

### Business Rules Implementation

```sql
SELECT 
    loan_id,
    applicant_name,
    credit_score,
    income,
    debt_ratio,
    CASE 
        WHEN credit_score >= 750 AND debt_ratio <= 0.3 THEN 'Auto-Approved'
        WHEN credit_score >= 700 AND debt_ratio <= 0.4 AND income >= 50000 THEN 'Likely Approved'
        WHEN credit_score >= 650 AND debt_ratio <= 0.5 AND income >= 40000 THEN 'Manual Review'
        WHEN credit_score >= 600 AND debt_ratio <= 0.6 AND income >= 35000 THEN 'High Risk Review'
        ELSE 'Declined'
    END as approval_status,
    CASE 
        WHEN credit_score >= 750 THEN 3.5
        WHEN credit_score >= 700 THEN 4.0
        WHEN credit_score >= 650 THEN 5.5
        WHEN credit_score >= 600 THEN 7.0
        ELSE NULL
    END as suggested_rate
FROM loan_applications;
```

## Performance Considerations

### Index-Friendly CASE Usage

```sql
-- Good: Uses indexed column in WHERE, CASE in SELECT
SELECT 
    id,
    name,
    CASE 
        WHEN status = 'A' THEN 'Active'
        WHEN status = 'I' THEN 'Inactive'
    END as status_desc
FROM users
WHERE status IN ('A', 'I')  -- Uses index
ORDER BY created_at DESC;    -- Uses index

-- Avoid: CASE in WHERE clause prevents index usage
-- SELECT * FROM users
-- WHERE CASE WHEN status = 'A' THEN 'Active' ELSE 'Inactive' END = 'Active';
```

### Optimizing Conditional Aggregation

```sql
-- Efficient conditional counting
SELECT 
    department,
    COUNT(*) as total,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_count,
    -- More efficient than COUNT(CASE WHEN ...)
    SUM(CASE WHEN salary > 50000 THEN 1 ELSE 0 END) as high_earners
FROM employees
GROUP BY department;
```

## Anti-Patterns

### Avoid These Common Mistakes

```sql
-- ❌ Bad: Redundant CASE statements
SELECT 
    CASE WHEN age > 18 THEN 'Adult' ELSE 'Minor' END,
    CASE WHEN age > 18 THEN TRUE ELSE FALSE END  -- Redundant!
FROM users;

-- ✅ Good: Reuse logic or combine conditions
SELECT 
    CASE WHEN age > 18 THEN 'Adult' ELSE 'Minor' END as age_group,
    age > 18 as is_adult  -- Simple boolean expression
FROM users;

-- ❌ Bad: Too many nested CASE statements
SELECT 
    CASE 
        WHEN status = 'A' THEN 
            CASE WHEN priority = 1 THEN 'High Active' 
                 ELSE CASE WHEN priority = 2 THEN 'Med Active' 
                           ELSE 'Low Active' END 
            END
        ELSE 'Inactive'
    END
FROM tasks;

-- ✅ Good: Use lookup table or simplified logic
WITH priority_labels AS (
    SELECT 1 as priority, 'High' as label
    UNION ALL SELECT 2, 'Medium'
    UNION ALL SELECT 3, 'Low'
)
SELECT 
    t.id,
    CASE 
        WHEN t.status = 'A' THEN CONCAT(pl.label, ' Active')
        ELSE 'Inactive'
    END as status_description
FROM tasks t
LEFT JOIN priority_labels pl ON t.priority = pl.priority;
```

### Performance Anti-Patterns

```sql
-- ❌ Bad: CASE in WHERE clause
SELECT * FROM orders
WHERE CASE 
    WHEN status = 'completed' THEN amount > 100
    WHEN status = 'pending' THEN TRUE
    ELSE FALSE
END;

-- ✅ Good: Rewrite with OR conditions
SELECT * FROM orders
WHERE (status = 'completed' AND amount > 100)
   OR status = 'pending';
```

## Best Practices

1. **Keep it Simple**: Prefer simple boolean expressions over CASE when possible
2. **Use Indexes**: Avoid CASE statements in WHERE clauses that prevent index usage
3. **Handle NULLs**: Always consider NULL values in your CASE logic
4. **Order Matters**: Place most selective conditions first
5. **Use COALESCE**: For simple NULL replacement, COALESCE is cleaner than CASE
6. **Document Complex Logic**: Add comments for business rules implemented in CASE statements
7. **Test Edge Cases**: Verify behavior with NULL, empty strings, and boundary values

## Related Patterns

- [Conditional Queries](query.md)
- [Data Transformation](../patterns/data-changes.md)
- [Sorting Patterns](sorting.md)
- [Aggregation Patterns](../patterns/group-and-sort.md)
