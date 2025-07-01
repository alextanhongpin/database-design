# Database Enums: Complete Guide

Database enums provide a way to constrain column values to a predefined set of options. This guide covers enum implementation patterns, best practices, and alternatives across different database systems.

## Table of Contents
- [PostgreSQL Enums](#postgresql-enums)
- [MySQL Enums](#mysql-enums)
- [Enum vs Check Constraints](#enum-vs-check-constraints)
- [Enum vs Reference Tables](#enum-vs-reference-tables)
- [Migration Strategies](#migration-strategies)
- [Best Practices](#best-practices)

## PostgreSQL Enums

### Creating and Using Enums

```sql
-- Create enum type
CREATE TYPE order_status AS ENUM (
    'pending',
    'confirmed', 
    'processing',
    'shipped',
    'delivered',
    'cancelled',
    'refunded'
);

-- Create enum type with more complex values
CREATE TYPE priority_level AS ENUM (
    'low',
    'medium', 
    'high',
    'urgent',
    'critical'
);

-- Use enum in table definition
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    status order_status DEFAULT 'pending',
    priority priority_level DEFAULT 'medium',
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert data using enum values
INSERT INTO orders (customer_id, status, priority, total_amount) VALUES
(1, 'pending', 'high', 299.99),
(2, 'confirmed', 'medium', 149.50),
(3, 'shipped', 'urgent', 89.99);
```

### Enum Operations

```sql
-- Display all enum values
SELECT enum_range(NULL::order_status);
-- Result: {pending,confirmed,processing,shipped,delivered,cancelled,refunded}

-- Get enum values as table
SELECT unnest(enum_range(NULL::order_status)) AS status_value;

-- Compare enum values (they have natural ordering)
SELECT 'pending'::order_status < 'confirmed'::order_status; -- true
SELECT 'high'::priority_level > 'medium'::priority_level;   -- true

-- Query using enum values
SELECT * FROM orders WHERE status = 'pending';
SELECT * FROM orders WHERE priority >= 'high';

-- Use in aggregations
SELECT 
    status,
    COUNT(*) as order_count,
    AVG(total_amount) as avg_amount
FROM orders 
GROUP BY status 
ORDER BY status;
```

### Modifying Enums

```sql
-- Add new enum values
ALTER TYPE order_status ADD VALUE 'on_hold';                    -- Append to end
ALTER TYPE order_status ADD VALUE 'preparing' BEFORE 'shipped'; -- Insert before
ALTER TYPE order_status ADD VALUE 'out_for_delivery' AFTER 'shipped'; -- Insert after

-- Rename enum values (PostgreSQL 10+)
ALTER TYPE order_status RENAME VALUE 'cancelled' TO 'canceled';

-- Note: Cannot drop enum values directly
-- Must recreate the enum type and migrate data
```

### Advanced Enum Usage

```sql
-- Enum with reference table for additional metadata
CREATE TYPE user_role AS ENUM ('admin', 'manager', 'employee', 'guest');

CREATE TABLE user_role_details (
    role user_role PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    permissions TEXT[],
    max_access_level INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO user_role_details VALUES
('admin', 'Administrator', 'Full system access', ARRAY['all'], 10),
('manager', 'Manager', 'Department management', ARRAY['read', 'write', 'approve'], 7),
('employee', 'Employee', 'Standard user access', ARRAY['read', 'write'], 5),
('guest', 'Guest User', 'Limited read access', ARRAY['read'], 1);

-- Query combining enum and reference table
SELECT 
    u.id,
    u.name,
    u.role,
    urd.display_name,
    urd.permissions
FROM users u
JOIN user_role_details urd ON urd.role = u.role
WHERE u.role >= 'manager';
```

## MySQL Enums

### Basic MySQL Enum Usage

```sql
-- MySQL enum syntax
CREATE TABLE products (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    status ENUM('draft', 'active', 'discontinued', 'deleted') DEFAULT 'draft',
    category ENUM('electronics', 'clothing', 'books', 'home', 'sports') NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert data
INSERT INTO products (name, status, category) VALUES
('Laptop', 'active', 'electronics'),
('T-Shirt', 'active', 'clothing'),
('Novel', 'draft', 'books');

-- Query enum columns
SELECT * FROM products WHERE status = 'active';
SELECT * FROM products WHERE category IN ('electronics', 'clothing');
```

### MySQL Enum Limitations

```sql
-- MySQL enums are stored as integers internally (1-based indexing)
SELECT status, status+0 as status_index FROM products;
-- Result: 'draft'=1, 'active'=2, 'discontinued'=3, 'deleted'=4

-- Adding enum values requires table alteration
ALTER TABLE products 
MODIFY COLUMN status ENUM('draft', 'active', 'discontinued', 'deleted', 'archived');

-- Cannot easily reorder enum values
-- Must recreate the column to change order
```

## Enum vs Check Constraints

### Check Constraints Alternative

```sql
-- PostgreSQL: Check constraint instead of enum
CREATE TABLE orders_with_check (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN (
        'pending', 'confirmed', 'processing', 'shipped', 
        'delivered', 'cancelled', 'refunded'
    )),
    priority TEXT DEFAULT 'medium' CHECK (priority IN (
        'low', 'medium', 'high', 'urgent', 'critical'
    )),
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Comparison: Enum vs Check Constraint
-- 
-- ENUM Pros:
-- ✅ Built-in ordering
-- ✅ Type safety
-- ✅ Better performance 
-- ✅ Reusable across tables
-- 
-- CHECK Constraint Pros:
-- ✅ Easier to modify values
-- ✅ No custom type management
-- ✅ More portable across databases
-- ✅ Simpler migration scripts
```

### Dynamic Check Constraints

```sql
-- Function to validate enum-like values from a reference table
CREATE OR REPLACE FUNCTION validate_status(status_value TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM valid_statuses 
        WHERE status_name = status_value AND is_active = TRUE
    );
END;
$$ LANGUAGE plpgsql;

-- Reference table for valid values
CREATE TABLE valid_statuses (
    status_name TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER,
    is_active BOOLEAN DEFAULT TRUE
);

INSERT INTO valid_statuses VALUES
('pending', 'Pending', 'Order received, awaiting confirmation', 1, TRUE),
('confirmed', 'Confirmed', 'Order confirmed and being prepared', 2, TRUE),
('shipped', 'Shipped', 'Order has been shipped', 3, TRUE),
('delivered', 'Delivered', 'Order delivered to customer', 4, TRUE);

-- Table with dynamic validation
CREATE TABLE flexible_orders (
    id SERIAL PRIMARY KEY,
    status TEXT NOT NULL CHECK (validate_status(status)),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Enum vs Reference Tables

### Reference Table Pattern

```sql
-- Full reference table approach
CREATE TABLE order_statuses (
    id SERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    color TEXT, -- For UI display
    icon TEXT,  -- For UI display
    sort_order INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO order_statuses (code, name, description, color, sort_order) VALUES
('pending', 'Pending', 'Order received, awaiting processing', '#fbbf24', 1),
('confirmed', 'Confirmed', 'Order confirmed and being prepared', '#3b82f6', 2),
('processing', 'Processing', 'Order is being processed', '#8b5cf6', 3),
('shipped', 'Shipped', 'Order has been shipped', '#10b981', 4),
('delivered', 'Delivered', 'Order delivered successfully', '#059669', 5),
('cancelled', 'Cancelled', 'Order was cancelled', '#ef4444', 6);

-- Orders table referencing status table
CREATE TABLE orders_with_ref (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    status_id INTEGER NOT NULL REFERENCES order_statuses(id),
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Query with status details
SELECT 
    o.id,
    o.total_amount,
    os.code as status_code,
    os.name as status_name,
    os.color as status_color
FROM orders_with_ref o
JOIN order_statuses os ON os.id = o.status_id
WHERE os.is_active = TRUE;
```

### Hybrid Approach: Enum + Reference Table

```sql
-- Best of both worlds: enum for performance, reference table for metadata
CREATE TYPE order_status_enum AS ENUM (
    'pending', 'confirmed', 'processing', 'shipped', 
    'delivered', 'cancelled', 'refunded'
);

CREATE TABLE order_status_metadata (
    status order_status_enum PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    color TEXT,
    icon TEXT,
    notification_template TEXT,
    is_final_state BOOLEAN DEFAULT FALSE,
    sort_order INTEGER
);

INSERT INTO order_status_metadata VALUES
('pending', 'Pending Payment', 'Awaiting payment confirmation', '#fbbf24', 'clock', 'payment_pending', FALSE, 1),
('confirmed', 'Order Confirmed', 'Payment received, order confirmed', '#3b82f6', 'check', 'order_confirmed', FALSE, 2),
('processing', 'Processing', 'Order is being prepared', '#8b5cf6', 'cog', 'order_processing', FALSE, 3),
('shipped', 'Shipped', 'Order has left the warehouse', '#10b981', 'truck', 'order_shipped', FALSE, 4),
('delivered', 'Delivered', 'Order delivered successfully', '#059669', 'check-circle', 'order_delivered', TRUE, 5),
('cancelled', 'Cancelled', 'Order was cancelled', '#ef4444', 'x-circle', 'order_cancelled', TRUE, 6),
('refunded', 'Refunded', 'Order refunded', '#6b7280', 'arrow-left', 'order_refunded', TRUE, 7);

-- Orders table using enum
CREATE TABLE orders_hybrid (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    status order_status_enum DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Efficient queries with rich metadata
SELECT 
    o.id,
    o.status,
    osm.display_name,
    osm.color,
    osm.is_final_state
FROM orders_hybrid o
JOIN order_status_metadata osm ON osm.status = o.status
WHERE o.customer_id = 123;
```

## Migration Strategies

### Safely Adding Enum Values

```sql
-- Safe enum modification in production
DO $$
BEGIN
    -- Check if enum value already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum 
        WHERE enumlabel = 'processing' 
        AND enumtypid = 'order_status'::regtype
    ) THEN
        ALTER TYPE order_status ADD VALUE 'processing' AFTER 'confirmed';
    END IF;
END $$;
```

### Migrating from Text to Enum

```sql
-- Step 1: Create enum type
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'banned');

-- Step 2: Add new enum column
ALTER TABLE users ADD COLUMN status_new user_status;

-- Step 3: Migrate existing data with validation
UPDATE users 
SET status_new = CASE 
    WHEN status_text = 'active' THEN 'active'::user_status
    WHEN status_text = 'inactive' THEN 'inactive'::user_status  
    WHEN status_text = 'suspended' THEN 'suspended'::user_status
    WHEN status_text = 'banned' THEN 'banned'::user_status
    ELSE 'inactive'::user_status -- Default for invalid values
END;

-- Step 4: Verify migration
SELECT status_text, status_new, COUNT(*) 
FROM users 
GROUP BY status_text, status_new;

-- Step 5: Drop old column and rename new one
ALTER TABLE users DROP COLUMN status_text;
ALTER TABLE users RENAME COLUMN status_new TO status;

-- Step 6: Add not null constraint if needed
ALTER TABLE users ALTER COLUMN status SET NOT NULL;
```

### Recreating Enum with Removed Values

```sql
-- When you need to remove enum values (complex operation)
-- Step 1: Create new enum type
CREATE TYPE order_status_new AS ENUM (
    'pending', 'confirmed', 'shipped', 'delivered' -- removed 'cancelled'
);

-- Step 2: Update affected tables
-- First, handle rows with the value being removed
UPDATE orders SET status = 'pending' WHERE status = 'cancelled';

-- Step 3: Add new column with new enum type
ALTER TABLE orders ADD COLUMN status_new order_status_new;

-- Step 4: Migrate data
UPDATE orders SET status_new = status::text::order_status_new;

-- Step 5: Drop old column and rename
ALTER TABLE orders DROP COLUMN status;
ALTER TABLE orders RENAME COLUMN status_new TO status;

-- Step 6: Drop old enum type
DROP TYPE order_status;

-- Step 7: Rename new enum type
ALTER TYPE order_status_new RENAME TO order_status;
```

## Best Practices

### 1. When to Use Enums

```sql
-- Use enums when:
-- ✅ Small, relatively stable set of values (< 50 values)
-- ✅ Values have natural ordering
-- ✅ Performance is important
-- ✅ Type safety is desired
-- ✅ Values are used across multiple tables

-- Examples of good enum candidates:
CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE day_of_week AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');
CREATE TYPE user_role AS ENUM ('guest', 'user', 'moderator', 'admin');

-- Avoid enums when:
-- ❌ Values change frequently
-- ❌ Large number of values (countries, categories with 100s of items)
-- ❌ Need rich metadata for values
-- ❌ Values are user-configurable
-- ❌ Need complex business logic per value
```

### 2. Enum Naming Conventions

```sql
-- Good enum naming
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered');
CREATE TYPE payment_method AS ENUM ('credit_card', 'debit_card', 'paypal', 'bank_transfer');
CREATE TYPE content_type AS ENUM ('article', 'video', 'podcast', 'infographic');

-- Consistent value naming (snake_case recommended)
CREATE TYPE notification_type AS ENUM (
    'order_confirmation',
    'payment_received', 
    'shipping_update',
    'delivery_notification',
    'account_activation'
);

-- Include sort hints in enum ordering when logical ordering matters
CREATE TYPE size_enum AS ENUM ('xs', 'small', 'medium', 'large', 'xl', 'xxl');
CREATE TYPE severity_level AS ENUM ('info', 'warning', 'error', 'critical');
```

### 3. Enum Documentation and Metadata

```sql
-- Document enums with comments
COMMENT ON TYPE order_status IS 'Possible states for order processing workflow';
COMMENT ON TYPE priority_level IS 'Priority levels for tasks and tickets, ordered from lowest to highest';

-- Create views for enum introspection
CREATE VIEW enum_values AS
SELECT 
    t.typname as enum_name,
    e.enumlabel as enum_value,
    e.enumsortorder as sort_order
FROM pg_type t 
JOIN pg_enum e ON t.oid = e.enumtypid
WHERE t.typtype = 'e'
ORDER BY t.typname, e.enumsortorder;

-- Query enum metadata
SELECT * FROM enum_values WHERE enum_name = 'order_status';
```

### 4. Testing Enum Constraints

```sql
-- Test enum constraints
DO $$
BEGIN
    -- Test valid enum values
    ASSERT (SELECT COUNT(*) FROM orders WHERE status = 'pending') >= 0;
    
    -- Test enum ordering
    ASSERT 'pending'::order_status < 'delivered'::order_status;
    
    -- Test invalid values (should raise exception)
    BEGIN
        INSERT INTO orders (customer_id, status, total_amount) 
        VALUES (1, 'invalid_status', 100.00);
        RAISE EXCEPTION 'Should not allow invalid enum value';
    EXCEPTION 
        WHEN invalid_text_representation THEN
            RAISE NOTICE 'Correctly rejected invalid enum value';
    END;
END $$;
```

## Conclusion

Enums are powerful tools for constraining values and improving data integrity:

**PostgreSQL Enums:**
- Excellent type safety and performance
- Natural ordering and comparison operations
- Can be challenging to modify in production
- Great for stable, well-defined value sets

**MySQL Enums:**
- Simpler implementation but less flexible
- Integer-based storage is efficient
- Limited comparison and ordering capabilities
- Suitable for basic use cases

**Best Practices:**
- Use enums for stable, small value sets
- Consider check constraints for more flexibility
- Use reference tables for rich metadata needs
- Plan enum modifications carefully in production
- Document enum meanings and usage patterns

Choose the approach that best balances your needs for performance, flexibility, and maintainability. 
