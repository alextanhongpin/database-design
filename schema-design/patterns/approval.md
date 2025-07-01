# Approval System Database Design

## Table of Contents
- [Overview](#overview)
- [Core Approval Model](#core-approval-model)
- [Workflow Management](#workflow-management)
- [Multi-Stage Approvals](#multi-stage-approvals)
- [Conditional Logic](#conditional-logic)
- [Real-World Examples](#real-world-examples)
- [Advanced Patterns](#advanced-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Overview

Approval systems are critical for business processes that require authorization, review, or sign-off before actions can be completed. They provide audit trails, enforce business rules, and ensure compliance with organizational policies.

### Common Use Cases
- **Document Approval**: Contracts, policies, marketing materials
- **Financial Approvals**: Expense reports, purchase orders, budget changes
- **HR Processes**: Leave requests, hiring approvals, performance reviews
- **Content Moderation**: Blog posts, comments, user-generated content
- **System Changes**: Configuration updates, user access requests

### Key Requirements
- **Flexible Workflows**: Support different approval chains
- **Audit Trail**: Complete history of all approval actions
- **Notifications**: Alert approvers and requesters of status changes
- **Delegation**: Allow approvers to delegate authority
- **Escalation**: Automatic escalation for overdue approvals
- **Conditional Logic**: Route based on request attributes

## Core Approval Model

### Basic Approval Tables

```sql
-- Approval workflows (templates for approval processes)
CREATE TABLE approval_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Workflow identification
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Workflow configuration
    workflow_type VARCHAR(100) NOT NULL, -- expense_report, document, etc.
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Workflow settings
    requires_all_approvers BOOLEAN DEFAULT FALSE, -- All vs. any approver
    auto_approve_threshold DECIMAL(19,4), -- Auto-approve below amount
    escalation_hours INTEGER DEFAULT 72, -- Hours before escalation
    
    -- Metadata
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_workflow_type (workflow_type, is_active)
);

-- Approval stages (steps in a workflow)
CREATE TABLE approval_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES approval_workflows(id) ON DELETE CASCADE,
    
    -- Stage details
    stage_name VARCHAR(255) NOT NULL,
    stage_order INTEGER NOT NULL,
    
    -- Stage configuration
    is_required BOOLEAN DEFAULT TRUE,
    can_skip BOOLEAN DEFAULT FALSE,
    
    -- Approval requirements
    min_approvers INTEGER DEFAULT 1,
    requires_all_approvers BOOLEAN DEFAULT FALSE,
    
    -- Conditional logic
    condition_rules JSONB, -- JSON rules for when this stage applies
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (workflow_id, stage_order),
    INDEX idx_workflow_order (workflow_id, stage_order)
);

-- Stage approvers (who can approve at each stage)
CREATE TABLE stage_approvers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id UUID NOT NULL REFERENCES approval_stages(id) ON DELETE CASCADE,
    
    -- Approver identification
    approver_type approver_type NOT NULL,
    user_id UUID REFERENCES users(id),
    role_id UUID REFERENCES roles(id),
    group_id UUID REFERENCES groups(id),
    
    -- Approval authority
    max_amount DECIMAL(19,4), -- Maximum amount this approver can approve
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure only one type of approver is specified
    CHECK (
        (user_id IS NOT NULL AND role_id IS NULL AND group_id IS NULL) OR
        (user_id IS NULL AND role_id IS NOT NULL AND group_id IS NULL) OR
        (user_id IS NULL AND role_id IS NULL AND group_id IS NOT NULL)
    ),
    
    INDEX idx_stage_approver (stage_id, approver_type)
);

CREATE TYPE approver_type AS ENUM ('user', 'role', 'group');

-- Approval requests (instances of workflows)
CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES approval_workflows(id),
    
    -- Request details
    title VARCHAR(500) NOT NULL,
    description TEXT,
    
    -- Request metadata
    request_type VARCHAR(100) NOT NULL,
    reference_id UUID, -- ID of the entity being approved (order, document, etc.)
    reference_data JSONB, -- Additional data for approval decision
    
    -- Financial information (if applicable)
    amount DECIMAL(19,4),
    currency_code CHAR(3),
    
    -- Status and timing
    status approval_status DEFAULT 'pending',
    priority priority_level DEFAULT 'normal',
    
    -- Key dates
    submitted_at TIMESTAMP DEFAULT NOW(),
    due_date TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- People involved
    requested_by UUID NOT NULL REFERENCES users(id),
    assigned_to UUID REFERENCES users(id), -- Current approver
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_status_priority (status, priority),
    INDEX idx_requested_by (requested_by, status),
    INDEX idx_assigned_to (assigned_to, status),
    INDEX idx_due_date (due_date, status)
);

CREATE TYPE approval_status AS ENUM (
    'draft', 'pending', 'in_progress', 'approved', 
    'rejected', 'cancelled', 'expired'
);

CREATE TYPE priority_level AS ENUM ('low', 'normal', 'high', 'urgent');

-- Individual approval actions
CREATE TABLE approval_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    stage_id UUID NOT NULL REFERENCES approval_stages(id),
    
    -- Action details
    action_type action_type NOT NULL,
    comment TEXT,
    
    -- Approver information
    approver_id UUID NOT NULL REFERENCES users(id),
    approver_role VARCHAR(100), -- Role they used to approve
    
    -- Timing
    action_date TIMESTAMP DEFAULT NOW(),
    
    -- Delegation (if approved on behalf of someone else)
    delegated_by UUID REFERENCES users(id),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    INDEX idx_request_stage (request_id, stage_id),
    INDEX idx_action_date (action_date),
    INDEX idx_approver (approver_id, action_date)
);

CREATE TYPE action_type AS ENUM (
    'submitted', 'approved', 'rejected', 'returned', 
    'escalated', 'delegated', 'cancelled', 'commented'
);
```

## Workflow Management

### Dynamic Workflow Routing

```sql
-- Function to start an approval workflow
CREATE OR REPLACE FUNCTION start_approval_workflow(
    workflow_name VARCHAR(255),
    request_title VARCHAR(500),
    request_description TEXT,
    reference_id UUID,
    reference_data JSONB,
    requested_by UUID,
    amount DECIMAL(19,4) DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    workflow_record RECORD;
    request_id UUID;
    first_stage_id UUID;
    next_approver UUID;
BEGIN
    -- Get workflow
    SELECT * INTO workflow_record
    FROM approval_workflows
    WHERE name = workflow_name AND is_active = TRUE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Workflow not found: %', workflow_name;
    END IF;
    
    -- Check auto-approval threshold
    IF amount IS NOT NULL AND workflow_record.auto_approve_threshold IS NOT NULL 
       AND amount <= workflow_record.auto_approve_threshold THEN
        -- Create auto-approved request
        INSERT INTO approval_requests (
            workflow_id, title, description, reference_id, reference_data,
            amount, status, requested_by, completed_at
        ) VALUES (
            workflow_record.id, request_title, request_description, 
            reference_id, reference_data, amount, 'approved', 
            requested_by, NOW()
        ) RETURNING id INTO request_id;
        
        -- Log auto-approval action
        INSERT INTO approval_actions (
            request_id, stage_id, action_type, approver_id, comment
        ) VALUES (
            request_id, 
            (SELECT id FROM approval_stages WHERE workflow_id = workflow_record.id ORDER BY stage_order LIMIT 1),
            'approved', requested_by, 'Auto-approved based on amount threshold'
        );
        
        RETURN request_id;
    END IF;
    
    -- Create pending request
    INSERT INTO approval_requests (
        workflow_id, title, description, reference_id, reference_data,
        amount, status, requested_by,
        due_date
    ) VALUES (
        workflow_record.id, request_title, request_description, 
        reference_id, reference_data, amount, 'pending', requested_by,
        NOW() + INTERVAL '%s hours' || workflow_record.escalation_hours
    ) RETURNING id INTO request_id;
    
    -- Get first stage
    SELECT id INTO first_stage_id
    FROM approval_stages
    WHERE workflow_id = workflow_record.id
    ORDER BY stage_order
    LIMIT 1;
    
    -- Find next approver
    SELECT next_approver INTO next_approver
    FROM find_next_approver(request_id, first_stage_id);
    
    -- Update request with assigned approver
    UPDATE approval_requests
    SET assigned_to = next_approver,
        status = 'in_progress'
    WHERE id = request_id;
    
    -- Log submission
    INSERT INTO approval_actions (
        request_id, stage_id, action_type, approver_id
    ) VALUES (
        request_id, first_stage_id, 'submitted', requested_by
    );
    
    RETURN request_id;
END;
$$ LANGUAGE plpgsql;

-- Function to find next approver
CREATE OR REPLACE FUNCTION find_next_approver(
    request_id UUID,
    stage_id UUID
) RETURNS TABLE(next_approver UUID) AS $$
DECLARE
    request_record RECORD;
BEGIN
    -- Get request details
    SELECT * INTO request_record
    FROM approval_requests
    WHERE id = request_id;
    
    RETURN QUERY
    WITH eligible_approvers AS (
        SELECT DISTINCT
            CASE sa.approver_type
                WHEN 'user' THEN sa.user_id
                WHEN 'role' THEN (
                    SELECT ur.user_id 
                    FROM user_roles ur 
                    WHERE ur.role_id = sa.role_id 
                    LIMIT 1
                )
                WHEN 'group' THEN (
                    SELECT ug.user_id 
                    FROM user_groups ug 
                    WHERE ug.group_id = sa.group_id 
                    LIMIT 1
                )
            END as approver_id
        FROM stage_approvers sa
        WHERE sa.stage_id = find_next_approver.stage_id
          AND sa.is_active = TRUE
          AND (sa.max_amount IS NULL OR sa.max_amount >= COALESCE(request_record.amount, 0))
    )
    SELECT ea.approver_id
    FROM eligible_approvers ea
    WHERE ea.approver_id IS NOT NULL
      AND ea.approver_id != request_record.requested_by -- Don't assign to self
    ORDER BY RANDOM() -- Simple assignment logic
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to process approval action
CREATE OR REPLACE FUNCTION process_approval_action(
    request_id UUID,
    approver_id UUID,
    action action_type,
    comment TEXT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    request_record RECORD;
    current_stage RECORD;
    next_stage_id UUID;
    next_approver UUID;
    approvals_needed INTEGER;
    approvals_received INTEGER;
BEGIN
    -- Get request details
    SELECT ar.*, ast.* INTO request_record
    FROM approval_requests ar
    JOIN approval_stages ast ON ast.workflow_id = ar.workflow_id
    WHERE ar.id = request_id
      AND ar.status = 'in_progress'
      AND ar.assigned_to = approver_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Invalid approval request or unauthorized approver';
    END IF;
    
    -- Record the action
    INSERT INTO approval_actions (
        request_id, stage_id, action_type, approver_id, comment
    ) VALUES (
        request_id, request_record.id, action, approver_id, comment
    );
    
    IF action = 'rejected' THEN
        -- Reject the entire request
        UPDATE approval_requests
        SET status = 'rejected',
            completed_at = NOW(),
            assigned_to = NULL
        WHERE id = request_id;
        
        RETURN TRUE;
    END IF;
    
    IF action = 'approved' THEN
        -- Check if we need more approvals for current stage
        SELECT min_approvers INTO approvals_needed
        FROM approval_stages
        WHERE id = request_record.id;
        
        SELECT COUNT(*) INTO approvals_received
        FROM approval_actions
        WHERE request_id = request_id
          AND stage_id = request_record.id
          AND action_type = 'approved';
        
        IF approvals_received >= approvals_needed THEN
            -- Move to next stage
            SELECT id INTO next_stage_id
            FROM approval_stages
            WHERE workflow_id = request_record.workflow_id
              AND stage_order > request_record.stage_order
            ORDER BY stage_order
            LIMIT 1;
            
            IF next_stage_id IS NULL THEN
                -- No more stages - approve the request
                UPDATE approval_requests
                SET status = 'approved',
                    completed_at = NOW(),
                    assigned_to = NULL
                WHERE id = request_id;
            ELSE
                -- Assign to next stage
                SELECT next_approver INTO next_approver
                FROM find_next_approver(request_id, next_stage_id);
                
                UPDATE approval_requests
                SET assigned_to = next_approver
                WHERE id = request_id;
            END IF;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## Multi-Stage Approvals

### Complex Approval Chains

```sql
-- Parallel approval stages (multiple approvers at same stage)
CREATE TABLE parallel_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES approval_requests(id),
    stage_id UUID NOT NULL REFERENCES approval_stages(id),
    
    -- Parallel approval tracking
    required_approvals INTEGER NOT NULL,
    received_approvals INTEGER DEFAULT 0,
    
    -- Status
    status parallel_status DEFAULT 'pending',
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    UNIQUE (request_id, stage_id)
);

CREATE TYPE parallel_status AS ENUM ('pending', 'in_progress', 'completed', 'failed');

-- Individual parallel approvers
CREATE TABLE parallel_approvers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parallel_approval_id UUID NOT NULL REFERENCES parallel_approvals(id),
    approver_id UUID NOT NULL REFERENCES users(id),
    
    -- Approval status
    status approver_status DEFAULT 'pending',
    approved_at TIMESTAMP,
    comment TEXT,
    
    -- Delegation
    delegated_to UUID REFERENCES users(id),
    delegated_at TIMESTAMP,
    
    INDEX idx_parallel_approver (parallel_approval_id, approver_id)
);

CREATE TYPE approver_status AS ENUM ('pending', 'approved', 'rejected', 'delegated');

-- Escalation rules
CREATE TABLE escalation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES approval_workflows(id),
    
    -- Escalation conditions
    trigger_hours INTEGER NOT NULL,
    stage_id UUID REFERENCES approval_stages(id), -- NULL for all stages
    
    -- Escalation actions
    escalation_type escalation_type NOT NULL,
    escalate_to UUID REFERENCES users(id),
    escalate_to_role UUID REFERENCES roles(id),
    
    -- Notification settings
    send_notification BOOLEAN DEFAULT TRUE,
    notification_template VARCHAR(255),
    
    is_active BOOLEAN DEFAULT TRUE,
    
    INDEX idx_workflow_trigger (workflow_id, trigger_hours)
);

CREATE TYPE escalation_type AS ENUM ('reassign', 'notify', 'auto_approve', 'add_approver');

-- Automated escalation function
CREATE OR REPLACE FUNCTION process_escalations() RETURNS INTEGER AS $$
DECLARE
    escalated_count INTEGER := 0;
    request_record RECORD;
    escalation_record RECORD;
BEGIN
    -- Find overdue requests
    FOR request_record IN
        SELECT ar.*, aw.escalation_hours
        FROM approval_requests ar
        JOIN approval_workflows aw ON aw.id = ar.workflow_id
        WHERE ar.status = 'in_progress'
          AND ar.due_date < NOW()
          AND NOT EXISTS (
              SELECT 1 FROM approval_actions aa 
              WHERE aa.request_id = ar.id 
                AND aa.action_type = 'escalated'
                AND aa.action_date > (NOW() - INTERVAL '1 hour')
          )
    LOOP
        -- Find applicable escalation rules
        FOR escalation_record IN
            SELECT er.*
            FROM escalation_rules er
            WHERE er.workflow_id = request_record.workflow_id
              AND er.is_active = TRUE
              AND (er.stage_id IS NULL OR er.stage_id = (
                  SELECT stage_id FROM approval_actions 
                  WHERE request_id = request_record.id 
                  ORDER BY action_date DESC LIMIT 1
              ))
            ORDER BY er.trigger_hours
        LOOP
            -- Process escalation based on type
            CASE escalation_record.escalation_type
                WHEN 'reassign' THEN
                    UPDATE approval_requests
                    SET assigned_to = escalation_record.escalate_to,
                        due_date = NOW() + INTERVAL '24 hours'
                    WHERE id = request_record.id;
                    
                WHEN 'auto_approve' THEN
                    UPDATE approval_requests
                    SET status = 'approved',
                        completed_at = NOW(),
                        assigned_to = NULL
                    WHERE id = request_record.id;
                    
                -- Add other escalation types as needed
                ELSE
                    CONTINUE;
            END CASE;
            
            -- Log escalation action
            INSERT INTO approval_actions (
                request_id, stage_id, action_type, approver_id, comment
            ) VALUES (
                request_record.id,
                (SELECT id FROM approval_stages WHERE workflow_id = request_record.workflow_id ORDER BY stage_order LIMIT 1),
                'escalated',
                COALESCE(escalation_record.escalate_to, request_record.requested_by),
                'Automatically escalated due to timeout'
            );
            
            escalated_count := escalated_count + 1;
            EXIT; -- Only apply first matching rule
        END LOOP;
    END LOOP;
    
    RETURN escalated_count;
END;
$$ LANGUAGE plpgsql;
```

## Real-World Examples

### Expense Report Approval

```sql
-- Expense report specific tables
CREATE TABLE expense_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES users(id),
    
    -- Report details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    total_amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Dates
    report_date DATE NOT NULL,
    submitted_at TIMESTAMP,
    
    -- Approval
    approval_request_id UUID REFERENCES approval_requests(id),
    approval_status approval_status DEFAULT 'draft',
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Expense line items
CREATE TABLE expense_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_report_id UUID NOT NULL REFERENCES expense_reports(id),
    
    -- Item details
    description VARCHAR(500) NOT NULL,
    category VARCHAR(100) NOT NULL,
    amount DECIMAL(19,4) NOT NULL,
    expense_date DATE NOT NULL,
    
    -- Receipts and documentation
    receipt_url VARCHAR(500),
    merchant_name VARCHAR(255),
    
    -- Approval flags
    requires_receipt BOOLEAN DEFAULT TRUE,
    is_personal BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Function to submit expense report for approval
CREATE OR REPLACE FUNCTION submit_expense_report(
    expense_report_id UUID
) RETURNS UUID AS $$
DECLARE
    report_record RECORD;
    approval_request_id UUID;
BEGIN
    -- Get expense report
    SELECT er.*, u.manager_id
    INTO report_record
    FROM expense_reports er
    JOIN users u ON u.id = er.employee_id
    WHERE er.id = expense_report_id;
    
    -- Determine workflow based on amount
    DECLARE
        workflow_name VARCHAR(255);
    BEGIN
        IF report_record.total_amount > 5000 THEN
            workflow_name := 'expense_report_executive';
        ELSIF report_record.total_amount > 1000 THEN
            workflow_name := 'expense_report_manager';
        ELSE
            workflow_name := 'expense_report_standard';
        END IF;
    END;
    
    -- Start approval workflow
    SELECT start_approval_workflow(
        workflow_name,
        'Expense Report: ' || report_record.title,
        report_record.description,
        expense_report_id,
        jsonb_build_object(
            'employee_id', report_record.employee_id,
            'manager_id', report_record.manager_id,
            'total_amount', report_record.total_amount
        ),
        report_record.employee_id,
        report_record.total_amount
    ) INTO approval_request_id;
    
    -- Update expense report
    UPDATE expense_reports
    SET approval_request_id = approval_request_id,
        approval_status = 'pending',
        submitted_at = NOW()
    WHERE id = expense_report_id;
    
    RETURN approval_request_id;
END;
$$ LANGUAGE plpgsql;
```

### Document Approval Workflow

```sql
-- Document management
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Document details
    title VARCHAR(500) NOT NULL,
    content TEXT,
    document_type VARCHAR(100) NOT NULL,
    version INTEGER DEFAULT 1,
    
    -- File information
    file_path VARCHAR(1000),
    file_size BIGINT,
    mime_type VARCHAR(100),
    
    -- Ownership
    created_by UUID NOT NULL REFERENCES users(id),
    department_id UUID REFERENCES departments(id),
    
    -- Approval workflow
    approval_request_id UUID REFERENCES approval_requests(id),
    publication_status publication_status DEFAULT 'draft',
    
    -- Important dates  
    effective_date DATE,
    expiry_date DATE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TYPE publication_status AS ENUM (
    'draft', 'pending_approval', 'approved', 'published', 
    'rejected', 'expired', 'archived'
);

-- Document reviewers (specific people who must review)
CREATE TABLE document_reviewers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id),
    reviewer_id UUID NOT NULL REFERENCES users(id),
    
    -- Review details
    review_type review_type NOT NULL,
    is_required BOOLEAN DEFAULT TRUE,
    
    -- Review status
    status reviewer_status DEFAULT 'pending',
    reviewed_at TIMESTAMP,
    comment TEXT,
    
    UNIQUE (document_id, reviewer_id, review_type)
);

CREATE TYPE review_type AS ENUM ('technical', 'legal', 'compliance', 'editorial');
CREATE TYPE reviewer_status AS ENUM ('pending', 'approved', 'rejected', 'not_required');

-- Multi-stakeholder approval for documents
CREATE OR REPLACE FUNCTION create_document_approval_workflow(
    document_id UUID
) RETURNS UUID AS $$
DECLARE
    doc_record RECORD;
    approval_request_id UUID;
    reviewer_record RECORD;
BEGIN
    -- Get document details
    SELECT * INTO doc_record FROM documents WHERE id = document_id;
    
    -- Create approval request
    SELECT start_approval_workflow(
        'document_approval',
        'Document Approval: ' || doc_record.title,
        'Review and approve document for publication',
        document_id,
        jsonb_build_object(
            'document_type', doc_record.document_type,
            'department_id', doc_record.department_id,
            'effective_date', doc_record.effective_date
        ),
        doc_record.created_by
    ) INTO approval_request_id;
    
    -- Add document-specific reviewers as parallel approvals
    FOR reviewer_record IN
        SELECT dr.*, u.name as reviewer_name
        FROM document_reviewers dr
        JOIN users u ON u.id = dr.reviewer_id
        WHERE dr.document_id = document_id
          AND dr.is_required = TRUE
    LOOP
        -- Create parallel approval assignments
        -- This would integrate with the parallel approval system
        -- Implementation details depend on specific requirements
        RAISE NOTICE 'Added reviewer: %', reviewer_record.reviewer_name;
    END LOOP;
    
    -- Update document status
    UPDATE documents
    SET approval_request_id = approval_request_id,
        publication_status = 'pending_approval'
    WHERE id = document_id;
    
    RETURN approval_request_id;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Workflow Design
- **Keep workflows simple**: Complex workflows are hard to maintain and understand
- **Support delegation**: Allow approvers to delegate their authority
- **Implement escalation**: Automatic escalation prevents bottlenecks
- **Track all actions**: Complete audit trail for compliance
- **Allow comments**: Enable communication between stakeholders

### 2. Performance Optimization
- **Index approval queries**: Focus on status, assigned_to, and due_date
- **Use database triggers**: For automatic status updates and notifications
- **Implement pagination**: For approval dashboards and reports
- **Cache workflow definitions**: Avoid repeated lookups
- **Batch process escalations**: Run escalation checks periodically

### 3. User Experience
- **Clear approval dashboards**: Show pending approvals prominently
- **Mobile-friendly**: Many approvals happen on mobile devices
- **Batch approvals**: Allow multiple approvals at once
- **Smart notifications**: Don't spam, but ensure timely alerts
- **Approval history**: Show complete approval trail

### 4. Security and Compliance
- **Role-based access**: Ensure only authorized users can approve
- **Audit everything**: Log all approval actions with timestamps
- **Handle conflicts of interest**: Prevent self-approval where inappropriate
- **Support compliance requirements**: Different industries have different needs
- **Secure sensitive data**: Protect confidential information in approval data

### 5. System Integration
- **Webhook notifications**: Integrate with external systems
- **API-first design**: Enable integration with other business systems
- **Event sourcing**: Consider event-driven architecture for complex workflows
- **Data synchronization**: Keep approval status in sync across systems
- **Backup and recovery**: Ensure approval data is properly backed up

This approval system design provides a flexible foundation for implementing various business approval processes while maintaining auditability and performance.
