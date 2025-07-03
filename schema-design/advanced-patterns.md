# Advanced Database Schema Patterns

Complex schema design patterns for sophisticated applications and business requirements.

## Deferrable Constraints

### What are Deferrable Constraints?

Deferrable constraints allow you to temporarily defer constraint checking until the end of a transaction, enabling complex multi-step operations that might temporarily violate constraints.

### Basic Syntax
```sql
-- Create deferrable constraint
ALTER TABLE orders 
ADD CONSTRAINT check_order_total 
CHECK (total >= 0) 
DEFERRABLE INITIALLY IMMEDIATE;

-- Use in transaction
BEGIN;
    SET CONSTRAINTS check_order_total DEFERRED;
    
    -- These operations might temporarily violate the constraint
    UPDATE orders SET total = -100 WHERE id = 1;  -- Temporarily negative
    UPDATE order_items SET quantity = 0 WHERE order_id = 1;
    UPDATE orders SET total = 0 WHERE id = 1;     -- Now valid
    
COMMIT; -- Constraint checked here
```

### Practical Example: Inventory Rebalancing
```sql
CREATE TABLE inventory (
    product_id INT PRIMARY KEY,
    warehouse_a_qty INT NOT NULL DEFAULT 0,
    warehouse_b_qty INT NOT NULL DEFAULT 0,
    total_qty INT NOT NULL DEFAULT 0,
    
    CONSTRAINT check_total_qty 
    CHECK (total_qty = warehouse_a_qty + warehouse_b_qty)
    DEFERRABLE INITIALLY IMMEDIATE
);

-- Transfer inventory between warehouses
BEGIN;
    SET CONSTRAINTS check_total_qty DEFERRED;
    
    -- Transfer 50 units from A to B
    UPDATE inventory 
    SET warehouse_a_qty = warehouse_a_qty - 50
    WHERE product_id = 123;  -- total_qty constraint violated temporarily
    
    UPDATE inventory 
    SET warehouse_b_qty = warehouse_b_qty + 50  
    WHERE product_id = 123;  -- Still violated
    
    UPDATE inventory 
    SET total_qty = warehouse_a_qty + warehouse_b_qty
    WHERE product_id = 123;  -- Now constraint satisfied
    
COMMIT; -- All constraints checked successfully
```

### Foreign Key Deferrals
```sql
-- Circular references require deferrable FKs
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    manager_id INT REFERENCES employees(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    head_employee_id INT REFERENCES employees(id) DEFERRABLE INITIALLY DEFERRED
);

ALTER TABLE employees 
ADD COLUMN department_id INT REFERENCES departments(id) DEFERRABLE INITIALLY DEFERRED;

-- Insert with circular dependencies
BEGIN;
    INSERT INTO departments (id, name) VALUES (1, 'Engineering');
    INSERT INTO employees (id, name, department_id) VALUES (1, 'Alice', 1);
    UPDATE departments SET head_employee_id = 1 WHERE id = 1;
COMMIT;
```

## User-Defined Ordering

### Custom Sort Orders with Explicit Positioning
```sql
CREATE TABLE menu_items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category_id INT NOT NULL,
    sort_order DECIMAL(10,5) NOT NULL,
    
    UNIQUE(category_id, sort_order)
);

-- Insert items with fractional ordering
INSERT INTO menu_items (name, category_id, sort_order) VALUES
('Appetizer 1', 1, 1.0),
('Appetizer 2', 1, 2.0),
('Appetizer 3', 1, 3.0);

-- Insert between items 1 and 2
INSERT INTO menu_items (name, category_id, sort_order) 
VALUES ('New Appetizer', 1, 1.5);

-- Query in custom order
SELECT name FROM menu_items 
WHERE category_id = 1 
ORDER BY sort_order;
```

### Hierarchical Ordering with Materialized Path
```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    path TEXT NOT NULL, -- e.g., '001.002.001'
    level INT NOT NULL,
    
    UNIQUE(path)
);

-- Function to generate next path at level
CREATE OR REPLACE FUNCTION next_category_path(parent_path TEXT DEFAULT '')
RETURNS TEXT AS $$
DECLARE
    next_num INT;
    new_path TEXT;
BEGIN
    IF parent_path = '' THEN
        -- Top level
        SELECT COALESCE(MAX(CAST(path AS INT)), 0) + 1 
        INTO next_num
        FROM categories 
        WHERE level = 1;
        
        RETURN LPAD(next_num::TEXT, 3, '0');
    ELSE
        -- Child level
        SELECT COALESCE(MAX(CAST(SPLIT_PART(path, '.', array_length(string_to_array(parent_path, '.'), 1) + 1) AS INT)), 0) + 1
        INTO next_num
        FROM categories 
        WHERE path LIKE parent_path || '.%' 
        AND level = array_length(string_to_array(parent_path, '.'), 1) + 1;
        
        RETURN parent_path || '.' || LPAD(next_num::TEXT, 3, '0');
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Insert hierarchical categories
INSERT INTO categories (name, path, level) VALUES
('Electronics', next_category_path(), 1),
('Computers', next_category_path('001'), 2),
('Laptops', next_category_path('001.001'), 3);
```

## Advanced Constraint Patterns

### Range Constraints with Overlaps
```sql
-- Prevent overlapping time periods
CREATE TABLE bookings (
    id SERIAL PRIMARY KEY,
    resource_id INT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    
    CONSTRAINT no_overlap 
    EXCLUDE USING GIST (
        resource_id WITH =,
        tsrange(start_time, end_time) WITH &&
    )
);

-- Example usage
INSERT INTO bookings (resource_id, start_time, end_time) VALUES
(1, '2024-07-01 09:00', '2024-07-01 11:00');

-- This will fail due to overlap
INSERT INTO bookings (resource_id, start_time, end_time) VALUES
(1, '2024-07-01 10:00', '2024-07-01 12:00');
```

### State Machine Constraints
```sql
CREATE TABLE order_states (
    state VARCHAR(20) PRIMARY KEY
);

INSERT INTO order_states VALUES 
('pending'), ('confirmed'), ('shipped'), ('delivered'), ('cancelled');

CREATE TABLE valid_transitions (
    from_state VARCHAR(20) REFERENCES order_states(state),
    to_state VARCHAR(20) REFERENCES order_states(state),
    PRIMARY KEY (from_state, to_state)
);

INSERT INTO valid_transitions VALUES
('pending', 'confirmed'),
('pending', 'cancelled'),
('confirmed', 'shipped'),
('confirmed', 'cancelled'),
('shipped', 'delivered');

-- Constraint to enforce valid state transitions
CREATE OR REPLACE FUNCTION validate_state_transition()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.state = NEW.state THEN
        RETURN NEW; -- No change
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM valid_transitions 
        WHERE from_state = OLD.state AND to_state = NEW.state
    ) THEN
        RAISE EXCEPTION 'Invalid state transition from % to %', OLD.state, NEW.state;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enforce_state_transitions
    BEFORE UPDATE OF state ON orders
    FOR EACH ROW
    EXECUTE FUNCTION validate_state_transition();
```

## Multi-Dimensional Data Patterns

### Time-Series with Multiple Granularities
```sql
-- Store data at multiple time granularities
CREATE TABLE metrics_raw (
    timestamp TIMESTAMP NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    value DECIMAL(15,4) NOT NULL,
    tags JSONB,
    
    PRIMARY KEY (timestamp, metric_name)
);

CREATE TABLE metrics_hourly (
    hour_timestamp TIMESTAMP NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    avg_value DECIMAL(15,4),
    min_value DECIMAL(15,4),
    max_value DECIMAL(15,4),
    sum_value DECIMAL(15,4),
    count_value INT,
    
    PRIMARY KEY (hour_timestamp, metric_name)
);

-- Automated aggregation
CREATE OR REPLACE FUNCTION aggregate_hourly_metrics()
RETURNS void AS $$
BEGIN
    INSERT INTO metrics_hourly (
        hour_timestamp, metric_name, avg_value, min_value, 
        max_value, sum_value, count_value
    )
    SELECT 
        date_trunc('hour', timestamp) as hour_timestamp,
        metric_name,
        AVG(value) as avg_value,
        MIN(value) as min_value,
        MAX(value) as max_value,
        SUM(value) as sum_value,
        COUNT(*) as count_value
    FROM metrics_raw
    WHERE timestamp >= (SELECT COALESCE(MAX(hour_timestamp), '1970-01-01') FROM metrics_hourly)
    GROUP BY date_trunc('hour', timestamp), metric_name
    ON CONFLICT (hour_timestamp, metric_name) DO UPDATE SET
        avg_value = EXCLUDED.avg_value,
        min_value = EXCLUDED.min_value,
        max_value = EXCLUDED.max_value,
        sum_value = EXCLUDED.sum_value,
        count_value = EXCLUDED.count_value;
END;
$$ LANGUAGE plpgsql;
```

### Spatial Data with Geohashing
```sql
-- Store locations with multiple precision levels
CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200),
    latitude DECIMAL(10,8) NOT NULL,
    longitude DECIMAL(11,8) NOT NULL,
    geohash_1 CHAR(1), -- ~5000km precision
    geohash_3 CHAR(3), -- ~150km precision  
    geohash_6 CHAR(6), -- ~1km precision
    geohash_9 CHAR(9), -- ~5m precision
    
    -- Indexes for different precision levels
    INDEX idx_geohash_1 (geohash_1),
    INDEX idx_geohash_3 (geohash_3),
    INDEX idx_geohash_6 (geohash_6),
    INDEX idx_geohash_9 (geohash_9)
);

-- Efficient proximity queries at different scales
-- Find nearby locations within ~1km
SELECT * FROM locations 
WHERE geohash_6 = 'drt2zt'
ORDER BY ST_Distance(
    ST_Point(longitude, latitude),
    ST_Point($user_lon, $user_lat)
);
```

## Dynamic Schema Patterns

### Polymorphic Associations
```sql
-- Generic association table
CREATE TABLE attachments (
    id SERIAL PRIMARY KEY,
    attachable_type VARCHAR(50) NOT NULL, -- 'Post', 'Comment', etc.
    attachable_id INT NOT NULL,
    filename VARCHAR(255) NOT NULL,
    file_size INT,
    content_type VARCHAR(100),
    
    INDEX idx_polymorphic (attachable_type, attachable_id)
);

-- Views for type safety
CREATE VIEW post_attachments AS
SELECT a.*, p.title as post_title
FROM attachments a
JOIN posts p ON a.attachable_id = p.id
WHERE a.attachable_type = 'Post';

CREATE VIEW comment_attachments AS  
SELECT a.*, c.content as comment_content
FROM attachments a
JOIN comments c ON a.attachable_id = c.id
WHERE a.attachable_type = 'Comment';
```

### Configuration Tables
```sql
-- Flexible configuration storage
CREATE TABLE configurations (
    id SERIAL PRIMARY KEY,
    context VARCHAR(100) NOT NULL,    -- 'user', 'tenant', 'global'
    context_id INT,                   -- NULL for global
    key VARCHAR(200) NOT NULL,
    value JSONB NOT NULL,
    data_type VARCHAR(20) NOT NULL,   -- 'string', 'number', 'boolean', 'json'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(context, context_id, key)
);

-- Helper function to get typed values
CREATE OR REPLACE FUNCTION get_config(
    p_context VARCHAR(100),
    p_context_id INT,
    p_key VARCHAR(200),
    p_default JSONB DEFAULT NULL
) RETURNS JSONB AS $$
DECLARE
    result JSONB;
BEGIN
    SELECT value INTO result
    FROM configurations
    WHERE context = p_context 
    AND (context_id = p_context_id OR (context_id IS NULL AND p_context_id IS NULL))
    AND key = p_key;
    
    RETURN COALESCE(result, p_default);
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Partitioning Strategies
```sql
-- Time-based partitioning for large datasets
CREATE TABLE events (
    id BIGSERIAL,
    event_type VARCHAR(50) NOT NULL,
    user_id INT NOT NULL,
    event_data JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE events_2024_07 PARTITION OF events
FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE events_2024_08 PARTITION OF events  
FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

-- Automated partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name TEXT, start_date DATE)
RETURNS void AS $$
DECLARE
    partition_name TEXT;
    end_date DATE;
BEGIN
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    end_date := start_date + INTERVAL '1 month';
    
    EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

1. **Use deferrable constraints sparingly** - Only when truly needed for complex operations
2. **Design for maintainability** - Complex patterns should be well-documented  
3. **Test thoroughly** - Advanced patterns can have subtle edge cases
4. **Monitor performance** - Complex constraints and triggers can impact performance
5. **Provide escape hatches** - Allow for manual intervention when needed
6. **Version your schema changes** - Advanced patterns often require careful migration

## Related Topics

- [Constraint Design](constraints.md) - Basic constraint patterns
- [Performance Optimization](../performance/README.md) - Performance implications
- [Migration Strategies](../operations/migrations.md) - Deploying advanced patterns
- [Monitoring](../operations/monitoring.md) - Tracking complex schema performance

## External References

- [Deferrable SQL Constraints](https://begriffs.com/posts/2017-08-27-deferrable-sql-constraints.html)
- [User-Defined Order](https://begriffs.com/posts/2018-03-20-user-defined-order.html)
