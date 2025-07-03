# Advanced UPDATE Patterns

Comprehensive patterns for performing efficient and safe UPDATE operations in SQL databases.

## Table of Contents

1. [Conditional Updates](#conditional-updates)
2. [Bulk Updates](#bulk-updates)
3. [Upsert Operations](#upsert-operations)
4. [Atomic Updates](#atomic-updates)
5. [Update with Joins](#update-with-joins)
6. [Performance Optimization](#performance-optimization)
7. [Best Practices](#best-practices)
8. [Anti-Patterns](#anti-patterns)

## Conditional Updates

### Update Only When Values Change

Prevent unnecessary updates by checking if values actually change:

```sql
-- Basic conditional update using COALESCE
UPDATE users SET
    name = COALESCE(@param_name, name),
    email = COALESCE(@param_email, email),
    phone = COALESCE(@param_phone, phone),
    updated_at = NOW()
WHERE id = @user_id;

-- Prevent empty updates by checking for actual changes
UPDATE users SET
    name = COALESCE(@param_name, name),
    email = COALESCE(@param_email, email),
    phone = COALESCE(@param_phone, phone),
    updated_at = NOW()
WHERE id = @user_id
    AND (@param_name IS NOT NULL AND @param_name IS DISTINCT FROM name
         OR @param_email IS NOT NULL AND @param_email IS DISTINCT FROM email
         OR @param_phone IS NOT NULL AND @param_phone IS DISTINCT FROM phone);

-- Return affected rows to confirm changes
UPDATE users SET
    name = COALESCE(@param_name, name),
    email = COALESCE(@param_email, email),
    updated_at = NOW()
WHERE id = @user_id
    AND (COALESCE(@param_name, name) != name 
         OR COALESCE(@param_email, email) != email)
RETURNING id, name, email, updated_at;
```

### Conditional Updates with CASE

```sql
-- Update based on conditions
UPDATE products SET
    price = CASE 
        WHEN category = 'electronics' THEN price * 0.9  -- 10% discount
        WHEN category = 'books' THEN price * 0.95       -- 5% discount
        ELSE price
    END,
    sale_price = CASE 
        WHEN price > 100 THEN price * 0.8               -- 20% off expensive items
        WHEN price > 50 THEN price * 0.85               -- 15% off mid-range
        ELSE price * 0.9                                -- 10% off everything else
    END,
    updated_at = NOW()
WHERE active = TRUE;
```

### Range-Based Updates

```sql
-- Update user levels based on points
UPDATE user_profiles SET
    level = CASE 
        WHEN points >= 10000 THEN 'platinum'
        WHEN points >= 5000 THEN 'gold'
        WHEN points >= 1000 THEN 'silver'
        ELSE 'bronze'
    END,
    level_updated_at = NOW()
WHERE level != CASE 
    WHEN points >= 10000 THEN 'platinum'
    WHEN points >= 5000 THEN 'gold'
    WHEN points >= 1000 THEN 'silver'
    ELSE 'bronze'
END;
```

## Bulk Updates

### Batch Updates with VALUES

```sql
-- PostgreSQL: Update multiple records efficiently
UPDATE products AS p SET
    price = v.new_price,
    stock_quantity = v.new_stock,
    updated_at = NOW()
FROM (VALUES
    (1, 29.99, 100),
    (2, 49.99, 50),
    (3, 19.99, 200)
) AS v(product_id, new_price, new_stock)
WHERE p.id = v.product_id;

-- MySQL equivalent using CASE statements
UPDATE products SET
    price = CASE id
        WHEN 1 THEN 29.99
        WHEN 2 THEN 49.99
        WHEN 3 THEN 19.99
        ELSE price
    END,
    stock_quantity = CASE id
        WHEN 1 THEN 100
        WHEN 2 THEN 50
        WHEN 3 THEN 200
        ELSE stock_quantity
    END,
    updated_at = NOW()
WHERE id IN (1, 2, 3);
```

### Batch Updates with Temporary Tables

```sql
-- Create temporary table for batch data
CREATE TEMPORARY TABLE temp_product_updates (
    product_id INTEGER,
    new_price DECIMAL(10,2),
    new_stock INTEGER
);

-- Insert batch data
INSERT INTO temp_product_updates VALUES
    (1, 29.99, 100),
    (2, 49.99, 50),
    (3, 19.99, 200);

-- Perform batch update
UPDATE products p 
SET price = t.new_price,
    stock_quantity = t.new_stock,
    updated_at = NOW()
FROM temp_product_updates t
WHERE p.id = t.product_id;

-- Cleanup
DROP TEMPORARY TABLE temp_product_updates;
```

### Chunked Updates for Large Tables

```sql
-- Update large tables in chunks to avoid long locks
DO $$
DECLARE
    batch_size INTEGER := 1000;
    total_updated INTEGER := 0;
    current_batch INTEGER;
BEGIN
    LOOP
        UPDATE products SET
            processed = TRUE,
            updated_at = NOW()
        WHERE id IN (
            SELECT id 
            FROM products 
            WHERE processed = FALSE 
            ORDER BY id 
            LIMIT batch_size
        );
        
        GET DIAGNOSTICS current_batch = ROW_COUNT;
        total_updated := total_updated + current_batch;
        
        -- Exit if no more rows to update
        EXIT WHEN current_batch = 0;
        
        -- Optional: Add delay between batches
        PERFORM pg_sleep(0.1);
        
        -- Log progress
        RAISE NOTICE 'Updated % rows, total: %', current_batch, total_updated;
    END LOOP;
    
    RAISE NOTICE 'Batch update completed. Total rows updated: %', total_updated;
END $$;
```

## Upsert Operations

### PostgreSQL UPSERT with ON CONFLICT

```sql
-- Basic upsert
INSERT INTO users (id, name, email, created_at)
VALUES (1, 'John Doe', 'john@example.com', NOW())
ON CONFLICT (email)
DO UPDATE SET
    name = EXCLUDED.name,
    updated_at = NOW();

-- Conditional upsert (only update if values changed)
INSERT INTO users (id, name, email, age, bio, created_at)
VALUES (1, 'John Doe', 'john@example.com', 30, 'Software Engineer', NOW())
ON CONFLICT (email)
DO UPDATE SET
    name = EXCLUDED.name,
    age = EXCLUDED.age,
    bio = EXCLUDED.bio,
    updated_at = NOW()
WHERE users IS DISTINCT FROM EXCLUDED
RETURNING *;

-- Upsert with additional logic
INSERT INTO product_inventory (product_id, quantity, last_updated)
VALUES (@product_id, @quantity, NOW())
ON CONFLICT (product_id)
DO UPDATE SET
    quantity = CASE 
        WHEN @operation = 'add' THEN product_inventory.quantity + EXCLUDED.quantity
        WHEN @operation = 'subtract' THEN product_inventory.quantity - EXCLUDED.quantity
        ELSE EXCLUDED.quantity
    END,
    last_updated = NOW()
WHERE product_inventory.quantity >= 0;  -- Prevent negative inventory
```

### MySQL UPSERT Patterns

```sql
-- MySQL: INSERT ... ON DUPLICATE KEY UPDATE
INSERT INTO users (id, name, email, created_at)
VALUES (1, 'John Doe', 'john@example.com', NOW())
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    email = VALUES(email),
    updated_at = NOW();

-- MySQL: REPLACE statement (deletes then inserts)
REPLACE INTO users (id, name, email, updated_at)
VALUES (1, 'John Doe', 'john@example.com', NOW());

-- MySQL: INSERT IGNORE (ignores if exists)
INSERT IGNORE INTO users (id, name, email, created_at)
VALUES (1, 'John Doe', 'john@example.com', NOW());
```

### Merge Operations (SQL Server, Oracle)

```sql
-- SQL Server MERGE statement
MERGE users AS target
USING (VALUES 
    (1, 'John Doe', 'john@example.com'),
    (2, 'Jane Smith', 'jane@example.com')
) AS source (id, name, email)
ON target.id = source.id
WHEN MATCHED THEN
    UPDATE SET 
        name = source.name,
        email = source.email,
        updated_at = GETDATE()
WHEN NOT MATCHED THEN
    INSERT (id, name, email, created_at)
    VALUES (source.id, source.name, source.email, GETDATE());
```

## Atomic Updates

### Counter Updates

```sql
-- Safe counter increment
UPDATE statistics SET
    view_count = view_count + 1,
    updated_at = NOW()
WHERE page_id = @page_id;

-- Conditional counter update with limits
UPDATE user_accounts SET
    login_attempts = login_attempts + 1,
    last_attempt_at = NOW()
WHERE username = @username
    AND login_attempts < 5;  -- Prevent overflow

-- Atomic balance transfer
BEGIN TRANSACTION;

UPDATE accounts SET
    balance = balance - @amount,
    updated_at = NOW()
WHERE id = @from_account_id
    AND balance >= @amount;  -- Ensure sufficient funds

UPDATE accounts SET
    balance = balance + @amount,
    updated_at = NOW()
WHERE id = @to_account_id;

-- Check if both updates succeeded
IF @@ROWCOUNT != 2 THEN
    ROLLBACK TRANSACTION;
    THROW 'Transfer failed';
ELSE
    COMMIT TRANSACTION;
END IF;
```

### Optimistic Locking

```sql
-- Update with version check
UPDATE documents SET
    title = @new_title,
    content = @new_content,
    version = version + 1,
    updated_at = NOW(),
    updated_by = @user_id
WHERE id = @document_id
    AND version = @expected_version;

-- Check if update succeeded
IF @@ROWCOUNT = 0 THEN
    THROW 'Document was modified by another user';
END IF;
```

## Update with Joins

### Update from Related Tables

```sql
-- Update based on related table data
UPDATE products p
SET 
    category_name = c.name,
    category_discount = c.default_discount,
    updated_at = NOW()
FROM categories c
WHERE p.category_id = c.id
    AND c.active = TRUE;

-- Update with aggregated data
UPDATE customers c
SET 
    total_orders = subq.order_count,
    total_spent = subq.total_amount,
    last_order_date = subq.last_order,
    updated_at = NOW()
FROM (
    SELECT 
        customer_id,
        COUNT(*) as order_count,
        SUM(total_amount) as total_amount,
        MAX(order_date) as last_order
    FROM orders
    WHERE status = 'completed'
    GROUP BY customer_id
) subq
WHERE c.id = subq.customer_id;
```

### Update with Window Functions

```sql
-- Update with ranking
WITH ranked_scores AS (
    SELECT 
        id,
        ROW_NUMBER() OVER (ORDER BY score DESC) as rank,
        DENSE_RANK() OVER (ORDER BY score DESC) as dense_rank
    FROM leaderboard
)
UPDATE leaderboard l
SET 
    rank = rs.rank,
    dense_rank = rs.dense_rank,
    updated_at = NOW()
FROM ranked_scores rs
WHERE l.id = rs.id;
```

## Performance Optimization

### Index-Friendly Updates

```sql
-- Good: Use indexed columns in WHERE clause
UPDATE users SET
    last_login = NOW(),
    login_count = login_count + 1
WHERE user_id = @user_id;  -- user_id is indexed

-- Avoid: Functions in WHERE clause prevent index usage
-- UPDATE users SET last_login = NOW() WHERE LOWER(email) = 'user@example.com';

-- Good: Use direct comparison
UPDATE users SET last_login = NOW() WHERE email = 'user@example.com';
```

### Minimize Lock Time

```sql
-- Update in smaller batches
CREATE OR REPLACE FUNCTION update_user_scores_batch()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER := 0;
    batch_size INTEGER := 1000;
    total_updated INTEGER := 0;
BEGIN
    LOOP
        UPDATE user_scores 
        SET calculated_score = (points * multiplier),
            updated_at = NOW()
        WHERE id IN (
            SELECT id 
            FROM user_scores 
            WHERE calculated_score IS NULL 
            ORDER BY id 
            LIMIT batch_size
        );
        
        GET DIAGNOSTICS updated_count = ROW_COUNT;
        total_updated := total_updated + updated_count;
        
        EXIT WHEN updated_count = 0;
        
        -- Release locks between batches
        COMMIT;
    END LOOP;
    
    RETURN total_updated;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Always Use WHERE Clauses

```sql
-- ❌ Dangerous: Updates all rows
UPDATE products SET price = price * 1.1;

-- ✅ Safe: Specific condition
UPDATE products SET 
    price = price * 1.1,
    updated_at = NOW()
WHERE category = 'electronics' 
    AND active = TRUE;
```

### 2. Use Transactions for Multi-Table Updates

```sql
BEGIN TRANSACTION;

-- Update order status
UPDATE orders SET 
    status = 'shipped',
    shipped_at = NOW()
WHERE id = @order_id;

-- Update inventory
UPDATE products SET 
    stock_quantity = stock_quantity - oi.quantity
FROM order_items oi
WHERE products.id = oi.product_id 
    AND oi.order_id = @order_id;

-- Log the shipment
INSERT INTO shipment_log (order_id, shipped_at, shipped_by)
VALUES (@order_id, NOW(), @user_id);

COMMIT TRANSACTION;
```

### 3. Validate Data Before Updates

```sql
-- Check constraints before updating
UPDATE accounts SET 
    balance = balance - @withdrawal_amount
WHERE account_id = @account_id
    AND balance >= @withdrawal_amount  -- Ensure sufficient funds
    AND status = 'active'              -- Ensure account is active
    AND NOT locked;                    -- Ensure account is not locked
```

## Anti-Patterns

### Common Mistakes to Avoid

```sql
-- ❌ Bad: No WHERE clause
UPDATE products SET price = 0;  -- Affects all products!

-- ❌ Bad: UPDATE without proper conditions
UPDATE users SET email = @new_email WHERE name = @name;  -- Multiple users might have same name

-- ❌ Bad: Not handling concurrent updates
UPDATE inventory SET quantity = @new_quantity WHERE product_id = @id;
-- Should use: quantity = quantity + @delta or optimistic locking

-- ❌ Bad: Large updates without batching
UPDATE huge_table SET processed = TRUE;  -- Locks table for too long

-- ❌ Bad: Not using indexes
UPDATE users SET status = 'inactive' WHERE created_at < '2020-01-01';  -- No index on created_at

-- ❌ Bad: Functions in WHERE clause
UPDATE users SET last_seen = NOW() WHERE DATE(last_seen) < '2024-01-01';  -- Prevents index usage

-- ✅ Good alternatives for above
UPDATE products SET price = 0 WHERE category = 'discontinued';
UPDATE users SET email = @new_email WHERE id = @user_id;
UPDATE inventory SET quantity = quantity + @delta WHERE product_id = @id;
-- Use batched updates for large tables
CREATE INDEX idx_users_created_at ON users(created_at);
UPDATE users SET last_seen = NOW() WHERE last_seen < '2024-01-01';
```

### Performance Anti-Patterns

```sql
-- ❌ Bad: N+1 update problem
-- FOR EACH user_id IN user_list LOOP
--     UPDATE users SET last_login = NOW() WHERE id = user_id;
-- END LOOP;

-- ✅ Good: Batch update
UPDATE users SET last_login = NOW() 
WHERE id = ANY(@user_id_array);

-- ❌ Bad: Updating unchanged values
UPDATE users SET 
    name = @name,
    email = @email,
    updated_at = NOW()
WHERE id = @user_id;  -- Updates even if values are the same

-- ✅ Good: Only update if values changed
UPDATE users SET 
    name = @name,
    email = @email,
    updated_at = NOW()
WHERE id = @user_id
    AND (name != @name OR email != @email);
```

## Related Patterns

- [Upsert Patterns](bulk-operations.md)
- [Transaction Patterns](transaction.md)
- [Locking Patterns](locks.md)
- [Batch Processing](../performance/README.md)
