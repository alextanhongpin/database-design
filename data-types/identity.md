# Identity and Primary Key Strategies

Comprehensive guide to choosing and implementing primary key strategies including serial, identity, and UUID approaches.

## 🎯 Overview

Primary key selection affects:
- **Security** - Exposure of business logic and data volumes
- **Performance** - Join performance and index efficiency
- **Scalability** - Distributed systems and replication
- **Maintainability** - Data integrity and foreign key relationships

## 🔢 Integer-Based Keys

### Serial vs Identity

```sql
-- PostgreSQL SERIAL (legacy approach)
CREATE TABLE users_serial (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- PostgreSQL IDENTITY (SQL standard approach)
CREATE TABLE users_identity (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- MySQL AUTO_INCREMENT
CREATE TABLE users_auto (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Sequence Management

```sql
-- PostgreSQL sequence operations
CREATE SEQUENCE custom_user_id_seq
    START WITH 1000
    INCREMENT BY 1
    MINVALUE 1000
    MAXVALUE 9999999
    CACHE 10;

CREATE TABLE users_custom_seq (
    id INTEGER DEFAULT nextval('custom_user_id_seq') PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reset sequence after data migration
SELECT setval('custom_user_id_seq', (SELECT MAX(id) FROM users_custom_seq));

-- Reset to start value
ALTER SEQUENCE custom_user_id_seq RESTART WITH 1000;
```

### Integer Key Advantages and Disadvantages

```sql
-- Advantages demonstration
EXPLAIN (ANALYZE, BUFFERS) 
SELECT u.email, p.title 
FROM users_identity u 
JOIN posts p ON u.id = p.user_id 
WHERE u.id = 12345;

-- Disadvantages - security through obscurity failure
CREATE TABLE vulnerable_orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Attacker can easily enumerate: /api/orders/1, /api/orders/2, etc.
-- Reveals business volume and allows unauthorized access
```

## 🆔 UUID-Based Keys

### UUID Generation Strategies

```sql
-- PostgreSQL UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Different UUID versions
CREATE TABLE uuid_examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- v4 (random)
    id_v1 UUID DEFAULT uuid_generate_v1(),         -- v1 (timestamp + MAC)
    id_v4 UUID DEFAULT uuid_generate_v4(),         -- v4 (random)
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Time-ordered UUIDs (PostgreSQL 13+)
CREATE TABLE time_ordered_uuids (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    time_uuid UUID DEFAULT uuid_generate_v1mc(), -- monotonic clock
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### UUID Performance Optimization

```sql
-- UUID storage optimization
CREATE TABLE optimized_uuids (
    id UUID PRIMARY KEY,
    -- Store UUID as text for display, binary for performance
    id_text TEXT GENERATED ALWAYS AS (id::text) STORED,
    -- Cluster-friendly ordering
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data JSONB
);

-- Index on time for better clustering
CREATE INDEX idx_optimized_uuids_time ON optimized_uuids (created_at);

-- UUID v6 simulation for better database clustering
CREATE OR REPLACE FUNCTION uuid_v6()
RETURNS UUID AS $$
DECLARE
    time_hi BIGINT;
    time_mid INTEGER;
    time_low INTEGER;
    clock_seq INTEGER;
    node_id BIGINT;
    result UUID;
BEGIN
    -- Get current timestamp in 100-nanosecond intervals since 1582-10-15
    SELECT EXTRACT(EPOCH FROM NOW()) * 10000000 + 122192928000000000 INTO time_hi;
    
    -- Generate random clock sequence and node ID
    SELECT (random() * 16383)::INTEGER INTO clock_seq;
    SELECT (random() * 281474976710655)::BIGINT INTO node_id;
    
    -- Construct UUID v6 (time-ordered)
    time_mid := (time_hi >> 32)::INTEGER;
    time_low := time_hi::INTEGER;
    
    result := (
        lpad(to_hex(time_hi >> 32), 8, '0') ||
        lpad(to_hex((time_hi >> 16) & 65535), 4, '0') ||
        '6' || lpad(to_hex(time_hi & 4095), 3, '0') ||
        lpad(to_hex(clock_seq | 32768), 4, '0') ||
        lpad(to_hex(node_id), 12, '0')
    )::UUID;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

## 🔀 Composite Keys

### Natural Composite Keys

```sql
-- Order line items with composite key
CREATE TABLE order_items (
    order_id UUID NOT NULL,
    item_sequence INTEGER NOT NULL,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (order_id, item_sequence),
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Multi-tenant data with composite key
CREATE TABLE tenant_users (
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (tenant_id, user_id),
    UNIQUE (tenant_id, email) -- Email unique within tenant
);
```

### Surrogate vs Natural Keys

```sql
-- Natural key example (can change)
CREATE TABLE countries_natural (
    country_code CHAR(3) PRIMARY KEY, -- ISO 3166-1 alpha-3
    country_name VARCHAR(100) NOT NULL,
    continent VARCHAR(50) NOT NULL
);

-- Surrogate key with natural key as unique constraint
CREATE TABLE countries_surrogate (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code CHAR(3) UNIQUE NOT NULL, -- Still enforce uniqueness
    country_name VARCHAR(100) NOT NULL,
    continent VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Foreign key references - compare approaches
CREATE TABLE addresses_natural (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code CHAR(3) REFERENCES countries_natural(country_code),
    address_line TEXT NOT NULL
);

CREATE TABLE addresses_surrogate (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_id UUID REFERENCES countries_surrogate(id),
    address_line TEXT NOT NULL
);
```

## 🔄 Key Generation Strategies

### Application-Generated Keys

```sql
-- Snowflake-style ID generation
CREATE OR REPLACE FUNCTION generate_snowflake_id(
    datacenter_id INTEGER DEFAULT 1,
    worker_id INTEGER DEFAULT 1
)
RETURNS BIGINT AS $$
DECLARE
    epoch_ms BIGINT := 1609459200000; -- 2021-01-01 00:00:00 UTC
    current_ms BIGINT;
    sequence_num INTEGER := 0;
    snowflake_id BIGINT;
BEGIN
    current_ms := EXTRACT(EPOCH FROM NOW()) * 1000;
    
    -- Snowflake format: timestamp(41) + datacenter(5) + worker(5) + sequence(12)
    snowflake_id := ((current_ms - epoch_ms) << 22) |
                   ((datacenter_id & 31) << 17) |
                   ((worker_id & 31) << 12) |
                   (sequence_num & 4095);
    
    RETURN snowflake_id;
END;
$$ LANGUAGE plpgsql;

-- Usage
CREATE TABLE snowflake_records (
    id BIGINT PRIMARY KEY DEFAULT generate_snowflake_id(),
    data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Database-Generated Keys

```sql
-- PostgreSQL with multiple generation strategies
CREATE TABLE multi_key_example (
    -- Sequential integer
    seq_id INTEGER GENERATED ALWAYS AS IDENTITY,
    -- Random UUID
    uuid_id UUID DEFAULT gen_random_uuid(),
    -- Time-based key
    time_key BIGINT DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000000 + EXTRACT(MICROSECONDS FROM NOW()),
    -- Hash-based key
    hash_key INTEGER DEFAULT abs(hashtext(gen_random_uuid()::text)),
    
    data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (uuid_id) -- Choose primary key strategy
);
```

## 🛡️ Security Considerations

### Preventing Enumeration Attacks

```sql
-- Vulnerable design
CREATE TABLE vulnerable_documents (
    id SERIAL PRIMARY KEY, -- Predictable, reveals volume
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    user_id INTEGER NOT NULL
);

-- Secure design
CREATE TABLE secure_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Unpredictable
    public_id VARCHAR(20) UNIQUE NOT NULL, -- For URLs, generated separately
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Generate public ID function
CREATE OR REPLACE FUNCTION generate_public_id()
RETURNS VARCHAR(20) AS $$
DECLARE
    chars TEXT := 'ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789';
    result TEXT := '';
    i INTEGER;
BEGIN
    FOR i IN 1..20 LOOP
        result := result || substring(chars, floor(random() * length(chars) + 1)::integer, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

### Access Control with Keys

```sql
-- Row-level security with UUIDs
CREATE TABLE user_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Enable row-level security
ALTER TABLE user_documents ENABLE ROW LEVEL SECURITY;

-- Policy: users can only access their own documents
CREATE POLICY user_documents_policy ON user_documents
    FOR ALL
    TO authenticated_users
    USING (user_id = current_setting('app.current_user_id')::UUID);
```

## 🚀 Performance Optimization

### Index Strategies

```sql
-- UUID performance optimization
CREATE TABLE uuid_performance (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data JSONB
);

-- Cluster table by time instead of UUID for better performance
CREATE INDEX idx_uuid_performance_time ON uuid_performance (created_at);
CLUSTER uuid_performance USING idx_uuid_performance_time;

-- Partial indexes for active records
CREATE INDEX idx_active_records ON uuid_performance (id)
WHERE created_at > CURRENT_DATE - INTERVAL '30 days';
```

### Join Performance

```sql
-- Compare join performance
EXPLAIN (ANALYZE, BUFFERS, COSTS)
SELECT u.email, COUNT(p.id) as post_count
FROM users_identity u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.email;

EXPLAIN (ANALYZE, BUFFERS, COSTS)
SELECT u.email, COUNT(p.id) as post_count
FROM users_uuid u
LEFT JOIN posts_uuid p ON u.id = p.user_id
GROUP BY u.id, u.email;
```

## 📊 Use Case Guidelines

### When to Use Each Strategy

```sql
-- Reference/lookup tables: Use integers
CREATE TABLE user_roles (
    id INTEGER PRIMARY KEY,
    role_name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

INSERT INTO user_roles VALUES
(1, 'admin', 'System administrator'),
(2, 'user', 'Regular user'),
(3, 'guest', 'Guest user');

-- User data: Use UUIDs
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    role_id INTEGER REFERENCES user_roles(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Log/audit tables: Use time-ordered keys
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_id UUID REFERENCES users(id),
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB
);

-- Create time-based index for log queries
CREATE INDEX idx_audit_logs_time ON audit_logs (event_time);
```

## 🔄 Migration Strategies

### Converting Between Key Types

```sql
-- Migration from integer to UUID
CREATE TABLE users_new (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    old_id INTEGER UNIQUE, -- Keep for migration
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Migration procedure
CREATE OR REPLACE FUNCTION migrate_to_uuid()
RETURNS VOID AS $$
DECLARE
    rec RECORD;
BEGIN
    -- Insert all existing records with new UUIDs
    FOR rec IN SELECT * FROM users_identity LOOP
        INSERT INTO users_new (old_id, email, created_at)
        VALUES (rec.id, rec.email, rec.created_at);
    END LOOP;
    
    -- Update all foreign key references
    -- This would need to be done for each referencing table
    -- using the old_id -> new UUID mapping
END;
$$ LANGUAGE plpgsql;
```

## 🎯 Best Practices

### Key Selection Decision Tree

1. **Public-facing APIs**: Use UUIDs
2. **Internal reference data**: Use integers
3. **High-volume transactional data**: Consider performance implications
4. **Multi-tenant systems**: Use composite keys or UUIDs
5. **Audit/logging systems**: Use time-ordered keys

### Common Pitfalls

```sql
-- ❌ Bad: Exposing internal IDs
SELECT id, title FROM articles WHERE id = 123; -- Reveals volume

-- ✅ Good: Use public-facing identifiers
SELECT id, title FROM articles WHERE public_id = 'abc123xyz';

-- ❌ Bad: UUID as clustering key in high-write scenario
CREATE TABLE high_write_uuid (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
); -- Random UUIDs cause index fragmentation

-- ✅ Good: Cluster by time, use UUID as alternate key
CREATE TABLE high_write_optimized (
    id UUID UNIQUE DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (created_at, id)
);
```

### Performance Monitoring

```sql
-- Monitor key performance
CREATE VIEW key_performance_metrics AS
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_tup_read DESC;

-- Monitor table sizes
CREATE VIEW table_size_metrics AS
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
    pg_total_relation_size(schemaname||'.'||tablename) as size_bytes
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY size_bytes DESC;
```

This comprehensive guide provides strategies for choosing and implementing appropriate primary key strategies based on your specific use case requirements.

## Human readable uuid, base32

https://github.com/solsson/uuid-base32
https://connect2id.com/blog/how-to-generate-human-friendly-identifiers


## Postgres Identity

- Don't use serial, use identity
- identity comes in two flavour - generated by default or generated always

```sql
drop table users;
create table if not exists users (
	id int generated by default as identity primary key,
--	id int generated always as identity primary key,
--ERROR:  cannot insert a non-DEFAULT value into column "id"
--DETAIL:  Column "id" is an identity column defined as GENERATED ALWAYS.
--HINT:  Use OVERRIDING SYSTEM VALUE to override.
	name text not null
);

insert into users(id, name) values (1, 'jane');
insert into users(name) values ('haha');
alter table users alter column id restart with 10;
table users;
```
