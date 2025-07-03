# Soft Delete Patterns

Soft delete is a pattern where records are marked as deleted rather than physically removed from the database. This approach preserves data integrity, maintains audit trails, and enables data recovery.

## Table of Contents

1. [Basic Soft Delete](#basic-soft-delete)
2. [Implementation Patterns](#implementation-patterns)
3. [Query Patterns](#query-patterns)
4. [Constraint Handling](#constraint-handling)
5. [Performance Considerations](#performance-considerations)
6. [Best Practices](#best-practices)
7. [Advanced Patterns](#advanced-patterns)
8. [Anti-Patterns](#anti-patterns)

## Basic Soft Delete

### Simple Soft Delete Column

```sql
-- Basic soft delete with boolean flag
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE
);

-- Alternative: Using timestamp
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);
```

### Basic Operations

```sql
-- Soft delete (mark as deleted)
UPDATE users SET is_deleted = TRUE, updated_at = NOW() WHERE id = 1;
UPDATE products SET deleted_at = NOW() WHERE id = 1;

-- Query active records
SELECT * FROM users WHERE is_deleted = FALSE;
SELECT * FROM products WHERE deleted_at IS NULL;

-- Query deleted records
SELECT * FROM users WHERE is_deleted = TRUE;
SELECT * FROM products WHERE deleted_at IS NOT NULL;

-- Restore (undelete)
UPDATE users SET is_deleted = FALSE, updated_at = NOW() WHERE id = 1;
UPDATE products SET deleted_at = NULL WHERE id = 1;
```

## Implementation Patterns

### Timestamp-Based Soft Delete

```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    order_number VARCHAR(20) UNIQUE NOT NULL,
    total_amount DECIMAL(10,2),
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    deleted_by INTEGER NULL,
    
    FOREIGN KEY (customer_id) REFERENCES users(id),
    FOREIGN KEY (deleted_by) REFERENCES users(id)
);

-- Soft delete with audit info
UPDATE orders 
SET deleted_at = NOW(), 
    deleted_by = @current_user_id,
    updated_at = NOW()
WHERE id = @order_id;
```

### Status-Based Soft Delete

```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    author_id INTEGER NOT NULL,
    status ENUM('draft', 'published', 'archived', 'deleted') DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (author_id) REFERENCES users(id)
);

-- Soft delete using status
UPDATE posts 
SET status = 'deleted', updated_at = NOW() 
WHERE id = 1;

-- Query active posts
SELECT * FROM posts 
WHERE status IN ('draft', 'published', 'archived');
```

### Versioned Soft Delete

```sql
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    version INTEGER DEFAULT 1,
    is_current BOOLEAN DEFAULT TRUE,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by INTEGER NOT NULL,
    
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE KEY unique_current_version (id, is_current, is_deleted)
);

-- Soft delete current version
UPDATE documents 
SET is_deleted = TRUE 
WHERE id = 1 AND is_current = TRUE;
```

## Query Patterns

### Views for Active Records

```sql
-- Create views to automatically filter deleted records
CREATE VIEW active_users AS
SELECT id, username, email, created_at, updated_at
FROM users
WHERE is_deleted = FALSE;

CREATE VIEW active_products AS
SELECT id, name, price, created_at, updated_at
FROM products
WHERE deleted_at IS NULL;

CREATE VIEW active_orders AS
SELECT id, customer_id, order_number, total_amount, status, created_at
FROM orders
WHERE deleted_at IS NULL;

-- Use views in queries
SELECT * FROM active_users WHERE email = 'user@example.com';
SELECT * FROM active_products WHERE price > 100;
```

### Complex Queries with Soft Delete

```sql
-- Join with soft delete consideration
SELECT 
    u.username,
    COUNT(o.id) as order_count,
    SUM(o.total_amount) as total_spent
FROM active_users u
LEFT JOIN active_orders o ON u.id = o.customer_id
GROUP BY u.id, u.username;

-- Subquery with soft delete
SELECT *
FROM active_products p
WHERE p.id IN (
    SELECT DISTINCT oi.product_id
    FROM order_items oi
    JOIN active_orders o ON oi.order_id = o.id
    WHERE o.created_at >= '2024-01-01'
);

-- Window functions with soft delete
SELECT 
    id,
    name,
    price,
    ROW_NUMBER() OVER (ORDER BY price DESC) as price_rank
FROM active_products
WHERE category_id = 1;
```

### Audit and Recovery Queries

```sql
-- Recently deleted items
SELECT 
    id,
    username,
    email,
    deleted_at,
    deleted_by
FROM users u
LEFT JOIN users deleter ON u.deleted_by = deleter.id
WHERE u.deleted_at > NOW() - INTERVAL '7 days';

-- Deletion statistics
SELECT 
    DATE(deleted_at) as deletion_date,
    COUNT(*) as deleted_count
FROM users
WHERE deleted_at IS NOT NULL
GROUP BY DATE(deleted_at)
ORDER BY deletion_date DESC;

-- Items deleted by user
SELECT 
    deleter.username as deleted_by,
    COUNT(*) as deletion_count
FROM orders o
JOIN users deleter ON o.deleted_by = deleter.id
WHERE o.deleted_at > NOW() - INTERVAL '30 days'
GROUP BY deleter.username
ORDER BY deletion_count DESC;
```

## Constraint Handling

### Unique Constraints with Soft Delete

One major benefit of soft delete is that unique constraints still apply, preventing users from re-adding the same data they previously deleted.

```sql
-- Problem: Unique constraint prevents re-insertion after soft delete
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,  -- This causes issues
    description TEXT,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- ❌ This fails if 'Electronics' was soft-deleted before
INSERT INTO categories (name) VALUES ('Electronics');

-- ✅ Solution 1: Conditional unique constraint (PostgreSQL)
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    UNIQUE (name) WHERE is_deleted = FALSE
);

-- ✅ Solution 2: Composite unique constraint with deleted flag
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    UNIQUE (name, is_deleted)
);

-- ✅ Solution 3: Unique constraint with timestamp (allows re-creation)
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    deleted_at TIMESTAMP NULL,
    
    UNIQUE (name) WHERE deleted_at IS NULL
);
```

### Foreign Key Constraints

```sql
-- Handle foreign keys with soft delete
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    price DECIMAL(10,2),
    deleted_at TIMESTAMP NULL,
    
    -- Foreign keys reference primary keys, not soft delete status
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Query respecting soft delete in related tables
SELECT 
    oi.id,
    oi.quantity,
    oi.price,
    p.name as product_name,
    o.order_number
FROM order_items oi
JOIN active_products p ON oi.product_id = p.id
JOIN active_orders o ON oi.order_id = o.id
WHERE oi.deleted_at IS NULL;
```

### Check Constraints

```sql
-- Ensure deleted records have proper metadata
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    department_id INTEGER,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP NULL,
    deleted_by INTEGER NULL,
    
    -- Ensure deleted records have deletion metadata
    CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL AND deleted_by IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
    )
);
```

## Performance Considerations

### Indexing for Soft Delete

```sql
-- Index on soft delete column for better query performance
CREATE INDEX idx_users_not_deleted ON users (id) WHERE is_deleted = FALSE;
CREATE INDEX idx_products_active ON products (id) WHERE deleted_at IS NULL;

-- Composite indexes for common query patterns
CREATE INDEX idx_orders_customer_active 
ON orders (customer_id, created_at) 
WHERE deleted_at IS NULL;

CREATE INDEX idx_posts_status_created 
ON posts (status, created_at) 
WHERE status != 'deleted';

-- Covering indexes for frequently accessed columns
CREATE INDEX idx_users_active_covering 
ON users (id) 
INCLUDE (username, email, created_at) 
WHERE is_deleted = FALSE;
```

### Partitioning by Deletion Status

```sql
-- PostgreSQL partitioning example
CREATE TABLE audit_logs (
    id BIGSERIAL,
    user_id INTEGER,
    action VARCHAR(50),
    details JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE
) PARTITION BY LIST (is_deleted);

CREATE TABLE audit_logs_active PARTITION OF audit_logs
FOR VALUES IN (FALSE);

CREATE TABLE audit_logs_deleted PARTITION OF audit_logs
FOR VALUES IN (TRUE);
```

## Best Practices

### 1. Consistent Soft Delete Implementation

```sql
-- Use consistent column names across tables
-- Option 1: Boolean flag
CREATE TABLE table1 (
    id SERIAL PRIMARY KEY,
    -- ... other columns ...
    is_deleted BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Option 2: Timestamp
CREATE TABLE table2 (
    id SERIAL PRIMARY KEY,
    -- ... other columns ...
    deleted_at TIMESTAMP NULL,
    deleted_by INTEGER NULL
);

-- Don't mix approaches within the same system
```

### 2. Application-Level Abstractions

```sql
-- Create application functions/procedures for consistency
CREATE OR REPLACE FUNCTION soft_delete_user(user_id INTEGER, deleted_by_user_id INTEGER)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE users 
    SET is_deleted = TRUE, 
        updated_at = NOW(),
        deleted_by = deleted_by_user_id
    WHERE id = user_id AND is_deleted = FALSE;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT soft_delete_user(123, 456);
```

### 3. Automated Cleanup Procedures

```sql
-- Hard delete old soft-deleted records
CREATE OR REPLACE FUNCTION cleanup_old_deleted_records()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Delete records soft-deleted more than 2 years ago
    DELETE FROM users 
    WHERE is_deleted = TRUE 
    AND updated_at < NOW() - INTERVAL '2 years';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Log cleanup activity
    INSERT INTO cleanup_log (table_name, records_deleted, cleanup_date)
    VALUES ('users', deleted_count, NOW());
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

## Advanced Patterns

### Hierarchical Soft Delete

```sql
-- Cascade soft delete to child records
CREATE OR REPLACE FUNCTION soft_delete_category_cascade(category_id INTEGER)
RETURNS VOID AS $$
BEGIN
    -- Soft delete the category
    UPDATE categories 
    SET is_deleted = TRUE, updated_at = NOW() 
    WHERE id = category_id;
    
    -- Soft delete all products in this category
    UPDATE products 
    SET is_deleted = TRUE, updated_at = NOW()
    WHERE category_id = category_id AND is_deleted = FALSE;
    
    -- Soft delete subcategories recursively
    UPDATE categories 
    SET is_deleted = TRUE, updated_at = NOW()
    WHERE parent_category_id = category_id AND is_deleted = FALSE;
END;
$$ LANGUAGE plpgsql;
```

### Batch Soft Delete

```sql
-- Efficient batch soft delete
CREATE OR REPLACE FUNCTION soft_delete_inactive_users(inactive_days INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    UPDATE users 
    SET is_deleted = TRUE, 
        updated_at = NOW(),
        deleted_by = 0  -- System deletion
    WHERE is_deleted = FALSE
    AND last_login < NOW() - (inactive_days || ' days')::INTERVAL
    AND created_at < NOW() - INTERVAL '30 days';  -- Grace period for new users
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

### Soft Delete with Approval Workflow

```sql
CREATE TABLE deletion_requests (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    record_id INTEGER NOT NULL,
    requested_by INTEGER NOT NULL,
    request_reason TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    approved_by INTEGER NULL,
    approval_reason TEXT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP NULL,
    
    FOREIGN KEY (requested_by) REFERENCES users(id),
    FOREIGN KEY (approved_by) REFERENCES users(id)
);

-- Request deletion
INSERT INTO deletion_requests (table_name, record_id, requested_by, request_reason)
VALUES ('products', 123, 456, 'Product discontinued');

-- Approve and execute deletion
UPDATE deletion_requests 
SET status = 'approved', 
    approved_by = @current_user_id,
    processed_at = NOW()
WHERE id = @request_id;

-- Then execute the actual soft delete
UPDATE products 
SET is_deleted = TRUE, updated_at = NOW()
WHERE id = (SELECT record_id FROM deletion_requests WHERE id = @request_id);
```

## Anti-Patterns

### Common Mistakes to Avoid

```sql
-- ❌ Bad: Inconsistent soft delete implementation
CREATE TABLE table1 (
    id SERIAL PRIMARY KEY,
    is_deleted BOOLEAN DEFAULT FALSE  -- Using boolean
);

CREATE TABLE table2 (
    id SERIAL PRIMARY KEY,
    deleted_at TIMESTAMP  -- Using timestamp
);

CREATE TABLE table3 (
    id SERIAL PRIMARY KEY,
    status VARCHAR(20)  -- Using status
);

-- ✅ Good: Consistent approach across all tables

-- ❌ Bad: Not indexing soft delete columns
CREATE TABLE large_table (
    id SERIAL PRIMARY KEY,
    data TEXT,
    is_deleted BOOLEAN DEFAULT FALSE
);
-- Queries will be slow without proper indexing

-- ✅ Good: Proper indexing
CREATE INDEX idx_large_table_active ON large_table (id) WHERE is_deleted = FALSE;

-- ❌ Bad: Forgetting soft delete in all queries
SELECT * FROM users WHERE username = 'john';  -- Returns deleted users too

-- ✅ Good: Always consider soft delete
SELECT * FROM users WHERE username = 'john' AND is_deleted = FALSE;
-- Or use views: SELECT * FROM active_users WHERE username = 'john';

-- ❌ Bad: No cleanup strategy
-- Soft deleted records accumulate forever, impacting performance

-- ✅ Good: Regular cleanup of old soft deleted records
```

### Performance Anti-Patterns

```sql
-- ❌ Bad: Using OR conditions with soft delete
SELECT * FROM products 
WHERE (category_id = 1 OR category_id = 2) 
AND deleted_at IS NULL;

-- ✅ Good: Use IN clause or separate queries
SELECT * FROM products 
WHERE category_id IN (1, 2) 
AND deleted_at IS NULL;

-- ❌ Bad: Functions in WHERE clause prevent index usage
SELECT * FROM orders 
WHERE DATE(created_at) = '2024-01-01' 
AND deleted_at IS NULL;

-- ✅ Good: Use range conditions
SELECT * FROM orders 
WHERE created_at >= '2024-01-01' 
AND created_at < '2024-01-02'
AND deleted_at IS NULL;
```

## Related Patterns

- [Audit Logging](../security/audit-logging.md)
- [Data Archival](../specialized/data-archival.md)
- [Versioning](../specialized/README.md)
- [Constraint Patterns](../schema-design/constraints.md)
