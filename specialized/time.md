# Time and Date Patterns in Database Design

This guide covers common patterns for representing time and date logic in database systems, particularly PostgreSQL.

## Core Time Patterns

### 1. Operation Timestamps

Standard timestamp fields for tracking database operations:

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    
    -- Core timestamp fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Optional soft delete
    deleted_at TIMESTAMPTZ
);

-- Trigger to automatically update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2. State Transition Timestamps

Use timestamps instead of boolean flags to track state changes:

```sql
-- Instead of using boolean flags
CREATE TABLE orders_bad (
    id SERIAL PRIMARY KEY,
    is_published BOOLEAN DEFAULT FALSE,
    is_confirmed BOOLEAN DEFAULT FALSE,
    is_shipped BOOLEAN DEFAULT FALSE
);

-- Use timestamps for richer information
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    
    -- State timestamps provide more information
    published_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    -- Constraints to enforce business rules
    CONSTRAINT valid_order_flow CHECK (
        (confirmed_at IS NULL OR published_at IS NOT NULL) AND
        (shipped_at IS NULL OR confirmed_at IS NOT NULL) AND
        (delivered_at IS NULL OR shipped_at IS NOT NULL) AND
        (cancelled_at IS NULL OR (delivered_at IS NULL AND cancelled_at > confirmed_at))
    )
);

-- Query patterns
SELECT * FROM orders WHERE published_at IS NOT NULL;  -- Published orders
SELECT * FROM orders WHERE confirmed_at IS NOT NULL AND shipped_at IS NULL;  -- Confirmed but not shipped
```

### 3. Time Periods with tstzrange

For data that is valid during specific time periods:

```sql
-- Employee assignments with time periods
CREATE TABLE employee_assignments (
    id SERIAL PRIMARY KEY,
    employee_id UUID NOT NULL,
    department_id UUID NOT NULL,
    assignment_period TSTZRANGE NOT NULL,
    
    -- Prevent overlapping assignments for the same employee
    EXCLUDE USING gist (
        employee_id WITH =,
        assignment_period WITH &&
    )
);

-- Insert assignment
INSERT INTO employee_assignments (employee_id, department_id, assignment_period)
VALUES ('emp-123', 'dept-456', tstzrange('2024-01-01', '2024-12-31', '[)'));

-- Query current assignments
SELECT * FROM employee_assignments 
WHERE assignment_period @> NOW();

-- Query assignments at specific time
SELECT * FROM employee_assignments 
WHERE assignment_period @> '2024-06-15'::TIMESTAMPTZ;
```

### 4. Continuous Data (Temporal Tables)

For data that changes over time but is always valid:

```sql
-- Product pricing with temporal validity
CREATE TABLE product_prices (
    id SERIAL PRIMARY KEY,
    product_id UUID NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Temporal validity
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ NOT NULL DEFAULT 'infinity',
    
    -- Prevent gaps and overlaps
    EXCLUDE USING gist (
        product_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    )
);

-- Insert new price (automatically closes previous)
CREATE OR REPLACE FUNCTION insert_product_price(
    p_product_id UUID,
    p_price DECIMAL(10,2),
    p_effective_from TIMESTAMPTZ DEFAULT NOW()
)
RETURNS VOID AS $$
BEGIN
    -- Close current price
    UPDATE product_prices 
    SET valid_to = p_effective_from
    WHERE product_id = p_product_id 
      AND valid_to = 'infinity';
    
    -- Insert new price
    INSERT INTO product_prices (product_id, price, valid_from)
    VALUES (p_product_id, p_price, p_effective_from);
END;
$$ LANGUAGE plpgsql;
```

## Advanced Time Patterns

### Scheduled Events

For events that need to be scheduled for future execution:

```sql
CREATE TABLE scheduled_tasks (
    id SERIAL PRIMARY KEY,
    task_name VARCHAR(100) NOT NULL,
    task_payload JSONB,
    
    -- Scheduling information
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    
    -- Retry logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- Status derived from timestamps
    status VARCHAR(20) GENERATED ALWAYS AS (
        CASE 
            WHEN failed_at IS NOT NULL THEN 'failed'
            WHEN completed_at IS NOT NULL THEN 'completed'
            WHEN started_at IS NOT NULL THEN 'running'
            WHEN scheduled_at > NOW() THEN 'scheduled'
            ELSE 'pending'
        END
    ) STORED
);

-- Index for efficient scheduling queries
CREATE INDEX idx_scheduled_tasks_pending 
ON scheduled_tasks (scheduled_at) 
WHERE started_at IS NULL AND failed_at IS NULL;
```

### Recurring Events

Handle recurring events with proper temporal logic:

```sql
CREATE TABLE recurring_events (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(100) NOT NULL,
    recurrence_pattern VARCHAR(50) NOT NULL, -- 'daily', 'weekly', 'monthly'
    recurrence_interval INTEGER DEFAULT 1,
    
    -- Time boundaries
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    
    -- Occurrence tracking
    last_occurrence TIMESTAMPTZ,
    next_occurrence TIMESTAMPTZ
);

-- Function to calculate next occurrence
CREATE OR REPLACE FUNCTION calculate_next_occurrence(
    event_id INTEGER
)
RETURNS TIMESTAMPTZ AS $$
DECLARE
    event_rec RECORD;
    next_time TIMESTAMPTZ;
BEGIN
    SELECT * INTO event_rec FROM recurring_events WHERE id = event_id;
    
    CASE event_rec.recurrence_pattern
        WHEN 'daily' THEN
            next_time = COALESCE(event_rec.last_occurrence, event_rec.start_date) + 
                       (event_rec.recurrence_interval || ' days')::INTERVAL;
        WHEN 'weekly' THEN
            next_time = COALESCE(event_rec.last_occurrence, event_rec.start_date) + 
                       (event_rec.recurrence_interval || ' weeks')::INTERVAL;
        WHEN 'monthly' THEN
            next_time = COALESCE(event_rec.last_occurrence, event_rec.start_date) + 
                       (event_rec.recurrence_interval || ' months')::INTERVAL;
    END CASE;
    
    RETURN next_time;
END;
$$ LANGUAGE plpgsql;
```

### Time-Based Expiration

Handle expiring data with automatic cleanup:

```sql
CREATE TABLE user_sessions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    
    -- Expiration handling
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Computed expiration status
    is_expired BOOLEAN GENERATED ALWAYS AS (
        expires_at < NOW() OR 
        last_accessed_at < NOW() - INTERVAL '2 hours'
    ) STORED
);

-- Index for efficient cleanup
CREATE INDEX idx_user_sessions_expired 
ON user_sessions (expires_at) 
WHERE NOT is_expired;

-- Cleanup function
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions 
    WHERE is_expired = TRUE;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

## Time Zone Handling

### Store UTC, Display Local

```sql
-- Always store in UTC
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(100) NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,  -- Stored in UTC
    timezone VARCHAR(50) NOT NULL,     -- User's timezone for display
    
    -- Computed local time for convenience
    local_time TIMESTAMPTZ GENERATED ALWAYS AS (
        event_time AT TIME ZONE timezone
    ) STORED
);

-- Query with timezone conversion
SELECT 
    event_name,
    event_time,
    event_time AT TIME ZONE 'America/New_York' AS ny_time,
    event_time AT TIME ZONE 'Asia/Singapore' AS sg_time
FROM events;
```

### Handle Daylight Saving Time

```sql
-- Store timezone-aware data
CREATE TABLE appointments (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    
    -- Store both UTC and local timezone
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(50) NOT NULL,
    
    -- Helper for displaying in original timezone
    local_start_time TIMESTAMPTZ GENERATED ALWAYS AS (
        start_time AT TIME ZONE timezone
    ) STORED
);
```

## Performance Optimization

### Indexing Time-Based Queries

```sql
-- Partial indexes for common queries
CREATE INDEX idx_orders_recent_created 
ON orders (created_at DESC) 
WHERE created_at > NOW() - INTERVAL '30 days';

CREATE INDEX idx_orders_pending_published 
ON orders (published_at) 
WHERE confirmed_at IS NULL AND cancelled_at IS NULL;

-- Range indexes for time period queries
CREATE INDEX idx_assignments_period 
ON employee_assignments USING gist (assignment_period);
```

### Partitioning by Time

```sql
-- Partition table by month
CREATE TABLE events_partitioned (
    id SERIAL,
    event_name VARCHAR(100) NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    data JSONB
) PARTITION BY RANGE (event_time);

-- Create monthly partitions
CREATE TABLE events_2024_01 PARTITION OF events_partitioned
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE events_2024_02 PARTITION OF events_partitioned
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

## Common Patterns and Use Cases

### Audit Trails

```sql
CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(10) NOT NULL, -- 'INSERT', 'UPDATE', 'DELETE'
    old_values JSONB,
    new_values JSONB,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger function for audit logging
CREATE OR REPLACE FUNCTION audit_trigger_function()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (table_name, record_id, action, old_values, changed_by)
        VALUES (TG_TABLE_NAME, OLD.id, TG_OP, to_jsonb(OLD), current_user_id());
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (table_name, record_id, action, old_values, new_values, changed_by)
        VALUES (TG_TABLE_NAME, NEW.id, TG_OP, to_jsonb(OLD), to_jsonb(NEW), current_user_id());
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (table_name, record_id, action, new_values, changed_by)
        VALUES (TG_TABLE_NAME, NEW.id, TG_OP, to_jsonb(NEW), current_user_id());
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
```

### Data Retention Policies

```sql
-- Implement data retention with time-based deletion
CREATE OR REPLACE FUNCTION enforce_data_retention()
RETURNS VOID AS $$
BEGIN
    -- Delete old log entries
    DELETE FROM audit_log 
    WHERE changed_at < NOW() - INTERVAL '7 years';
    
    -- Delete old sessions
    DELETE FROM user_sessions 
    WHERE created_at < NOW() - INTERVAL '30 days';
    
    -- Archive old orders
    INSERT INTO orders_archive 
    SELECT * FROM orders 
    WHERE created_at < NOW() - INTERVAL '2 years';
    
    DELETE FROM orders 
    WHERE created_at < NOW() - INTERVAL '2 years';
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

1. **Always use TIMESTAMPTZ** - Store all timestamps with timezone information
2. **Default to UTC** - Store times in UTC, convert for display
3. **Use appropriate granularity** - Choose precision based on your needs
4. **Index time-based queries** - Create indexes for common date range queries
5. **Handle null values carefully** - Use 'infinity' for open-ended ranges
6. **Consider partitioning** - For large time-series data
7. **Implement proper constraints** - Prevent invalid time ranges
8. **Document time semantics** - Clearly define what each timestamp represents

## Common Pitfalls

- Mixing timezone-aware and timezone-naive timestamps
- Not handling daylight saving time transitions
- Poor indexing on time-based queries
- Allowing invalid time ranges (end before start)
- Not considering clock skew in distributed systems
- Inadequate handling of leap seconds
- Confusing transaction time with valid time
