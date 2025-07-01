# Soft Delete Patterns: Complete Guide

Soft deletes allow you to "delete" records by marking them as deleted rather than physically removing them from the database. This approach preserves data for audit trails, compliance, and potential recovery while maintaining referential integrity.

## Table of Contents
- [Soft Delete vs Hard Delete](#soft-delete-vs-hard-delete)
- [Implementation Strategies](#implementation-strategies)
- [Real-World Applications](#real-world-applications)
- [Advanced Patterns](#advanced-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Soft Delete vs Hard Delete

### Hard Delete Characteristics
- **Immediate removal**: Data is permanently deleted from the database
- **Cascade deletion**: Foreign key constraints handle related records
- **Storage efficient**: No additional storage required for deleted records
- **Simple queries**: No need to filter deleted records
- **Irreversible**: Data cannot be recovered without backups

### Soft Delete Characteristics
- **Logical deletion**: Records are marked as deleted but remain in the database
- **Manual cascade**: Must handle related records explicitly
- **Audit trail**: Complete history of all data changes
- **Complex queries**: Must filter deleted records in most queries
- **Recoverable**: Deleted records can be easily restored

## Implementation Strategies

### 1. Simple Boolean Flag

The most basic approach uses a boolean column:

```sql
-- Basic soft delete with boolean flag
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for performance
CREATE INDEX idx_users_not_deleted ON users(id) WHERE NOT is_deleted;

-- Soft delete operation
UPDATE users SET is_deleted = TRUE WHERE id = 1;

-- Query active users
SELECT * FROM users WHERE NOT is_deleted;
```

### 2. Timestamp-Based Soft Delete

More informative approach using timestamps:

```sql
-- Soft delete with timestamp
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

-- Partial index for active records
CREATE INDEX idx_products_active ON products(id, name) WHERE deleted_at IS NULL;

-- Soft delete with timestamp
UPDATE products SET deleted_at = NOW() WHERE id = 1;

-- Query active products
SELECT * FROM products WHERE deleted_at IS NULL;

-- Query deleted products
SELECT * FROM products WHERE deleted_at IS NOT NULL;
```

### 3. Status-Based Approach

More flexible approach using status enums:

```sql
-- Status-based soft delete
CREATE TYPE record_status AS ENUM ('active', 'inactive', 'deleted', 'archived');

CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    status record_status DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    status_changed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for each status
CREATE INDEX idx_documents_active ON documents(id) WHERE status = 'active';
CREATE INDEX idx_documents_deleted ON documents(id) WHERE status = 'deleted';

-- Soft delete operation
UPDATE documents 
SET status = 'deleted', status_changed_at = NOW() 
WHERE id = 1;

-- Query by status
SELECT * FROM documents WHERE status = 'active';
SELECT * FROM documents WHERE status IN ('active', 'inactive');
```

## Real-World Applications

### E-commerce Product Management

```sql
-- Product lifecycle management
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    status TEXT DEFAULT 'active' CHECK (status IN (
        'draft', 'active', 'discontinued', 'archived', 'deleted'
    )),
    
    -- Lifecycle timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    discontinued_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    
    -- Audit fields
    created_by INTEGER,
    deleted_by INTEGER,
    delete_reason TEXT
);

-- Business logic for product lifecycle
CREATE OR REPLACE FUNCTION update_product_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Set discontinued timestamp
    IF NEW.status = 'discontinued' AND OLD.status != 'discontinued' THEN
        NEW.discontinued_at = NOW();
    END IF;
    
    -- Set deleted timestamp and require reason
    IF NEW.status = 'deleted' AND OLD.status != 'deleted' THEN
        NEW.deleted_at = NOW();
        IF NEW.delete_reason IS NULL THEN
            RAISE EXCEPTION 'Delete reason is required when deleting products';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER product_status_trigger
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_product_status();

-- Example operations
-- Discontinue product (still visible with label)
UPDATE products SET status = 'discontinued' WHERE sku = 'IPHONE-12';

-- Soft delete product (hidden from catalog)
UPDATE products 
SET status = 'deleted', deleted_by = 1, delete_reason = 'Copyright violation' 
WHERE sku = 'COUNTERFEIT-ITEM';
```

### User Account Management

```sql
-- User account soft delete with cascading
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    status TEXT DEFAULT 'active' CHECK (status IN (
        'active', 'suspended', 'deactivated', 'deleted'
    )),
    deleted_at TIMESTAMPTZ,
    deletion_type TEXT CHECK (deletion_type IN (
        'user_requested', 'admin_action', 'gdpr_compliance', 'inactivity'
    )),
    
    -- Unique constraint only for active users
    UNIQUE(email) DEFERRABLE INITIALLY DEFERRED
);

-- Create partial unique index for active users only
DROP INDEX IF EXISTS users_email_key;
CREATE UNIQUE INDEX users_email_active_unique ON users(email) WHERE status != 'deleted';

-- User posts with soft delete cascade
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title TEXT NOT NULL,
    content TEXT,
    status TEXT DEFAULT 'published' CHECK (status IN (
        'draft', 'published', 'archived', 'deleted'
    )),
    deleted_at TIMESTAMPTZ,
    cascade_deleted BOOLEAN DEFAULT FALSE -- Deleted due to user deletion
);

-- Function to cascade soft delete
CREATE OR REPLACE FUNCTION cascade_user_deletion()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'deleted' AND OLD.status != 'deleted' THEN
        -- Soft delete all user's posts
        UPDATE posts 
        SET status = 'deleted', deleted_at = NOW(), cascade_deleted = TRUE
        WHERE user_id = NEW.id AND status != 'deleted';
        
        -- You could also soft delete other related entities here
        -- UPDATE comments SET deleted_at = NOW() WHERE user_id = NEW.id;
        -- UPDATE user_sessions SET deleted_at = NOW() WHERE user_id = NEW.id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cascade_user_deletion_trigger
    AFTER UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION cascade_user_deletion();
```

### Content Management System

```sql
-- CMS with soft delete and versioning
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    author_id INTEGER NOT NULL,
    
    -- Publication status
    status TEXT DEFAULT 'draft' CHECK (status IN (
        'draft', 'published', 'archived', 'deleted'
    )),
    published_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    
    -- Soft delete metadata
    deleted_by INTEGER,
    delete_reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique slug only for non-deleted articles
    UNIQUE(slug) DEFERRABLE INITIALLY DEFERRED
);

-- Partial unique index for active articles
CREATE UNIQUE INDEX articles_slug_active_unique ON articles(slug) 
WHERE status != 'deleted';

-- Archive old articles automatically
CREATE OR REPLACE FUNCTION auto_archive_old_articles()
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    UPDATE articles 
    SET status = 'archived', archived_at = NOW()
    WHERE status = 'published' 
    AND published_at < NOW() - INTERVAL '2 years'
    AND archived_at IS NULL;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Schedule this function to run periodically
-- SELECT cron.schedule('archive-old-articles', '0 2 * * *', 'SELECT auto_archive_old_articles();');
```

## Advanced Patterns

### 1. Temporal Soft Delete

For records that should be deleted after a certain time:

```sql
-- Temporary soft delete with automatic cleanup
CREATE TABLE temporary_uploads (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    file_path TEXT NOT NULL,
    uploaded_by INTEGER NOT NULL,
    
    -- Automatic cleanup fields
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '24 hours'),
    deleted_at TIMESTAMPTZ,
    auto_deleted BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to clean up expired uploads
CREATE OR REPLACE FUNCTION cleanup_expired_uploads()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    UPDATE temporary_uploads 
    SET deleted_at = NOW(), auto_deleted = TRUE
    WHERE expires_at < NOW() 
    AND deleted_at IS NULL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Log cleanup operation
    INSERT INTO cleanup_log (table_name, deleted_count, cleaned_at)
    VALUES ('temporary_uploads', deleted_count, NOW());
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

### 2. Hierarchical Soft Delete

For tree structures where parent deletion affects children:

```sql
-- Category tree with hierarchical soft delete
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id INTEGER REFERENCES categories(id),
    path TEXT, -- Materialized path for efficient queries
    depth INTEGER DEFAULT 0,
    
    deleted_at TIMESTAMPTZ,
    deleted_by INTEGER,
    cascade_deleted BOOLEAN DEFAULT FALSE
);

-- Function to soft delete category and all descendants
CREATE OR REPLACE FUNCTION soft_delete_category_tree(category_id INTEGER, deleted_by_user INTEGER)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete the category and all its descendants
    WITH RECURSIVE category_tree AS (
        -- Base case: the category to delete
        SELECT id, name, parent_id, path, depth
        FROM categories 
        WHERE id = category_id AND deleted_at IS NULL
        
        UNION ALL
        
        -- Recursive case: all descendants
        SELECT c.id, c.name, c.parent_id, c.path, c.depth
        FROM categories c
        INNER JOIN category_tree ct ON c.parent_id = ct.id
        WHERE c.deleted_at IS NULL
    )
    UPDATE categories 
    SET deleted_at = NOW(), 
        deleted_by = deleted_by_user,
        cascade_deleted = (categories.id != category_id)
    FROM category_tree
    WHERE categories.id = category_tree.id;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT soft_delete_category_tree(5, 1); -- Delete category 5 and all subcategories
```

### 3. Soft Delete with Archival

Move old deleted records to archive tables:

```sql
-- Main table
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Archive table with same structure
CREATE TABLE orders_archive (
    archived_at TIMESTAMPTZ DEFAULT NOW(),
    LIKE orders INCLUDING ALL
);

-- Function to archive old soft-deleted records
CREATE OR REPLACE FUNCTION archive_old_deleted_orders()
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    -- Move records older than 1 year to archive
    WITH old_deleted AS (
        DELETE FROM orders 
        WHERE deleted_at IS NOT NULL 
        AND deleted_at < NOW() - INTERVAL '1 year'
        RETURNING *
    )
    INSERT INTO orders_archive SELECT NOW() as archived_at, * FROM old_deleted;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;
```

## Using Views for Abstraction

### 1. Create Views for Active Records

```sql
-- Create clean views that automatically filter deleted records
CREATE VIEW active_users AS
SELECT id, email, name, created_at, updated_at
FROM users 
WHERE status != 'deleted';

CREATE VIEW active_products AS  
SELECT id, sku, name, price, status, created_at
FROM products 
WHERE status NOT IN ('deleted', 'archived');

-- Use views in application code
SELECT * FROM active_users WHERE email = 'user@example.com';
SELECT * FROM active_products WHERE price > 100;
```

### 2. Updateable Views with Soft Delete

```sql
-- Create updateable view that handles soft delete
CREATE VIEW users_managed AS
SELECT id, email, name, created_at, updated_at
FROM users 
WHERE status != 'deleted';

-- Create INSTEAD OF trigger for soft delete
CREATE OR REPLACE FUNCTION soft_delete_user_via_view()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users 
    SET status = 'deleted', deleted_at = NOW()
    WHERE id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER soft_delete_user_trigger
    INSTEAD OF DELETE ON users_managed
    FOR EACH ROW
    EXECUTE FUNCTION soft_delete_user_via_view();

-- Now you can use standard DELETE syntax
DELETE FROM users_managed WHERE id = 1;
```

## Performance Considerations

### 1. Indexing Strategy

```sql
-- Index design for soft delete patterns
CREATE TABLE performance_test (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category_id INTEGER,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Partial indexes for better performance
CREATE INDEX idx_performance_active ON performance_test(id, category_id) 
WHERE deleted_at IS NULL;

CREATE INDEX idx_performance_deleted ON performance_test(deleted_at) 
WHERE deleted_at IS NOT NULL;

-- Composite index for common queries
CREATE INDEX idx_performance_category_active ON performance_test(category_id, created_at)
WHERE deleted_at IS NULL;
```

### 2. Query Optimization

```sql
-- Use EXISTS instead of JOIN for better performance
-- Good
SELECT p.* FROM products p 
WHERE p.deleted_at IS NULL 
AND EXISTS (
    SELECT 1 FROM categories c 
    WHERE c.id = p.category_id AND c.deleted_at IS NULL
);

-- Avoid expensive operations on deleted records
-- Use partial indexes and WHERE clauses effectively
EXPLAIN ANALYZE 
SELECT COUNT(*) FROM orders 
WHERE deleted_at IS NULL 
AND created_at > NOW() - INTERVAL '30 days';
```

### 3. Maintenance Operations

```sql
-- Regular maintenance for soft delete tables
CREATE OR REPLACE FUNCTION soft_delete_maintenance()
RETURNS TABLE(table_name TEXT, operation TEXT, row_count INTEGER) AS $$
BEGIN
    -- Update statistics for partial indexes
    ANALYZE users;
    ANALYZE products;
    ANALYZE orders;
    
    -- Archive old deleted records
    RETURN QUERY SELECT 'orders'::TEXT, 'archived'::TEXT, archive_old_deleted_orders();
    RETURN QUERY SELECT 'uploads'::TEXT, 'cleaned'::TEXT, cleanup_expired_uploads();
    
    -- Vacuum tables with many deleted records
    PERFORM pg_stat_reset_single_table_counters('users'::regclass);
END;
$$ LANGUAGE plpgsql;

-- Schedule maintenance
-- SELECT cron.schedule('soft-delete-maintenance', '0 3 * * 0', 'SELECT soft_delete_maintenance();');
```

## Best Practices

### 1. Naming Conventions

```sql
-- Consistent naming for soft delete columns
-- Option 1: Timestamp approach
deleted_at TIMESTAMPTZ
archived_at TIMESTAMPTZ
deactivated_at TIMESTAMPTZ

-- Option 2: Status approach  
status TEXT CHECK (status IN ('active', 'deleted', 'archived'))
record_state TEXT DEFAULT 'active'

-- Option 3: Boolean approach (least recommended)
is_deleted BOOLEAN DEFAULT FALSE
is_active BOOLEAN DEFAULT TRUE
```

### 2. Migration Strategy

```sql
-- Safe migration to add soft delete
-- Step 1: Add column with default value
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Step 2: Create partial index
CREATE INDEX CONCURRENTLY idx_users_active ON users(id) WHERE deleted_at IS NULL;

-- Step 3: Update application code to use new column
-- Step 4: Create view for backward compatibility
CREATE VIEW users_legacy AS SELECT * FROM users WHERE deleted_at IS NULL;

-- Step 5: Gradually migrate queries to use deleted_at filter
```

### 3. Data Consistency

```sql
-- Ensure data consistency with constraints
ALTER TABLE orders ADD CONSTRAINT check_deleted_orders 
CHECK (
    (deleted_at IS NULL AND status NOT IN ('deleted', 'archived')) OR
    (deleted_at IS NOT NULL AND status IN ('deleted', 'archived'))
);

-- Use triggers to maintain consistency
CREATE OR REPLACE FUNCTION maintain_soft_delete_consistency()
RETURNS TRIGGER AS $$
BEGIN
    -- Auto-set deleted_at when status changes to deleted
    IF NEW.status = 'deleted' AND OLD.status != 'deleted' THEN
        NEW.deleted_at = COALESCE(NEW.deleted_at, NOW());
    END IF;
    
    -- Clear deleted_at when status changes from deleted
    IF NEW.status != 'deleted' AND OLD.status = 'deleted' THEN
        NEW.deleted_at = NULL;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### 4. Testing Strategy

```sql
-- Test soft delete functionality
DO $$
DECLARE
    user_id INTEGER;
    post_count INTEGER;
BEGIN
    -- Create test user
    INSERT INTO users (email, name) VALUES ('test@example.com', 'Test User')
    RETURNING id INTO user_id;
    
    -- Create test posts
    INSERT INTO posts (user_id, title, content) 
    VALUES (user_id, 'Test Post 1', 'Content 1'),
           (user_id, 'Test Post 2', 'Content 2');
    
    -- Verify posts exist
    SELECT COUNT(*) INTO post_count FROM posts WHERE user_id = user_id AND status != 'deleted';
    ASSERT post_count = 2, 'Expected 2 active posts';
    
    -- Soft delete user
    UPDATE users SET status = 'deleted' WHERE id = user_id;
    
    -- Verify cascade deletion
    SELECT COUNT(*) INTO post_count FROM posts WHERE user_id = user_id AND status != 'deleted';
    ASSERT post_count = 0, 'Expected 0 active posts after user deletion';
    
    -- Verify posts still exist but are soft deleted
    SELECT COUNT(*) INTO post_count FROM posts WHERE user_id = user_id AND cascade_deleted = TRUE;
    ASSERT post_count = 2, 'Expected 2 cascade-deleted posts';
    
    -- Cleanup
    DELETE FROM posts WHERE user_id = user_id;
    DELETE FROM users WHERE id = user_id;
    
    RAISE NOTICE 'All soft delete tests passed';
END $$;
```

## Common Anti-Patterns to Avoid

### 1. Forgetting to Filter Deleted Records

```sql
-- DON'T: Forget to filter deleted records
SELECT COUNT(*) FROM users; -- Includes deleted users

-- DO: Always filter deleted records
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;
-- Or use views
SELECT COUNT(*) FROM active_users;
```

### 2. Inconsistent Soft Delete Implementation

```sql
-- DON'T: Mix different soft delete approaches
CREATE TABLE mixed_approach (
    id SERIAL PRIMARY KEY,
    is_deleted BOOLEAN DEFAULT FALSE,  -- Boolean approach
    deleted_at TIMESTAMPTZ,            -- Timestamp approach  
    status TEXT                        -- Status approach
);

-- DO: Choose one approach and use it consistently
CREATE TABLE consistent_approach (
    id SERIAL PRIMARY KEY,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'deleted', 'archived')),
    status_changed_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. Not Handling Unique Constraints

```sql
-- DON'T: Ignore unique constraints with soft delete
CREATE TABLE bad_example (
    email TEXT UNIQUE,  -- Will prevent re-using email after soft delete
    deleted_at TIMESTAMPTZ
);

-- DO: Use partial unique constraints
CREATE TABLE good_example (
    email TEXT,
    deleted_at TIMESTAMPTZ,
    UNIQUE(email) WHERE deleted_at IS NULL
);
```

## Conclusion

Soft delete patterns provide valuable benefits for data retention, audit trails, and compliance requirements. However, they come with complexity costs that must be carefully managed:

**When to use soft delete:**
- Regulatory compliance requires data retention
- Audit trails are essential for business operations
- Data recovery is frequently needed
- Cascade deletion would be destructive

**When to avoid soft delete:**
- High-volume transactional data where performance is critical
- Simple applications without audit requirements
- Data that truly should be permanently removed (e.g., sensitive personal data)

**Key implementation principles:**
- Choose one soft delete strategy and use it consistently
- Always create proper indexes for soft delete columns
- Use views to abstract the complexity from application code
- Plan for data archival and cleanup of old soft-deleted records
- Test thoroughly, especially cascade deletion scenarios

Remember that soft deletes are a tool for specific use cases. Don't use them everywhere—sometimes a hard delete is the right choice.
