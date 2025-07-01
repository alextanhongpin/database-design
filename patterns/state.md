# State Management Database Design

## Table of Contents
- [Overview](#overview)
- [State vs Event Storage](#state-vs-event-storage)
- [State Machine Implementation](#state-machine-implementation)
- [Audit Trail Patterns](#audit-trail-patterns)
- [Event Sourcing Basics](#event-sourcing-basics)
- [Real-World Examples](#real-world-examples)
- [Advanced Patterns](#advanced-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Overview

State management is crucial for tracking entity lifecycle, ensuring data consistency, and maintaining audit trails. This guide covers patterns for storing and managing state transitions in database systems.

### Key Concepts
- **Current State**: The present condition of an entity
- **State Transitions**: Valid moves between states
- **Event History**: Sequence of actions that led to current state
- **Audit Trail**: Complete record of all changes
- **State Machine**: Rules governing valid state transitions

## State vs Event Storage

### Single Source of Truth: Final State Only

Simple approach storing only the current state of entities.

```sql
-- Basic state-only approach
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Order details
    total_amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Current state only
    status order_status DEFAULT 'pending',
    
    -- Basic timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_orders_status (status, created_at),
    INDEX idx_orders_customer (customer_id, status)
);

CREATE TYPE order_status AS ENUM (
    'pending', 'confirmed', 'processing', 'shipped', 
    'delivered', 'cancelled', 'refunded'
);

-- Simple state update
UPDATE orders 
SET status = 'shipped', updated_at = NOW() 
WHERE id = ? AND status = 'processing';
```

**Pros:**
- Simple to implement and query
- Minimal storage overhead
- Fast queries for current state
- Easy to understand

**Cons:**
- No audit trail
- Can't answer "why" questions
- No rollback capability
- Loss of historical context

### Hybrid Approach: State + Event Log

Maintains current state for performance while preserving event history.

```sql
-- Main entity table with current state
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Order details
    total_amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Current state (derived from events)
    status order_status DEFAULT 'pending',
    version INTEGER DEFAULT 0, -- For optimistic locking
    
    -- Key timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_orders_status (status, created_at),
    INDEX idx_orders_customer (customer_id, status)
);

-- Event log for complete history
CREATE TABLE order_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Event details
    event_type VARCHAR(100) NOT NULL,
    from_status order_status,
    to_status order_status NOT NULL,
    
    -- Event metadata
    event_data JSONB DEFAULT '{}',
    reason TEXT,
    
    -- Who and when
    triggered_by UUID REFERENCES users(id),
    occurred_at TIMESTAMP DEFAULT NOW(),
    
    -- Event ordering
    sequence_number BIGINT GENERATED ALWAYS AS IDENTITY,
    
    INDEX idx_order_events_order (order_id, sequence_number),
    INDEX idx_order_events_type (event_type, occurred_at),
    INDEX idx_order_events_user (triggered_by, occurred_at)
);

-- Function to transition order state
CREATE OR REPLACE FUNCTION transition_order_state(
    order_id UUID,
    new_status order_status,
    event_type VARCHAR(100),
    triggered_by UUID,
    reason TEXT DEFAULT NULL,
    event_data JSONB DEFAULT '{}'
) RETURNS BOOLEAN AS $$
DECLARE
    current_order RECORD;
    is_valid_transition BOOLEAN;
BEGIN
    -- Get current order state
    SELECT * INTO current_order 
    FROM orders 
    WHERE id = order_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Order not found: %', order_id;
    END IF;
    
    -- Validate state transition
    SELECT validate_order_transition(current_order.status, new_status) 
    INTO is_valid_transition;
    
    IF NOT is_valid_transition THEN
        RAISE EXCEPTION 'Invalid state transition from % to %', 
            current_order.status, new_status;
    END IF;
    
    -- Record the event
    INSERT INTO order_events (
        order_id, event_type, from_status, to_status,
        event_data, reason, triggered_by
    ) VALUES (
        order_id, event_type, current_order.status, new_status,
        event_data, reason, triggered_by
    );
    
    -- Update current state
    UPDATE orders 
    SET status = new_status,
        version = version + 1,
        updated_at = NOW()
    WHERE id = order_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- State transition validation
CREATE OR REPLACE FUNCTION validate_order_transition(
    from_status order_status,
    to_status order_status
) RETURNS BOOLEAN AS $$
BEGIN
    -- Define valid transitions
    RETURN CASE from_status
        WHEN 'pending' THEN to_status IN ('confirmed', 'cancelled')
        WHEN 'confirmed' THEN to_status IN ('processing', 'cancelled')
        WHEN 'processing' THEN to_status IN ('shipped', 'cancelled')
        WHEN 'shipped' THEN to_status IN ('delivered', 'cancelled')
        WHEN 'delivered' THEN to_status IN ('refunded')
        WHEN 'cancelled' THEN to_status IN ('confirmed') -- Allow reactivation
        WHEN 'refunded' THEN FALSE -- Terminal state
        ELSE FALSE
    END;
END;
$$ LANGUAGE plpgsql;
```

## State Machine Implementation

### Finite State Machine with Rules

```sql
-- State machine definition
CREATE TABLE state_machines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    entity_type VARCHAR(100) NOT NULL, -- orders, invoices, etc.
    
    -- Configuration
    initial_state VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- States in the machine
CREATE TABLE states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state_machine_id UUID NOT NULL REFERENCES state_machines(id),
    
    -- State definition
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- State properties
    is_initial BOOLEAN DEFAULT FALSE,
    is_final BOOLEAN DEFAULT FALSE,
    requires_approval BOOLEAN DEFAULT FALSE,
    
    -- UI/UX properties
    color VARCHAR(7), -- Hex color
    icon VARCHAR(50),
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (state_machine_id, name)
);

-- Valid transitions between states
CREATE TABLE state_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state_machine_id UUID NOT NULL REFERENCES state_machines(id),
    from_state_id UUID NOT NULL REFERENCES states(id),
    to_state_id UUID NOT NULL REFERENCES states(id),
    
    -- Transition properties
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(150),
    description TEXT,
    
    -- Conditions and requirements
    requires_permission VARCHAR(100),
    condition_function VARCHAR(100), -- Function name to check conditions
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (state_machine_id, from_state_id, to_state_id)
);

-- Advanced state transition function
CREATE OR REPLACE FUNCTION execute_state_transition(
    entity_table TEXT,
    entity_id UUID,
    transition_name VARCHAR(100),
    user_id UUID,
    context JSONB DEFAULT '{}'
) RETURNS BOOLEAN AS $$
DECLARE
    current_state VARCHAR(50);
    target_state VARCHAR(50);
    state_machine_id UUID;
    transition_record RECORD;
    can_transition BOOLEAN;
BEGIN
    -- Get current state (assumes entity has 'status' column)
    EXECUTE format('SELECT status FROM %I WHERE id = $1', entity_table) 
    INTO current_state USING entity_id;
    
    IF current_state IS NULL THEN
        RAISE EXCEPTION 'Entity not found: %', entity_id;
    END IF;
    
    -- Find the transition
    SELECT st.*, s_from.name as from_state, s_to.name as to_state
    INTO transition_record
    FROM state_transitions st
    JOIN states s_from ON s_from.id = st.from_state_id
    JOIN states s_to ON s_to.id = st.to_state_id
    WHERE st.name = transition_name
      AND s_from.name = current_state;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Invalid transition % from state %', transition_name, current_state;
    END IF;
    
    -- Check permissions if required
    IF transition_record.requires_permission IS NOT NULL THEN
        -- Implementation depends on your permission system
        -- SELECT check_user_permission(user_id, transition_record.requires_permission) INTO can_transition;
        can_transition := TRUE; -- Simplified
    ELSE
        can_transition := TRUE;
    END IF;
    
    IF NOT can_transition THEN
        RAISE EXCEPTION 'Insufficient permissions for transition %', transition_name;
    END IF;
    
    -- Execute the transition
    EXECUTE format('UPDATE %I SET status = $1, updated_at = NOW() WHERE id = $2', entity_table)
    USING transition_record.to_state, entity_id;
    
    -- Log the transition (using generic event log)
    INSERT INTO entity_state_changes (
        entity_table, entity_id, from_state, to_state,
        transition_name, changed_by, context
    ) VALUES (
        entity_table, entity_id, current_state, transition_record.to_state,
        transition_name, user_id, context
    );
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Generic state change log
CREATE TABLE entity_state_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Entity identification
    entity_table VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    
    -- State change details
    from_state VARCHAR(50) NOT NULL,
    to_state VARCHAR(50) NOT NULL,
    transition_name VARCHAR(100),
    
    -- Context
    changed_by UUID REFERENCES users(id),
    context JSONB DEFAULT '{}',
    
    -- Timing
    changed_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_entity_changes (entity_table, entity_id, changed_at),
    INDEX idx_state_changes_user (changed_by, changed_at)
);
```

## Audit Trail Patterns

### Comprehensive Audit Trail

```sql
-- Comprehensive audit trail for any table
CREATE TABLE audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- What was changed
    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    
    -- Type of change
    operation operation_type NOT NULL,
    
    -- Change details
    old_values JSONB,
    new_values JSONB,
    changed_columns TEXT[],
    
    -- Who and when
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMP DEFAULT NOW(),
    
    -- Context
    transaction_id UUID, -- Group related changes
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    
    INDEX idx_audit_table_record (table_name, record_id, changed_at),
    INDEX idx_audit_user (changed_by, changed_at),
    INDEX idx_audit_transaction (transaction_id)
);

CREATE TYPE operation_type AS ENUM ('INSERT', 'UPDATE', 'DELETE', 'TRUNCATE');

-- Generic audit trigger function
CREATE OR REPLACE FUNCTION audit_trigger()
RETURNS TRIGGER AS $$
DECLARE
    audit_row audit_trail%ROWTYPE;
    old_data JSONB;
    new_data JSONB;
    changed_cols TEXT[] := '{}';
    col_name TEXT;
BEGIN
    -- Determine operation type
    audit_row.operation = TG_OP::operation_type;
    audit_row.table_name = TG_TABLE_NAME;
    audit_row.changed_at = NOW();
    
    -- Get user from session (requires setting session variable)
    audit_row.changed_by = current_setting('app.current_user_id', true)::UUID;
    
    IF TG_OP = 'DELETE' THEN
        audit_row.record_id = OLD.id;
        old_data = to_jsonb(OLD);
        new_data = NULL;
    ELSIF TG_OP = 'INSERT' THEN
        audit_row.record_id = NEW.id;
        old_data = NULL;
        new_data = to_jsonb(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        audit_row.record_id = NEW.id;
        old_data = to_jsonb(OLD);
        new_data = to_jsonb(NEW);
        
        -- Find changed columns
        FOR col_name IN 
            SELECT key FROM jsonb_each(old_data) 
            WHERE jsonb_each.value != (new_data->>key)::jsonb
        LOOP
            changed_cols := array_append(changed_cols, col_name);
        END LOOP;
    END IF;
    
    audit_row.old_values = old_data;
    audit_row.new_values = new_data;
    audit_row.changed_columns = changed_cols;
    
    INSERT INTO audit_trail VALUES (audit_row.*);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Apply audit trigger to any table
-- CREATE TRIGGER audit_orders AFTER INSERT OR UPDATE OR DELETE ON orders 
-- FOR EACH ROW EXECUTE FUNCTION audit_trigger();
```

## Real-World Examples

### Order Management System

```sql
-- Complete order lifecycle management
CREATE TABLE order_statuses (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER,
    
    -- Status properties
    is_terminal BOOLEAN DEFAULT FALSE,
    requires_payment BOOLEAN DEFAULT FALSE,
    allows_cancellation BOOLEAN DEFAULT TRUE,
    
    -- Visual properties
    color VARCHAR(7),
    icon VARCHAR(50)
);

-- Insert predefined statuses
INSERT INTO order_statuses (code, name, description, sort_order, is_terminal, requires_payment, allows_cancellation) VALUES
('pending', 'Pending', 'Order created but not yet confirmed', 1, FALSE, FALSE, TRUE),
('confirmed', 'Confirmed', 'Order confirmed and payment received', 2, FALSE, TRUE, TRUE),
('processing', 'Processing', 'Order is being prepared', 3, FALSE, TRUE, TRUE),
('shipped', 'Shipped', 'Order has been shipped', 4, FALSE, TRUE, FALSE),
('delivered', 'Delivered', 'Order has been delivered', 5, TRUE, TRUE, FALSE),
('cancelled', 'Cancelled', 'Order has been cancelled', 6, TRUE, FALSE, FALSE),
('refunded', 'Refunded', 'Order has been refunded', 7, TRUE, FALSE, FALSE);

-- Orders with rich state management
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Order details
    total_amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- State management
    status VARCHAR(20) NOT NULL DEFAULT 'pending' REFERENCES order_statuses(code),
    status_changed_at TIMESTAMP DEFAULT NOW(),
    status_changed_by UUID REFERENCES users(id),
    
    -- Cancellation details
    can_be_cancelled BOOLEAN GENERATED ALWAYS AS (
        status IN ('pending', 'confirmed', 'processing')
    ) STORED,
    cancelled_at TIMESTAMP,
    cancellation_reason TEXT,
    
    -- Payment tracking
    payment_required BOOLEAN GENERATED ALWAYS AS (
        (SELECT requires_payment FROM order_statuses WHERE code = status)
    ) STORED,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_orders_status (status, created_at),
    INDEX idx_orders_customer_status (customer_id, status)
);

-- Order state change log
CREATE TABLE order_state_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- State change details
    from_status VARCHAR(20) REFERENCES order_statuses(code),
    to_status VARCHAR(20) NOT NULL REFERENCES order_statuses(code),
    
    -- Change context
    reason TEXT,
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    
    -- Who and when
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_order_state_changes (order_id, changed_at)
);

-- Order state transition with validation
CREATE OR REPLACE FUNCTION change_order_status(
    order_id UUID,
    new_status VARCHAR(20),
    changed_by UUID,
    reason TEXT DEFAULT NULL,
    notes TEXT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    current_order RECORD;
    valid_transitions TEXT[];
BEGIN
    -- Get current order
    SELECT o.*, os.is_terminal, os.allows_cancellation
    INTO current_order
    FROM orders o
    JOIN order_statuses os ON os.code = o.status
    WHERE o.id = order_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Order not found: %', order_id;
    END IF;
    
    -- Check if current status is terminal
    IF current_order.is_terminal THEN
        RAISE EXCEPTION 'Cannot change status of order in terminal state: %', current_order.status;
    END IF;
    
    -- Define valid transitions based on business rules
    valid_transitions := CASE current_order.status
        WHEN 'pending' THEN ARRAY['confirmed', 'cancelled']
        WHEN 'confirmed' THEN ARRAY['processing', 'cancelled']
        WHEN 'processing' THEN ARRAY['shipped', 'cancelled']
        WHEN 'shipped' THEN ARRAY['delivered']
        ELSE ARRAY[]::TEXT[]
    END;
    
    -- Special case: allow refund from delivered
    IF current_order.status = 'delivered' AND new_status = 'refunded' THEN
        valid_transitions := array_append(valid_transitions, 'refunded');
    END IF;
    
    -- Validate transition
    IF NOT (new_status = ANY(valid_transitions)) THEN
        RAISE EXCEPTION 'Invalid status transition from % to %', current_order.status, new_status;
    END IF;
    
    -- Record state change
    INSERT INTO order_state_changes (
        order_id, from_status, to_status, reason, notes, changed_by
    ) VALUES (
        order_id, current_order.status, new_status, reason, notes, changed_by
    );
    
    -- Update order
    UPDATE orders 
    SET status = new_status,
        status_changed_at = NOW(),
        status_changed_by = changed_by,
        updated_at = NOW(),
        cancelled_at = CASE WHEN new_status = 'cancelled' THEN NOW() ELSE cancelled_at END,
        cancellation_reason = CASE WHEN new_status = 'cancelled' THEN reason ELSE cancellation_reason END
    WHERE id = order_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### User Account Lifecycle

```sql
-- User account state management
CREATE TYPE user_status AS ENUM (
    'pending_verification', 'active', 'suspended', 
    'deactivated', 'banned', 'deleted'
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    
    -- State management
    status user_status DEFAULT 'pending_verification',
    status_reason TEXT,
    status_changed_at TIMESTAMP DEFAULT NOW(),
    status_changed_by UUID, -- Can reference users(id) or admin users
    
    -- Account restrictions
    can_login BOOLEAN GENERATED ALWAYS AS (
        status IN ('active')
    ) STORED,
    can_post BOOLEAN GENERATED ALWAYS AS (
        status IN ('active')
    ) STORED,
    
    -- Suspension details
    suspended_until TIMESTAMP,
    suspension_reason TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_users_status (status, created_at),
    INDEX idx_users_suspended_until (suspended_until) WHERE suspended_until IS NOT NULL
);

-- User status change history
CREATE TABLE user_status_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    from_status user_status,
    to_status user_status NOT NULL,
    
    -- Change details
    reason TEXT NOT NULL,
    duration INTERVAL, -- For temporary suspensions
    notes TEXT,
    
    -- Administrative info
    changed_by UUID, -- Admin or system user
    changed_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP, -- For temporary status changes
    
    INDEX idx_user_status_changes (user_id, changed_at)
);

-- Function to change user status with business logic
CREATE OR REPLACE FUNCTION change_user_status(
    user_id UUID,
    new_status user_status,
    reason TEXT,
    changed_by UUID DEFAULT NULL,
    duration INTERVAL DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    current_user RECORD;
    expires_at TIMESTAMP;
BEGIN
    -- Get current user
    SELECT * INTO current_user FROM users WHERE id = user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'User not found: %', user_id;
    END IF;
    
    -- Calculate expiration for temporary changes
    IF duration IS NOT NULL THEN
        expires_at := NOW() + duration;
    END IF;
    
    -- Business logic validations
    IF current_user.status = 'deleted' AND new_status != 'deleted' THEN
        RAISE EXCEPTION 'Cannot restore deleted user account';
    END IF;
    
    -- Record status change
    INSERT INTO user_status_changes (
        user_id, from_status, to_status, reason, 
        duration, changed_by, expires_at
    ) VALUES (
        user_id, current_user.status, new_status, reason,
        duration, changed_by, expires_at
    );
    
    -- Update user
    UPDATE users 
    SET status = new_status,
        status_reason = reason,
        status_changed_at = NOW(),
        status_changed_by = changed_by,
        suspended_until = CASE WHEN new_status = 'suspended' THEN expires_at ELSE NULL END,
        suspension_reason = CASE WHEN new_status = 'suspended' THEN reason ELSE NULL END,
        updated_at = NOW()
    WHERE id = user_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Automatic status restoration for temporary suspensions
CREATE OR REPLACE FUNCTION restore_expired_suspensions() RETURNS INTEGER AS $$
DECLARE
    restored_count INTEGER := 0;
BEGIN
    -- Restore users whose suspension has expired
    WITH expired_suspensions AS (
        UPDATE users 
        SET status = 'active',
            status_reason = 'Suspension expired automatically',
            status_changed_at = NOW(),
            suspended_until = NULL,
            suspension_reason = NULL,
            updated_at = NOW()
        WHERE status = 'suspended' 
          AND suspended_until IS NOT NULL 
          AND suspended_until <= NOW()
        RETURNING id
    )
    SELECT COUNT(*) INTO restored_count FROM expired_suspensions;
    
    -- Log automatic restorations
    INSERT INTO user_status_changes (
        user_id, from_status, to_status, reason, changed_at
    )
    SELECT 
        u.id, 'suspended', 'active', 
        'Automatic restoration after suspension expired',
        NOW()
    FROM users u
    WHERE u.status = 'active' 
      AND u.status_changed_at >= NOW() - INTERVAL '1 minute'
      AND u.status_reason = 'Suspension expired automatically';
    
    RETURN restored_count;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. State Design
- **Keep states simple**: Avoid overly complex state machines
- **Use descriptive names**: Clear, business-meaningful state names
- **Define terminal states**: Clearly mark states that cannot transition
- **Validate transitions**: Implement business rule validation
- **Document state meanings**: Each state should have clear business semantics

### 2. Event Logging
- **Log all changes**: Never lose audit trail information
- **Include context**: Store why changes happened, not just what
- **Store metadata**: Additional information for debugging and analysis
- **Use consistent format**: Standardize event data structure
- **Enable querying**: Index events for efficient historical queries

### 3. Performance Optimization
- **Index current state**: Fast queries on current entity state
- **Partition event logs**: By date or entity type for large systems
- **Use materialized views**: For complex state-based reports
- **Cache frequently accessed states**: In application or database
- **Batch state changes**: Group related transitions when possible

### 4. Data Integrity
- **Use database constraints**: Enforce valid states at database level
- **Implement optimistic locking**: Prevent concurrent state changes
- **Validate state transitions**: Check business rules before changes
- **Handle failures gracefully**: Rollback on validation errors
- **Test state machines**: Comprehensive testing of all transitions

### 5. Scalability Considerations
- **Design for eventual consistency**: In distributed systems
- **Use event sourcing**: For systems requiring complete audit trails
- **Consider CQRS**: Separate read and write models for complex domains
- **Implement compensating actions**: For handling failures in distributed transactions
- **Monitor state distribution**: Track state patterns for optimization

This comprehensive state management design provides robust foundations for tracking entity lifecycles while maintaining performance and data integrity.
