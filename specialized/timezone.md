# Timezone Handling in PostgreSQL

Proper timezone handling is crucial for global applications. This guide covers PostgreSQL's timezone features and best practices for managing temporal data across different time zones.

## PostgreSQL Timezone Types

### TIMESTAMP vs TIMESTAMPTZ

```sql
-- TIMESTAMP (without timezone) - stores local time
CREATE TABLE events_local (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(100),
    event_time TIMESTAMP  -- No timezone information
);

-- TIMESTAMPTZ (with timezone) - stores UTC + timezone info
CREATE TABLE events_global (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(100),
    event_time TIMESTAMPTZ,  -- Timezone aware
    user_timezone VARCHAR(50) -- Store user's timezone for display
);
```

**Best Practice**: Always use `TIMESTAMPTZ` for new applications.

## Timezone Conversion

### Basic Conversion Syntax

```sql
-- Convert timestamp to specific timezone
SELECT '2024-04-28 09:27:58.597317'::TIMESTAMPTZ AT TIME ZONE 'Asia/Singapore';
SELECT '2024-04-28 09:27:58.597317'::TIMESTAMPTZ AT TIME ZONE 'sgt';

-- Convert between timezones
SELECT 
    NOW() as utc_time,
    NOW() AT TIME ZONE 'America/New_York' as ny_time,
    NOW() AT TIME ZONE 'Asia/Singapore' as sg_time,
    NOW() AT TIME ZONE 'Europe/London' as london_time;
```

### Timezone Name Formats

PostgreSQL supports multiple timezone name formats:

```sql
-- POSIX-style
SELECT NOW() AT TIME ZONE 'EST5EDT';
SELECT NOW() AT TIME ZONE 'PST8PDT';

-- IANA timezone names (preferred)
SELECT NOW() AT TIME ZONE 'America/New_York';
SELECT NOW() AT TIME ZONE 'Europe/London';
SELECT NOW() AT TIME ZONE 'Asia/Singapore';

-- Common abbreviations
SELECT NOW() AT TIME ZONE 'UTC';
SELECT NOW() AT TIME ZONE 'SGT';
SELECT NOW() AT TIME ZONE 'EDT';
```

## Session and Database Settings

### Current Timezone Settings

```sql
-- Check current timezone setting
SELECT current_setting('timezone');
SHOW timezone;

-- Set session timezone
SET timezone TO 'Asia/Singapore';
SET timezone TO 'UTC';
SET timezone TO 'America/New_York';

-- Reset to default
RESET timezone;
```

### Database-Level Settings

```sql
-- Set default timezone for database
ALTER DATABASE mydb SET timezone TO 'UTC';

-- Set timezone for specific user
ALTER USER myuser SET timezone TO 'Asia/Singapore';
```

## Migration Patterns

### Converting TIMESTAMP to TIMESTAMPTZ

Safe migration from timezone-naive to timezone-aware:

```sql
-- Create test table
CREATE TABLE migration_test (
    id SERIAL PRIMARY KEY,
    t1 TIMESTAMP WITHOUT TIME ZONE,
    t2 TIMESTAMPTZ
);

-- Insert test data
INSERT INTO migration_test (t1) VALUES ('2024-04-28 09:27:58.597317');

-- Copy data before migration
UPDATE migration_test SET t2 = t1;

-- Check data before migration
SELECT * FROM migration_test;

-- Safe migration: add timezone information
ALTER TABLE migration_test ALTER COLUMN t1 TYPE TIMESTAMPTZ;

-- Verify migration
SELECT * FROM migration_test;
```

### Handling Ambiguous Times

```sql
-- Handle daylight saving time transitions
SELECT 
    t1 AT TIME ZONE 'Asia/Singapore' AT TIME ZONE 'UTC' as adjusted_utc,
    t2 AT TIME ZONE 'SGT' AT TIME ZONE 'UTC' as sgt_to_utc,
    t2 AT TIME ZONE 'SGT' as local_sgt,
    t2 AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Singapore' as utc_to_local,
    t1 as original_t1,
    t2 as original_t2
FROM migration_test;
```

## Application Patterns

### Storing User Timezone Preferences

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User events with timezone context
CREATE TABLE user_events (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Store user's timezone for display purposes
    user_timezone VARCHAR(50) NOT NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Function to get user events in their timezone
CREATE OR REPLACE FUNCTION get_user_events_local(p_user_id UUID)
RETURNS TABLE(
    event_type VARCHAR(50),
    event_time_utc TIMESTAMPTZ,
    event_time_local TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ue.event_type,
        ue.event_time,
        ue.event_time AT TIME ZONE ue.user_timezone
    FROM user_events ue
    WHERE ue.user_id = p_user_id
    ORDER BY ue.event_time DESC;
END;
$$ LANGUAGE plpgsql;
```

### Recurring Events with Timezone

```sql
CREATE TABLE recurring_events (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    
    -- Store in user's timezone
    scheduled_time TIME NOT NULL,
    timezone VARCHAR(50) NOT NULL,
    
    -- Recurring pattern
    recurrence_pattern VARCHAR(20) NOT NULL, -- 'daily', 'weekly', 'monthly'
    
    -- Next occurrence in UTC for efficient querying
    next_occurrence_utc TIMESTAMPTZ NOT NULL,
    
    -- User context
    user_id UUID NOT NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Function to calculate next occurrence
CREATE OR REPLACE FUNCTION calculate_next_occurrence(
    p_event_id INTEGER
)
RETURNS TIMESTAMPTZ AS $$
DECLARE
    event_rec RECORD;
    next_local TIMESTAMPTZ;
    next_utc TIMESTAMPTZ;
BEGIN
    SELECT * INTO event_rec FROM recurring_events WHERE id = p_event_id;
    
    -- Calculate next occurrence in user's timezone
    CASE event_rec.recurrence_pattern
        WHEN 'daily' THEN
            next_local = (CURRENT_DATE + INTERVAL '1 day' + event_rec.scheduled_time) AT TIME ZONE event_rec.timezone;
        WHEN 'weekly' THEN
            next_local = (CURRENT_DATE + INTERVAL '7 days' + event_rec.scheduled_time) AT TIME ZONE event_rec.timezone;
        WHEN 'monthly' THEN
            next_local = (CURRENT_DATE + INTERVAL '1 month' + event_rec.scheduled_time) AT TIME ZONE event_rec.timezone;
    END CASE;
    
    -- Convert to UTC
    next_utc = next_local AT TIME ZONE 'UTC';
    
    RETURN next_utc;
END;
$$ LANGUAGE plpgsql;
```

## Date Range Queries with Timezone

### Proper Date Range Handling

```sql
-- When querying between dates, always consider timezone
-- BAD: Naive date comparison
SELECT * FROM orders 
WHERE created_at BETWEEN '2024-01-01' AND '2024-01-31';

-- GOOD: Explicit timezone handling
SELECT * FROM orders 
WHERE created_at >= '2024-01-01 00:00:00 Asia/Singapore'::TIMESTAMPTZ
  AND created_at < '2024-02-01 00:00:00 Asia/Singapore'::TIMESTAMPTZ;

-- Even better: Use date functions
SELECT * FROM orders 
WHERE created_at AT TIME ZONE 'Asia/Singapore' >= '2024-01-01'::DATE
  AND created_at AT TIME ZONE 'Asia/Singapore' < '2024-02-01'::DATE;
```

### Business Hours Queries

```sql
-- Find orders during business hours in user's timezone
SELECT 
    o.id,
    o.created_at,
    o.created_at AT TIME ZONE u.timezone as local_time,
    EXTRACT(HOUR FROM o.created_at AT TIME ZONE u.timezone) as local_hour
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE EXTRACT(HOUR FROM o.created_at AT TIME ZONE u.timezone) BETWEEN 9 AND 17
  AND EXTRACT(DOW FROM o.created_at AT TIME ZONE u.timezone) BETWEEN 1 AND 5;
```

## Advanced Timezone Handling

### Daylight Saving Time Considerations

```sql
-- Handle DST transitions
CREATE OR REPLACE FUNCTION is_dst_transition(
    p_timestamp TIMESTAMPTZ,
    p_timezone VARCHAR(50)
)
RETURNS BOOLEAN AS $$
DECLARE
    local_time TIMESTAMPTZ;
    utc_offset_before INTERVAL;
    utc_offset_after INTERVAL;
BEGIN
    -- Get UTC offset before and after the timestamp
    local_time := p_timestamp AT TIME ZONE p_timezone;
    
    -- Check if there's a DST transition
    SELECT 
        (p_timestamp - INTERVAL '1 day') AT TIME ZONE p_timezone - (p_timestamp - INTERVAL '1 day'),
        (p_timestamp + INTERVAL '1 day') AT TIME ZONE p_timezone - (p_timestamp + INTERVAL '1 day')
    INTO utc_offset_before, utc_offset_after;
    
    RETURN utc_offset_before != utc_offset_after;
END;
$$ LANGUAGE plpgsql;
```

### Multiple Timezone Support

```sql
-- Store events with multiple timezone representations
CREATE TABLE global_events (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    
    -- Store in UTC
    event_time_utc TIMESTAMPTZ NOT NULL,
    
    -- Cache common timezone representations
    event_time_ny TIMESTAMPTZ GENERATED ALWAYS AS (
        event_time_utc AT TIME ZONE 'America/New_York'
    ) STORED,
    
    event_time_london TIMESTAMPTZ GENERATED ALWAYS AS (
        event_time_utc AT TIME ZONE 'Europe/London'
    ) STORED,
    
    event_time_singapore TIMESTAMPTZ GENERATED ALWAYS AS (
        event_time_utc AT TIME ZONE 'Asia/Singapore'
    ) STORED,
    
    -- Original timezone for context
    original_timezone VARCHAR(50) NOT NULL
);

-- Index for efficient timezone queries
CREATE INDEX idx_global_events_ny ON global_events(event_time_ny);
CREATE INDEX idx_global_events_london ON global_events(event_time_london);
CREATE INDEX idx_global_events_singapore ON global_events(event_time_singapore);
```

## Performance Optimization

### Efficient Timezone Queries

```sql
-- Pre-compute timezone conversions for better performance
CREATE MATERIALIZED VIEW daily_stats_by_timezone AS
SELECT 
    DATE(created_at AT TIME ZONE 'UTC') as date_utc,
    DATE(created_at AT TIME ZONE 'America/New_York') as date_ny,
    DATE(created_at AT TIME ZONE 'Europe/London') as date_london,
    DATE(created_at AT TIME ZONE 'Asia/Singapore') as date_singapore,
    COUNT(*) as order_count,
    SUM(total_amount) as total_amount
FROM orders
GROUP BY 
    DATE(created_at AT TIME ZONE 'UTC'),
    DATE(created_at AT TIME ZONE 'America/New_York'),
    DATE(created_at AT TIME ZONE 'Europe/London'),
    DATE(created_at AT TIME ZONE 'Asia/Singapore');

-- Refresh periodically
REFRESH MATERIALIZED VIEW daily_stats_by_timezone;
```

### Partitioning by Timezone

```sql
-- Partition table by timezone-adjusted date
CREATE TABLE orders_partitioned (
    id SERIAL,
    user_id UUID NOT NULL,
    total_amount DECIMAL(10,2),
    created_at TIMESTAMPTZ NOT NULL,
    user_timezone VARCHAR(50) NOT NULL
) PARTITION BY RANGE (DATE(created_at AT TIME ZONE 'UTC'));

-- Create partitions
CREATE TABLE orders_2024_01 PARTITION OF orders_partitioned
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE orders_2024_02 PARTITION OF orders_partitioned
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

## Common Queries and Examples

### Business Reporting

```sql
-- Daily sales report by timezone
SELECT 
    DATE(created_at AT TIME ZONE 'Asia/Singapore') as business_date,
    COUNT(*) as order_count,
    SUM(total_amount) as total_sales
FROM orders
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at AT TIME ZONE 'Asia/Singapore')
ORDER BY business_date DESC;

-- Peak hours analysis
SELECT 
    EXTRACT(HOUR FROM created_at AT TIME ZONE 'Asia/Singapore') as hour,
    COUNT(*) as order_count
FROM orders
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY EXTRACT(HOUR FROM created_at AT TIME ZONE 'Asia/Singapore')
ORDER BY hour;
```

### User Activity Patterns

```sql
-- User login patterns by their timezone
SELECT 
    u.timezone,
    EXTRACT(HOUR FROM l.login_time AT TIME ZONE u.timezone) as local_hour,
    COUNT(*) as login_count
FROM user_logins l
JOIN users u ON l.user_id = u.id
WHERE l.login_time >= NOW() - INTERVAL '30 days'
GROUP BY u.timezone, EXTRACT(HOUR FROM l.login_time AT TIME ZONE u.timezone)
ORDER BY u.timezone, local_hour;
```

## Best Practices

1. **Always use TIMESTAMPTZ** - Store all timestamps with timezone information
2. **Store in UTC** - Keep data in UTC, convert for display
3. **Store user timezones** - Keep user timezone preferences for display
4. **Be explicit with conversions** - Always specify timezone in queries
5. **Handle DST transitions** - Consider daylight saving time in calculations
6. **Use IANA timezone names** - Prefer 'America/New_York' over 'EST'
7. **Test across timezones** - Verify behavior during DST transitions
8. **Index timezone-converted columns** - For frequently queried timezone-specific data

## Common Pitfalls

1. **Mixing timezone-aware and naive timestamps**
2. **Not considering daylight saving time**
3. **Using abbreviations instead of full timezone names**
4. **Forgetting to convert for date range queries**
5. **Not storing user timezone preferences**
6. **Assuming all users are in the same timezone**
7. **Not handling DST transitions properly**
8. **Poor indexing on timezone-converted columns**

## Testing Timezone Code

```sql
-- Test DST transitions
SELECT 
    generate_series(
        '2024-03-10 00:00:00 America/New_York'::TIMESTAMPTZ,
        '2024-03-10 05:00:00 America/New_York'::TIMESTAMPTZ,
        INTERVAL '1 hour'
    ) as timestamp_series;

-- Test timezone conversions
SELECT 
    NOW() as utc_now,
    NOW() AT TIME ZONE 'America/New_York' as ny_now,
    NOW() AT TIME ZONE 'Europe/London' as london_now,
    NOW() AT TIME ZONE 'Asia/Singapore' as singapore_now;
```

Remember: When querying date ranges or filtering by time, always consider the user's timezone context to ensure accurate results.
