# Foreign Key Design Patterns: Complete Guide

Foreign keys are fundamental database constraints that maintain referential integrity between related tables. This guide explores when to use foreign keys, alternatives for complex scenarios, and best practices for maintaining data consistency.

## Table of Contents
- [Foreign Key Fundamentals](#foreign-key-fundamentals)
- [When to Use Foreign Keys](#when-to-use-foreign-keys)
- [When to Avoid Foreign Keys](#when-to-avoid-foreign-keys)
- [Polymorphic Associations](#polymorphic-associations)
- [Composite Foreign Keys](#composite-foreign-keys)
- [Soft References](#soft-references)
- [Performance Considerations](#performance-considerations)
- [Migration Strategies](#migration-strategies)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)

## Foreign Key Fundamentals

### Basic Foreign Key Relationships

```sql
-- One-to-Many: Classic foreign key relationship
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Many-to-Many: Junction table with composite foreign keys
CREATE TABLE product_tags (
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (product_id, tag_id)
);

-- Self-referencing: Hierarchical data
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    manager_id INTEGER REFERENCES employees(id) ON DELETE SET NULL,
    department_id INTEGER NOT NULL REFERENCES departments(id),
    hired_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Foreign Key Constraints and Actions

```sql
-- Different referential actions
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    shipping_address_id INTEGER REFERENCES addresses(id) ON DELETE SET NULL,
    billing_address_id INTEGER REFERENCES addresses(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Deferred constraint checking
CREATE TABLE parent_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE child_table (
    id INTEGER PRIMARY KEY,
    parent_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    CONSTRAINT fk_child_parent 
        FOREIGN KEY (parent_id) 
        REFERENCES parent_table(id) 
        DEFERRABLE INITIALLY DEFERRED
);
```

## When to Use Foreign Keys

### 1. Strong Referential Integrity Requirements

```sql
-- Financial transactions requiring strict consistency
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    account_number TEXT UNIQUE NOT NULL,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    balance DECIMAL(15,2) NOT NULL DEFAULT 0,
    account_type TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    from_account_id INTEGER REFERENCES accounts(id) ON DELETE RESTRICT,
    to_account_id INTEGER REFERENCES accounts(id) ON DELETE RESTRICT,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    transaction_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure at least one account is specified
    CHECK (from_account_id IS NOT NULL OR to_account_id IS NOT NULL)
);
```

### 2. Master-Detail Relationships

```sql
-- Invoice and line items with cascade delete
CREATE TABLE invoices (
    id SERIAL PRIMARY KEY,
    invoice_number TEXT UNIQUE NOT NULL,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE invoice_line_items (
    id SERIAL PRIMARY KEY,
    invoice_id INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    quantity DECIMAL(10,3) NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10,2) NOT NULL,
    line_total DECIMAL(10,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Trigger to update invoice total
CREATE OR REPLACE FUNCTION update_invoice_total()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices 
    SET total_amount = (
        SELECT COALESCE(SUM(line_total), 0)
        FROM invoice_line_items 
        WHERE invoice_id = COALESCE(NEW.invoice_id, OLD.invoice_id)
    )
    WHERE id = COALESCE(NEW.invoice_id, OLD.invoice_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_invoice_total
    AFTER INSERT OR UPDATE OR DELETE ON invoice_line_items
    FOR EACH ROW EXECUTE FUNCTION update_invoice_total();
```

## When to Avoid Foreign Keys

### 1. Cross-Service Boundaries (Microservices)

```sql
-- Avoid foreign keys across service boundaries
CREATE TABLE order_service_orders (
    id SERIAL PRIMARY KEY,
    order_number TEXT UNIQUE NOT NULL,
    customer_service_customer_id INTEGER NOT NULL, -- Reference to external service
    total_amount DECIMAL(10,2) NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Instead, use eventual consistency and saga patterns
CREATE TABLE order_events (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES order_service_orders(id),
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to publish domain events instead of relying on foreign keys
CREATE OR REPLACE FUNCTION publish_order_event(
    p_order_id INTEGER,
    p_event_type TEXT,
    p_payload JSONB
) RETURNS VOID AS $$
BEGIN
    INSERT INTO order_events (order_id, event_type, payload)
    VALUES (p_order_id, p_event_type, p_payload);
    
    -- Trigger external service notification
    PERFORM pg_notify('order_events', json_build_object(
        'order_id', p_order_id,
        'event_type', p_event_type,
        'payload', p_payload
    )::text);
END;
$$ LANGUAGE plpgsql;
```

### 2. High-Volume Logging and Analytics

```sql
-- Audit logs without foreign keys for performance
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL, -- Store as text to avoid FK overhead
    action TEXT NOT NULL,
    user_id TEXT, -- External user identifier
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Partitioned table for high volume
CREATE TABLE audit_logs_partitioned (
    id BIGSERIAL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    action TEXT NOT NULL,
    user_id TEXT,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Index for efficient querying without FK constraints
CREATE INDEX idx_audit_logs_entity ON audit_logs_partitioned (entity_type, entity_id, created_at);
```

### 3. Flexible Content Systems

```sql
-- CMS with flexible relationships
CREATE TABLE content_items (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content_type TEXT NOT NULL,
    body TEXT,
    author_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Flexible relationships without foreign keys
CREATE TABLE content_relationships (
    id SERIAL PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(source_type, source_id, target_type, target_id, relationship_type)
);

-- Composite identifier pattern
CREATE OR REPLACE FUNCTION make_entity_id(entity_type TEXT, entity_id INTEGER)
RETURNS TEXT AS $$
BEGIN
    RETURN entity_type || '-' || entity_id::TEXT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Usage examples
INSERT INTO content_relationships (source_type, source_id, target_type, target_id, relationship_type)
VALUES 
    ('article', make_entity_id('article', 1), 'category', make_entity_id('category', 5), 'belongs_to'),
    ('article', make_entity_id('article', 1), 'tag', make_entity_id('tag', 10), 'tagged_with'),
    ('article', make_entity_id('article', 1), 'user', make_entity_id('user', 123), 'authored_by');
```

## Polymorphic Associations

### 1. Single Table Polymorphism

```sql
-- Comments that can belong to different entities
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    commentable_type TEXT NOT NULL,
    commentable_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Create composite index for polymorphic queries
    INDEX idx_comments_polymorphic (commentable_type, commentable_id)
);

-- Separate index for each type for better performance
CREATE INDEX idx_comments_articles ON comments (commentable_id) 
WHERE commentable_type = 'article';

CREATE INDEX idx_comments_products ON comments (commentable_id) 
WHERE commentable_type = 'product';

-- Function to safely insert polymorphic comments
CREATE OR REPLACE FUNCTION create_comment(
    p_commentable_type TEXT,
    p_commentable_id INTEGER,
    p_author_id INTEGER,
    p_content TEXT
) RETURNS INTEGER AS $$
DECLARE
    entity_exists BOOLEAN := FALSE;
    comment_id INTEGER;
BEGIN
    -- Validate entity exists based on type
    CASE p_commentable_type
        WHEN 'article' THEN
            SELECT EXISTS(SELECT 1 FROM articles WHERE id = p_commentable_id) INTO entity_exists;
        WHEN 'product' THEN
            SELECT EXISTS(SELECT 1 FROM products WHERE id = p_commentable_id) INTO entity_exists;
        WHEN 'user' THEN
            SELECT EXISTS(SELECT 1 FROM users WHERE id = p_commentable_id) INTO entity_exists;
        ELSE
            RAISE EXCEPTION 'Invalid commentable_type: %', p_commentable_type;
    END CASE;
    
    IF NOT entity_exists THEN
        RAISE EXCEPTION 'Entity % with id % does not exist', p_commentable_type, p_commentable_id;
    END IF;
    
    INSERT INTO comments (commentable_type, commentable_id, author_id, content)
    VALUES (p_commentable_type, p_commentable_id, p_author_id, p_content)
    RETURNING id INTO comment_id;
    
    RETURN comment_id;
END;
$$ LANGUAGE plpgsql;
```

### 2. Multiple Foreign Key Approach

```sql
-- Alternative: Separate nullable foreign keys
CREATE TABLE activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    activity_type TEXT NOT NULL,
    
    -- Mutually exclusive foreign keys
    article_id INTEGER REFERENCES articles(id),
    product_id INTEGER REFERENCES products(id),
    comment_id INTEGER REFERENCES comments(id),
    
    action TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure exactly one foreign key is set
    CHECK (
        (article_id IS NOT NULL)::INTEGER + 
        (product_id IS NOT NULL)::INTEGER + 
        (comment_id IS NOT NULL)::INTEGER = 1
    )
);

-- Indexes for each relationship
CREATE INDEX idx_activities_article ON activities (article_id) WHERE article_id IS NOT NULL;
CREATE INDEX idx_activities_product ON activities (product_id) WHERE product_id IS NOT NULL;
CREATE INDEX idx_activities_comment ON activities (comment_id) WHERE comment_id IS NOT NULL;
```

### 3. Hybrid Approach with Validation

```sql
-- Combine polymorphic pattern with validation
CREATE TABLE attachments (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    
    -- Polymorphic relationship
    attachable_type TEXT NOT NULL,
    attachable_id INTEGER NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_attachments_polymorphic (attachable_type, attachable_id)
);

-- Validation function to ensure referential integrity
CREATE OR REPLACE FUNCTION validate_attachment_reference()
RETURNS TRIGGER AS $$
DECLARE
    entity_exists BOOLEAN := FALSE;
BEGIN
    -- Check if the referenced entity exists
    CASE NEW.attachable_type
        WHEN 'user' THEN
            SELECT EXISTS(SELECT 1 FROM users WHERE id = NEW.attachable_id) INTO entity_exists;
        WHEN 'article' THEN
            SELECT EXISTS(SELECT 1 FROM articles WHERE id = NEW.attachable_id) INTO entity_exists;
        WHEN 'product' THEN
            SELECT EXISTS(SELECT 1 FROM products WHERE id = NEW.attachable_id) INTO entity_exists;
        ELSE
            RAISE EXCEPTION 'Invalid attachable_type: %', NEW.attachable_type;
    END CASE;
    
    IF NOT entity_exists THEN
        RAISE EXCEPTION 'Referenced entity % with id % does not exist', 
                       NEW.attachable_type, NEW.attachable_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_attachment_reference
    BEFORE INSERT OR UPDATE ON attachments
    FOR EACH ROW EXECUTE FUNCTION validate_attachment_reference();
```

## Composite Foreign Keys

### Multi-Column References

```sql
-- Composite primary key scenario
CREATE TABLE organizations (
    id SERIAL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (id, code)
);

CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    org_id INTEGER NOT NULL,
    org_code TEXT NOT NULL,
    name TEXT NOT NULL,
    
    -- Composite foreign key
    FOREIGN KEY (org_id, org_code) REFERENCES organizations(id, code)
);

-- Temporal foreign keys with time periods
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    department_id INTEGER NOT NULL,
    valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
    valid_to DATE,
    
    FOREIGN KEY (department_id) REFERENCES departments(id)
);

CREATE TABLE employee_assignments (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    assignment_date DATE NOT NULL,
    end_date DATE,
    
    FOREIGN KEY (employee_id) REFERENCES employees(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    
    -- Ensure no overlapping assignments for same employee
    EXCLUDE USING gist (
        employee_id WITH =,
        daterange(assignment_date, end_date, '[]') WITH &&
    )
);
```

## Soft References

### Weak References with Validation

```sql
-- Notifications that reference multiple entity types
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    recipient_id INTEGER NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    
    -- Soft reference using composite identifier
    related_entity_type TEXT,
    related_entity_id TEXT,
    
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to create entity reference
CREATE OR REPLACE FUNCTION create_entity_reference(
    entity_type TEXT,
    entity_id INTEGER
) RETURNS TEXT AS $$
BEGIN
    RETURN entity_type || ':' || entity_id::TEXT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to parse entity reference
CREATE OR REPLACE FUNCTION parse_entity_reference(entity_ref TEXT)
RETURNS TABLE(entity_type TEXT, entity_id INTEGER) AS $$
DECLARE
    parts TEXT[];
BEGIN
    IF entity_ref IS NULL THEN
        RETURN;
    END IF;
    
    parts := string_to_array(entity_ref, ':');
    
    IF array_length(parts, 1) != 2 THEN
        RAISE EXCEPTION 'Invalid entity reference format: %', entity_ref;
    END IF;
    
    RETURN QUERY SELECT parts[1], parts[2]::INTEGER;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Usage
INSERT INTO notifications (recipient_id, title, message, related_entity_type, related_entity_id)
VALUES (
    123, 
    'New Order', 
    'Your order has been processed',
    'order',
    create_entity_reference('order', 456)
);
```

### Event Sourcing Pattern

```sql
-- Events without traditional foreign keys
CREATE TABLE domain_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_version INTEGER NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique constraint to prevent duplicate events
    UNIQUE(aggregate_type, aggregate_id, event_version)
);

-- Function to append events
CREATE OR REPLACE FUNCTION append_event(
    p_aggregate_type TEXT,
    p_aggregate_id TEXT,
    p_event_type TEXT,
    p_event_data JSONB,
    p_expected_version INTEGER DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    current_version INTEGER;
    event_id UUID;
BEGIN
    -- Get current version
    SELECT COALESCE(MAX(event_version), 0) INTO current_version
    FROM domain_events
    WHERE aggregate_type = p_aggregate_type AND aggregate_id = p_aggregate_id;
    
    -- Check expected version for optimistic concurrency
    IF p_expected_version IS NOT NULL AND current_version != p_expected_version THEN
        RAISE EXCEPTION 'Concurrency conflict: expected version %, current version %', 
                       p_expected_version, current_version;
    END IF;
    
    -- Insert new event
    INSERT INTO domain_events (aggregate_type, aggregate_id, event_type, event_version, event_data)
    VALUES (p_aggregate_type, p_aggregate_id, p_event_type, current_version + 1, p_event_data)
    RETURNING event_id INTO event_id;
    
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Foreign Key Index Strategy

```sql
-- Foreign keys automatically create indexes on referencing columns
-- But you may need additional indexes for performance

-- Multi-column index for common query patterns
CREATE INDEX idx_order_items_order_product ON order_items (order_id, product_id);

-- Partial indexes for active records only
CREATE INDEX idx_active_subscriptions_user ON subscriptions (user_id) 
WHERE status = 'active';

-- Covering indexes to avoid table lookups
CREATE INDEX idx_comments_with_content ON comments (commentable_type, commentable_id) 
INCLUDE (author_id, content, created_at);
```

### Foreign Key Performance Impact

```sql
-- Measure foreign key constraint checking overhead
EXPLAIN (ANALYZE, BUFFERS) 
INSERT INTO order_items (order_id, product_id, quantity, unit_price)
VALUES (12345, 67890, 2, 29.99);

-- Temporarily disable foreign key checks for bulk operations
-- (Use with extreme caution)
ALTER TABLE order_items DISABLE TRIGGER ALL;
-- Bulk insert operations
ALTER TABLE order_items ENABLE TRIGGER ALL;

-- Alternative: Use COPY for bulk inserts with validation
CREATE OR REPLACE FUNCTION bulk_insert_with_validation(
    p_table_name TEXT,
    p_data JSONB[]
) RETURNS INTEGER AS $$
DECLARE
    inserted_count INTEGER := 0;
    data_item JSONB;
BEGIN
    -- Validate all foreign key references first
    FOR data_item IN SELECT unnest(p_data) LOOP
        -- Custom validation logic here
        NULL;
    END LOOP;
    
    -- Then perform bulk insert
    -- Implementation depends on specific table structure
    
    RETURN inserted_count;
END;
$$ LANGUAGE plpgsql;
```

## Migration Strategies

### Adding Foreign Keys to Existing Data

```sql
-- Safe migration strategy for adding foreign keys
BEGIN;

-- Step 1: Add the column without constraint
ALTER TABLE existing_table ADD COLUMN new_foreign_key_id INTEGER;

-- Step 2: Populate the column with valid references
UPDATE existing_table 
SET new_foreign_key_id = (
    SELECT ref_table.id 
    FROM ref_table 
    WHERE ref_table.legacy_field = existing_table.legacy_reference
);

-- Step 3: Handle orphaned records
UPDATE existing_table 
SET new_foreign_key_id = NULL 
WHERE new_foreign_key_id IS NULL;

-- Step 4: Add the foreign key constraint
ALTER TABLE existing_table 
ADD CONSTRAINT fk_existing_new_reference 
FOREIGN KEY (new_foreign_key_id) REFERENCES ref_table(id);

-- Step 5: Create indexes
CREATE INDEX idx_existing_new_fk ON existing_table (new_foreign_key_id);

COMMIT;
```

### Removing Foreign Keys Safely

```sql
-- Safe foreign key removal strategy
BEGIN;

-- Step 1: Drop the constraint but keep the column
ALTER TABLE child_table DROP CONSTRAINT fk_child_parent;

-- Step 2: Add application-level validation if needed
CREATE OR REPLACE FUNCTION validate_parent_reference()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_id IS NOT NULL THEN
        IF NOT EXISTS(SELECT 1 FROM parent_table WHERE id = NEW.parent_id) THEN
            RAISE EXCEPTION 'Invalid parent reference: %', NEW.parent_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_parent_ref
    BEFORE INSERT OR UPDATE ON child_table
    FOR EACH ROW EXECUTE FUNCTION validate_parent_reference();

-- Step 3: Optionally convert to soft reference
ALTER TABLE child_table ADD COLUMN parent_reference TEXT;
UPDATE child_table SET parent_reference = 'parent:' || parent_id::TEXT WHERE parent_id IS NOT NULL;

COMMIT;
```

## Real-World Examples

### 1. E-commerce Order System

```sql
-- Strong foreign keys for critical business relationships
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    order_number TEXT UNIQUE NOT NULL,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    order_date TIMESTAMPTZ DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_sku TEXT NOT NULL, -- Soft reference to product catalog service
    product_name TEXT NOT NULL, -- Denormalized for performance
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10,2) NOT NULL,
    line_total DECIMAL(10,2) GENERATED ALWAYS AS (quantity * unit_price) STORED
);

-- Payments with soft references to external services
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    payment_method TEXT NOT NULL,
    external_transaction_id TEXT, -- Reference to payment processor
    amount DECIMAL(10,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL DEFAULT 'pending',
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 2. Content Management System

```sql
-- Flexible content system with mixed FK approaches
CREATE TABLE content_types (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    schema JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE content_items (
    id SERIAL PRIMARY KEY,
    content_type_id INTEGER NOT NULL REFERENCES content_types(id),
    title TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    content JSONB NOT NULL,
    author_id INTEGER NOT NULL, -- Soft reference to user service
    status TEXT NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Flexible tagging without foreign keys
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    category TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE taggings (
    id SERIAL PRIMARY KEY,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    taggable_type TEXT NOT NULL,
    taggable_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(tag_id, taggable_type, taggable_id)
);

-- Comments with validation
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    commentable_type TEXT NOT NULL,
    commentable_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL, -- Soft reference
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'published',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Validation trigger for comments
CREATE OR REPLACE FUNCTION validate_commentable_reference()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.commentable_type = 'content_item' THEN
        IF NOT EXISTS(SELECT 1 FROM content_items WHERE id = NEW.commentable_id) THEN
            RAISE EXCEPTION 'Content item % does not exist', NEW.commentable_id;
        END IF;
    ELSE
        RAISE EXCEPTION 'Invalid commentable_type: %', NEW.commentable_type;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_commentable
    BEFORE INSERT OR UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION validate_commentable_reference();
```

## Best Practices

### 1. Foreign Key Naming Conventions

```sql
-- Consistent naming pattern: fk_<child_table>_<parent_table>_<column>
ALTER TABLE order_items 
ADD CONSTRAINT fk_order_items_orders_order_id 
FOREIGN KEY (order_id) REFERENCES orders(id);

ALTER TABLE order_items 
ADD CONSTRAINT fk_order_items_products_product_id 
FOREIGN KEY (product_id) REFERENCES products(id);

-- For composite foreign keys
ALTER TABLE employee_departments 
ADD CONSTRAINT fk_employee_departments_employees_employee_id 
FOREIGN KEY (employee_id) REFERENCES employees(id);
```

### 2. Choosing Referential Actions

```sql
-- Guidelines for ON DELETE actions:

-- CASCADE: Use for dependent data that has no meaning without parent
CREATE TABLE invoice_line_items (
    invoice_id INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE
);

-- RESTRICT: Use for critical references that should never be orphaned
CREATE TABLE orders (
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE RESTRICT
);

-- SET NULL: Use for optional references
CREATE TABLE employees (
    manager_id INTEGER REFERENCES employees(id) ON DELETE SET NULL
);

-- SET DEFAULT: Use when a default value makes sense
CREATE TABLE products (
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE SET DEFAULT DEFAULT 1
);
```

### 3. Performance Optimization

```sql
-- Always index foreign key columns (if not done automatically)
CREATE INDEX idx_order_items_order_id ON order_items (order_id);
CREATE INDEX idx_order_items_product_id ON order_items (product_id);

-- Use covering indexes for common queries
CREATE INDEX idx_order_items_covering 
ON order_items (order_id) 
INCLUDE (product_id, quantity, unit_price);

-- Consider partial indexes for filtered queries
CREATE INDEX idx_active_orders_customer 
ON orders (customer_id) 
WHERE status IN ('pending', 'processing');
```

### 4. Documentation and Governance

```sql
-- Document foreign key relationships
COMMENT ON CONSTRAINT fk_orders_customers_customer_id ON orders 
IS 'Links orders to customers. RESTRICT prevents customer deletion if orders exist.';

-- Create views to document relationships
CREATE VIEW order_relationships AS
SELECT 
    o.id AS order_id,
    o.order_number,
    c.email AS customer_email,
    COUNT(oi.id) AS item_count,
    SUM(oi.line_total) AS calculated_total
FROM orders o
JOIN customers c ON o.customer_id = c.id
LEFT JOIN order_items oi ON o.id = oi.order_id
GROUP BY o.id, o.order_number, c.email;

-- Function to check referential integrity
CREATE OR REPLACE FUNCTION check_referential_integrity()
RETURNS TABLE(
    table_name TEXT,
    constraint_name TEXT,
    violated_rows INTEGER
) AS $$
BEGIN
    -- Implementation would check for orphaned records
    -- This is a simplified example
    RETURN QUERY
    SELECT 
        'order_items'::TEXT,
        'fk_order_items_orders_order_id'::TEXT,
        COUNT(*)::INTEGER
    FROM order_items oi
    LEFT JOIN orders o ON oi.order_id = o.id
    WHERE o.id IS NULL;
END;
$$ LANGUAGE plpgsql;
```

This comprehensive guide covers all aspects of foreign key design, from basic relationships to complex scenarios requiring alternative approaches. The key is to understand when foreign keys provide value (referential integrity, performance) versus when they create unnecessary coupling or performance overhead. Choose the right approach based on your specific requirements for consistency, performance, and architectural flexibility.

## References

- [Ardalis: Related Data Without Foreign Keys](https://ardalis.com/related-data-without-foreign-keys/)
- [Alibaba Cloud: Aggregation in Domain Driven Design](https://www.alibabacloud.com/blog/an-in-depth-understanding-of-aggregation-in-domain-driven-design_598034)
- [Ruby on Rails: Association Basics](https://guides.rubyonrails.org/association_basics.html)
- [PostgreSQL Documentation: Foreign Keys](https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-FK)
