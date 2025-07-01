# Database Views

Database views are virtual tables that provide a way to present data from one or more tables in a specific format. They are powerful tools for data abstraction, security, and simplifying complex queries.

## Table of Contents

1. [Basic View Concepts](#basic-view-concepts)
2. [Simple Views](#simple-views)
3. [Complex Views](#complex-views)
4. [Updatable Views](#updatable-views)
5. [Security Views](#security-views)
6. [Performance Considerations](#performance-considerations)
7. [Best Practices](#best-practices)
8. [Anti-Patterns](#anti-patterns)

## Basic View Concepts

### What are Views?

Views are virtual tables that don't store data themselves but display data from underlying tables. They are defined by SQL queries and can be used like regular tables in most contexts.

**Benefits:**
- Data abstraction and simplified access
- Security through controlled data exposure
- Consistent business logic application
- Simplified complex joins and calculations
- Backwards compatibility when schema changes

**Limitations:**
- Performance overhead for complex views
- Update restrictions on complex views
- Dependency management complexity

## Simple Views

### Basic Single-Table View

```sql
-- Simple filtering view
CREATE VIEW active_users AS
SELECT 
    id,
    username,
    email,
    created_at,
    last_login
FROM users
WHERE status = 'active' 
    AND deleted_at IS NULL;

-- Usage
SELECT * FROM active_users WHERE last_login > '2024-01-01';
```

### Column Transformation View

```sql
-- Data transformation and formatting
CREATE VIEW user_profiles AS
SELECT 
    id,
    CONCAT(first_name, ' ', last_name) AS full_name,
    LOWER(email) AS email,
    DATE_FORMAT(created_at, '%Y-%m-%d') AS registration_date,
    TIMESTAMPDIFF(YEAR, date_of_birth, CURDATE()) AS age,
    CASE 
        WHEN status = 'A' THEN 'Active'
        WHEN status = 'I' THEN 'Inactive'
        WHEN status = 'S' THEN 'Suspended'
        ELSE 'Unknown'
    END AS status_description
FROM users
WHERE deleted_at IS NULL;
```

## Complex Views

### Multi-Table Join Views

```sql
-- User address view with proper joins
CREATE VIEW user_addresses AS
SELECT 
    u.id AS user_id,
    u.username,
    u.email,
    a.id AS address_id,
    a.street_address,
    a.city,
    a.state,
    a.postal_code,
    a.country,
    a.address_type,
    a.is_primary
FROM users u
INNER JOIN addresses a ON u.id = a.user_id
WHERE u.status = 'active' 
    AND u.deleted_at IS NULL
    AND a.deleted_at IS NULL;

-- Order summary view
CREATE VIEW order_summaries AS
SELECT 
    o.id AS order_id,
    o.order_number,
    o.order_date,
    o.status AS order_status,
    u.id AS customer_id,
    u.username AS customer_name,
    u.email AS customer_email,
    COUNT(oi.id) AS item_count,
    SUM(oi.quantity) AS total_quantity,
    SUM(oi.price * oi.quantity) AS subtotal,
    o.tax_amount,
    o.shipping_amount,
    (SUM(oi.price * oi.quantity) + o.tax_amount + o.shipping_amount) AS total_amount
FROM orders o
JOIN users u ON o.customer_id = u.id
LEFT JOIN order_items oi ON o.id = oi.order_id
WHERE o.deleted_at IS NULL
GROUP BY o.id, o.order_number, o.order_date, o.status, u.id, u.username, u.email, o.tax_amount, o.shipping_amount;
```

### Aggregation Views

```sql
-- Customer analytics view
CREATE VIEW customer_analytics AS
SELECT 
    c.id AS customer_id,
    c.username,
    c.email,
    c.created_at AS registration_date,
    COUNT(DISTINCT o.id) AS total_orders,
    COALESCE(SUM(o.total_amount), 0) AS lifetime_value,
    COALESCE(AVG(o.total_amount), 0) AS average_order_value,
    MIN(o.order_date) AS first_order_date,
    MAX(o.order_date) AS last_order_date,
    CASE 
        WHEN MAX(o.order_date) > DATE_SUB(NOW(), INTERVAL 30 DAY) THEN 'Active'
        WHEN MAX(o.order_date) > DATE_SUB(NOW(), INTERVAL 90 DAY) THEN 'At Risk'
        WHEN MAX(o.order_date) IS NOT NULL THEN 'Inactive'
        ELSE 'Never Ordered'
    END AS customer_status,
    CASE 
        WHEN COALESCE(SUM(o.total_amount), 0) > 5000 THEN 'VIP'
        WHEN COALESCE(SUM(o.total_amount), 0) > 1000 THEN 'Premium'
        WHEN COALESCE(SUM(o.total_amount), 0) > 100 THEN 'Regular'
        ELSE 'Bronze'
    END AS customer_tier
FROM users c
LEFT JOIN orders o ON c.id = o.customer_id AND o.status = 'completed'
WHERE c.deleted_at IS NULL
GROUP BY c.id, c.username, c.email, c.created_at;
```

### Time-Series Views

```sql
-- Daily sales metrics view
CREATE VIEW daily_sales_metrics AS
SELECT 
    DATE(order_date) AS sale_date,
    COUNT(DISTINCT id) AS order_count,
    COUNT(DISTINCT customer_id) AS unique_customers,
    SUM(total_amount) AS daily_revenue,
    AVG(total_amount) AS average_order_value,
    MIN(total_amount) AS min_order_value,
    MAX(total_amount) AS max_order_value,
    SUM(CASE WHEN status = 'completed' THEN total_amount ELSE 0 END) AS completed_revenue,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) AS completed_orders,
    COUNT(CASE WHEN status = 'cancelled' THEN 1 END) AS cancelled_orders
FROM orders
WHERE deleted_at IS NULL
GROUP BY DATE(order_date);
```

## Updatable Views

### Simple Updatable Views

```sql
-- Updatable view (single table, no aggregation)
CREATE VIEW editable_user_profiles AS
SELECT 
    id,
    username,
    email,
    first_name,
    last_name,
    phone,
    date_of_birth
FROM users
WHERE status = 'active' 
    AND deleted_at IS NULL;

-- These operations work on updatable views
INSERT INTO editable_user_profiles (username, email, first_name, last_name)
VALUES ('newuser', 'new@example.com', 'New', 'User');

UPDATE editable_user_profiles 
SET phone = '+1-555-0123' 
WHERE username = 'newuser';

DELETE FROM editable_user_profiles 
WHERE username = 'newuser';
```

### View Update Restrictions

```sql
-- Example of non-updatable view (multiple tables)
CREATE VIEW user_order_details AS
SELECT 
    u.id AS user_id,
    u.username,
    o.id AS order_id,
    o.order_date,
    o.total_amount
FROM users u
JOIN orders o ON u.id = o.customer_id;

-- This will fail: "Can not modify more than one base table through a join view"
-- UPDATE user_order_details SET username = 'newname', total_amount = 100.00 WHERE user_id = 1;

-- These individual updates work:
UPDATE user_order_details SET username = 'newname' WHERE user_id = 1;
UPDATE user_order_details SET total_amount = 100.00 WHERE order_id = 1;
```

### INSTEAD OF Triggers for Complex Updates

```sql
-- PostgreSQL example with INSTEAD OF trigger
CREATE VIEW user_address_edit AS
SELECT 
    u.id AS user_id,
    u.username,
    u.email,
    a.id AS address_id,
    a.street_address,
    a.city,
    a.country
FROM users u
JOIN addresses a ON u.id = a.user_id;

-- Create trigger function
CREATE OR REPLACE FUNCTION update_user_address()
RETURNS TRIGGER AS $$
BEGIN
    -- Update user table
    UPDATE users 
    SET username = NEW.username, email = NEW.email
    WHERE id = NEW.user_id;
    
    -- Update address table
    UPDATE addresses
    SET street_address = NEW.street_address, 
        city = NEW.city, 
        country = NEW.country
    WHERE id = NEW.address_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
CREATE TRIGGER user_address_update_trigger
    INSTEAD OF UPDATE ON user_address_edit
    FOR EACH ROW
    EXECUTE FUNCTION update_user_address();
```

## Security Views

### Data Masking Views

```sql
-- Sensitive data protection view
CREATE VIEW public_user_info AS
SELECT 
    id,
    username,
    CONCAT(LEFT(first_name, 1), REPEAT('*', LENGTH(first_name) - 1)) AS first_name,
    CONCAT(LEFT(last_name, 1), REPEAT('*', LENGTH(last_name) - 1)) AS last_name,
    CONCAT(LEFT(email, 3), '***@', SUBSTRING_INDEX(email, '@', -1)) AS masked_email,
    DATE_FORMAT(created_at, '%Y-%m') AS registration_month
FROM users
WHERE status = 'active';

-- Row-level security view
CREATE VIEW department_users AS
SELECT 
    id,
    username,
    email,
    department_id,
    created_at
FROM users u
WHERE u.department_id = GET_CURRENT_USER_DEPARTMENT()  -- Custom function
    AND u.status = 'active';
```

### Role-Based Views

```sql
-- Manager view with additional sensitive data
CREATE VIEW manager_user_view AS
SELECT 
    id,
    username,
    email,
    first_name,
    last_name,
    phone,
    salary,
    department_id,
    hire_date,
    performance_rating
FROM users
WHERE status = 'active';

-- Employee view with limited data
CREATE VIEW employee_user_view AS
SELECT 
    id,
    username,
    email,
    first_name,
    last_name,
    department_id,
    hire_date
FROM users
WHERE status = 'active';

-- Grant appropriate permissions
GRANT SELECT ON manager_user_view TO manager_role;
GRANT SELECT ON employee_user_view TO employee_role;
```

## Performance Considerations

### Indexed Views (SQL Server)

```sql
-- SQL Server indexed view example
CREATE VIEW dbo.ProductSales
WITH SCHEMABINDING AS
SELECT 
    p.ProductID,
    p.ProductName,
    SUM(od.Quantity) AS TotalQuantity,
    SUM(od.Quantity * od.UnitPrice) AS TotalSales,
    COUNT_BIG(*) AS OrderCount
FROM dbo.Products p
JOIN dbo.OrderDetails od ON p.ProductID = od.ProductID
GROUP BY p.ProductID, p.ProductName;

-- Create clustered index to make it a materialized view
CREATE UNIQUE CLUSTERED INDEX IX_ProductSales_ProductID 
ON dbo.ProductSales (ProductID);
```

### View Optimization Tips

```sql
-- Good: Use covering indexes on base tables
CREATE INDEX idx_orders_customer_date_status 
ON orders (customer_id, order_date, status) 
INCLUDE (total_amount);

-- Good: Filter early in the view
CREATE VIEW recent_active_orders AS
SELECT 
    id,
    customer_id,
    order_date,
    total_amount,
    status
FROM orders
WHERE order_date >= DATE_SUB(NOW(), INTERVAL 30 DAY)
    AND status IN ('pending', 'processing', 'completed')
    AND deleted_at IS NULL;

-- Avoid: Complex calculations in frequently used views
-- Consider using materialized views or pre-computed columns instead
```

## Best Practices

### 1. Naming Conventions

```sql
-- Use clear, descriptive names
CREATE VIEW active_customer_orders AS ...;      -- Good
CREATE VIEW v_aco AS ...;                      -- Bad

-- Use consistent prefixes for different view types
CREATE VIEW rpt_monthly_sales AS ...;          -- Report view
CREATE VIEW sec_masked_customers AS ...;       -- Security view
CREATE VIEW api_user_profiles AS ...;          -- API view
```

### 2. Documentation and Comments

```sql
-- Document complex views
CREATE VIEW customer_lifetime_analytics AS
-- Purpose: Provides customer analytics for business intelligence
-- Dependencies: users, orders, order_items tables
-- Updates: Refreshed nightly via ETL process
-- Owner: Analytics Team
SELECT 
    u.id AS customer_id,
    u.email,
    -- Customer registration cohort for retention analysis
    DATE_FORMAT(u.created_at, '%Y-%m') AS registration_cohort,
    -- Lifetime metrics
    COUNT(DISTINCT o.id) AS total_orders,
    COALESCE(SUM(o.total_amount), 0) AS lifetime_value
    -- ... rest of query
FROM users u
LEFT JOIN orders o ON u.id = o.customer_id;
```

### 3. Dependency Management

```sql
-- Create views in dependency order
-- 1. Base views first
CREATE VIEW clean_users AS
SELECT * FROM users WHERE deleted_at IS NULL;

-- 2. Views that depend on base views
CREATE VIEW user_profiles AS
SELECT 
    id,
    CONCAT(first_name, ' ', last_name) AS full_name,
    email
FROM clean_users;

-- 3. Complex views that depend on multiple views
CREATE VIEW user_analytics AS
SELECT 
    up.id,
    up.full_name,
    up.email,
    COUNT(o.id) AS order_count
FROM user_profiles up
LEFT JOIN orders o ON up.id = o.customer_id
GROUP BY up.id, up.full_name, up.email;
```

## Anti-Patterns

### Common Mistakes to Avoid

```sql
-- ❌ Bad: Overly complex views that are hard to maintain
CREATE VIEW monster_view AS
SELECT 
    -- 50+ columns with complex calculations
    -- Multiple subqueries and window functions
    -- Joins to 10+ tables
    ...
FROM table1 t1
JOIN table2 t2 ON ...
-- ... many more joins
WHERE -- complex conditions
-- This becomes a maintenance nightmare

-- ✅ Good: Break into smaller, focused views
CREATE VIEW customer_base AS ...;
CREATE VIEW order_metrics AS ...;
CREATE VIEW customer_order_summary AS
SELECT cb.*, om.order_count, om.total_spent
FROM customer_base cb
LEFT JOIN order_metrics om ON cb.customer_id = om.customer_id;

-- ❌ Bad: Views without proper filtering
CREATE VIEW all_user_data AS
SELECT * FROM users;  -- Exposes all data, including deleted records

-- ✅ Good: Views with appropriate filters
CREATE VIEW active_users AS
SELECT * FROM users 
WHERE status = 'active' AND deleted_at IS NULL;

-- ❌ Bad: Updatable views without considering side effects
CREATE VIEW user_summary AS
SELECT id, username, status FROM users;
-- Updates to this view affect the base table directly

-- ✅ Good: Use INSTEAD OF triggers for controlled updates
```

### Performance Anti-Patterns

```sql
-- ❌ Bad: Views that prevent index usage
CREATE VIEW slow_user_search AS
SELECT 
    id,
    UPPER(username) AS username,  -- Function prevents index usage
    email
FROM users;

-- ✅ Good: Let the application handle transformations
CREATE VIEW fast_user_search AS
SELECT id, username, email FROM users;
-- Apply UPPER() in the application or query that uses the view

-- ❌ Bad: Nested views creating query complexity
CREATE VIEW view1 AS SELECT ... FROM big_table;
CREATE VIEW view2 AS SELECT ... FROM view1;
CREATE VIEW view3 AS SELECT ... FROM view2;
-- Each level adds complexity

-- ✅ Good: Direct views or materialized views for complex cases
```

## Related Patterns

- [Materialized Views](materialized.md)
- [Query Optimization](query.md)
- [Security Patterns](../authorization/README.md)
- [Data Access Patterns](../patterns/README.md)
