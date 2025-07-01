# Workflow Pattern

A comprehensive guide to implementing flexible workflow systems that manage entity states through configurable processes.

## Table of Contents

1. [Overview](#overview)
2. [Core Concepts](#core-concepts)
3. [Basic Schema Design](#basic-schema-design)
4. [Advanced Workflow Engine](#advanced-workflow-engine)
5. [Real-World Examples](#real-world-examples)
6. [State Machine Implementation](#state-machine-implementation)
7. [Workflow Analytics](#workflow-analytics)
8. [Best Practices](#best-practices)
9. [Performance Considerations](#performance-considerations)
10. [Migration Strategies](#migration-strategies)

## Overview

The Workflow Pattern provides a flexible way to model and manage the lifecycle of business entities through configurable states and transitions. This pattern is essential for applications requiring approval processes, order fulfillment, content publishing, or any multi-step business process.

### Key Benefits

- **Flexibility**: Easy to modify workflows without code changes
- **Auditability**: Complete history of state changes
- **Parallelism**: Support for concurrent workflow paths
- **Scalability**: Handles complex multi-stage processes
- **Reusability**: Common workflows can be shared across entities

## Core Concepts

### Workflow Components

```sql
-- Core workflow definition tables
CREATE TABLE workflow_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT valid_version CHECK (version > 0)
);

CREATE TABLE workflow_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    description TEXT,
    state_type VARCHAR(50) NOT NULL DEFAULT 'normal',
    is_initial BOOLEAN NOT NULL DEFAULT false,
    is_terminal BOOLEAN NOT NULL DEFAULT false,
    timeout_minutes INTEGER,
    color_code VARCHAR(7), -- For UI visualization
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(workflow_id, name),
    CONSTRAINT valid_state_type CHECK (state_type IN ('initial', 'normal', 'terminal', 'error')),
    CONSTRAINT valid_color_code CHECK (color_code IS NULL OR color_code ~ '^#[0-9A-Fa-f]{6}$')
);

CREATE TABLE workflow_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id),
    from_state_id UUID NOT NULL REFERENCES workflow_states(id),
    to_state_id UUID NOT NULL REFERENCES workflow_states(id),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    condition_rules JSONB, -- Business rules for transition
    auto_transition BOOLEAN NOT NULL DEFAULT false,
    requires_approval BOOLEAN NOT NULL DEFAULT false,
    notification_template VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(workflow_id, from_state_id, name),
    CONSTRAINT no_self_loop CHECK (from_state_id != to_state_id OR name = 'retry')
);
```

## Basic Schema Design

### Entity State Management

```sql
-- Generic entity that follows a workflow
CREATE TABLE workflow_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL, -- 'order', 'application', 'document', etc.
    entity_id UUID NOT NULL, -- Reference to actual entity
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id),
    current_state_id UUID NOT NULL REFERENCES workflow_states(id),
    previous_state_id UUID REFERENCES workflow_states(id),
    assigned_to UUID, -- User responsible for next action
    assigned_at TIMESTAMPTZ,
    due_date TIMESTAMPTZ,
    priority INTEGER NOT NULL DEFAULT 3,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(entity_type, entity_id),
    CONSTRAINT valid_priority CHECK (priority BETWEEN 1 AND 5)
);

-- Complete audit trail of state changes
CREATE TABLE workflow_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_entity_id UUID NOT NULL REFERENCES workflow_entities(id),
    from_state_id UUID REFERENCES workflow_states(id),
    to_state_id UUID NOT NULL REFERENCES workflow_states(id),
    transition_id UUID REFERENCES workflow_transitions(id),
    action_taken VARCHAR(100) NOT NULL,
    comment TEXT,
    performed_by UUID NOT NULL,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_minutes INTEGER,
    metadata JSONB,
    
    CONSTRAINT valid_duration CHECK (duration_minutes IS NULL OR duration_minutes >= 0)
);

-- Active workflow tasks and assignments
CREATE TABLE workflow_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_entity_id UUID NOT NULL REFERENCES workflow_entities(id),
    task_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    assigned_to UUID,
    assigned_by UUID,
    assigned_at TIMESTAMPTZ,
    due_date TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    priority INTEGER NOT NULL DEFAULT 3,
    form_data JSONB,
    
    CONSTRAINT valid_task_status CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled', 'expired')),
    CONSTRAINT valid_priority CHECK (priority BETWEEN 1 AND 5)
);
```

## Advanced Workflow Engine

### Comprehensive Workflow System

```sql
-- Workflow conditions and rules
CREATE TABLE workflow_conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transition_id UUID NOT NULL REFERENCES workflow_transitions(id),
    condition_type VARCHAR(50) NOT NULL,
    field_name VARCHAR(100),
    operator VARCHAR(20) NOT NULL,
    expected_value TEXT,
    error_message TEXT,
    is_required BOOLEAN NOT NULL DEFAULT true,
    
    CONSTRAINT valid_condition_type CHECK (condition_type IN ('field_value', 'user_role', 'time_based', 'custom_function')),
    CONSTRAINT valid_operator CHECK (operator IN ('eq', 'ne', 'gt', 'lt', 'gte', 'lte', 'in', 'not_in', 'exists', 'not_exists'))
);

-- Workflow notifications and alerts
CREATE TABLE workflow_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_entity_id UUID NOT NULL REFERENCES workflow_entities(id),
    notification_type VARCHAR(50) NOT NULL,
    recipient_type VARCHAR(20) NOT NULL, -- 'user', 'role', 'email'
    recipient_value TEXT NOT NULL,
    subject VARCHAR(200),
    message TEXT,
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    metadata JSONB,
    
    CONSTRAINT valid_recipient_type CHECK (recipient_type IN ('user', 'role', 'email', 'webhook')),
    CONSTRAINT valid_notification_status CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'cancelled'))
);

-- Parallel workflow branches
CREATE TABLE workflow_branches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_entity_id UUID NOT NULL REFERENCES workflow_entities(id),
    branch_name VARCHAR(100) NOT NULL,
    parent_branch_id UUID REFERENCES workflow_branches(id),
    current_state_id UUID NOT NULL REFERENCES workflow_states(id),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    
    UNIQUE(workflow_entity_id, branch_name)
);
```

### State Machine Functions

```sql
-- Function to transition entity to new state
CREATE OR REPLACE FUNCTION transition_workflow_state(
    p_entity_id UUID,
    p_transition_name VARCHAR,
    p_performed_by UUID,
    p_comment TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_workflow_entity workflow_entities%ROWTYPE;
    v_transition workflow_transitions%ROWTYPE;
    v_new_state workflow_states%ROWTYPE;
    v_can_transition BOOLEAN;
BEGIN
    -- Get current workflow entity
    SELECT * INTO v_workflow_entity
    FROM workflow_entities
    WHERE id = p_entity_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Workflow entity not found: %', p_entity_id;
    END IF;
    
    -- Find valid transition
    SELECT t.* INTO v_transition
    FROM workflow_transitions t
    WHERE t.workflow_id = v_workflow_entity.workflow_id
      AND t.from_state_id = v_workflow_entity.current_state_id
      AND t.name = p_transition_name;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Invalid transition: % from current state', p_transition_name;
    END IF;
    
    -- Check transition conditions
    SELECT check_transition_conditions(v_transition.id, p_entity_id, p_metadata) INTO v_can_transition;
    
    IF NOT v_can_transition THEN
        RAISE EXCEPTION 'Transition conditions not met for: %', p_transition_name;
    END IF;
    
    -- Get target state
    SELECT * INTO v_new_state
    FROM workflow_states
    WHERE id = v_transition.to_state_id;
    
    -- Update entity state
    UPDATE workflow_entities
    SET current_state_id = v_new_state.id,
        previous_state_id = v_workflow_entity.current_state_id,
        updated_at = NOW()
    WHERE id = p_entity_id;
    
    -- Record history
    INSERT INTO workflow_history (
        workflow_entity_id,
        from_state_id,
        to_state_id,
        transition_id,
        action_taken,
        comment,
        performed_by,
        metadata
    ) VALUES (
        p_entity_id,
        v_workflow_entity.current_state_id,
        v_new_state.id,
        v_transition.id,
        p_transition_name,
        p_comment,
        p_performed_by,
        p_metadata
    );
    
    -- Handle post-transition actions
    PERFORM handle_post_transition_actions(p_entity_id, v_transition.id);
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to check transition conditions
CREATE OR REPLACE FUNCTION check_transition_conditions(
    p_transition_id UUID,
    p_entity_id UUID,
    p_metadata JSONB
) RETURNS BOOLEAN AS $$
DECLARE
    v_condition workflow_conditions%ROWTYPE;
    v_result BOOLEAN := TRUE;
BEGIN
    FOR v_condition IN
        SELECT * FROM workflow_conditions
        WHERE transition_id = p_transition_id
        ORDER BY id
    LOOP
        CASE v_condition.condition_type
            WHEN 'field_value' THEN
                -- Check metadata field values
                v_result := evaluate_field_condition(v_condition, p_metadata);
            WHEN 'time_based' THEN
                -- Check time-based conditions
                v_result := evaluate_time_condition(v_condition, p_entity_id);
            WHEN 'user_role' THEN
                -- Check user permissions
                v_result := evaluate_user_condition(v_condition, p_metadata);
            ELSE
                v_result := TRUE;
        END CASE;
        
        IF NOT v_result AND v_condition.is_required THEN
            RETURN FALSE;
        END IF;
    END LOOP;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to get available transitions for an entity
CREATE OR REPLACE FUNCTION get_available_transitions(p_entity_id UUID)
RETURNS TABLE(
    transition_id UUID,
    transition_name VARCHAR,
    target_state VARCHAR,
    description TEXT,
    requires_approval BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.name,
        s.name,
        t.description,
        t.requires_approval
    FROM workflow_entities we
    JOIN workflow_transitions t ON t.workflow_id = we.workflow_id 
        AND t.from_state_id = we.current_state_id
    JOIN workflow_states s ON s.id = t.to_state_id
    WHERE we.id = p_entity_id
      AND check_transition_conditions(t.id, p_entity_id, NULL)
    ORDER BY t.name;
END;
$$ LANGUAGE plpgsql;
```

## Real-World Examples

### 1. Job Application Workflow

```sql
-- Create a job application workflow
INSERT INTO workflow_definitions (name, description, created_by) VALUES
('job_application_process', 'Standard job application and hiring process', 'system');

-- Define states for job application
INSERT INTO workflow_states (workflow_id, name, display_name, description, is_initial, is_terminal, color_code) VALUES
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'submitted', 'Application Submitted', 'Initial application received', true, false, '#007bff'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'screening', 'Initial Screening', 'HR initial review', false, false, '#ffc107'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'interview_scheduled', 'Interview Scheduled', 'Interview arranged', false, false, '#17a2b8'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'interviewed', 'Interviewed', 'Interview completed', false, false, '#6f42c1'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'reference_check', 'Reference Check', 'Checking references', false, false, '#fd7e14'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'offer_made', 'Offer Made', 'Job offer extended', false, false, '#20c997'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'hired', 'Hired', 'Successfully hired', false, true, '#28a745'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'), 'rejected', 'Rejected', 'Application rejected', false, true, '#dc3545');

-- Define transitions
INSERT INTO workflow_transitions (workflow_id, from_state_id, to_state_id, name, description) VALUES
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'submitted'),
 (SELECT id FROM workflow_states WHERE name = 'screening'),
 'start_screening', 'Begin initial screening process'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'screening'),
 (SELECT id FROM workflow_states WHERE name = 'interview_scheduled'),
 'schedule_interview', 'Schedule candidate for interview'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'screening'),
 (SELECT id FROM workflow_states WHERE name = 'rejected'),
 'reject_screening', 'Reject after initial screening'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'interview_scheduled'),
 (SELECT id FROM workflow_states WHERE name = 'interviewed'),
 'complete_interview', 'Mark interview as completed'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'interviewed'),
 (SELECT id FROM workflow_states WHERE name = 'reference_check'),
 'check_references', 'Begin reference verification'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'interviewed'),
 (SELECT id FROM workflow_states WHERE name = 'rejected'),
 'reject_interview', 'Reject after interview'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'reference_check'),
 (SELECT id FROM workflow_states WHERE name = 'offer_made'),
 'make_offer', 'Extend job offer'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'offer_made'),
 (SELECT id FROM workflow_states WHERE name = 'hired'),
 'accept_offer', 'Candidate accepts offer'),
((SELECT id FROM workflow_definitions WHERE name = 'job_application_process'),
 (SELECT id FROM workflow_states WHERE name = 'offer_made'),
 (SELECT id FROM workflow_states WHERE name = 'rejected'),
 'decline_offer', 'Candidate declines offer');
```

### 2. Order Fulfillment Workflow

```sql
-- E-commerce order processing workflow
INSERT INTO workflow_definitions (name, description, created_by) VALUES
('order_fulfillment', 'E-commerce order processing and fulfillment', 'system');

-- Order states
INSERT INTO workflow_states (workflow_id, name, display_name, is_initial, is_terminal, timeout_minutes) VALUES
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'pending', 'Pending Payment', true, false, 1440), -- 24 hours
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'paid', 'Payment Confirmed', false, false, NULL),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'processing', 'Processing', false, false, 4320), -- 3 days
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'shipped', 'Shipped', false, false, NULL),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'delivered', 'Delivered', false, true, NULL),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'cancelled', 'Cancelled', false, true, NULL),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'), 'refunded', 'Refunded', false, true, NULL);

-- Order transitions with conditions
INSERT INTO workflow_transitions (workflow_id, from_state_id, to_state_id, name, auto_transition, requires_approval) VALUES
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'),
 (SELECT id FROM workflow_states WHERE name = 'pending'),
 (SELECT id FROM workflow_states WHERE name = 'paid'),
 'confirm_payment', true, false),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'),
 (SELECT id FROM workflow_states WHERE name = 'paid'),
 (SELECT id FROM workflow_states WHERE name = 'processing'),
 'start_processing', true, false),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'),
 (SELECT id FROM workflow_states WHERE name = 'processing'),
 (SELECT id FROM workflow_states WHERE name = 'shipped'),
 'ship_order', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'),
 (SELECT id FROM workflow_states WHERE name = 'shipped'),
 (SELECT id FROM workflow_states WHERE name = 'delivered'),
 'confirm_delivery', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'),
 (SELECT id FROM workflow_states WHERE name = 'pending'),
 (SELECT id FROM workflow_states WHERE name = 'cancelled'),
 'cancel_unpaid', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'order_fulfillment'),
 (SELECT id FROM workflow_states WHERE name = 'delivered'),
 (SELECT id FROM workflow_states WHERE name = 'refunded'),
 'process_refund', false, true);
```

### 3. Document Approval Workflow

```sql
-- Document review and approval workflow
INSERT INTO workflow_definitions (name, description, created_by) VALUES
('document_approval', 'Multi-level document approval process', 'system');

-- Document approval states
INSERT INTO workflow_states (workflow_id, name, display_name, is_initial, is_terminal) VALUES
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'draft', 'Draft', true, false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'submitted', 'Submitted for Review', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'manager_review', 'Manager Review', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'director_review', 'Director Review', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'legal_review', 'Legal Review', false, false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'approved', 'Approved', false, true),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'rejected', 'Rejected', false, true),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'), 'revision_required', 'Revision Required', false, false);

-- Approval transitions
INSERT INTO workflow_transitions (workflow_id, from_state_id, to_state_id, name, requires_approval) VALUES
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'draft'),
 (SELECT id FROM workflow_states WHERE name = 'submitted'),
 'submit_for_review', false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'submitted'),
 (SELECT id FROM workflow_states WHERE name = 'manager_review'),
 'assign_manager', false),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'manager_review'),
 (SELECT id FROM workflow_states WHERE name = 'director_review'),
 'escalate_director', true),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'director_review'),
 (SELECT id FROM workflow_states WHERE name = 'legal_review'),
 'legal_review_required', true),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'legal_review'),
 (SELECT id FROM workflow_states WHERE name = 'approved'),
 'final_approval', true),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'manager_review'),
 (SELECT id FROM workflow_states WHERE name = 'revision_required'),
 'request_revision', true),
((SELECT id FROM workflow_definitions WHERE name = 'document_approval'),
 (SELECT id FROM workflow_states WHERE name = 'revision_required'),
 (SELECT id FROM workflow_states WHERE name = 'submitted'),
 'resubmit_after_revision', false);
```

## State Machine Implementation

### Workflow Execution Engine

```sql
-- Trigger to handle automatic transitions
CREATE OR REPLACE FUNCTION handle_workflow_auto_transitions()
RETURNS TRIGGER AS $$
DECLARE
    v_auto_transition workflow_transitions%ROWTYPE;
BEGIN
    -- Check for auto transitions from new state
    FOR v_auto_transition IN
        SELECT t.*
        FROM workflow_transitions t
        WHERE t.workflow_id = NEW.workflow_id
          AND t.from_state_id = NEW.current_state_id
          AND t.auto_transition = true
    LOOP
        -- Execute auto transition
        PERFORM transition_workflow_state(
            NEW.id,
            v_auto_transition.name,
            'system', -- system user for auto transitions
            'Automatic transition',
            NULL
        );
    END LOOP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workflow_auto_transition_trigger
    AFTER UPDATE ON workflow_entities
    FOR EACH ROW
    WHEN (OLD.current_state_id IS DISTINCT FROM NEW.current_state_id)
    EXECUTE FUNCTION handle_workflow_auto_transitions();

-- Function to handle timeout-based transitions
CREATE OR REPLACE FUNCTION process_workflow_timeouts()
RETURNS INTEGER AS $$
DECLARE
    v_entity workflow_entities%ROWTYPE;
    v_state workflow_states%ROWTYPE;
    v_timeout_count INTEGER := 0;
BEGIN
    FOR v_entity IN
        SELECT we.*
        FROM workflow_entities we
        JOIN workflow_states ws ON ws.id = we.current_state_id
        WHERE ws.timeout_minutes IS NOT NULL
          AND we.updated_at < NOW() - INTERVAL '1 minute' * ws.timeout_minutes
          AND NOT EXISTS (
              SELECT 1 FROM workflow_states 
              WHERE id = we.current_state_id AND is_terminal = true
          )
    LOOP
        SELECT * INTO v_state FROM workflow_states WHERE id = v_entity.current_state_id;
        
        -- Handle timeout based on state type
        CASE v_state.name
            WHEN 'pending' THEN
                PERFORM transition_workflow_state(
                    v_entity.id,
                    'timeout_cancel',
                    'system',
                    'Timeout: Payment not received'
                );
            WHEN 'processing' THEN
                -- Send notification but don't auto-transition
                INSERT INTO workflow_notifications (
                    workflow_entity_id,
                    notification_type,
                    recipient_type,
                    recipient_value,
                    subject,
                    message
                ) VALUES (
                    v_entity.id,
                    'timeout_warning',
                    'user',
                    v_entity.assigned_to::text,
                    'Order Processing Timeout',
                    'Order has been in processing state for too long'
                );
            ELSE
                NULL; -- No action for other states
        END CASE;
        
        v_timeout_count := v_timeout_count + 1;
    END LOOP;
    
    RETURN v_timeout_count;
END;
$$ LANGUAGE plpgsql;

-- Schedule timeout processing (pseudo-code for cron job)
-- SELECT cron.schedule('process_workflow_timeouts', '*/15 * * * *', 'SELECT process_workflow_timeouts();');
```

## Workflow Analytics

### Comprehensive Reporting and Metrics

```sql
-- View for workflow performance analytics
CREATE VIEW workflow_analytics AS
SELECT 
    wd.name as workflow_name,
    ws.name as state_name,
    ws.display_name,
    COUNT(we.id) as current_entities,
    AVG(EXTRACT(EPOCH FROM (NOW() - we.updated_at))/3600) as avg_hours_in_state,
    COUNT(wh.id) as total_transitions,
    COUNT(CASE WHEN wh.performed_at > NOW() - INTERVAL '24 hours' THEN 1 END) as transitions_24h,
    COUNT(CASE WHEN wh.performed_at > NOW() - INTERVAL '7 days' THEN 1 END) as transitions_7d
FROM workflow_definitions wd
JOIN workflow_states ws ON ws.workflow_id = wd.id
LEFT JOIN workflow_entities we ON we.workflow_id = wd.id AND we.current_state_id = ws.id
LEFT JOIN workflow_history wh ON wh.to_state_id = ws.id
WHERE wd.is_active = true
GROUP BY wd.id, wd.name, ws.id, ws.name, ws.display_name
ORDER BY wd.name, ws.name;

-- Function to get workflow bottlenecks
CREATE OR REPLACE FUNCTION get_workflow_bottlenecks(p_workflow_id UUID, p_days INTEGER DEFAULT 30)
RETURNS TABLE(
    state_name VARCHAR,
    avg_duration_hours NUMERIC,
    entity_count BIGINT,
    bottleneck_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH state_durations AS (
        SELECT 
            ws.name,
            AVG(EXTRACT(EPOCH FROM (
                COALESCE(wh2.performed_at, NOW()) - wh1.performed_at
            ))/3600) as avg_hours,
            COUNT(*) as transitions
        FROM workflow_history wh1
        JOIN workflow_states ws ON ws.id = wh1.to_state_id
        LEFT JOIN workflow_history wh2 ON wh2.workflow_entity_id = wh1.workflow_entity_id
            AND wh2.performed_at > wh1.performed_at
            AND wh2.id = (
                SELECT MIN(id) FROM workflow_history 
                WHERE workflow_entity_id = wh1.workflow_entity_id 
                AND performed_at > wh1.performed_at
            )
        WHERE ws.workflow_id = p_workflow_id
          AND wh1.performed_at > NOW() - INTERVAL '1 day' * p_days
        GROUP BY ws.id, ws.name
    )
    SELECT 
        sd.name,
        ROUND(sd.avg_hours, 2),
        sd.transitions,
        ROUND(sd.avg_hours * sd.transitions / 100.0, 2) -- Simple bottleneck score
    FROM state_durations sd
    ORDER BY (sd.avg_hours * sd.transitions) DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to get workflow completion rate
CREATE OR REPLACE FUNCTION get_workflow_completion_rate(p_workflow_id UUID, p_days INTEGER DEFAULT 30)
RETURNS TABLE(
    total_started BIGINT,
    total_completed BIGINT,
    completion_rate NUMERIC,
    avg_completion_time_hours NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH workflow_stats AS (
        SELECT 
            COUNT(*) as started,
            COUNT(CASE WHEN ws.is_terminal = true THEN 1 END) as completed,
            AVG(CASE 
                WHEN ws.is_terminal = true THEN 
                    EXTRACT(EPOCH FROM (we.updated_at - we.created_at))/3600 
            END) as avg_hours
        FROM workflow_entities we
        JOIN workflow_states ws ON ws.id = we.current_state_id
        WHERE we.workflow_id = p_workflow_id
          AND we.created_at > NOW() - INTERVAL '1 day' * p_days
    )
    SELECT 
        started,
        completed,
        CASE WHEN started > 0 THEN ROUND(completed * 100.0 / started, 2) ELSE 0 END,
        ROUND(avg_hours, 2)
    FROM workflow_stats;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Design Principles

```sql
-- Example of a well-designed workflow with proper constraints
CREATE TABLE workflow_validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id),
    rule_type VARCHAR(50) NOT NULL,
    rule_expression TEXT NOT NULL,
    error_message TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    CONSTRAINT valid_rule_type CHECK (rule_type IN (
        'state_constraint', 'transition_guard', 'completion_rule', 'timeout_rule'
    ))
);

-- Function to validate workflow integrity
CREATE OR REPLACE FUNCTION validate_workflow_integrity(p_workflow_id UUID)
RETURNS TABLE(
    issue_type VARCHAR,
    issue_description TEXT,
    severity VARCHAR
) AS $$
BEGIN
    -- Check for orphaned states
    RETURN QUERY
    SELECT 
        'orphaned_state'::VARCHAR,
        'State "' || ws.name || '" has no incoming transitions'::TEXT,
        'warning'::VARCHAR
    FROM workflow_states ws
    WHERE ws.workflow_id = p_workflow_id
      AND NOT ws.is_initial
      AND NOT EXISTS (
          SELECT 1 FROM workflow_transitions 
          WHERE workflow_id = p_workflow_id AND to_state_id = ws.id
      );
    
    -- Check for unreachable terminal states
    RETURN QUERY
    SELECT 
        'unreachable_terminal'::VARCHAR,
        'Terminal state "' || ws.name || '" is not reachable'::TEXT,
        'error'::VARCHAR
    FROM workflow_states ws
    WHERE ws.workflow_id = p_workflow_id
      AND ws.is_terminal
      AND NOT EXISTS (
          SELECT 1 FROM workflow_transitions 
          WHERE workflow_id = p_workflow_id AND to_state_id = ws.id
      );
    
    -- Check for missing initial state
    IF NOT EXISTS (
        SELECT 1 FROM workflow_states 
        WHERE workflow_id = p_workflow_id AND is_initial = true
    ) THEN
        RETURN QUERY
        SELECT 
            'no_initial_state'::VARCHAR,
            'Workflow has no initial state'::TEXT,
            'error'::VARCHAR;
    END IF;
END;
$$ LANGUAGE plpgsql;
```

### 2. Security and Permissions

```sql
-- Role-based workflow permissions
CREATE TABLE workflow_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id),
    role_name VARCHAR(100) NOT NULL,
    permission_type VARCHAR(50) NOT NULL,
    state_id UUID REFERENCES workflow_states(id),
    transition_id UUID REFERENCES workflow_transitions(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID NOT NULL,
    
    CONSTRAINT valid_permission_type CHECK (permission_type IN (
        'view', 'edit', 'transition', 'assign', 'approve', 'admin'
    ))
);

-- Function to check workflow permissions
CREATE OR REPLACE FUNCTION check_workflow_permission(
    p_user_id UUID,
    p_workflow_entity_id UUID,
    p_permission_type VARCHAR,
    p_transition_id UUID DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_has_permission BOOLEAN := FALSE;
BEGIN
    -- Implementation would check user roles against workflow_permissions
    -- This is a simplified version
    SELECT EXISTS (
        SELECT 1 
        FROM workflow_permissions wp
        JOIN workflow_entities we ON we.workflow_id = wp.workflow_id
        WHERE we.id = p_workflow_entity_id
          AND wp.permission_type = p_permission_type
          AND (p_transition_id IS NULL OR wp.transition_id = p_transition_id)
          -- Add user role checking logic here
    ) INTO v_has_permission;
    
    RETURN v_has_permission;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Indexing Strategy

```sql
-- Essential indexes for workflow performance
CREATE INDEX idx_workflow_entities_current_state ON workflow_entities(current_state_id);
CREATE INDEX idx_workflow_entities_workflow_state ON workflow_entities(workflow_id, current_state_id);
CREATE INDEX idx_workflow_entities_assigned ON workflow_entities(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_workflow_entities_due_date ON workflow_entities(due_date) WHERE due_date IS NOT NULL;

CREATE INDEX idx_workflow_history_entity_time ON workflow_history(workflow_entity_id, performed_at);
CREATE INDEX idx_workflow_history_state_time ON workflow_history(to_state_id, performed_at);

CREATE INDEX idx_workflow_tasks_assigned ON workflow_tasks(assigned_to, status) WHERE status != 'completed';
CREATE INDEX idx_workflow_tasks_due ON workflow_tasks(due_date) WHERE due_date IS NOT NULL;

-- Partial indexes for active workflows
CREATE INDEX idx_workflow_entities_active ON workflow_entities(workflow_id, current_state_id) 
WHERE current_state_id NOT IN (
    SELECT id FROM workflow_states WHERE is_terminal = true
);
```

### Archiving Strategy

```sql
-- Archive completed workflows
CREATE TABLE workflow_entities_archive (
    LIKE workflow_entities INCLUDING ALL
);

CREATE TABLE workflow_history_archive (
    LIKE workflow_history INCLUDING ALL
);

-- Function to archive completed workflows
CREATE OR REPLACE FUNCTION archive_completed_workflows(p_days_old INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    v_archived_count INTEGER := 0;
BEGIN
    -- Move completed entities to archive
    WITH archived_entities AS (
        DELETE FROM workflow_entities we
        WHERE EXISTS (
            SELECT 1 FROM workflow_states ws 
            WHERE ws.id = we.current_state_id 
            AND ws.is_terminal = true
        )
        AND we.updated_at < NOW() - INTERVAL '1 day' * p_days_old
        RETURNING *
    )
    INSERT INTO workflow_entities_archive 
    SELECT * FROM archived_entities;
    
    GET DIAGNOSTICS v_archived_count = ROW_COUNT;
    
    -- Move related history to archive
    WITH archived_history AS (
        DELETE FROM workflow_history wh
        WHERE NOT EXISTS (
            SELECT 1 FROM workflow_entities we 
            WHERE we.id = wh.workflow_entity_id
        )
        RETURNING *
    )
    INSERT INTO workflow_history_archive 
    SELECT * FROM archived_history;
    
    RETURN v_archived_count;
END;
$$ LANGUAGE plpgsql;
```

## Migration Strategies

### Schema Evolution

```sql
-- Example migration for adding new workflow features
-- Migration: Add priority levels to workflows
ALTER TABLE workflow_entities 
ADD COLUMN IF NOT EXISTS priority_level VARCHAR(20) DEFAULT 'normal';

ALTER TABLE workflow_entities 
ADD CONSTRAINT valid_priority_level 
CHECK (priority_level IN ('low', 'normal', 'high', 'urgent'));

-- Migration: Add workflow versioning
ALTER TABLE workflow_definitions 
ADD COLUMN IF NOT EXISTS parent_version_id UUID REFERENCES workflow_definitions(id);

-- Migration: Add SLA tracking
CREATE TABLE IF NOT EXISTS workflow_slas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id),
    state_id UUID NOT NULL REFERENCES workflow_states(id),
    max_duration_hours INTEGER NOT NULL,
    escalation_rules JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(workflow_id, state_id)
);

-- Data migration function
CREATE OR REPLACE FUNCTION migrate_workflow_data()
RETURNS VOID AS $$
BEGIN
    -- Update existing records with default values
    UPDATE workflow_entities 
    SET priority_level = 'normal' 
    WHERE priority_level IS NULL;
    
    -- Create default SLAs for existing workflows
    INSERT INTO workflow_slas (workflow_id, state_id, max_duration_hours)
    SELECT DISTINCT 
        ws.workflow_id,
        ws.id,
        CASE 
            WHEN ws.timeout_minutes IS NOT NULL THEN ws.timeout_minutes / 60
            ELSE 24 -- Default 24 hours
        END
    FROM workflow_states ws
    WHERE NOT EXISTS (
        SELECT 1 FROM workflow_slas sla 
        WHERE sla.workflow_id = ws.workflow_id 
        AND sla.state_id = ws.id
    );
END;
$$ LANGUAGE plpgsql;
```

---

**References:**
- [Workflow Pattern Part 1](https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-1-using-workflow-patterns-to-manage-the-state-of-any-entity)
- [Workflow Pattern Part 2](https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-2-using-configuration-tables-to-define-the-actual-workflow)
- [State Machine Design Patterns](https://en.wikipedia.org/wiki/State_pattern)
- [Business Process Management](https://en.wikipedia.org/wiki/Business_process_management)
```


## Modelling it in Golang

```go
package main

import (
	"fmt"
)

type Status string

type Transition func() Status

type Statuses map[Status][]Status

func main() {
	var statuses = Statuses{
		ApplicationReceived: []Status{ApplicationReview, ApplicationClosed},
		ApplicationReview:   []Status{InvitedToInterview, ApplicationClosed},
		InvitedToInterview:  []Status{Interview, ApplicationClosed},
		Interview:           []Status{MakeOffer, SeekReferences, ApplicationClosed, InvitedToInterview},
		MakeOffer:           []Status{SeekReferences, ApplicationClosed},
		SeekReferences:      []Status{Hired, ApplicationClosed},
		Hired:               []Status{},
		ApplicationClosed:   []Status{},
	}
}
```

REFERENCES: 
- https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-1-using-workflow-patterns-to-manage-the-state-of-any-entity
- https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-2-using-configuration-tables-to-define-the-actual-workflow


