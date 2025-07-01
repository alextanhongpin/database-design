# Existence Checking Patterns

Efficiently checking for the existence of data is a fundamental database operation. This guide covers different approaches, performance considerations, and best practices for existence queries.

## 🎯 Core Patterns

### 1. EXISTS Clause (Recommended)

**Best for**: General existence checking with optimal performance

```sql
-- Basic existence check
SELECT EXISTS(
    SELECT 1 FROM users 
    WHERE email = 'john@example.com'
);

-- With LIMIT 1 for safety (though EXISTS already stops at first match)
SELECT EXISTS(
    SELECT 1 FROM orders 
    WHERE user_id = '123' AND status = 'pending'
    LIMIT 1
);

-- Complex existence with joins
SELECT EXISTS(
    SELECT 1 FROM orders o
    JOIN order_items oi ON o.id = oi.order_id
    WHERE o.user_id = '123' 
    AND oi.product_id = '456'
);
```

### 2. COUNT-Based Existence

**Best for**: When you also need the count

```sql
-- Simple count check
SELECT COUNT(*) > 0 AS exists_flag
FROM users 
WHERE status = 'active';

-- More efficient with LIMIT 1
SELECT COUNT(*) > 0 AS exists_flag
FROM (
    SELECT 1 FROM large_table 
    WHERE complex_condition = true
    LIMIT 1
) AS subquery;

-- Get both existence and count
SELECT 
    COUNT(*) AS total_count,
    COUNT(*) > 0 AS exists_flag
FROM user_sessions 
WHERE expires_at > NOW();
```

### 3. Conditional Existence Patterns

```sql
-- Check multiple conditions
SELECT 
    EXISTS(SELECT 1 FROM users WHERE role = 'admin') AS has_admin,
    EXISTS(SELECT 1 FROM users WHERE status = 'banned') AS has_banned,
    EXISTS(SELECT 1 FROM users WHERE last_login > NOW() - INTERVAL '1 day') AS has_recent_login;

-- Existence with aggregation
SELECT 
    category_id,
    COUNT(*) AS product_count,
    EXISTS(
        SELECT 1 FROM products p2 
        WHERE p2.category_id = p.category_id 
        AND p2.in_stock = true
    ) AS has_stock
FROM products p
GROUP BY category_id;
```

## 🚀 Performance Optimization

### Index Usage for Existence Checks

```sql
-- Ensure proper indexing for existence queries
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_user_status ON orders(user_id, status);
CREATE INDEX idx_user_sessions_expires ON user_sessions(expires_at);

-- Partial indexes for common existence patterns
CREATE INDEX idx_active_users ON users(id) WHERE status = 'active';
CREATE INDEX idx_pending_orders ON orders(user_id) WHERE status = 'pending';
```

### Existence Subquery Optimization

```sql
-- Efficient existence check in WHERE clause
SELECT u.id, u.name
FROM users u
WHERE EXISTS(
    SELECT 1 FROM orders o
    WHERE o.user_id = u.id
    AND o.created_at > NOW() - INTERVAL '30 days'
);

-- Using NOT EXISTS for finding missing relationships
SELECT p.id, p.name
FROM products p
WHERE NOT EXISTS(
    SELECT 1 FROM order_items oi
    WHERE oi.product_id = p.id
);
```

## 🏗️ Advanced Existence Patterns

### 1. Hierarchical Existence

```sql
-- Check if category has any active products (recursive)
WITH RECURSIVE category_tree AS (
    SELECT id, name, parent_id, 0 as level
    FROM categories 
    WHERE id = '123'
    
    UNION ALL
    
    SELECT c.id, c.name, c.parent_id, ct.level + 1
    FROM categories c
    JOIN category_tree ct ON c.parent_id = ct.id
)
SELECT EXISTS(
    SELECT 1 FROM products p
    JOIN category_tree ct ON p.category_id = ct.id
    WHERE p.status = 'active'
) AS has_active_products;
```

### 2. Time-Window Existence

```sql
-- Check for events within different time windows
SELECT 
    user_id,
    EXISTS(
        SELECT 1 FROM user_activities 
        WHERE user_id = u.id 
        AND activity_type = 'login'
        AND created_at > NOW() - INTERVAL '1 hour'
    ) AS active_in_last_hour,
    EXISTS(
        SELECT 1 FROM user_activities 
        WHERE user_id = u.id 
        AND activity_type = 'purchase'
        AND created_at > NOW() - INTERVAL '7 days'
    ) AS purchased_this_week
FROM users u;
```

### 3. Conditional Existence with CASE

```sql
-- Complex existence logic
SELECT 
    order_id,
    CASE 
        WHEN EXISTS(
            SELECT 1 FROM payments 
            WHERE order_id = o.id AND status = 'completed'
        ) THEN 'paid'
        WHEN EXISTS(
            SELECT 1 FROM payments 
            WHERE order_id = o.id AND status = 'pending'
        ) THEN 'payment_pending'
        ELSE 'unpaid'
    END AS payment_status
FROM orders o;
```

## 🔍 Existence in Different Contexts

### Application-Level Existence Checks

```sql
-- Before insert operations
DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM users WHERE email = 'new@example.com') THEN
        RAISE EXCEPTION 'User with email already exists';
    END IF;
    
    INSERT INTO users (email, name) VALUES ('new@example.com', 'New User');
END $$;

-- Conditional updates
UPDATE products 
SET status = 'discontinued'
WHERE id = '123'
AND EXISTS(
    SELECT 1 FROM product_replacements 
    WHERE old_product_id = '123'
);
```

### Bulk Existence Operations

```sql
-- Check existence for multiple values
SELECT 
    value,
    EXISTS(SELECT 1 FROM products WHERE sku = value) AS exists_flag
FROM (VALUES 
    ('SKU001'),
    ('SKU002'), 
    ('SKU003')
) AS input_values(value);

-- Find missing relationships in bulk
SELECT u.id, u.email
FROM users u
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE up.user_id IS NULL; -- Users without profiles
```

## ⚠️ Common Pitfalls

### 1. Inefficient COUNT Usage
```sql
-- ❌ Inefficient - counts all rows
SELECT COUNT(*) > 0 FROM large_table WHERE condition = true;

-- ✅ Efficient - stops at first match
SELECT EXISTS(SELECT 1 FROM large_table WHERE condition = true);
```

### 2. Missing LIMIT in Subqueries
```sql
-- ❌ Could scan many rows unnecessarily
SELECT EXISTS(
    SELECT user_id FROM user_activities 
    WHERE activity_type = 'login'
);

-- ✅ Limits scanning (though EXISTS already optimizes this)
SELECT EXISTS(
    SELECT 1 FROM user_activities 
    WHERE activity_type = 'login'
    LIMIT 1
);
```

### 3. Incorrect NULL Handling
```sql
-- ❌ Won't match NULL values
SELECT EXISTS(SELECT 1 FROM users WHERE middle_name = 'value');

-- ✅ Explicit NULL handling
SELECT EXISTS(
    SELECT 1 FROM users 
    WHERE middle_name = 'value' OR middle_name IS NULL
);
```

## 📊 Performance Comparison

| Pattern | Use Case | Performance | Notes |
|---------|----------|-------------|-------|
| `EXISTS` | General existence | ⭐⭐⭐⭐⭐ | Optimal, stops at first match |
| `COUNT(*) > 0` | Need count too | ⭐⭐⭐ | Scans all matching rows |
| `COUNT(*) > 0 LIMIT 1` | Safety net | ⭐⭐⭐⭐ | Good compromise |
| `LEFT JOIN IS NULL` | Missing relationships | ⭐⭐⭐⭐ | Efficient for anti-joins |
| `IN (SELECT ...)` | Value membership | ⭐⭐⭐ | Can be slower than EXISTS |

## 🎯 Best Practices

1. **Use EXISTS for pure existence checks** - It's optimized to stop at the first match
2. **Add proper indexes** - Ensure WHERE clause columns are indexed
3. **Use LIMIT 1 defensively** - Adds safety for complex subqueries
4. **Avoid SELECT *** - Use `SELECT 1` or `SELECT NULL` in EXISTS clauses
5. **Consider partial indexes** - For common existence patterns
6. **Profile your queries** - Use EXPLAIN ANALYZE to verify performance
7. **Handle NULLs explicitly** - Be clear about NULL behavior in conditions

## 🔗 References

- [PostgreSQL EXISTS Documentation](https://www.postgresql.org/docs/current/functions-subquery.html)
- [SQL Performance Explained - EXISTS vs IN](https://use-the-index-luke.com/sql/where-clause/functions/exists)
- [MySQL EXISTS Optimization](https://dev.mysql.com/doc/refman/8.0/en/exists-and-not-exists-subqueries.html)
