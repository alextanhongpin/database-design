# Database ID Patterns: Complete Guide

Choosing the right identifier strategy is crucial for database design. This guide covers different approaches to primary keys, their trade-offs, and when to use each pattern.

## Table of Contents
- [Integer vs UUID Comparison](#integer-vs-uuid-comparison)
- [UUID Patterns](#uuid-patterns)
- [Hybrid Approaches](#hybrid-approaches)
- [Performance Considerations](#performance-considerations)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)

## Integer vs UUID Comparison

### Auto-Increment Integer IDs

```sql
-- Traditional integer primary key
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pros:
-- ✅ Small storage footprint (4-8 bytes)
-- ✅ Fast indexing and joins
-- ✅ Human-readable
-- ✅ Sequential ordering
-- ✅ Database-native generation

-- Cons:
-- ❌ Predictable/enumerable
-- ❌ Exposes business information (user count)
-- ❌ Difficult in distributed systems
-- ❌ Merge conflicts in multi-master setups
```

### UUID Primary Keys

```sql
-- UUID primary key (PostgreSQL)
CREATE TABLE products (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pros:
-- ✅ Globally unique
-- ✅ Non-predictable
-- ✅ Great for distributed systems
-- ✅ No coordination needed for generation
-- ✅ Privacy-friendly

-- Cons:
-- ❌ Larger storage (16 bytes)
-- ❌ Slower indexing due to randomness
-- ❌ Not human-friendly
-- ❌ No natural ordering
```

## UUID Patterns

### UUID with Sequential Component

```sql
-- Hybrid approach: UUID primary key with integer sort column
CREATE TABLE orders (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    order_number BIGINT GENERATED ALWAYS AS IDENTITY,
    customer_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Index on sequential column for efficient ordering
    INDEX idx_orders_number (order_number),
    INDEX idx_orders_created (created_at, order_number)
);

-- Query using human-friendly order number
SELECT * FROM orders WHERE order_number = 12345;

-- Efficient pagination using sequential ordering
SELECT * FROM orders 
ORDER BY order_number DESC 
LIMIT 20 OFFSET 100;

-- Foreign key references still use UUID for security
CREATE TABLE order_items (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id),
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL
);
```

### Time-Ordered UUIDs (ULID Pattern)

```sql
-- Custom ULID-like function for PostgreSQL
CREATE OR REPLACE FUNCTION generate_ulid()
RETURNS TEXT AS $$
DECLARE
    timestamp_part BIGINT;
    random_part TEXT;
BEGIN
    -- Get current timestamp in milliseconds
    timestamp_part := EXTRACT(EPOCH FROM NOW()) * 1000;
    
    -- Generate random part
    random_part := encode(gen_random_bytes(10), 'base32');
    
    -- Combine timestamp and random parts
    RETURN lpad(timestamp_part::TEXT, 10, '0') || random_part;
END;
$$ LANGUAGE plpgsql;

-- Table using time-ordered identifiers
CREATE TABLE events (
    id TEXT DEFAULT generate_ulid() PRIMARY KEY,
    event_type TEXT NOT NULL,
    user_id UUID,
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Natural chronological ordering without additional timestamp column
SELECT * FROM events ORDER BY id DESC LIMIT 10;
```

### UUID v7 (Time-Ordered, Future Standard)

```sql
-- PostgreSQL extension for UUID v7 (when available)
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Simulated UUID v7 function (approximation)
CREATE OR REPLACE FUNCTION generate_uuid_v7()
RETURNS UUID AS $$
DECLARE
    timestamp_ms BIGINT;
    random_bytes BYTEA;
    uuid_bytes BYTEA;
BEGIN
    -- Current timestamp in milliseconds since Unix epoch
    timestamp_ms := EXTRACT(EPOCH FROM NOW()) * 1000;
    
    -- Generate 10 random bytes
    random_bytes := gen_random_bytes(10);
    
    -- Construct UUID v7 bytes (simplified)
    uuid_bytes := 
        decode(lpad(to_hex(timestamp_ms >> 16), 8, '0'), 'hex') ||
        decode(lpad(to_hex(timestamp_ms & 65535), 4, '0'), 'hex') ||
        decode('7', 'hex') || substring(random_bytes from 1 for 1) ||
        decode('8', 'hex') || substring(random_bytes from 2 for 9);
    
    RETURN encode(uuid_bytes, 'hex')::UUID;
END;
$$ LANGUAGE plpgsql;

-- Usage
CREATE TABLE notifications (
    id UUID DEFAULT generate_uuid_v7() PRIMARY KEY,
    user_id UUID NOT NULL,
    title TEXT NOT NULL,
    message TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Hybrid Approaches

### Dual Key Pattern

```sql
-- Best of both worlds: UUID for security, integer for usability
CREATE TABLE customers (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    customer_number BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    company_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Application logic considerations:
-- - Use UUID in URLs and API responses: /api/customers/123e4567-e89b-12d3-a456-426614174000
-- - Use customer_number for human communication: "Customer #12345"
-- - Use customer_number for efficient sorting and pagination
-- - Use UUID for all foreign key relationships

-- Efficient queries using either identifier
CREATE INDEX idx_customers_number ON customers(customer_number);
CREATE INDEX idx_customers_uuid ON customers(id);

-- Customer lookup by number (human-friendly)
SELECT * FROM customers WHERE customer_number = 12345;

-- API endpoint using UUID (secure)
SELECT * FROM customers WHERE id = '123e4567-e89b-12d3-a456-426614174000';
```

### Domain-Specific IDs

```sql
-- Generate meaningful IDs for different entity types
CREATE OR REPLACE FUNCTION generate_typed_id(prefix TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN prefix || '_' || 
           replace(gen_random_uuid()::TEXT, '-', '') ||
           extract(epoch from now())::BIGINT;
END;
$$ LANGUAGE plpgsql;

-- Usage in different tables
CREATE TABLE users (
    id TEXT DEFAULT generate_typed_id('usr') PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE products (
    id TEXT DEFAULT generate_typed_id('prd') PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE orders (
    id TEXT DEFAULT generate_typed_id('ord') PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending'
);

-- Example IDs generated:
-- usr_a1b2c3d4e5f6789012345678901234561672531200
-- prd_f6e5d4c3b2a1987654321098765432101672531201
-- ord_9876543210abcdef1234567890abcdef1672531202
```

### Masked Integer IDs

```sql
-- Hash/encode integer IDs for external use
CREATE OR REPLACE FUNCTION encode_id(id INTEGER, salt TEXT DEFAULT 'your-secret-salt')
RETURNS TEXT AS $$
BEGIN
    -- Simple encoding (use a proper library in production)
    RETURN encode(digest((id::TEXT || salt), 'sha256'), 'base64')::TEXT;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION decode_id(encoded_id TEXT, salt TEXT DEFAULT 'your-secret-salt')
RETURNS INTEGER AS $$
DECLARE
    test_id INTEGER;
    test_encoded TEXT;
BEGIN
    -- This is a simplified example - in production, use proper hash ID libraries
    -- like HashIds or similar reversible encoding
    FOR test_id IN 1..1000000 LOOP
        test_encoded := encode_id(test_id, salt);
        IF test_encoded = encoded_id THEN
            RETURN test_id;
        END IF;
    END LOOP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Usage
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    author_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- In application, expose encoded IDs
-- Internal: SELECT *, encode_id(id) as public_id FROM articles;
-- External API: /articles/aGk3Y2F0cy5jb20=
```

## Performance Considerations

### Indexing Strategies

```sql
-- UUID indexing considerations
CREATE TABLE performance_test_uuid (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    data TEXT,
    category_id INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Random UUIDs cause index fragmentation
-- Use covering indexes to reduce I/O
CREATE INDEX idx_perf_uuid_covering 
ON performance_test_uuid (category_id, created_at) 
INCLUDE (id, data);

-- For time-ordered UUIDs, standard B-tree works well
CREATE TABLE performance_test_ulid (
    id TEXT DEFAULT generate_ulid() PRIMARY KEY,
    data TEXT,
    category_id INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Standard B-tree index performs well with ordered IDs
CREATE INDEX idx_perf_ulid_category ON performance_test_ulid (category_id, id);
```

### Storage Optimization

```sql
-- Compare storage requirements
SELECT 
    pg_size_pretty(pg_total_relation_size('performance_test_uuid')) as uuid_size,
    pg_size_pretty(pg_total_relation_size('performance_test_ulid')) as ulid_size;

-- Monitor index fragmentation
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_blks_read,
    idx_blks_hit,
    round((idx_blks_hit::DECIMAL / NULLIF(idx_blks_hit + idx_blks_read, 0)) * 100, 2) as cache_hit_ratio
FROM pg_stat_user_indexes
WHERE tablename IN ('performance_test_uuid', 'performance_test_ulid');
```

## Real-World Examples

### E-commerce System

```sql
-- E-commerce with hybrid ID strategy
CREATE TABLE customers (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    customer_number BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    email TEXT UNIQUE NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE orders (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    order_number TEXT GENERATED ALWAYS AS (
        'ORD-' || lpad(extract(year from created_at)::TEXT, 4, '0') ||
        '-' || lpad(order_sequence::TEXT, 6, '0')
    ) STORED,
    order_sequence BIGINT GENERATED ALWAYS AS IDENTITY,
    customer_id UUID NOT NULL REFERENCES customers(id),
    status TEXT NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(order_number)
);

-- Customer service can reference ORD-2024-000123
-- APIs use UUID: /api/orders/550e8400-e29b-41d4-a716-446655440000
-- Database relationships use UUID for security and consistency
```

### SaaS Multi-Tenant System

```sql
-- Tenant-aware ID generation
CREATE TABLE tenants (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION generate_tenant_scoped_id(
    tenant_id UUID,
    prefix TEXT DEFAULT ''
)
RETURNS TEXT AS $$
DECLARE
    tenant_slug TEXT;
    timestamp_part TEXT;
    random_part TEXT;
BEGIN
    -- Get tenant slug for readable IDs
    SELECT slug INTO tenant_slug FROM tenants WHERE id = tenant_id;
    
    timestamp_part := extract(epoch from now())::BIGINT::TEXT;
    random_part := encode(gen_random_bytes(4), 'hex');
    
    RETURN CASE 
        WHEN prefix = '' THEN tenant_slug || '_' || timestamp_part || '_' || random_part
        ELSE tenant_slug || '_' || prefix || '_' || timestamp_part || '_' || random_part
    END;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE projects (
    id TEXT, -- Will be set by trigger
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (id)
);

-- Trigger to generate tenant-scoped IDs
CREATE OR REPLACE FUNCTION set_tenant_scoped_id()
RETURNS TRIGGER AS $$
BEGIN
    NEW.id := generate_tenant_scoped_id(NEW.tenant_id, 'proj');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_project_id
    BEFORE INSERT ON projects
    FOR EACH ROW
    EXECUTE FUNCTION set_tenant_scoped_id();

-- Example generated IDs:
-- acme_proj_1672531200_a1b2c3d4 (for tenant 'acme')
-- widgets_proj_1672531201_e5f6g7h8 (for tenant 'widgets-inc')
```

### Social Media Platform

```sql
-- Social media with ULIDs for chronological ordering
CREATE TABLE posts (
    id TEXT DEFAULT generate_ulid() PRIMARY KEY,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    likes_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE comments (
    id TEXT DEFAULT generate_ulid() PRIMARY KEY,
    post_id TEXT NOT NULL REFERENCES posts(id),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Natural chronological ordering without additional sorting
SELECT * FROM posts ORDER BY id DESC LIMIT 10;

-- Efficient range queries using ID
SELECT * FROM comments 
WHERE post_id = 'specific_post_id'
AND id > 'last_seen_comment_id'
ORDER BY id ASC
LIMIT 20;
```

## Best Practices

### 1. Choose Based on Requirements

```sql
-- Decision matrix for ID strategy

-- Use INTEGER IDs when:
-- - Small to medium datasets
-- - Single database instance
-- - Human readability is important
-- - Performance is critical
-- - Simple pagination requirements

-- Use UUID when:
-- - Distributed systems
-- - Multi-tenant applications
-- - High security requirements
-- - External API exposure
-- - Merge/replication scenarios

-- Use HYBRID approach when:
-- - Need both security and usability
-- - Customer-facing applications
-- - Internal tools need readable IDs
-- - APIs need non-predictable IDs
```

### 2. Consistent ID Patterns

```sql
-- Establish consistent patterns across your application
CREATE OR REPLACE FUNCTION standard_uuid()
RETURNS UUID AS $$
BEGIN
    RETURN gen_random_uuid();
END;
$$ LANGUAGE plpgsql;

-- Use consistent naming conventions
CREATE TABLE entities_with_standard_ids (
    id UUID DEFAULT standard_uuid() PRIMARY KEY,
    entity_number BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    -- ... other columns
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. Migration Strategies

```sql
-- Safe migration from INTEGER to UUID
-- Step 1: Add UUID column
ALTER TABLE legacy_table ADD COLUMN uuid_id UUID DEFAULT gen_random_uuid();

-- Step 2: Populate UUIDs for existing records
UPDATE legacy_table SET uuid_id = gen_random_uuid() WHERE uuid_id IS NULL;

-- Step 3: Add unique constraint
ALTER TABLE legacy_table ADD CONSTRAINT unique_uuid_id UNIQUE (uuid_id);

-- Step 4: Update application to use UUID
-- Step 5: Create new tables with UUID primary key
-- Step 6: Migrate foreign key references
-- Step 7: Drop old integer ID column (when safe)
```

### 4. Testing and Monitoring

```sql
-- Monitor ID generation performance
CREATE OR REPLACE FUNCTION benchmark_id_generation(iterations INTEGER DEFAULT 10000)
RETURNS TABLE(
    strategy TEXT,
    duration_ms NUMERIC,
    ids_per_second NUMERIC
) AS $$
DECLARE
    start_time TIMESTAMPTZ;
    end_time TIMESTAMPTZ;
    i INTEGER;
    dummy_uuid UUID;
    dummy_text TEXT;
BEGIN
    -- Test UUID generation
    start_time := clock_timestamp();
    FOR i IN 1..iterations LOOP
        dummy_uuid := gen_random_uuid();
    END LOOP;
    end_time := clock_timestamp();
    
    RETURN QUERY SELECT 
        'UUID'::TEXT,
        extract(milliseconds from (end_time - start_time))::NUMERIC,
        (iterations / extract(seconds from (end_time - start_time)))::NUMERIC;
    
    -- Test ULID generation
    start_time := clock_timestamp();
    FOR i IN 1..iterations LOOP
        dummy_text := generate_ulid();
    END LOOP;
    end_time := clock_timestamp();
    
    RETURN QUERY SELECT 
        'ULID'::TEXT,
        extract(milliseconds from (end_time - start_time))::NUMERIC,
        (iterations / extract(seconds from (end_time - start_time)))::NUMERIC;
END;
$$ LANGUAGE plpgsql;

-- Run benchmark
SELECT * FROM benchmark_id_generation(1000);
```

## Conclusion

The choice of ID strategy significantly impacts your application's scalability, security, and usability:

**Key Decision Factors:**
- **Scale**: Distributed systems favor UUIDs
- **Security**: UUIDs prevent enumeration attacks
- **Performance**: Integers are faster for joins and sorting
- **Usability**: Hybrid approaches balance technical and human needs

**Recommended Patterns:**
- **Small apps**: Auto-increment integers with encoding for external use
- **Distributed systems**: UUIDs with sequential component where needed
- **Customer-facing**: Hybrid UUID + human-readable number
- **High-throughput**: Time-ordered UUIDs (ULID/UUID v7)

Remember to consider your specific requirements for consistency, performance, security, and developer experience when choosing an ID strategy.


