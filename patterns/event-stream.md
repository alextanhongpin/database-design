# Event Stream Database Design

## Table of Contents
- [Overview](#overview)
- [Event Stream Fundamentals](#event-stream-fundamentals)
- [Event Sourcing Patterns](#event-sourcing-patterns)
- [Stream Processing](#stream-processing)
- [Event Store Implementation](#event-store-implementation)
- [CQRS Integration](#cqrs-integration)
- [Real-World Examples](#real-world-examples)
- [Advanced Patterns](#advanced-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Overview

Event streaming is a pattern for storing data as events in an append-only log. Unlike state-oriented databases that only keep the latest version of entity state, event streams preserve the complete history of all changes, enabling powerful analytics, audit trails, and system recovery capabilities.

### Key Concepts
- **Events**: Immutable facts about what happened
- **Streams**: Ordered sequences of related events
- **Event Store**: Database optimized for event storage and retrieval
- **Projections**: Read models built from event streams
- **Snapshots**: Optimized state reconstructions

### When to Use Event Streams
- **Audit Requirements**: Need complete change history
- **Complex Business Logic**: Multiple systems need to react to changes
- **Temporal Queries**: "What was the state at time X?"
- **Event-Driven Architecture**: Microservices communication
- **Analytics**: Historical data analysis and ML

## Event Stream Fundamentals

### Basic Event Structure

```sql
-- Core event stream table
CREATE TABLE event_streams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Stream identification
    stream_id VARCHAR(255) NOT NULL, -- Logical grouping of events
    stream_type VARCHAR(100) NOT NULL, -- user, order, product, etc.
    
    -- Event details
    event_type VARCHAR(100) NOT NULL,
    event_version INTEGER DEFAULT 1,
    
    -- Event data
    event_data JSONB NOT NULL,
    event_metadata JSONB DEFAULT '{}',
    
    -- Ordering and versioning
    sequence_number BIGINT GENERATED ALWAYS AS IDENTITY,
    stream_version INTEGER NOT NULL,
    
    -- Timing
    occurred_at TIMESTAMP NOT NULL DEFAULT NOW(),
    recorded_at TIMESTAMP DEFAULT NOW(),
    
    -- Causality
    correlation_id UUID, -- Groups related events across streams
    causation_id UUID,   -- The event that caused this event
    
    -- Source information
    source_system VARCHAR(100),
    source_user UUID,
    
    -- Constraints
    UNIQUE (stream_id, stream_version),
    
    -- Indexes
    INDEX idx_stream_events (stream_id, stream_version),
    INDEX idx_event_type_time (event_type, occurred_at),
    INDEX idx_sequence (sequence_number),
    INDEX idx_correlation (correlation_id) WHERE correlation_id IS NOT NULL
);

-- Event types registry for validation
CREATE TABLE event_types (
    name VARCHAR(100) PRIMARY KEY,
    description TEXT,
    schema_version INTEGER DEFAULT 1,
    event_schema JSONB, -- JSON Schema for validation
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    deprecated_at TIMESTAMP
);

-- Stream metadata
CREATE TABLE stream_metadata (
    stream_id VARCHAR(255) PRIMARY KEY,
    stream_type VARCHAR(100) NOT NULL,
    
    -- Stream state
    current_version INTEGER DEFAULT 0,
    event_count INTEGER DEFAULT 0,
    
    -- Stream lifecycle
    created_at TIMESTAMP DEFAULT NOW(),
    last_event_at TIMESTAMP,
    
    -- Stream properties
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    
    INDEX idx_stream_type (stream_type, last_event_at)
);

-- Function to append events to stream
CREATE OR REPLACE FUNCTION append_to_stream(
    stream_id VARCHAR(255),
    stream_type VARCHAR(100),
    event_type VARCHAR(100),
    event_data JSONB,
    expected_version INTEGER DEFAULT NULL,
    event_metadata JSONB DEFAULT '{}',
    correlation_id UUID DEFAULT NULL,
    causation_id UUID DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    current_version INTEGER;
    new_version INTEGER;
    event_id UUID;
BEGIN
    -- Get current stream version
    SELECT sm.current_version INTO current_version
    FROM stream_metadata sm
    WHERE sm.stream_id = append_to_stream.stream_id;
    
    -- Create stream metadata if doesn't exist
    IF current_version IS NULL THEN
        INSERT INTO stream_metadata (stream_id, stream_type, current_version, event_count)
        VALUES (append_to_stream.stream_id, append_to_stream.stream_type, 0, 0);
        current_version := 0;
    END IF;
    
    -- Check expected version for optimistic concurrency
    IF expected_version IS NOT NULL AND current_version != expected_version THEN
        RAISE EXCEPTION 'Stream version mismatch. Expected: %, Actual: %', 
            expected_version, current_version;
    END IF;
    
    new_version := current_version + 1;
    
    -- Insert event
    INSERT INTO event_streams (
        stream_id, stream_type, event_type, event_data, event_metadata,
        stream_version, correlation_id, causation_id, source_user
    ) VALUES (
        append_to_stream.stream_id, append_to_stream.stream_type, 
        append_to_stream.event_type, append_to_stream.event_data, 
        append_to_stream.event_metadata, new_version, 
        append_to_stream.correlation_id, append_to_stream.causation_id,
        current_setting('app.current_user_id', true)::UUID
    ) RETURNING id INTO event_id;
    
    -- Update stream metadata
    UPDATE stream_metadata 
    SET current_version = new_version,
        event_count = event_count + 1,
        last_event_at = NOW()
    WHERE stream_id = append_to_stream.stream_id;
    
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;
```

### Event Sourcing Implementation

```sql
-- Event-sourced aggregate root
CREATE TABLE user_streams (
    stream_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Current state (projection)
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE,
    full_name VARCHAR(255),
    status user_status DEFAULT 'pending',
    
    -- Event sourcing metadata
    version INTEGER DEFAULT 0,
    last_event_at TIMESTAMP,
    
    -- Snapshot optimization
    snapshot_version INTEGER DEFAULT 0,
    snapshot_data JSONB,
    snapshot_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_user_status (status),
    INDEX idx_user_email (email)
);

-- User events
INSERT INTO event_types (name, description, event_schema) VALUES
('UserRegistered', 'User account was created', '{
    "type": "object",
    "required": ["email", "username"],
    "properties": {
        "email": {"type": "string", "format": "email"},
        "username": {"type": "string", "minLength": 3},
        "full_name": {"type": "string"}
    }
}'),
('UserEmailVerified', 'User verified their email address', '{}'),
('UserProfileUpdated', 'User updated their profile', '{
    "type": "object",
    "properties": {
        "full_name": {"type": "string"},
        "username": {"type": "string"}
    }
}'),
('UserDeactivated', 'User account was deactivated', '{
    "type": "object",
    "properties": {
        "reason": {"type": "string"}
    }
}');

-- Function to register new user (event-sourced)
CREATE OR REPLACE FUNCTION register_user(
    email VARCHAR(255),
    username VARCHAR(100),
    full_name VARCHAR(255) DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    user_id UUID;
    event_data JSONB;
BEGIN
    user_id := gen_random_uuid();
    
    -- Build event data
    event_data := jsonb_build_object(
        'email', email,
        'username', username,
        'full_name', full_name
    );
    
    -- Append registration event
    PERFORM append_to_stream(
        user_id::text,
        'user',
        'UserRegistered',
        event_data
    );
    
    -- Create projection (current state)
    INSERT INTO user_streams (
        stream_id, email, username, full_name, 
        version, last_event_at
    ) VALUES (
        user_id, email, username, full_name,
        1, NOW()
    );
    
    RETURN user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to rebuild user state from events
CREATE OR REPLACE FUNCTION rebuild_user_from_events(
    user_id UUID
) RETURNS user_streams AS $$
DECLARE
    user_record user_streams%ROWTYPE;
    event_record RECORD;
    event_data JSONB;
BEGIN
    -- Initialize user record
    user_record.stream_id := user_id;
    user_record.version := 0;
    user_record.status := 'pending';
    
    -- Replay all events for this user
    FOR event_record IN
        SELECT event_type, event_data, stream_version, occurred_at
        FROM event_streams
        WHERE stream_id = user_id::text
          AND stream_type = 'user'
        ORDER BY stream_version
    LOOP
        event_data := event_record.event_data;
        
        -- Apply event to state
        CASE event_record.event_type
            WHEN 'UserRegistered' THEN
                user_record.email := event_data->>'email';
                user_record.username := event_data->>'username';
                user_record.full_name := event_data->>'full_name';
                user_record.status := 'pending';
                user_record.created_at := event_record.occurred_at;
                
            WHEN 'UserEmailVerified' THEN
                user_record.status := 'active';
                
            WHEN 'UserProfileUpdated' THEN
                user_record.full_name := COALESCE(event_data->>'full_name', user_record.full_name);
                user_record.username := COALESCE(event_data->>'username', user_record.username);
                
            WHEN 'UserDeactivated' THEN
                user_record.status := 'deactivated';
        END CASE;
        
        user_record.version := event_record.stream_version;
        user_record.last_event_at := event_record.occurred_at;
        user_record.updated_at := event_record.occurred_at;
    END LOOP;
    
    RETURN user_record;
END;
$$ LANGUAGE plpgsql;
```

## Stream Processing

### Event Projections

```sql
-- Read model projections
CREATE TABLE user_profile_projection (
    user_id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    username VARCHAR(100),
    full_name VARCHAR(255),
    
    -- Derived fields
    display_name VARCHAR(255) GENERATED ALWAYS AS (
        COALESCE(full_name, username, split_part(email, '@', 1))
    ) STORED,
    
    -- Metrics
    profile_completeness INTEGER DEFAULT 0,
    last_login_at TIMESTAMP,
    
    -- Projection metadata
    last_processed_event BIGINT,
    projected_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_user_profile_email (email),
    INDEX idx_user_profile_username (username)
);

-- User activity projection
CREATE TABLE user_activity_projection (
    user_id UUID PRIMARY KEY,
    
    -- Activity counters
    login_count INTEGER DEFAULT 0,
    profile_updates_count INTEGER DEFAULT 0,
    last_activity_at TIMESTAMP,
    
    -- Activity patterns
    most_active_hour INTEGER,
    most_active_day INTEGER,
    
    -- Projection metadata
    last_processed_event BIGINT,
    projected_at TIMESTAMP DEFAULT NOW()
);

-- Event projection processor
CREATE OR REPLACE FUNCTION process_user_events() RETURNS INTEGER AS $$
DECLARE
    processed_count INTEGER := 0;
    event_record RECORD;
    last_processed BIGINT;
BEGIN
    -- Get last processed event
    SELECT COALESCE(MAX(last_processed_event), 0) INTO last_processed
    FROM user_profile_projection;
    
    -- Process new events
    FOR event_record IN
        SELECT *
        FROM event_streams
        WHERE stream_type = 'user'
          AND sequence_number > last_processed
        ORDER BY sequence_number
    LOOP
        -- Update profile projection
        CASE event_record.event_type
            WHEN 'UserRegistered' THEN
                INSERT INTO user_profile_projection (
                    user_id, email, username, full_name, last_processed_event
                ) VALUES (
                    event_record.stream_id::UUID,
                    event_record.event_data->>'email',
                    event_record.event_data->>'username',
                    event_record.event_data->>'full_name',
                    event_record.sequence_number
                ) ON CONFLICT (user_id) DO UPDATE SET
                    email = EXCLUDED.email,
                    username = EXCLUDED.username,
                    full_name = EXCLUDED.full_name,
                    last_processed_event = EXCLUDED.last_processed_event;
                    
            WHEN 'UserProfileUpdated' THEN
                UPDATE user_profile_projection 
                SET full_name = COALESCE(event_record.event_data->>'full_name', full_name),
                    username = COALESCE(event_record.event_data->>'username', username),
                    last_processed_event = event_record.sequence_number,
                    projected_at = NOW()
                WHERE user_id = event_record.stream_id::UUID;
                
                -- Update activity projection
                INSERT INTO user_activity_projection (
                    user_id, profile_updates_count, last_activity_at, last_processed_event
                ) VALUES (
                    event_record.stream_id::UUID, 1, event_record.occurred_at, event_record.sequence_number
                ) ON CONFLICT (user_id) DO UPDATE SET
                    profile_updates_count = user_activity_projection.profile_updates_count + 1,
                    last_activity_at = event_record.occurred_at,
                    last_processed_event = event_record.sequence_number;
        END CASE;
        
        processed_count := processed_count + 1;
    END LOOP;
    
    RETURN processed_count;
END;
$$ LANGUAGE plpgsql;
```

### Snapshots for Performance

```sql
-- Snapshot table for large aggregates
CREATE TABLE aggregate_snapshots (
    aggregate_id VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    
    -- Snapshot data
    version INTEGER NOT NULL,
    snapshot_data JSONB NOT NULL,
    
    -- Snapshot metadata
    created_at TIMESTAMP DEFAULT NOW(),
    events_count INTEGER,
    
    PRIMARY KEY (aggregate_id, aggregate_type),
    INDEX idx_snapshots_version (aggregate_type, version DESC)
);

-- Function to create snapshot
CREATE OR REPLACE FUNCTION create_snapshot(
    aggregate_id VARCHAR(255),
    aggregate_type VARCHAR(100),
    snapshot_interval INTEGER DEFAULT 100
) RETURNS BOOLEAN AS $$
DECLARE
    current_version INTEGER;
    last_snapshot_version INTEGER := 0;
    aggregate_data JSONB;
BEGIN
    -- Get current version
    SELECT current_version INTO current_version
    FROM stream_metadata
    WHERE stream_id = aggregate_id;
    
    -- Get last snapshot version
    SELECT version INTO last_snapshot_version
    FROM aggregate_snapshots
    WHERE aggregate_id = create_snapshot.aggregate_id
      AND aggregate_type = create_snapshot.aggregate_type;
    
    -- Check if snapshot is needed
    IF current_version - COALESCE(last_snapshot_version, 0) < snapshot_interval THEN
        RETURN FALSE;
    END IF;
    
    -- Rebuild aggregate state
    CASE aggregate_type
        WHEN 'user' THEN
            SELECT to_jsonb(rebuild_user_from_events(aggregate_id::UUID)) INTO aggregate_data;
        -- Add other aggregate types as needed
        ELSE
            RAISE EXCEPTION 'Unknown aggregate type: %', aggregate_type;
    END CASE;
    
    -- Save snapshot
    INSERT INTO aggregate_snapshots (
        aggregate_id, aggregate_type, version, 
        snapshot_data, events_count
    ) VALUES (
        aggregate_id, aggregate_type, current_version,
        aggregate_data, current_version
    ) ON CONFLICT (aggregate_id, aggregate_type) DO UPDATE SET
        version = EXCLUDED.version,
        snapshot_data = EXCLUDED.snapshot_data,
        created_at = NOW(),
        events_count = EXCLUDED.events_count;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Optimized function to load aggregate with snapshot
CREATE OR REPLACE FUNCTION load_user_with_snapshot(
    user_id UUID
) RETURNS user_streams AS $$
DECLARE
    user_record user_streams%ROWTYPE;
    snapshot_data JSONB;
    snapshot_version INTEGER := 0;
    event_record RECORD;
BEGIN
    -- Try to load from snapshot
    SELECT snapshot_data, version INTO snapshot_data, snapshot_version
    FROM aggregate_snapshots
    WHERE aggregate_id = user_id::text
      AND aggregate_type = 'user';
    
    IF snapshot_data IS NOT NULL THEN
        -- Load from snapshot
        SELECT * INTO user_record FROM jsonb_populate_record(NULL::user_streams, snapshot_data);
    ELSE
        -- Initialize empty record
        user_record.stream_id := user_id;
        user_record.version := 0;
    END IF;
    
    -- Apply events since snapshot
    FOR event_record IN
        SELECT event_type, event_data, stream_version, occurred_at
        FROM event_streams
        WHERE stream_id = user_id::text
          AND stream_type = 'user'
          AND stream_version > snapshot_version
        ORDER BY stream_version
    LOOP
        -- Apply event logic (same as rebuild_user_from_events)
        -- ... event handling code ...
        user_record.version := event_record.stream_version;
        user_record.last_event_at := event_record.occurred_at;
    END LOOP;
    
    RETURN user_record;
END;
$$ LANGUAGE plpgsql;
```

## Real-World Examples

### E-commerce Order Events

```sql
-- Order event types
INSERT INTO event_types (name, description, event_schema) VALUES
('OrderCreated', 'Customer created a new order', '{
    "type": "object",
    "required": ["customer_id", "items", "total_amount"],
    "properties": {
        "customer_id": {"type": "string"},
        "items": {"type": "array"},
        "total_amount": {"type": "number"},
        "currency": {"type": "string"}
    }
}'),
('OrderItemAdded', 'Item was added to order', '{}'),
('OrderItemRemoved', 'Item was removed from order', '{}'),
('OrderConfirmed', 'Order was confirmed and payment processed', '{}'),
('OrderShipped', 'Order was shipped to customer', '{}'),
('OrderDelivered', 'Order was delivered successfully', '{}'),
('OrderCancelled', 'Order was cancelled', '{}');

-- Order projection
CREATE TABLE order_projection (
    order_id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    
    -- Order details
    status order_status NOT NULL,
    total_amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    -- Order items (denormalized)
    items JSONB NOT NULL DEFAULT '[]',
    item_count INTEGER GENERATED ALWAYS AS (jsonb_array_length(items)) STORED,
    
    -- Key timestamps
    created_at TIMESTAMP NOT NULL,
    confirmed_at TIMESTAMP,
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    
    -- Event sourcing metadata
    version INTEGER NOT NULL,
    last_event_at TIMESTAMP NOT NULL,
    
    INDEX idx_order_customer (customer_id, created_at),
    INDEX idx_order_status (status, created_at)
);

-- Function to create order
CREATE OR REPLACE FUNCTION create_order(
    customer_id UUID,
    items JSONB,
    total_amount DECIMAL(19,4),
    currency_code CHAR(3) DEFAULT 'USD'
) RETURNS UUID AS $$
DECLARE
    order_id UUID;
    event_data JSONB;
BEGIN
    order_id := gen_random_uuid();
    
    event_data := jsonb_build_object(
        'customer_id', customer_id,
        'items', items,
        'total_amount', total_amount,
        'currency_code', currency_code
    );
    
    -- Append creation event
    PERFORM append_to_stream(
        order_id::text,
        'order',
        'OrderCreated',
        event_data
    );
    
    -- Create projection
    INSERT INTO order_projection (
        order_id, customer_id, status, total_amount, 
        currency_code, items, version, created_at, last_event_at
    ) VALUES (
        order_id, customer_id, 'pending', total_amount,
        currency_code, items, 1, NOW(), NOW()
    );
    
    RETURN order_id;
END;
$$ LANGUAGE plpgsql;
```

### Banking Transaction Events

```sql
-- Account event stream for banking
CREATE TABLE account_projection (
    account_id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    account_number VARCHAR(20) UNIQUE NOT NULL,
    
    -- Account state
    balance DECIMAL(19,4) NOT NULL DEFAULT 0,
    status account_status DEFAULT 'active',
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Transaction counters
    transaction_count INTEGER DEFAULT 0,
    last_transaction_at TIMESTAMP,
    
    -- Event sourcing metadata
    version INTEGER DEFAULT 0,
    last_event_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_account_customer (customer_id),
    INDEX idx_account_number (account_number)
);

-- Function to process account transaction
CREATE OR REPLACE FUNCTION process_account_transaction(
    account_id UUID,
    transaction_type VARCHAR(50),
    amount DECIMAL(19,4),
    reference VARCHAR(255) DEFAULT NULL,
    metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    current_balance DECIMAL(19,4);
    new_balance DECIMAL(19,4);
    event_data JSONB;
    correlation_id UUID;
BEGIN
    -- Get current balance
    SELECT balance INTO current_balance
    FROM account_projection
    WHERE account_id = process_account_transaction.account_id;
    
    IF current_balance IS NULL THEN
        RAISE EXCEPTION 'Account not found: %', account_id;
    END IF;
    
    -- Calculate new balance
    CASE transaction_type
        WHEN 'deposit', 'credit' THEN
            new_balance := current_balance + amount;
        WHEN 'withdrawal', 'debit' THEN
            new_balance := current_balance - amount;
            IF new_balance < 0 THEN
                RAISE EXCEPTION 'Insufficient funds. Current balance: %', current_balance;
            END IF;
        ELSE
            RAISE EXCEPTION 'Invalid transaction type: %', transaction_type;
    END CASE;
    
    correlation_id := gen_random_uuid();
    
    event_data := jsonb_build_object(
        'transaction_type', transaction_type,
        'amount', amount,
        'previous_balance', current_balance,
        'new_balance', new_balance,
        'reference', reference,
        'metadata', metadata
    );
    
    -- Append transaction event
    PERFORM append_to_stream(
        account_id::text,
        'account',
        'TransactionProcessed',
        event_data,
        NULL, -- expected_version
        '{}', -- event_metadata
        correlation_id
    );
    
    -- Update projection
    UPDATE account_projection
    SET balance = new_balance,
        transaction_count = transaction_count + 1,
        last_transaction_at = NOW(),
        version = version + 1,
        last_event_at = NOW()
    WHERE account_id = process_account_transaction.account_id;
    
    RETURN correlation_id;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Event Design
- **Make events immutable**: Never modify events after creation
- **Use descriptive names**: Events should clearly describe what happened
- **Include sufficient data**: Events should be self-contained
- **Version your events**: Plan for schema evolution
- **Use correlation IDs**: Track related events across streams

### 2. Stream Management
- **Partition by stream ID**: Distribute load across database partitions
- **Implement retention policies**: Archive or delete old events
- **Use snapshots wisely**: Balance between storage and rebuild time
- **Monitor stream health**: Track stream sizes and processing lag
- **Design for idempotency**: Handle duplicate event processing

### 3. Performance Optimization
- **Index on sequence numbers**: Enable efficient range queries
- **Use projections**: Pre-compute read models for common queries
- **Batch process events**: Group event processing for efficiency
- **Cache hot data**: Keep frequently accessed projections in memory
- **Partition large streams**: Split very large streams for better performance

### 4. Consistency and Reliability
- **Use optimistic concurrency**: Prevent conflicting concurrent updates
- **Implement compensating actions**: Handle failures in distributed systems
- **Validate event schemas**: Ensure data quality at write time
- **Monitor processing lag**: Ensure projections stay up to date
- **Plan for replay**: Design systems to handle event reprocessing

### 5. Operational Considerations
- **Monitor storage growth**: Event streams grow continuously
- **Implement backup strategies**: Ensure event data is protected
- **Plan for debugging**: Make it easy to trace event flow
- **Document event semantics**: Clear business meaning for each event type
- **Test event replay**: Ensure system recovery procedures work

This comprehensive event stream design provides a solid foundation for building event-driven systems with complete audit trails and powerful analytics capabilities.
