# Date and Time Types: Complete Guide

Proper handling of dates and times is crucial for any application. This guide covers date/time type selection, timezone handling, common operations, and best practices across different database systems.

## Table of Contents
- [Date/Time Type Selection](#datetime-type-selection)
- [Timezone Handling](#timezone-handling)
- [Common Patterns](#common-patterns)
- [Temporal Queries](#temporal-queries)
- [Business Logic Implementation](#business-logic-implementation)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Date/Time Type Selection

### PostgreSQL Date/Time Types

```sql
-- PostgreSQL temporal types
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    
    -- Date only (no time component)
    event_date DATE,                    -- '2024-07-04'
    
    -- Time only (no date component)
    event_time TIME,                    -- '14:30:00'
    event_time_tz TIME WITH TIME ZONE,  -- '14:30:00+02:00'
    
    -- Timestamp without timezone (naive datetime)
    created_at TIMESTAMP,               -- '2024-07-04 14:30:00'
    
    -- Timestamp with timezone (recommended for most use cases)
    updated_at TIMESTAMPTZ,             -- '2024-07-04 14:30:00+02:00'
    
    -- Interval (duration)
    duration INTERVAL,                  -- '2 hours 30 minutes'
    
    CONSTRAINT future_events CHECK (event_date >= CURRENT_DATE)
);

-- Recommended: Use TIMESTAMPTZ for most datetime columns
CREATE TABLE user_activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    activity_type TEXT NOT NULL,
    
    -- Always use TIMESTAMPTZ for audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Use DATE for business dates (birthdays, deadlines)
    due_date DATE,
    birth_date DATE,
    
    -- Use TIME for business times (opening hours)
    opens_at TIME,
    closes_at TIME
);
```

### MySQL Date/Time Types

```sql
-- MySQL temporal types
CREATE TABLE mysql_events (
    id INT AUTO_INCREMENT PRIMARY KEY,
    
    -- Date only
    event_date DATE,                    -- '2024-07-04'
    
    -- Time only  
    event_time TIME,                    -- '14:30:00'
    
    -- Year only (legacy, avoid in new designs)
    event_year YEAR,                    -- 2024
    
    -- Datetime (no timezone info)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Timestamp (UTC, affected by timezone settings)
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Fractional seconds (MySQL 5.6+)
    precise_time DATETIME(6),           -- Microsecond precision
    logged_at TIMESTAMP(3)              -- Millisecond precision
);

-- Best practice: Use DATETIME for application times
CREATE TABLE user_sessions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    
    -- Store in UTC, handle timezone in application
    session_start DATETIME NOT NULL,
    session_end DATETIME,
    last_activity DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP()),
    
    -- Use TIMESTAMP for audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

## Timezone Handling

### PostgreSQL Timezone Best Practices

```sql
-- Set database timezone
SET timezone = 'UTC';

-- Store everything in UTC, convert for display
CREATE TABLE global_events (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    
    -- Store in UTC
    event_time TIMESTAMPTZ NOT NULL,
    
    -- Store user's timezone for display
    user_timezone TEXT NOT NULL DEFAULT 'UTC',
    
    -- Computed local time for queries
    local_event_time TIMESTAMPTZ GENERATED ALWAYS AS (
        event_time AT TIME ZONE 'UTC' AT TIME ZONE user_timezone
    ) STORED
);

-- Timezone conversion examples
SELECT 
    event_time as utc_time,
    event_time AT TIME ZONE 'America/New_York' as eastern_time,
    event_time AT TIME ZONE 'Europe/London' as london_time,
    event_time AT TIME ZONE user_timezone as local_time
FROM global_events;

-- Working with timezone-aware data
INSERT INTO global_events (title, event_time, user_timezone) VALUES
('Conference Start', '2024-07-04 09:00:00-04:00', 'America/New_York'),
('Workshop', '2024-07-04 14:00:00+01:00', 'Europe/London'),
('Webinar', '2024-07-04 20:00:00+09:00', 'Asia/Tokyo');
```

### Timezone Handling Functions

```sql
-- Utility functions for timezone handling
CREATE OR REPLACE FUNCTION convert_to_user_timezone(
    utc_time TIMESTAMPTZ,
    user_tz TEXT
) RETURNS TIMESTAMPTZ AS $$
BEGIN
    RETURN utc_time AT TIME ZONE 'UTC' AT TIME ZONE user_tz;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION convert_from_user_timezone(
    local_time TIMESTAMP,
    user_tz TEXT
) RETURNS TIMESTAMPTZ AS $$
BEGIN
    RETURN (local_time AT TIME ZONE user_tz) AT TIME ZONE 'UTC';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Business hours across timezones
CREATE TABLE business_hours (
    location_id INTEGER PRIMARY KEY,
    location_name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    
    -- Store in local business timezone
    monday_open TIME,
    monday_close TIME,
    tuesday_open TIME,
    tuesday_close TIME,
    -- ... other days
    
    -- Convert to UTC for global queries
    monday_open_utc TIME GENERATED ALWAYS AS (
        (CURRENT_DATE + monday_open) AT TIME ZONE timezone AT TIME ZONE 'UTC'
    ) STORED
);
```

## Common Patterns

### Audit Timestamps

```sql
-- Standard audit pattern
CREATE TABLE auditable_table (
    id SERIAL PRIMARY KEY,
    data TEXT NOT NULL,
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by INTEGER REFERENCES users(id),
    updated_by INTEGER REFERENCES users(id)
);

-- Automatic update trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_updated_at
    BEFORE UPDATE ON auditable_table
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

### Soft Deletion with Timestamps

```sql
-- Soft deletion pattern
CREATE TABLE soft_deletable (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ, -- NULL = active, timestamp = deleted
    
    -- Ensure unique names among active records only
    CONSTRAINT unique_active_name 
    EXCLUDE (name WITH =) WHERE (deleted_at IS NULL)
);

-- Helper functions
CREATE OR REPLACE FUNCTION soft_delete(table_name TEXT, record_id INTEGER)
RETURNS VOID AS $$
BEGIN
    EXECUTE format(
        'UPDATE %I SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL',
        table_name
    ) USING record_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION restore_record(table_name TEXT, record_id INTEGER)
RETURNS VOID AS $$
BEGIN
    EXECUTE format(
        'UPDATE %I SET deleted_at = NULL WHERE id = $1',
        table_name
    ) USING record_id;
END;
$$ LANGUAGE plpgsql;
```

### Versioning with Temporal Data

```sql
-- Temporal versioning pattern
CREATE TABLE document_versions (
    id SERIAL PRIMARY KEY,
    document_id INTEGER NOT NULL,
    version_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    
    -- Temporal validity
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by INTEGER REFERENCES users(id),
    
    CONSTRAINT unique_current_version 
    EXCLUDE (document_id WITH =) WHERE (valid_to IS NULL),
    
    CONSTRAINT valid_temporal_range 
    CHECK (valid_to IS NULL OR valid_to > valid_from)
);

-- Get current version
CREATE OR REPLACE FUNCTION get_current_document_version(doc_id INTEGER)
RETURNS document_versions AS $$
DECLARE
    result document_versions;
BEGIN
    SELECT * INTO result
    FROM document_versions
    WHERE document_id = doc_id AND valid_to IS NULL;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Get version at specific time
CREATE OR REPLACE FUNCTION get_document_version_at(
    doc_id INTEGER,
    at_time TIMESTAMPTZ
) RETURNS document_versions AS $$
DECLARE
    result document_versions;
BEGIN
    SELECT * INTO result
    FROM document_versions
    WHERE document_id = doc_id
      AND valid_from <= at_time
      AND (valid_to IS NULL OR valid_to > at_time)
    ORDER BY valid_from DESC
    LIMIT 1;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

## Temporal Queries

### Common Date/Time Queries

```sql
-- Recent data queries
-- Last 24 hours
SELECT * FROM events 
WHERE created_at >= NOW() - INTERVAL '24 hours';

-- Last week
SELECT * FROM events 
WHERE created_at >= DATE_TRUNC('week', NOW()) - INTERVAL '1 week';

-- This month
SELECT * FROM events 
WHERE DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW());

-- Yesterday
SELECT * FROM events 
WHERE DATE(created_at) = CURRENT_DATE - INTERVAL '1 day';

-- Business days only (Monday-Friday)
SELECT * FROM events 
WHERE EXTRACT(DOW FROM created_at) BETWEEN 1 AND 5;
```

### Age and Duration Calculations

```sql
-- Age calculations
SELECT 
    user_id,
    birth_date,
    
    -- Age in years
    EXTRACT(YEAR FROM AGE(birth_date)) as age_years,
    
    -- Precise age
    AGE(birth_date) as precise_age,
    
    -- Days since birth
    CURRENT_DATE - birth_date as days_lived,
    
    -- Next birthday
    CASE 
        WHEN EXTRACT(DOY FROM birth_date) >= EXTRACT(DOY FROM CURRENT_DATE)
        THEN DATE_TRUNC('year', CURRENT_DATE) + 
             (EXTRACT(DOY FROM birth_date) - 1) * INTERVAL '1 day'
        ELSE DATE_TRUNC('year', CURRENT_DATE) + INTERVAL '1 year' +
             (EXTRACT(DOY FROM birth_date) - 1) * INTERVAL '1 day'
    END as next_birthday
FROM user_profiles;

-- Session duration analysis
SELECT 
    user_id,
    session_start,
    session_end,
    
    -- Duration in different units
    session_end - session_start as duration,
    EXTRACT(EPOCH FROM (session_end - session_start)) as duration_seconds,
    EXTRACT(EPOCH FROM (session_end - session_start)) / 60 as duration_minutes,
    
    -- Categorize session length
    CASE 
        WHEN session_end - session_start < INTERVAL '5 minutes' THEN 'quick'
        WHEN session_end - session_start < INTERVAL '30 minutes' THEN 'short'
        WHEN session_end - session_start < INTERVAL '2 hours' THEN 'medium'
        ELSE 'long'
    END as session_category
FROM user_sessions
WHERE session_end IS NOT NULL;
```

### Time-based Aggregations

```sql
-- Daily statistics
SELECT 
    DATE(created_at) as date,
    COUNT(*) as daily_count,
    COUNT(DISTINCT user_id) as unique_users,
    AVG(EXTRACT(EPOCH FROM duration)) as avg_duration_seconds
FROM user_sessions
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date;

-- Hourly patterns
SELECT 
    EXTRACT(HOUR FROM created_at) as hour,
    EXTRACT(DOW FROM created_at) as day_of_week,
    COUNT(*) as event_count,
    AVG(COUNT(*)) OVER (PARTITION BY EXTRACT(HOUR FROM created_at)) as avg_hourly
FROM events
WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY EXTRACT(HOUR FROM created_at), EXTRACT(DOW FROM created_at)
ORDER BY day_of_week, hour;

-- Monthly trends with year-over-year comparison
SELECT 
    DATE_TRUNC('month', created_at) as month,
    COUNT(*) as current_count,
    LAG(COUNT(*), 12) OVER (ORDER BY DATE_TRUNC('month', created_at)) as previous_year_count,
    COUNT(*) - LAG(COUNT(*), 12) OVER (ORDER BY DATE_TRUNC('month', created_at)) as yoy_change
FROM orders
GROUP BY DATE_TRUNC('month', created_at)
ORDER BY month;
```

## Business Logic Implementation

### Opening Hours Management

```sql
-- Comprehensive opening hours system
CREATE TYPE day_of_week AS ENUM (
    'monday', 'tuesday', 'wednesday', 'thursday', 
    'friday', 'saturday', 'sunday'
);

CREATE TABLE business_locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE opening_hours (
    id SERIAL PRIMARY KEY,
    location_id INTEGER REFERENCES business_locations(id),
    day_of_week day_of_week NOT NULL,
    
    -- Handle multiple shifts per day
    shift_number INTEGER NOT NULL DEFAULT 1,
    
    opens_at TIME NOT NULL,
    closes_at TIME NOT NULL,
    
    -- Handle overnight hours (e.g., 22:00 to 02:00 next day)
    closes_next_day BOOLEAN DEFAULT FALSE,
    
    -- Effective date range for seasonal hours
    effective_from DATE DEFAULT CURRENT_DATE,
    effective_to DATE,
    
    UNIQUE(location_id, day_of_week, shift_number, effective_from),
    
    CONSTRAINT valid_hours CHECK (
        (NOT closes_next_day AND closes_at > opens_at) OR
        (closes_next_day AND closes_at < opens_at)
    )
);

-- Exception dates (holidays, special hours)
CREATE TABLE opening_hour_exceptions (
    id SERIAL PRIMARY KEY,
    location_id INTEGER REFERENCES business_locations(id),
    exception_date DATE NOT NULL,
    
    -- NULL means closed all day
    opens_at TIME,
    closes_at TIME,
    closes_next_day BOOLEAN DEFAULT FALSE,
    
    reason TEXT,
    
    UNIQUE(location_id, exception_date)
);

-- Function to check if location is open
CREATE OR REPLACE FUNCTION is_location_open(
    p_location_id INTEGER,
    p_check_time TIMESTAMPTZ DEFAULT NOW()
) RETURNS BOOLEAN AS $$
DECLARE
    location_tz TEXT;
    local_time TIMESTAMP;
    local_date DATE;
    local_time_only TIME;
    day_name TEXT;
    is_open BOOLEAN := FALSE;
BEGIN
    -- Get location timezone
    SELECT timezone INTO location_tz
    FROM business_locations WHERE id = p_location_id;
    
    IF location_tz IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- Convert to local time
    local_time := p_check_time AT TIME ZONE 'UTC' AT TIME ZONE location_tz;
    local_date := local_time::DATE;
    local_time_only := local_time::TIME;
    
    -- Check for exceptions first
    SELECT CASE
        WHEN opens_at IS NULL THEN FALSE  -- Closed all day
        WHEN NOT closes_next_day THEN 
            local_time_only BETWEEN opens_at AND closes_at
        ELSE 
            local_time_only >= opens_at OR local_time_only <= closes_at
    END INTO is_open
    FROM opening_hour_exceptions
    WHERE location_id = p_location_id AND exception_date = local_date;
    
    IF FOUND THEN
        RETURN is_open;
    END IF;
    
    -- Check regular hours
    day_name := LOWER(TO_CHAR(local_time, 'Day'));
    day_name := TRIM(day_name);
    
    SELECT TRUE INTO is_open
    FROM opening_hours oh
    WHERE oh.location_id = p_location_id
      AND oh.day_of_week = day_name::day_of_week
      AND (oh.effective_from <= local_date)
      AND (oh.effective_to IS NULL OR oh.effective_to >= local_date)
      AND (
          (NOT oh.closes_next_day AND local_time_only BETWEEN oh.opens_at AND oh.closes_at) OR
          (oh.closes_next_day AND (local_time_only >= oh.opens_at OR local_time_only <= oh.closes_at))
      )
    LIMIT 1;
    
    RETURN COALESCE(is_open, FALSE);
END;
$$ LANGUAGE plpgsql;
```

### Scheduling and Recurring Events

```sql
-- Recurring event pattern
CREATE TYPE recurrence_type AS ENUM (
    'none', 'daily', 'weekly', 'monthly', 'yearly'
);

CREATE TABLE recurring_events (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    
    -- Base event details
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    
    -- Recurrence configuration
    recurrence recurrence_type NOT NULL DEFAULT 'none',
    recurrence_interval INTEGER DEFAULT 1, -- Every N units
    recurrence_days INTEGER[], -- For weekly: [1,3,5] = Mon,Wed,Fri
    recurrence_end_date DATE,
    max_occurrences INTEGER,
    
    -- Timezone for recurring calculations
    timezone TEXT NOT NULL DEFAULT 'UTC',
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Generate event instances
CREATE OR REPLACE FUNCTION generate_event_instances(
    event_id INTEGER,
    from_date DATE DEFAULT CURRENT_DATE,
    to_date DATE DEFAULT CURRENT_DATE + INTERVAL '1 year'
) RETURNS TABLE(
    instance_date DATE,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ
) AS $$
DECLARE
    event_record recurring_events;
    current_date DATE;
    instance_count INTEGER := 0;
BEGIN
    SELECT * INTO event_record FROM recurring_events WHERE id = event_id;
    
    IF NOT FOUND THEN
        RETURN;
    END IF;
    
    current_date := from_date;
    
    WHILE current_date <= to_date AND 
          (event_record.recurrence_end_date IS NULL OR current_date <= event_record.recurrence_end_date) AND
          (event_record.max_occurrences IS NULL OR instance_count < event_record.max_occurrences) LOOP
        
        -- Check if this date matches the recurrence pattern
        IF event_record.recurrence = 'daily' OR
           (event_record.recurrence = 'weekly' AND 
            EXTRACT(DOW FROM current_date) = ANY(event_record.recurrence_days)) OR
           (event_record.recurrence = 'monthly' AND 
            EXTRACT(DAY FROM current_date) = EXTRACT(DAY FROM event_record.start_time)) OR
           (event_record.recurrence = 'yearly' AND 
            EXTRACT(MONTH FROM current_date) = EXTRACT(MONTH FROM event_record.start_time) AND
            EXTRACT(DAY FROM current_date) = EXTRACT(DAY FROM event_record.start_time)) THEN
            
            instance_date := current_date;
            start_time := (current_date + EXTRACT(HOUR FROM event_record.start_time) * INTERVAL '1 hour' +
                          EXTRACT(MINUTE FROM event_record.start_time) * INTERVAL '1 minute') 
                          AT TIME ZONE event_record.timezone;
            end_time := (current_date + EXTRACT(HOUR FROM event_record.end_time) * INTERVAL '1 hour' +
                        EXTRACT(MINUTE FROM event_record.end_time) * INTERVAL '1 minute') 
                        AT TIME ZONE event_record.timezone;
            
            instance_count := instance_count + 1;
            RETURN NEXT;
        END IF;
        
        current_date := current_date + INTERVAL '1 day';
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Indexing Temporal Data

```sql
-- Effective indexing strategies for temporal data
CREATE TABLE time_series_data (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Compound index for common query patterns
CREATE INDEX idx_time_series_user_time ON time_series_data (user_id, created_at DESC);
CREATE INDEX idx_time_series_type_time ON time_series_data (event_type, created_at DESC);

-- Partial indexes for recent data
CREATE INDEX idx_time_series_recent ON time_series_data (created_at) 
WHERE created_at >= NOW() - INTERVAL '30 days';

-- Expression indexes for date truncation queries
CREATE INDEX idx_time_series_daily ON time_series_data (DATE_TRUNC('day', created_at));
CREATE INDEX idx_time_series_hourly ON time_series_data (DATE_TRUNC('hour', created_at));

-- BRIN indexes for large time-series tables (PostgreSQL)
CREATE INDEX idx_time_series_brin ON time_series_data USING BRIN (created_at);
```

### Partitioning by Time

```sql
-- PostgreSQL: Partition large tables by time
CREATE TABLE event_logs (
    id BIGSERIAL,
    user_id INTEGER NOT NULL,
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE event_logs_2024_01 PARTITION OF event_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE event_logs_2024_02 PARTITION OF event_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Automated partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition(
    table_name TEXT,
    start_date DATE
) RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    end_date DATE;
BEGIN
    partition_name := table_name || '_' || TO_CHAR(start_date, 'YYYY_MM');
    end_date := start_date + INTERVAL '1 month';
    
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
        partition_name, table_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Always Use Timezone-Aware Types

```sql
-- ✅ Good: Timezone-aware
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    event_start TIMESTAMPTZ NOT NULL,  -- Stores timezone info
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ❌ Bad: Timezone-naive  
CREATE TABLE bad_events (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    event_start TIMESTAMP NOT NULL,    -- No timezone info
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 2. Store in UTC, Display in Local Time

```sql
-- Application pattern: Store UTC, convert for display
CREATE TABLE user_activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    activity_type TEXT NOT NULL,
    
    -- Always store in UTC
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Store user's timezone for display purposes
    user_timezone TEXT NOT NULL
);

-- Query with timezone conversion
SELECT 
    id,
    activity_type,
    occurred_at as utc_time,
    occurred_at AT TIME ZONE user_timezone as local_time
FROM user_activities
WHERE user_id = 123;
```

### 3. Use Appropriate Precision

```sql
-- Choose precision based on requirements
CREATE TABLE precise_measurements (
    id SERIAL PRIMARY KEY,
    
    -- Microsecond precision for high-frequency data
    measurement_time TIMESTAMPTZ(6) NOT NULL,
    
    -- Second precision for most business applications
    created_at TIMESTAMPTZ(0) NOT NULL DEFAULT NOW(),
    
    -- Date only for business dates
    effective_date DATE NOT NULL
);
```

### 4. Handle Edge Cases

```sql
-- Account for leap years, month boundaries, DST changes
CREATE OR REPLACE FUNCTION safe_add_months(
    base_date DATE,
    months_to_add INTEGER
) RETURNS DATE AS $$
DECLARE
    result_date DATE;
    original_day INTEGER;
BEGIN
    original_day := EXTRACT(DAY FROM base_date);
    
    -- Add months
    result_date := base_date + (months_to_add || ' months')::INTERVAL;
    
    -- Handle month-end dates (e.g., Jan 31 + 1 month should be Feb 28/29)
    IF EXTRACT(DAY FROM result_date) != original_day THEN
        -- Go to last day of target month
        result_date := DATE_TRUNC('month', result_date) + INTERVAL '1 month' - INTERVAL '1 day';
    END IF;
    
    RETURN result_date;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Test edge cases
SELECT 
    safe_add_months('2024-01-31'::DATE, 1) as feb_result, -- 2024-02-29
    safe_add_months('2024-01-31'::DATE, 2) as mar_result, -- 2024-03-31
    safe_add_months('2024-01-30'::DATE, 1) as feb_result2; -- 2024-02-29
```

### 5. Validation and Constraints

```sql
-- Add meaningful constraints
CREATE TABLE scheduled_events (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    
    -- Ensure logical time ordering
    CONSTRAINT valid_time_range CHECK (end_time > start_time),
    
    -- Ensure reasonable event duration (not longer than 24 hours)
    CONSTRAINT reasonable_duration CHECK (
        end_time - start_time <= INTERVAL '24 hours'
    ),
    
    -- Ensure events are not too far in the past or future
    CONSTRAINT reasonable_timeframe CHECK (
        start_time >= CURRENT_DATE - INTERVAL '10 years' AND
        start_time <= CURRENT_DATE + INTERVAL '10 years'
    )
);
```

## Conclusion

Effective date/time handling requires:

1. **Type Selection**: Use TIMESTAMPTZ for most datetime fields, DATE for business dates
2. **Timezone Strategy**: Store in UTC, convert for display
3. **Indexing**: Create appropriate indexes for temporal queries
4. **Validation**: Add constraints to ensure data integrity
5. **Edge Cases**: Handle leap years, DST, month boundaries
6. **Performance**: Use partitioning for large time-series data
7. **Business Logic**: Implement timezone-aware business rules

The key is consistency in your approach and clear understanding of how your application handles time across different contexts and user locations.
- what if it has two opening hours on the same day? create the same entry for the same weekday, with different time
- how to set if the store is closed on a particular date? set the closed dates
- what if the store is not open on a day? don't create the entry

https://stackoverflow.com/questions/19545597/way-to-store-various-shop-opening-times-in-a-database
http://www.remy-mellet.com/blog/288-storing-opening-and-closing-times-in-database-for-stores/
https://stackoverflow.com/questions/4464898/best-way-to-store-working-hours-and-query-it-efficiently

## Calculate Age

```
SELECT TIMESTAMPDIFF(YEAR, '1970-02-01', CURDATE()) AS age
```

## Group by age bucket

```sql
SELECT
    SUM(IF(age < 20,1,0)) as 'Under 20',
    SUM(IF(age BETWEEN 20 and 29,1,0)) as '20 - 29',
    SUM(IF(age BETWEEN 30 and 39,1,0)) as '30 - 39',
    SUM(IF(age BETWEEN 40 and 49,1,0)) as '40 - 49',
...etc.
FROM inquiries;
```


## Golang test time
Go time.Time has nanosecond resolution, MySQL datetime has second resolution (use datetime(6) for microseconds). Go has a timezone, MySQL doesn't.
```
  Expected: '2019-04-04 11:45:41.170518 +0800 +08 m=+0.223437462'
  Actual:   '2019-04-04 03:45:41 +0000 UTC'
```

To make the test work, round the seconds and convert to UTC:
```
time.Now().Round(time.Second).UTC())
```

## SQL Date

Hoping to get all records within a month...
```sql
BETWEEN 2019-01-01 AND 2019-01-31
```

But this will only take records less than `2019-01-31 00:00:00`. The below query is correct:
```sql
BETWEEN 2019-01-01 AND 2019-02-01
```

## SQL Timezone

Common mistake is to query the start to end of the date in UTC, which is different when comparing against local timezone.
```sql
SELECT * FROM mysql.time_zone;
SELECT * FROM mysql.time_zone_name;
select current_timestamp;
-- 2019-04-09 02:08:18

select CONVERT_TZ(current_timestamp, 'GMT', 'Singapore');
-- 2019-04-09 10:08:24


-- To query the difference
SELECT
id, created_at, 
DATE(created_at) AS date_utc, 
DATE(convert_tz(created_at, 'GMT', 'Singapore')) AS date_local
FROM employee_activity 
WHERE DATE(convert_tz(created_at, 'GMT', 'Singapore')) != DATE(created_at);
```

To find the records on a specific date (local timezone!):

```sql
-- This query is incorrect, because it will select those dates that are based on UTC.
SELECT
id, created_at, 
DATE(created_at) AS date_utc, 
DATE(convert_tz(created_at, 'GMT', 'Singapore')) AS date_local
FROM employee_activity 
WHERE DATE(created_at) = '2019-03-04';

-- This query is correct, because the dates are first converted into local timezone before queried.
SELECT
id, created_at, 
DATE(created_at) AS date_utc, 
DATE(convert_tz(created_at, 'GMT', 'Singapore')) AS date_local
FROM employee_activity 
WHERE DATE(CONVERT_TZ(created_at, 'GMT', 'Singapore')) = '2019-03-04';
```


## Difference in days, hours ...
```
mysql> SELECT TIMESTAMPDIFF(MONTH,'2003-02-01','2003-05-01');
        -> 3
mysql> SELECT TIMESTAMPDIFF(YEAR,'2002-05-01','2001-01-01');
        -> -1
mysql> SELECT TIMESTAMPDIFF(MINUTE,'2003-02-01','2003-05-01 12:05:55');
        -> 128885
```

For difference in days:

```sql
datediff(current_timestamp, created_at)
```

To check how many days have elapsed (10 is the number of days elapsed):

```sql
datediff(current_timestamp, created_at) > 10
```


## Default/Null Date

There are some disadvantages of using NULL date, eg. it cannot be indexed (you might need it later), and marshalling them can be a pain when using a strongly typed language (null types needs to be type asserted). 

There are some cases where the NULL values can be [optimized](https://dev.mysql.com/doc/refman/8.0/en/is-null-optimization.html).

For dates, it's best to use a default date range rather than `null`, with the only exception being the `deleted_at` date (since it is easier to check if `deleted_at IS NULL` rather than `deleted_at = 9999-12-31').

TL;DR;

- valid_from: `1000-01-01`
- valid_till: `9999-12-31`

## DATE vs DATETIME

For validity period that has a period ranging within days/weeks/months/year, using `DATE` will be sufficient. 

For actions (approvals, update, creation, logging, audit), use `DATETIME` for better accuracy.


## Date Elapsed

```mysql
-- Get the last day of the month.
select last_day(current_date) as last_day;

-- Get the first day of the month.
select DATE_ADD(
	DATE_ADD(LAST_DAY(current_date),INTERVAL 1 DAY),
	INTERVAL - 1 MONTH) AS first_day;

-- Get the max date (if registered on the same month) or the start of the month
select GREATEST('2019-03-12', DATE_ADD(
	DATE_ADD(LAST_DAY(current_date),INTERVAL 1 DAY),
	INTERVAL - 1 MONTH));
	
-- Find the difference in date between the current date and the last day.

-- Get the max date (if registered on the same month) or the start of the month.
-- The order matters - the end of the month must be first.
select datediff(
	last_day(current_date), 
	-- Compare the subscription date vs the start of the month, the greater one takes priority
	GREATEST('2019-04-12', DATE_ADD(
	DATE_ADD(LAST_DAY(current_date),INTERVAL 1 DAY),
	INTERVAL - 1 MONTH))
);

-- Latest date - now.
select datediff('2019-05-31', current_date);
```

## Timezone difference

Rather than setting the timezone information for the user, set the timezone information on the products/events instead. So if the user purchases the product with the said timezone, it is much easier to process the difference in the timezone. With that said, this means that for each product, there is a need to create different product with different timezone, and there's a logic required to show the different products by different countries too, possibly by the user location or ip information.

Why does this matter? Because if there's a promotion in Malaysia (GMT+8), then if the sale is supposed to end at 12:00 am GMT+8, if the server time is set to UTC instead, the closing time would have been different (it would end earlier) and this could cause a lot of miscommunication.

That said, it is best to store the dates as UTC in the database. But the timezone information should be stored somewhere else too so that the dates can be computed correctly.


## Date Range


When calculating the date difference Use `curdate()/current_date()`, not `now()` since `now()` includes the time.

```sql
# Get policy expiring in 7 days, UTC time
policy.end_date = DATE_ADD(CURDATE(), INTERVAL 7 DAY);

# Get policy expiring in 30 days, UTC time
policy.end_date = DATE_ADD(CURDATE(), INTERVAL 30 DAY);
```


## Get Unix Timestamp (Postgres)

```sql
select extract(epoch from created_at) from your_table;
```

## Postgres End of month 

```sql
  SELECT TO_CHAR(
    DATE_TRUNC('month', CURRENT_DATE)  
      + INTERVAL '1 month'            
      - INTERVAL '1 day',              
    'YYYY-MM-DD HH-MM-SS'              
  ) AS end_of_month
```

## Postgres Start of month

```sql
  SELECT date_trunc('month', current_date) AS start_of_month
```


## Pretty print dates

```sql
SELECT to_char(now(), 'YYYY-MM-DD HH24:MI:SS');

-- Saturday , 25 July      2020 04:00 AM
SELECT
to_char(lower(appointment_at), 'Day, DD Month YYYY HH12:MI AM'),
to_char(upper(appointment_at), 'Day, DD Month YYYY HH12:MI AM')
FROM party_appointment;
```

## Interval

```sql
SELECT INTERVAL '1 day';
SELECT '1 day'::INTERVAL;
-- How many seconds are there in one day?
SELECT EXTRACT(epoch FROM '1 day'::interval);
SELECT EXTRACT(epoch FROM '1 hour'::interval);
```

## Timezone with Postgres

```sql
  SELECT   
  lower(appointment_at)::timestamptz,
  upper(appointment_at)::timestamptz,
  lower(appointment_at) AS appointment_start_date,
  upper(appointment_at) AS appointment_end_date,
  lower(appointment_at) AT TIME ZONE 'Singapore' AS appointment_start_date,
  upper(appointment_at) AT TIME ZONE 'Singapore' AS appointment_end_date
  FROM party_appointment
```

## Postgres round to nearest minute/hour

Rounding the time to nearest hour and setting the column to unique ensure there's only unique value per hour:
```sql

-- Round up to nearest minute.
SELECT date_trunc('minute', now());

-- Round up to nearest hour.
SELECT date_trunc('hour', now());

CREATE TABLE test (
	name text,
	effective_date timestamptz NOT NULL DEFAULT date_trunc('minute', now()),
	UNIQUE (effective_date)
);
```

## Postgres timestamp range default with timezone

```sql
validity tstzrange NOT NULL DEFAULT tstzrange(now(),null),
```
