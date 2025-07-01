# State Machine Patterns in Database Design

State machines are fundamental to many business processes - from order processing to content moderation, user onboarding to payment flows. This guide shows how to implement robust, scalable state machines in your database.

## 🎯 Why State Machines Matter

### Common Use Cases
- **E-commerce Orders**: draft → pending → paid → shipped → delivered
- **Content Management**: draft → review → published → archived
- **User Onboarding**: registered → verified → active → suspended
- **Payment Processing**: pending → processing → completed → refunded
- **Support Tickets**: open → assigned → in_progress → resolved → closed

### Benefits of Database State Machines
- **Data Integrity** - Invalid state transitions are impossible
- **Audit Trail** - Track when and why states changed
- **Business Rules** - Enforce complex business logic
- **Consistency** - Same rules across all applications
- **Performance** - Database-native validation is fast

## 🏗️ Implementation Patterns

### Pattern 1: Simple Status Column (Level 1)

**Best for**: Simple linear workflows with few states

```sql
-- Basic state tracking with timestamps
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    
    -- Current state
    status TEXT NOT NULL DEFAULT 'draft' 
        CHECK (status IN ('draft', 'review', 'published', 'archived')),
    
    -- State transition timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    submitted_for_review_at TIMESTAMP,
    published_at TIMESTAMP,
    archived_at TIMESTAMP,
    
    -- Metadata
    author_id UUID NOT NULL,
    reviewer_id UUID,
    
    -- Business rules as constraints
    CONSTRAINT valid_review_transition 
        CHECK (
            (status != 'review') OR 
            (status = 'review' AND submitted_for_review_at IS NOT NULL)
        ),
    
    CONSTRAINT valid_published_transition 
        CHECK (
            (status != 'published') OR 
            (status = 'published' AND published_at IS NOT NULL AND reviewer_id IS NOT NULL)
        )
);

-- Index for common queries
CREATE INDEX idx_posts_status ON posts (status);
CREATE INDEX idx_posts_published_at ON posts (published_at) WHERE status = 'published';
```

**Pros**: Simple, performant, easy to understand
**Cons**: Hard to track history, limited business rule enforcement

### Pattern 2: State History Table (Level 2)

**Best for**: When you need audit trails and complex business rules

```sql
-- Main entity table (current state only)
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    total_cents INTEGER NOT NULL CHECK (total_cents > 0),
    
    -- Current state information
    current_status TEXT NOT NULL DEFAULT 'draft',
    current_status_since TIMESTAMP DEFAULT NOW(),
    status_updated_by UUID,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- State definitions table
CREATE TABLE order_status_types (
    status_code TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    is_terminal BOOLEAN DEFAULT FALSE, -- Can't transition from terminal states
    sort_order INTEGER NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO order_status_types (status_code, display_name, sort_order, is_terminal) VALUES
('draft', 'Draft', 1, FALSE),
('pending_payment', 'Pending Payment', 2, FALSE),
('paid', 'Paid', 3, FALSE),
('processing', 'Processing', 4, FALSE),
('shipped', 'Shipped', 5, FALSE),
('delivered', 'Delivered', 6, TRUE),
('cancelled', 'Cancelled', 7, TRUE),
('refunded', 'Refunded', 8, TRUE);

-- Valid state transitions
CREATE TABLE order_status_transitions (
    from_status TEXT NOT NULL REFERENCES order_status_types(status_code),
    to_status TEXT NOT NULL REFERENCES order_status_types(status_code),
    
    -- Business rules
    requires_approval BOOLEAN DEFAULT FALSE,
    allowed_roles TEXT[], -- Which roles can make this transition
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (from_status, to_status)
);

INSERT INTO order_status_transitions (from_status, to_status, allowed_roles) VALUES
('draft', 'pending_payment', ARRAY['customer', 'admin']),
('pending_payment', 'paid', ARRAY['payment_system', 'admin']),
('paid', 'processing', ARRAY['fulfillment', 'admin']),
('processing', 'shipped', ARRAY['fulfillment', 'admin']),
('shipped', 'delivered', ARRAY['delivery_system', 'customer', 'admin']),
('draft', 'cancelled', ARRAY['customer', 'admin']),
('pending_payment', 'cancelled', ARRAY['customer', 'admin']),
('paid', 'refunded', ARRAY['admin']),
('processing', 'cancelled', ARRAY['admin']);

-- State history for audit trail
CREATE TABLE order_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    from_status TEXT REFERENCES order_status_types(status_code),
    to_status TEXT NOT NULL REFERENCES order_status_types(status_code),
    
    -- Who made the change and why
    changed_by UUID NOT NULL,
    reason TEXT,
    metadata JSONB, -- Additional context (payment_id, tracking_number, etc.)
    
    -- When the change happened
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure chronological order
    CONSTRAINT status_history_order CHECK (created_at >= NOW() - INTERVAL '1 minute')
);

-- Indexes for performance
CREATE INDEX idx_order_status_history_order_id ON order_status_history (order_id, created_at);
CREATE INDEX idx_order_status_history_timeline ON order_status_history (created_at);
```

### Pattern 3: State Machine Functions

**Best for**: Complex business logic and validation

```sql
-- Function to validate and execute state transitions
CREATE OR REPLACE FUNCTION transition_order_status(
    p_order_id UUID,
    p_new_status TEXT,
    p_changed_by UUID,
    p_reason TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_current_status TEXT;
    v_is_valid_transition BOOLEAN := FALSE;
    v_is_terminal BOOLEAN := FALSE;
    v_user_roles TEXT[];
BEGIN
    -- Get current status
    SELECT current_status INTO v_current_status
    FROM orders
    WHERE id = p_order_id
    FOR UPDATE; -- Lock the row
    
    IF v_current_status IS NULL THEN
        RAISE EXCEPTION 'Order not found: %', p_order_id;
    END IF;
    
    -- Check if current status is terminal
    SELECT is_terminal INTO v_is_terminal
    FROM order_status_types
    WHERE status_code = v_current_status;
    
    IF v_is_terminal THEN
        RAISE EXCEPTION 'Cannot transition from terminal status: %', v_current_status;
    END IF;
    
    -- Validate transition is allowed
    SELECT COUNT(*) > 0 INTO v_is_valid_transition
    FROM order_status_transitions
    WHERE from_status = v_current_status AND to_status = p_new_status;
    
    IF NOT v_is_valid_transition THEN
        RAISE EXCEPTION 'Invalid transition from % to %', v_current_status, p_new_status;
    END IF;
    
    -- TODO: Add role-based authorization check here
    -- This would typically involve checking p_changed_by against allowed_roles
    
    -- Record the transition
    INSERT INTO order_status_history (
        order_id, from_status, to_status, changed_by, reason, metadata
    ) VALUES (
        p_order_id, v_current_status, p_new_status, p_changed_by, p_reason, p_metadata
    );
    
    -- Update current status
    UPDATE orders 
    SET 
        current_status = p_new_status,
        current_status_since = NOW(),
        status_updated_by = p_changed_by,
        updated_at = NOW()
    WHERE id = p_order_id;
    
    -- Trigger any side effects (notifications, etc.)
    PERFORM notify_status_change(p_order_id, v_current_status, p_new_status);
    
    RETURN TRUE;
    
EXCEPTION
    WHEN OTHERS THEN
        -- Log the error for debugging
        INSERT INTO system_errors (error_message, context, created_at)
        VALUES (SQLERRM, jsonb_build_object(
            'function', 'transition_order_status',
            'order_id', p_order_id,
            'from_status', v_current_status,
            'to_status', p_new_status,
            'user_id', p_changed_by
        ), NOW());
        
        RAISE;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT transition_order_status(
    'order-uuid',
    'paid',
    'user-uuid',
    'Payment confirmed via Stripe',
    '{"payment_id": "pi_123456", "amount_cents": 2999}'::jsonb
);
```

## 🌍 Real-World Examples

### E-Commerce Order Processing
```sql
-- Complete order state machine with business logic
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    
    -- Order details
    items JSONB NOT NULL, -- Store order items for historical accuracy
    subtotal_cents INTEGER NOT NULL CHECK (subtotal_cents > 0),
    tax_cents INTEGER NOT NULL CHECK (tax_cents >= 0),
    shipping_cents INTEGER NOT NULL CHECK (shipping_cents >= 0),
    total_cents INTEGER GENERATED ALWAYS AS (subtotal_cents + tax_cents + shipping_cents) STORED,
    
    -- Current state
    status TEXT NOT NULL DEFAULT 'cart' CHECK (
        status IN ('cart', 'checkout', 'pending_payment', 'payment_failed', 
                  'paid', 'fulfillment', 'shipped', 'delivered', 'returned', 'refunded')
    ),
    
    -- Payment information
    payment_method_id UUID,
    payment_intent_id TEXT, -- External payment processor ID
    
    -- Shipping information
    shipping_address JSONB,
    tracking_number TEXT,
    carrier TEXT,
    
    -- Timestamps for key transitions
    checkout_started_at TIMESTAMP,
    payment_attempted_at TIMESTAMP,
    paid_at TIMESTAMP,
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    
    -- SLA tracking
    estimated_delivery DATE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Business rule: Can't ship without payment
ALTER TABLE orders ADD CONSTRAINT cannot_ship_unpaid
    CHECK (
        status NOT IN ('shipped', 'delivered') OR 
        (status IN ('shipped', 'delivered') AND paid_at IS NOT NULL)
    );

-- Business rule: Must have shipping address to ship
ALTER TABLE orders ADD CONSTRAINT must_have_shipping_address
    CHECK (
        status NOT IN ('shipped', 'delivered') OR
        (status IN ('shipped', 'delivered') AND shipping_address IS NOT NULL)
    );
```

### Content Approval Workflow
```sql
-- Blog post with editorial workflow
CREATE TABLE blog_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    
    -- Author information
    author_id UUID NOT NULL,
    
    -- Editorial workflow status
    status TEXT NOT NULL DEFAULT 'draft' CHECK (
        status IN ('draft', 'submitted', 'reviewing', 'needs_revision', 
                  'approved', 'published', 'unpublished', 'archived')
    ),
    
    -- Editorial assignments
    assigned_editor_id UUID,
    
    -- Publishing details
    published_at TIMESTAMP,
    unpublished_at TIMESTAMP,
    scheduled_publish_at TIMESTAMP,
    
    -- SEO and metadata
    slug TEXT UNIQUE,
    meta_description TEXT,
    tags TEXT[],
    
    -- Engagement metrics (computed separately)
    view_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Editorial feedback
CREATE TABLE post_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES blog_posts(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL,
    
    status TEXT NOT NULL CHECK (status IN ('approved', 'needs_revision', 'rejected')),
    feedback TEXT,
    
    -- Specific review points
    review_points JSONB, -- {"grammar": "good", "seo": "needs_work", "accuracy": "excellent"}
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Auto-generate slug on publish
CREATE OR REPLACE FUNCTION generate_post_slug()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'published' AND (OLD.slug IS NULL OR NEW.slug IS NULL) THEN
        NEW.slug := lower(regexp_replace(NEW.title, '[^a-zA-Z0-9]+', '-', 'g'));
        NEW.slug := trim(both '-' from NEW.slug);
        
        -- Ensure uniqueness
        WHILE EXISTS (SELECT 1 FROM blog_posts WHERE slug = NEW.slug AND id != NEW.id) LOOP
            NEW.slug := NEW.slug || '-' || extract(epoch from now())::integer;
        END LOOP;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER post_slug_trigger
    BEFORE UPDATE ON blog_posts
    FOR EACH ROW EXECUTE FUNCTION generate_post_slug();
```

### User Account Lifecycle
```sql
-- User account with comprehensive state management
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    
    -- Account status
    status TEXT NOT NULL DEFAULT 'pending_verification' CHECK (
        status IN ('pending_verification', 'active', 'suspended', 
                  'deactivated', 'banned', 'deleted')
    ),
    
    -- Verification tracking
    email_verified_at TIMESTAMP,
    phone_verified_at TIMESTAMP,
    identity_verified_at TIMESTAMP,
    
    -- Security
    password_hash TEXT NOT NULL,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    
    -- Profile information
    first_name TEXT,
    last_name TEXT,
    date_of_birth DATE,
    phone TEXT,
    
    -- Preferences
    preferences JSONB DEFAULT '{}',
    
    -- Timestamps
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Account actions history
CREATE TABLE user_account_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    action_type TEXT NOT NULL CHECK (
        action_type IN ('verification_sent', 'verified', 'login', 'logout', 
                       'password_reset', 'suspended', 'reactivated', 'banned')
    ),
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    reason TEXT, -- For admin actions
    performed_by UUID, -- Admin who performed the action
    
    -- Additional data
    metadata JSONB,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Function to safely suspend user account
CREATE OR REPLACE FUNCTION suspend_user_account(
    p_user_id UUID,
    p_reason TEXT,
    p_admin_id UUID,
    p_duration INTERVAL DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_current_status TEXT;
BEGIN
    -- Get current status
    SELECT status INTO v_current_status
    FROM users
    WHERE id = p_user_id
    FOR UPDATE;
    
    IF v_current_status IS NULL THEN
        RAISE EXCEPTION 'User not found: %', p_user_id;
    END IF;
    
    IF v_current_status IN ('banned', 'deleted') THEN
        RAISE EXCEPTION 'Cannot suspend user in status: %', v_current_status;
    END IF;
    
    -- Update user status
    UPDATE users 
    SET 
        status = 'suspended',
        updated_at = NOW()
    WHERE id = p_user_id;
    
    -- Log the action
    INSERT INTO user_account_actions (
        user_id, action_type, reason, performed_by, metadata
    ) VALUES (
        p_user_id, 'suspended', p_reason, p_admin_id,
        jsonb_build_object(
            'previous_status', v_current_status,
            'duration', extract(epoch from p_duration)
        )
    );
    
    -- Schedule automatic reactivation if duration specified
    IF p_duration IS NOT NULL THEN
        -- This would typically trigger a background job
        PERFORM schedule_user_reactivation(p_user_id, NOW() + p_duration);
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Advanced Patterns

### Parallel State Tracking
```sql
-- When entities can be in multiple states simultaneously
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    
    -- Separate state dimensions
    approval_status TEXT DEFAULT 'draft' CHECK (
        approval_status IN ('draft', 'submitted', 'approved', 'rejected')
    ),
    
    publication_status TEXT DEFAULT 'unpublished' CHECK (
        publication_status IN ('unpublished', 'scheduled', 'published', 'archived')
    ),
    
    security_status TEXT DEFAULT 'unrestricted' CHECK (
        security_status IN ('unrestricted', 'internal', 'confidential', 'classified')
    ),
    
    -- Timestamps for each dimension
    submitted_at TIMESTAMP,
    approved_at TIMESTAMP,
    published_at TIMESTAMP,
    classified_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Business rule: Can't publish without approval
ALTER TABLE documents ADD CONSTRAINT publication_requires_approval
    CHECK (
        publication_status = 'unpublished' OR
        (publication_status != 'unpublished' AND approval_status = 'approved')
    );
```

### Time-Based State Transitions
```sql
-- Automatic state transitions based on time
CREATE TABLE promotional_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    
    -- Time-based status
    status TEXT NOT NULL DEFAULT 'draft' CHECK (
        status IN ('draft', 'scheduled', 'active', 'paused', 'ended', 'cancelled')
    ),
    
    -- Time boundaries
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP NOT NULL CHECK (ends_at > starts_at),
    
    -- Auto-computed status based on time
    computed_status TEXT GENERATED ALWAYS AS (
        CASE 
            WHEN status = 'cancelled' THEN 'cancelled'
            WHEN status = 'paused' THEN 'paused'
            WHEN NOW() < starts_at THEN 'scheduled'
            WHEN NOW() BETWEEN starts_at AND ends_at THEN 'active'
            WHEN NOW() > ends_at THEN 'ended'
            ELSE status
        END
    ) STORED,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Function to sync time-based statuses
CREATE OR REPLACE FUNCTION sync_campaign_statuses()
RETURNS INTEGER AS $$
DECLARE
    campaigns_updated INTEGER := 0;
BEGIN
    -- Start scheduled campaigns
    UPDATE promotional_campaigns 
    SET status = 'active'
    WHERE status = 'scheduled' 
      AND starts_at <= NOW()
      AND ends_at > NOW();
    
    GET DIAGNOSTICS campaigns_updated = ROW_COUNT;
    
    -- End active campaigns
    UPDATE promotional_campaigns 
    SET status = 'ended'
    WHERE status = 'active' 
      AND ends_at <= NOW();
    
    GET DIAGNOSTICS campaigns_updated = campaigns_updated + ROW_COUNT;
    
    RETURN campaigns_updated;
END;
$$ LANGUAGE plpgsql;

-- Schedule this function to run periodically
-- SELECT cron.schedule('sync-campaigns', '* * * * *', 'SELECT sync_campaign_statuses();');
```

## 🔍 Querying State Machines

### Common Query Patterns
```sql
-- Current state distribution
SELECT 
    status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM orders
GROUP BY status
ORDER BY count DESC;

-- State transition analytics
SELECT 
    from_status,
    to_status,
    COUNT(*) as transition_count,
    AVG(EXTRACT(EPOCH FROM (created_at - LAG(created_at) OVER (
        PARTITION BY order_id ORDER BY created_at
    )))) as avg_duration_seconds
FROM order_status_history
WHERE from_status IS NOT NULL
GROUP BY from_status, to_status
ORDER BY transition_count DESC;

-- Orders stuck in a state
SELECT 
    id,
    current_status,
    current_status_since,
    NOW() - current_status_since as time_in_status
FROM orders
WHERE current_status = 'processing'
  AND current_status_since < NOW() - INTERVAL '2 days'
ORDER BY current_status_since ASC;

-- State machine bottlenecks
WITH state_durations AS (
    SELECT 
        order_id,
        to_status,
        created_at,
        LEAD(created_at) OVER (
            PARTITION BY order_id ORDER BY created_at
        ) as next_transition_at
    FROM order_status_history
)
SELECT 
    to_status,
    COUNT(*) as orders,
    AVG(EXTRACT(EPOCH FROM (next_transition_at - created_at))) as avg_duration_seconds,
    PERCENTILE_CONT(0.5) WITHIN GROUP (
        ORDER BY EXTRACT(EPOCH FROM (next_transition_at - created_at))
    ) as median_duration_seconds
FROM state_durations
WHERE next_transition_at IS NOT NULL
GROUP BY to_status
ORDER BY avg_duration_seconds DESC;
```

### Reporting Views
```sql
-- Order funnel analysis
CREATE VIEW order_funnel AS
SELECT 
    DATE_TRUNC('day', created_at) as date,
    COUNT(*) FILTER (WHERE status IN ('cart', 'checkout')) as browsing,
    COUNT(*) FILTER (WHERE status = 'pending_payment') as payment_initiated,
    COUNT(*) FILTER (WHERE status = 'paid') as paid,
    COUNT(*) FILTER (WHERE status = 'shipped') as shipped,
    COUNT(*) FILTER (WHERE status = 'delivered') as delivered,
    COUNT(*) FILTER (WHERE status IN ('cancelled', 'refunded')) as cancelled
FROM orders
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY date DESC;

-- State machine health metrics
CREATE VIEW state_machine_health AS
SELECT 
    'orders' as entity_type,
    COUNT(*) as total_entities,
    COUNT(*) FILTER (WHERE current_status_since < NOW() - INTERVAL '1 hour') as stale_entities,
    COUNT(DISTINCT current_status) as active_states,
    AVG(EXTRACT(EPOCH FROM (NOW() - current_status_since))) as avg_time_in_current_state
FROM orders
WHERE current_status NOT IN ('delivered', 'cancelled', 'refunded');
```

## 🚨 Common Pitfalls & Solutions

### Pitfall 1: No State History
```sql
-- ❌ Bad: Only current state
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL
);

-- ✅ Good: With history tracking
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    current_status TEXT NOT NULL,
    current_status_since TIMESTAMP DEFAULT NOW()
);

CREATE TABLE order_status_history (
    order_id UUID REFERENCES orders(id),
    from_status TEXT,
    to_status TEXT,
    changed_by UUID,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Pitfall 2: No Validation
```sql
-- ❌ Bad: No transition validation
UPDATE orders SET status = 'delivered' WHERE id = ?;

-- ✅ Good: Validated transitions
SELECT transition_order_status(?, 'delivered', ?, 'Package delivered');
```

### Pitfall 3: Race Conditions
```sql
-- ❌ Bad: Race condition possible
UPDATE orders SET status = 'processing' 
WHERE id = ? AND status = 'paid';

-- ✅ Good: Proper locking
CREATE OR REPLACE FUNCTION safe_status_transition(...)
RETURNS BOOLEAN AS $$
BEGIN
    -- Lock the row first
    SELECT status FROM orders WHERE id = p_order_id FOR UPDATE;
    -- Then validate and update
    -- ... rest of logic
END;
$$ LANGUAGE plpgsql;
```

## 📈 Performance Optimization

### Indexing Strategy
```sql
-- Index current status for filtering
CREATE INDEX idx_orders_current_status ON orders (current_status);

-- Partial indexes for active states
CREATE INDEX idx_orders_active ON orders (current_status_since) 
WHERE current_status NOT IN ('delivered', 'cancelled', 'refunded');

-- Composite index for common queries
CREATE INDEX idx_orders_customer_status ON orders (customer_id, current_status);

-- History table optimization
CREATE INDEX idx_status_history_order_timeline ON order_status_history (order_id, created_at);
```

### Archival Strategy
```sql
-- Archive completed orders to separate table
CREATE TABLE orders_archived (LIKE orders INCLUDING ALL);

-- Move old completed orders
INSERT INTO orders_archived 
SELECT * FROM orders 
WHERE current_status IN ('delivered', 'cancelled', 'refunded')
  AND current_status_since < NOW() - INTERVAL '1 year';

DELETE FROM orders 
WHERE id IN (SELECT id FROM orders_archived);
```

## 💡 Best Practices

1. **Design for Auditability** - Always track who changed what and when
2. **Validate Transitions** - Use database constraints and functions
3. **Handle Concurrency** - Use proper locking to prevent race conditions
4. **Plan for Scale** - Consider archival and partitioning strategies
5. **Monitor Performance** - Track state distribution and transition times
6. **Document Business Rules** - Make state machine logic explicit
7. **Test Edge Cases** - Handle concurrent transitions and error conditions
8. **Version Control** - Track changes to state machine rules
9. **Separate Concerns** - Keep state logic separate from business logic
10. **Plan for Evolution** - Design state machines that can be extended

State machines are powerful tools for modeling complex business processes. When implemented correctly in the database, they provide strong consistency guarantees while maintaining flexibility for future requirements.
