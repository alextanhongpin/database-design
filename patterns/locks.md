# Database Locking Patterns: Comprehensive Guide

Database locking is crucial for maintaining data consistency in concurrent environments. This guide covers various locking strategies, from optimistic concurrency control to advanced PostgreSQL-specific locking mechanisms.

## Table of Contents
- [Locking Overview](#locking-overview)
- [Optimistic Locking](#optimistic-locking)
- [Pessimistic Locking](#pessimistic-locking)
- [Row-Level Locking](#row-level-locking)
- [Advisory Locks](#advisory-locks)
- [Skip Locked Pattern](#skip-locked-pattern)
- [Serializable Isolation](#serializable-isolation)
- [Real-World Examples](#real-world-examples)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Locking Overview

### Types of Locks

1. **Optimistic Locking** - Assume conflicts are rare, check at commit time
2. **Pessimistic Locking** - Lock resources before accessing them
3. **Shared Locks** - Multiple readers, no writers
4. **Exclusive Locks** - Single reader/writer, blocks all others

### When to Use Each Approach

```sql
-- Decision matrix example
CREATE TABLE locking_strategies (
    scenario TEXT,
    read_frequency TEXT,
    write_frequency TEXT,
    conflict_probability TEXT,
    recommended_strategy TEXT
);

INSERT INTO locking_strategies VALUES
('Web application user profiles', 'High', 'Low', 'Low', 'Optimistic'),
('Banking transactions', 'Medium', 'High', 'Medium', 'Pessimistic'),
('Inventory management', 'High', 'High', 'High', 'Pessimistic + Queuing'),
('Content management', 'High', 'Low', 'Low', 'Optimistic'),
('Job queue processing', 'Low', 'High', 'High', 'Skip Locked');
```

## Optimistic Locking

### Version-Based Optimistic Locking

Best for scenarios with low conflict probability:

```sql
-- Basic versioned table
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    balance DECIMAL(10,2) NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Automatic version increment trigger
CREATE OR REPLACE FUNCTION update_row_version() 
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.version = OLD.version AND NEW.* IS DISTINCT FROM OLD.* THEN
        NEW.version = NEW.version + 1;
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_account_version
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_row_version();

-- Safe update with version check
CREATE OR REPLACE FUNCTION update_account_balance(
    p_account_id INTEGER,
    p_new_balance DECIMAL(10,2),
    p_expected_version INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    rows_affected INTEGER;
BEGIN
    UPDATE accounts 
    SET balance = p_new_balance
    WHERE id = p_account_id AND version = p_expected_version;
    
    GET DIAGNOSTICS rows_affected = ROW_COUNT;
    
    IF rows_affected = 0 THEN
        RAISE EXCEPTION 'Optimistic lock failure: Account % was modified by another transaction', p_account_id;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Usage example with application-level retry
DO $$
DECLARE
    current_account RECORD;
    retry_count INTEGER := 0;
    max_retries INTEGER := 3;
BEGIN
    LOOP
        -- Read current state
        SELECT id, balance, version INTO current_account
        FROM accounts WHERE id = 1;
        
        BEGIN
            -- Attempt update
            PERFORM update_account_balance(
                current_account.id, 
                current_account.balance - 100.00, 
                current_account.version
            );
            
            RAISE NOTICE 'Successfully updated account balance';
            EXIT; -- Success, exit loop
            
        EXCEPTION WHEN OTHERS THEN
            retry_count := retry_count + 1;
            
            IF retry_count >= max_retries THEN
                RAISE EXCEPTION 'Failed to update after % retries: %', max_retries, SQLERRM;
            END IF;
            
            RAISE NOTICE 'Retry % due to: %', retry_count, SQLERRM;
            PERFORM pg_sleep(0.1 * retry_count); -- Exponential backoff
        END;
    END LOOP;
END;
$$;
```

### Timestamp-Based Optimistic Locking

Alternative to version numbers:

```sql
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    last_modified TIMESTAMPTZ DEFAULT NOW(),
    modified_by INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Update with timestamp check
CREATE OR REPLACE FUNCTION update_document_safe(
    p_doc_id INTEGER,
    p_title TEXT,
    p_content TEXT,
    p_expected_timestamp TIMESTAMPTZ,
    p_user_id INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    rows_affected INTEGER;
BEGIN
    UPDATE documents 
    SET 
        title = p_title,
        content = p_content,
        last_modified = NOW(),
        modified_by = p_user_id
    WHERE id = p_doc_id 
      AND last_modified = p_expected_timestamp;
    
    GET DIAGNOSTICS rows_affected = ROW_COUNT;
    
    IF rows_affected = 0 THEN
        RAISE EXCEPTION 'Document was modified by another user. Please refresh and try again.';
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## Pessimistic Locking

### SELECT FOR UPDATE

Explicit row locking for critical operations:

```sql
-- Basic SELECT FOR UPDATE
BEGIN;

-- Lock specific rows
SELECT id, balance FROM accounts 
WHERE user_id = 123 
FOR UPDATE;

-- Perform operations knowing rows are locked
UPDATE accounts SET balance = balance - 100 
WHERE user_id = 123;

COMMIT;
```

### Advanced SELECT FOR UPDATE Patterns

```sql
-- SELECT FOR UPDATE with different lock modes
CREATE OR REPLACE FUNCTION transfer_funds(
    p_from_account INTEGER,
    p_to_account INTEGER,
    p_amount DECIMAL(10,2)
) RETURNS BOOLEAN AS $$
DECLARE
    from_balance DECIMAL(10,2);
    to_balance DECIMAL(10,2);
BEGIN
    -- Lock accounts in consistent order to prevent deadlocks
    IF p_from_account < p_to_account THEN
        SELECT balance INTO from_balance 
        FROM accounts WHERE id = p_from_account FOR UPDATE;
        
        SELECT balance INTO to_balance 
        FROM accounts WHERE id = p_to_account FOR UPDATE;
    ELSE
        SELECT balance INTO to_balance 
        FROM accounts WHERE id = p_to_account FOR UPDATE;
        
        SELECT balance INTO from_balance 
        FROM accounts WHERE id = p_from_account FOR UPDATE;
    END IF;
    
    -- Check sufficient funds
    IF from_balance < p_amount THEN
        RAISE EXCEPTION 'Insufficient funds. Available: %, Required: %', from_balance, p_amount;
    END IF;
    
    -- Perform transfer
    UPDATE accounts SET balance = balance - p_amount WHERE id = p_from_account;
    UPDATE accounts SET balance = balance + p_amount WHERE id = p_to_account;
    
    -- Log transaction
    INSERT INTO transaction_log (from_account, to_account, amount, created_at)
    VALUES (p_from_account, p_to_account, p_amount, NOW());
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### Lock Modes and Waiting

```sql
-- Different lock modes
SELECT * FROM accounts WHERE id = 1 FOR UPDATE; -- Exclusive lock
SELECT * FROM accounts WHERE id = 1 FOR SHARE;  -- Shared lock

-- Non-blocking locks
SELECT * FROM accounts WHERE id = 1 FOR UPDATE NOWAIT;
SELECT * FROM accounts WHERE id = 1 FOR UPDATE SKIP LOCKED;

-- Lock specific columns only (PostgreSQL 9.5+)
SELECT id, balance FROM accounts 
WHERE id = 1 
FOR UPDATE OF accounts;
```

## Row-Level Locking

### Implementing Row-Level Locks

```sql
-- Table with built-in locking mechanism
CREATE TABLE resources (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    is_locked BOOLEAN DEFAULT FALSE,
    locked_by INTEGER,
    locked_at TIMESTAMPTZ,
    lock_expires_at TIMESTAMPTZ,
    data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to acquire lock
CREATE OR REPLACE FUNCTION acquire_resource_lock(
    p_resource_id INTEGER,
    p_user_id INTEGER,
    p_lock_duration INTERVAL DEFAULT '30 minutes'
) RETURNS BOOLEAN AS $$
DECLARE
    lock_acquired BOOLEAN := FALSE;
BEGIN
    UPDATE resources 
    SET 
        is_locked = TRUE,
        locked_by = p_user_id,
        locked_at = NOW(),
        lock_expires_at = NOW() + p_lock_duration
    WHERE id = p_resource_id 
      AND (NOT is_locked OR lock_expires_at < NOW());
    
    GET DIAGNOSTICS lock_acquired = FOUND;
    
    IF NOT lock_acquired THEN
        RAISE EXCEPTION 'Resource is currently locked by another user';
    END IF;
    
    RETURN lock_acquired;
END;
$$ LANGUAGE plpgsql;

-- Function to release lock
CREATE OR REPLACE FUNCTION release_resource_lock(
    p_resource_id INTEGER,
    p_user_id INTEGER
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE resources 
    SET 
        is_locked = FALSE,
        locked_by = NULL,
        locked_at = NULL,
        lock_expires_at = NULL
    WHERE id = p_resource_id 
      AND locked_by = p_user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Cannot release lock - not owned by user or already released';
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Cleanup expired locks
CREATE OR REPLACE FUNCTION cleanup_expired_locks()
RETURNS INTEGER AS $$
DECLARE
    cleaned_count INTEGER;
BEGIN
    UPDATE resources 
    SET 
        is_locked = FALSE,
        locked_by = NULL,
        locked_at = NULL,
        lock_expires_at = NULL
    WHERE is_locked = TRUE 
      AND lock_expires_at < NOW();
    
    GET DIAGNOSTICS cleaned_count = ROW_COUNT;
    
    RETURN cleaned_count;
END;
$$ LANGUAGE plpgsql;
```

## Advisory Locks

### Session-Level Advisory Locks

```sql
-- Acquire session-level advisory lock
SELECT pg_advisory_lock(12345);

-- Try to acquire without blocking
SELECT pg_try_advisory_lock(12345) AS lock_acquired;

-- Release session-level lock
SELECT pg_advisory_unlock(12345);

-- Use case: Database migrations
CREATE OR REPLACE FUNCTION run_migration(migration_name TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    lock_key INTEGER;
    migration_exists BOOLEAN;
BEGIN
    -- Generate consistent lock key from migration name
    lock_key := abs(hashtext(migration_name));
    
    -- Try to acquire lock
    IF NOT pg_try_advisory_lock(lock_key) THEN
        RAISE NOTICE 'Migration % is already running', migration_name;
        RETURN FALSE;
    END IF;
    
    BEGIN
        -- Check if migration already completed
        SELECT EXISTS(
            SELECT 1 FROM migration_history 
            WHERE name = migration_name
        ) INTO migration_exists;
        
        IF migration_exists THEN
            RAISE NOTICE 'Migration % already completed', migration_name;
            RETURN TRUE;
        END IF;
        
        -- Run migration logic here
        RAISE NOTICE 'Running migration %', migration_name;
        
        -- Record migration completion
        INSERT INTO migration_history (name, executed_at)
        VALUES (migration_name, NOW());
        
        RETURN TRUE;
        
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION 'Migration failed: %', SQLERRM;
    END;
    
    -- Lock is automatically released when session ends
    -- But we can release it explicitly
    PERFORM pg_advisory_unlock(lock_key);
END;
$$ LANGUAGE plpgsql;
```

### Transaction-Level Advisory Locks

```sql
-- Transaction-level advisory locks (auto-released on commit/rollback)
BEGIN;
SELECT pg_advisory_xact_lock(54321);

-- Do work that needs coordination across multiple application instances
INSERT INTO job_queue (task_name, created_at) 
VALUES ('process_payments', NOW());

COMMIT; -- Lock automatically released
```

### Distributed Locking with Advisory Locks

```sql
-- Implement distributed mutex
CREATE OR REPLACE FUNCTION distributed_mutex_execute(
    p_lock_name TEXT,
    p_timeout_seconds INTEGER DEFAULT 30
) RETURNS BOOLEAN AS $$
DECLARE
    lock_key INTEGER;
    start_time TIMESTAMPTZ;
BEGIN
    lock_key := abs(hashtext(p_lock_name));
    start_time := NOW();
    
    -- Try to acquire lock with timeout
    LOOP
        IF pg_try_advisory_lock(lock_key) THEN
            RETURN TRUE;
        END IF;
        
        IF EXTRACT(EPOCH FROM (NOW() - start_time)) > p_timeout_seconds THEN
            RETURN FALSE;
        END IF;
        
        PERFORM pg_sleep(0.1); -- Wait 100ms before retry
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Usage example
DO $$
BEGIN
    IF distributed_mutex_execute('payment_processing', 60) THEN
        -- Critical section - only one instance can execute this
        RAISE NOTICE 'Processing payments...';
        
        -- Your critical code here
        
        -- Release lock
        PERFORM pg_advisory_unlock(abs(hashtext('payment_processing')));
    ELSE
        RAISE NOTICE 'Could not acquire lock within timeout';
    END IF;
END;
$$;
```

## Skip Locked Pattern

### Job Queue Implementation

```sql
-- Job queue table
CREATE TABLE job_queue (
    id SERIAL PRIMARY KEY,
    job_type TEXT NOT NULL,
    payload JSONB,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Index for efficient job processing
CREATE INDEX idx_job_queue_processing 
ON job_queue (status, next_retry_at) 
WHERE status IN ('pending', 'failed');

-- Function to claim and process a job
CREATE OR REPLACE FUNCTION claim_next_job()
RETURNS TABLE(
    job_id INTEGER,
    job_type TEXT,
    payload JSONB
) AS $$
DECLARE
    claimed_job RECORD;
BEGIN
    -- Claim next available job atomically
    UPDATE job_queue 
    SET 
        status = 'processing',
        attempts = attempts + 1,
        updated_at = NOW()
    WHERE id = (
        SELECT id FROM job_queue
        WHERE status IN ('pending', 'failed')
          AND next_retry_at <= NOW()
        ORDER BY created_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING id, job_type, payload INTO claimed_job;
    
    IF claimed_job IS NOT NULL THEN
        RETURN QUERY SELECT claimed_job.id, claimed_job.job_type, claimed_job.payload;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to complete a job
CREATE OR REPLACE FUNCTION complete_job(p_job_id INTEGER)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE job_queue 
    SET 
        status = 'completed',
        completed_at = NOW(),
        updated_at = NOW()
    WHERE id = p_job_id;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Function to fail a job with retry logic
CREATE OR REPLACE FUNCTION fail_job(
    p_job_id INTEGER,
    p_error_message TEXT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    job_record RECORD;
BEGIN
    SELECT attempts, max_attempts INTO job_record
    FROM job_queue WHERE id = p_job_id;
    
    IF job_record.attempts >= job_record.max_attempts THEN
        -- Max attempts reached, mark as permanently failed
        UPDATE job_queue 
        SET 
            status = 'failed',
            updated_at = NOW()
        WHERE id = p_job_id;
    ELSE
        -- Schedule retry with exponential backoff
        UPDATE job_queue 
        SET 
            status = 'failed',
            next_retry_at = NOW() + (INTERVAL '1 minute' * POWER(2, attempts)),
            updated_at = NOW()
        WHERE id = p_job_id;
    END IF;
    
    -- Log error if provided
    IF p_error_message IS NOT NULL THEN
        INSERT INTO job_errors (job_id, error_message, created_at)
        VALUES (p_job_id, p_error_message, NOW());
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### High-Throughput Processing

```sql
-- Batch job processing with SKIP LOCKED
CREATE OR REPLACE FUNCTION process_batch_jobs(p_batch_size INTEGER DEFAULT 10)
RETURNS TABLE(processed_jobs INTEGER) AS $$
DECLARE
    job_ids INTEGER[];
    current_job RECORD;
BEGIN
    -- Claim a batch of jobs
    WITH claimed_jobs AS (
        UPDATE job_queue 
        SET 
            status = 'processing',
            attempts = attempts + 1,
            updated_at = NOW()
        WHERE id IN (
            SELECT id FROM job_queue
            WHERE status = 'pending'
              AND next_retry_at <= NOW()
            ORDER BY created_at
            FOR UPDATE SKIP LOCKED
            LIMIT p_batch_size
        )
        RETURNING id, job_type, payload
    )
    SELECT ARRAY_AGG(id) INTO job_ids FROM claimed_jobs;
    
    -- Process each job
    FOR current_job IN 
        SELECT id, job_type, payload 
        FROM job_queue 
        WHERE id = ANY(job_ids)
    LOOP
        BEGIN
            -- Process job based on type
            CASE current_job.job_type
                WHEN 'send_email' THEN
                    PERFORM process_email_job(current_job.payload);
                WHEN 'process_payment' THEN
                    PERFORM process_payment_job(current_job.payload);
                ELSE
                    RAISE EXCEPTION 'Unknown job type: %', current_job.job_type;
            END CASE;
            
            -- Mark job as completed
            PERFORM complete_job(current_job.id);
            
        EXCEPTION WHEN OTHERS THEN
            -- Mark job as failed
            PERFORM fail_job(current_job.id, SQLERRM);
        END;
    END LOOP;
    
    RETURN QUERY SELECT COALESCE(array_length(job_ids, 1), 0);
END;
$$ LANGUAGE plpgsql;
```

## Serializable Isolation

### Preventing Write Skew with Serializable Transactions

```sql
-- Example: Preventing double-booking
CREATE TABLE bookings (
    id SERIAL PRIMARY KEY,
    room_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status TEXT DEFAULT 'confirmed',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function that prevents overlapping bookings
CREATE OR REPLACE FUNCTION create_booking(
    p_room_id INTEGER,
    p_user_id INTEGER,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ
) RETURNS INTEGER AS $$
DECLARE
    new_booking_id INTEGER;
    conflict_count INTEGER;
BEGIN
    -- Use serializable isolation to prevent write skew
    SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    
    -- Check for conflicts
    SELECT COUNT(*) INTO conflict_count
    FROM bookings
    WHERE room_id = p_room_id
      AND status = 'confirmed'
      AND (
          (start_time <= p_start_time AND end_time > p_start_time) OR
          (start_time < p_end_time AND end_time >= p_end_time) OR
          (start_time >= p_start_time AND end_time <= p_end_time)
      );
    
    IF conflict_count > 0 THEN
        RAISE EXCEPTION 'Room is already booked during the requested time period';
    END IF;
    
    -- Create booking
    INSERT INTO bookings (room_id, user_id, start_time, end_time)
    VALUES (p_room_id, p_user_id, p_start_time, p_end_time)
    RETURNING id INTO new_booking_id;
    
    RETURN new_booking_id;
END;
$$ LANGUAGE plpgsql;
```

## Real-World Examples

### 1. E-commerce Inventory Management

```sql
-- Inventory table with optimistic locking
CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    warehouse_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(product_id, warehouse_id)
);

-- Reserve inventory with proper locking
CREATE OR REPLACE FUNCTION reserve_inventory(
    p_product_id INTEGER,
    p_warehouse_id INTEGER,
    p_quantity INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    current_inventory RECORD;
    retry_count INTEGER := 0;
    max_retries INTEGER := 3;
BEGIN
    LOOP
        -- Read current inventory
        SELECT quantity, reserved_quantity, version 
        INTO current_inventory
        FROM inventory 
        WHERE product_id = p_product_id AND warehouse_id = p_warehouse_id;
        
        -- Check availability
        IF current_inventory.quantity < p_quantity THEN
            RAISE EXCEPTION 'Insufficient inventory. Available: %, Requested: %', 
                current_inventory.quantity, p_quantity;
        END IF;
        
        BEGIN
            -- Attempt to reserve with optimistic locking
            UPDATE inventory
            SET 
                reserved_quantity = reserved_quantity + p_quantity,
                version = version + 1,
                updated_at = NOW()
            WHERE product_id = p_product_id 
              AND warehouse_id = p_warehouse_id
              AND version = current_inventory.version;
            
            IF NOT FOUND THEN
                RAISE EXCEPTION 'Inventory was modified by another transaction';
            END IF;
            
            RETURN TRUE;
            
        EXCEPTION WHEN OTHERS THEN
            retry_count := retry_count + 1;
            
            IF retry_count >= max_retries THEN
                RAISE;
            END IF;
            
            PERFORM pg_sleep(0.01 * retry_count);
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

### 2. Bank Account Transfers

```sql
-- Account transfers with proper deadlock prevention
CREATE OR REPLACE FUNCTION bank_transfer(
    p_from_account INTEGER,
    p_to_account INTEGER,
    p_amount DECIMAL(12,2),
    p_reference TEXT DEFAULT NULL
) RETURNS INTEGER AS $$
DECLARE
    from_balance DECIMAL(12,2);
    to_balance DECIMAL(12,2);
    transfer_id INTEGER;
BEGIN
    -- Always lock accounts in the same order to prevent deadlocks
    IF p_from_account < p_to_account THEN
        SELECT balance INTO from_balance 
        FROM accounts WHERE id = p_from_account FOR UPDATE;
        
        SELECT balance INTO to_balance 
        FROM accounts WHERE id = p_to_account FOR UPDATE;
    ELSE
        SELECT balance INTO to_balance 
        FROM accounts WHERE id = p_to_account FOR UPDATE;
        
        SELECT balance INTO from_balance 
        FROM accounts WHERE id = p_from_account FOR UPDATE;
    END IF;
    
    -- Validate sufficient funds
    IF from_balance < p_amount THEN
        RAISE EXCEPTION 'Insufficient funds. Available: %, Required: %', 
            from_balance, p_amount;
    END IF;
    
    -- Create transfer record first
    INSERT INTO transfers (from_account, to_account, amount, reference, status, created_at)
    VALUES (p_from_account, p_to_account, p_amount, p_reference, 'processing', NOW())
    RETURNING id INTO transfer_id;
    
    -- Perform the transfer
    UPDATE accounts SET balance = balance - p_amount WHERE id = p_from_account;
    UPDATE accounts SET balance = balance + p_amount WHERE id = p_to_account;
    
    -- Mark transfer as completed
    UPDATE transfers SET status = 'completed', completed_at = NOW() 
    WHERE id = transfer_id;
    
    RETURN transfer_id;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Lock Monitoring and Analysis

```sql
-- View current locks
SELECT 
    l.locktype,
    l.database,
    l.relation,
    l.page,
    l.tuple,
    l.transactionid,
    l.mode,
    l.granted,
    a.query,
    a.query_start,
    a.application_name,
    a.client_addr
FROM pg_locks l
JOIN pg_stat_activity a ON l.pid = a.pid
WHERE NOT l.granted
ORDER BY l.relation, l.mode;

-- Find blocking queries
WITH blocking_locks AS (
    SELECT 
        blocked_locks.pid AS blocked_pid,
        blocked_activity.usename AS blocked_user,
        blocking_locks.pid AS blocking_pid,
        blocking_activity.usename AS blocking_user,
        blocked_activity.query AS blocked_statement,
        blocking_activity.query AS blocking_statement,
        blocked_activity.application_name AS blocked_application,
        blocking_activity.application_name AS blocking_application
    FROM pg_catalog.pg_locks blocked_locks
    JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
    JOIN pg_catalog.pg_locks blocking_locks ON (
        blocking_locks.locktype = blocked_locks.locktype
        AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
        AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
        AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
        AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
        AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
        AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
        AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
        AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
        AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
        AND blocking_locks.pid != blocked_locks.pid
    )
    JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
    WHERE NOT blocked_locks.granted
)
SELECT * FROM blocking_locks;
```

### Deadlock Detection and Prevention

```sql
-- Enable deadlock logging
-- postgresql.conf: deadlock_timeout = 1s, log_lock_waits = on

-- Function to detect potential deadlock scenarios
CREATE OR REPLACE FUNCTION check_deadlock_risk()
RETURNS TABLE(
    waiting_pid INTEGER,
    waiting_query TEXT,
    blocking_pid INTEGER,
    blocking_query TEXT,
    lock_duration INTERVAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        w.pid,
        w.query,
        b.pid,
        b.query,
        NOW() - b.query_start
    FROM pg_stat_activity w
    JOIN pg_locks wl ON w.pid = wl.pid
    JOIN pg_locks bl ON (
        wl.locktype = bl.locktype
        AND wl.database IS NOT DISTINCT FROM bl.database
        AND wl.relation IS NOT DISTINCT FROM bl.relation
    )
    JOIN pg_stat_activity b ON bl.pid = b.pid
    WHERE wl.granted = FALSE
      AND bl.granted = TRUE
      AND w.pid != b.pid;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Lock Ordering

```sql
-- Always acquire locks in a consistent order
CREATE OR REPLACE FUNCTION transfer_with_proper_ordering(
    account1 INTEGER,
    account2 INTEGER,
    amount DECIMAL
) RETURNS VOID AS $$
BEGIN
    -- Lock accounts in ascending ID order
    IF account1 < account2 THEN
        PERFORM * FROM accounts WHERE id = account1 FOR UPDATE;
        PERFORM * FROM accounts WHERE id = account2 FOR UPDATE;
    ELSE
        PERFORM * FROM accounts WHERE id = account2 FOR UPDATE;
        PERFORM * FROM accounts WHERE id = account1 FOR UPDATE;
    END IF;
    
    -- Perform transfer logic
END;
$$ LANGUAGE plpgsql;
```

### 2. Timeout and Retry Logic

```sql
-- Set lock timeout
SET lock_timeout = '30s';

-- Retry logic with exponential backoff
CREATE OR REPLACE FUNCTION retry_with_backoff(
    operation_sql TEXT,
    max_retries INTEGER DEFAULT 3
) RETURNS BOOLEAN AS $$
DECLARE
    retry_count INTEGER := 0;
    wait_time NUMERIC;
BEGIN
    LOOP
        BEGIN
            EXECUTE operation_sql;
            RETURN TRUE;
            
        EXCEPTION 
            WHEN lock_not_available OR serialization_failure THEN
                retry_count := retry_count + 1;
                
                IF retry_count > max_retries THEN
                    RAISE;
                END IF;
                
                wait_time := 0.1 * POWER(2, retry_count - 1);
                PERFORM pg_sleep(wait_time);
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

### 3. Lock Granularity

```sql
-- Fine-grained locking for better concurrency
CREATE TABLE account_balances (
    account_id INTEGER PRIMARY KEY,
    available_balance DECIMAL(12,2) NOT NULL,
    pending_balance DECIMAL(12,2) NOT NULL DEFAULT 0
);

-- Lock only what you need
CREATE OR REPLACE FUNCTION reserve_funds(
    p_account_id INTEGER,
    p_amount DECIMAL(12,2)
) RETURNS BOOLEAN AS $$
DECLARE
    current_balance DECIMAL(12,2);
BEGIN
    -- Lock only the specific account
    SELECT available_balance INTO current_balance
    FROM account_balances 
    WHERE account_id = p_account_id
    FOR UPDATE;
    
    IF current_balance >= p_amount THEN
        UPDATE account_balances 
        SET 
            available_balance = available_balance - p_amount,
            pending_balance = pending_balance + p_amount
        WHERE account_id = p_account_id;
        
        RETURN TRUE;
    ELSE
        RETURN FALSE;
    END IF;
END;
$$ LANGUAGE plpgsql;
```

This comprehensive guide covers all major locking patterns in PostgreSQL, from basic optimistic and pessimistic locking to advanced techniques like advisory locks and the SKIP LOCKED pattern. Choose the appropriate locking strategy based on your specific use case, considering factors like conflict probability, performance requirements, and consistency needs.

## References

1. [PostgreSQL: Explicit Locking](https://www.postgresql.org/docs/current/explicit-locking.html)
2. [Hibernate Oplocks](https://wiki.postgresql.org/wiki/Hibernate_oplocks)
3. [2ndquadrant: PostgreSQL Anti-pattern: Read-modify-write-cycle](https://www.2ndquadrant.com/en/blog/postgresql-anti-patterns-read-modify-write-cycles/)
4. [StackOverflow: Optimistic concurrency control across tables in postgres](https://stackoverflow.com/questions/37801598/optimistic-concurrency-control-across-tables-in-postgres)
5. [Optimistic-pessimistic locking SQL](https://learning-notes.mistermicheels.com/data/sql/optimistic-pessimistic-locking-sql/)
6. [Particular: Optimizations to scatter-gather sagas](https://particular.net/blog/optimizations-to-scatter-gather-sagas)
7. [Engineering QubeCinema: Unlocking advisory locks](https://engineering.qubecinema.com/2019/08/26/unlocking-advisory-locks.html)
8. [2ndquadrant: What is SELECT SKIP LOCKED for in PostgreSQL 9.5](https://www.2ndquadrant.com/en/blog/what-is-select-skip-locked-for-in-postgresql-9-5/)
9. [Spin Atomic Object: Redis PostgreSQL](https://spin.atomicobject.com/2021/02/04/redis-postgresql/)
