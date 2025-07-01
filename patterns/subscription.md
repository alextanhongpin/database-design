# Subscription System Database Design

## Table of Contents
- [Overview](#overview)
- [Core Subscription Model](#core-subscription-model)
- [Subscription Plans](#subscription-plans)
- [Billing and Invoicing](#billing-and-invoicing)
- [Usage-Based Billing](#usage-based-billing)
- [Subscription Lifecycle](#subscription-lifecycle)
- [Proration and Upgrades](#proration-and-upgrades)
- [Trial Management](#trial-management)
- [Real-World Examples](#real-world-examples)
- [Advanced Patterns](#advanced-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Overview

Subscription systems are complex business models that require careful database design to handle recurring billing, plan changes, trials, usage tracking, and customer lifecycle management. This guide covers patterns for building scalable subscription platforms.

### Key Requirements
- **Flexible Billing Cycles**: Monthly, quarterly, annual, custom periods
- **Plan Management**: Upgrades, downgrades, add-ons, custom pricing
- **Proration**: Fair billing for mid-cycle changes
- **Usage Tracking**: Metered billing and usage limits
- **Trial Management**: Free trials with automatic conversion
- **Dunning Management**: Failed payment handling and retry logic
- **Revenue Recognition**: Accurate financial reporting

## Core Subscription Model

### Base Tables

```sql
-- Subscription plans (the products/services offered)
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Plan identification
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    
    -- Pricing
    base_price DECIMAL(19,4) NOT NULL CHECK (base_price >= 0),
    setup_fee DECIMAL(19,4) DEFAULT 0 CHECK (setup_fee >= 0),
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Billing configuration
    billing_interval interval_type NOT NULL DEFAULT 'monthly',
    billing_interval_count INTEGER DEFAULT 1 CHECK (billing_interval_count > 0),
    
    -- Plan features and limits
    features JSONB DEFAULT '{}',
    usage_limits JSONB DEFAULT '{}',
    
    -- Trial configuration
    trial_period_days INTEGER DEFAULT 0 CHECK (trial_period_days >= 0),
    
    -- Plan status
    is_active BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT TRUE, -- Can customers see and select this plan?
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_active_public (is_active, is_public),
    INDEX idx_slug (slug)
);

-- Billing intervals
CREATE TYPE interval_type AS ENUM ('day', 'week', 'month', 'quarter', 'year');

-- Customers/Users (simplified)
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    
    -- Customer details
    company_name VARCHAR(255),
    billing_address JSONB,
    tax_id VARCHAR(100),
    
    -- Account status
    status customer_status DEFAULT 'active',
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TYPE customer_status AS ENUM ('active', 'suspended', 'closed');

-- Main subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    
    -- Subscription pricing (can override plan pricing)
    current_price DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    -- Billing configuration
    billing_interval interval_type NOT NULL,
    billing_interval_count INTEGER NOT NULL DEFAULT 1,
    
    -- Current billing period
    current_period_start TIMESTAMP NOT NULL,
    current_period_end TIMESTAMP NOT NULL,
    
    -- Status and lifecycle
    status subscription_status DEFAULT 'active',
    
    -- Trial information
    trial_start TIMESTAMP,
    trial_end TIMESTAMP,
    
    -- Cancellation
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    cancelled_at TIMESTAMP,
    ended_at TIMESTAMP,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_customer_status (customer_id, status),
    INDEX idx_period_end (current_period_end, status),
    INDEX idx_trial_end (trial_end) WHERE trial_end IS NOT NULL
);

CREATE TYPE subscription_status AS ENUM (
    'incomplete', 'incomplete_expired', 'trialing', 'active', 
    'past_due', 'canceled', 'unpaid', 'paused'
);

-- Subscription items (for multi-item subscriptions and add-ons)
CREATE TABLE subscription_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    
    -- Item details
    price_id UUID REFERENCES subscription_plans(id), -- If this is a standard plan
    quantity INTEGER DEFAULT 1 CHECK (quantity > 0),
    
    -- Custom pricing (for enterprise deals)
    unit_price DECIMAL(19,4),
    currency_code CHAR(3),
    
    -- Usage-based billing
    usage_type usage_type DEFAULT 'licensed',
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure consistent currency
    CHECK (
        (price_id IS NOT NULL AND unit_price IS NULL) OR
        (price_id IS NULL AND unit_price IS NOT NULL AND currency_code IS NOT NULL)
    ),
    
    INDEX idx_subscription (subscription_id)
);

CREATE TYPE usage_type AS ENUM ('licensed', 'metered');
```

## Subscription Plans

### Plan Variations and Pricing

```sql
-- Plan variations (for different billing cycles of the same plan)
CREATE TABLE plan_variations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    
    -- Variation details
    billing_interval interval_type NOT NULL,
    billing_interval_count INTEGER DEFAULT 1,
    
    -- Pricing (can be discounted for annual plans)
    price DECIMAL(19,4) NOT NULL,
    discount_percentage DECIMAL(5,2) DEFAULT 0,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (base_plan_id, billing_interval, billing_interval_count)
);

-- Add-ons and plan features
CREATE TABLE plan_add_ons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Add-on details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Pricing
    price DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) DEFAULT 'USD',
    billing_type billing_type DEFAULT 'recurring',
    
    -- Applicability
    compatible_plans UUID[] DEFAULT '{}', -- Empty array means compatible with all
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TYPE billing_type AS ENUM ('one_time', 'recurring', 'usage_based');

-- Subscription add-ons (many-to-many)
CREATE TABLE subscription_add_ons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    add_on_id UUID NOT NULL REFERENCES plan_add_ons(id),
    
    -- Add-on configuration
    quantity INTEGER DEFAULT 1,
    unit_price DECIMAL(19,4) NOT NULL,
    
    -- Lifecycle
    added_at TIMESTAMP DEFAULT NOW(),
    removed_at TIMESTAMP,
    
    -- Prevent duplicates
    UNIQUE (subscription_id, add_on_id) WHERE removed_at IS NULL
);
```

## Billing and Invoicing

### Invoice Management

```sql
-- Invoices
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID REFERENCES subscriptions(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Invoice identification
    invoice_number VARCHAR(100) UNIQUE NOT NULL,
    
    -- Billing period
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    
    -- Financial details
    subtotal DECIMAL(19,4) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    total_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    amount_paid DECIMAL(19,4) NOT NULL DEFAULT 0,
    amount_due DECIMAL(19,4) GENERATED ALWAYS AS (total_amount - amount_paid) STORED,
    currency_code CHAR(3) NOT NULL,
    
    -- Invoice status
    status invoice_status DEFAULT 'draft',
    
    -- Important dates
    due_date TIMESTAMP,
    paid_at TIMESTAMP,
    
    -- Metadata
    description TEXT,
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_customer_status (customer_id, status),
    INDEX idx_subscription (subscription_id),
    INDEX idx_due_date (due_date, status),
    INDEX idx_invoice_number (invoice_number)
);

CREATE TYPE invoice_status AS ENUM (
    'draft', 'open', 'paid', 'void', 'uncollectible'
);

-- Invoice line items
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    subscription_item_id UUID REFERENCES subscription_items(id),
    
    -- Item details
    description TEXT NOT NULL,
    quantity DECIMAL(10,3) DEFAULT 1,
    unit_price DECIMAL(19,4) NOT NULL,
    amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    -- Period for this line item
    period_start TIMESTAMP,
    period_end TIMESTAMP,
    
    -- Item type
    item_type item_type DEFAULT 'subscription',
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_invoice (invoice_id)
);

CREATE TYPE item_type AS ENUM (
    'subscription', 'add_on', 'usage', 'credit', 'tax', 'discount'
);

-- Invoice generation function
CREATE OR REPLACE FUNCTION generate_invoice_for_subscription(
    subscription_id UUID,
    period_start TIMESTAMP,
    period_end TIMESTAMP
) RETURNS UUID AS $$
DECLARE
    invoice_id UUID;
    subscription_record RECORD;
    item_record RECORD;
    subtotal DECIMAL(19,4) := 0;
BEGIN
    -- Get subscription details
    SELECT s.*, c.id as customer_id
    INTO subscription_record
    FROM subscriptions s
    JOIN customers c ON c.id = s.customer_id
    WHERE s.id = subscription_id;
    
    -- Create invoice
    INSERT INTO invoices (
        subscription_id, customer_id, invoice_number,
        period_start, period_end, currency_code, status
    ) VALUES (
        subscription_id, subscription_record.customer_id,
        'INV-' || EXTRACT(YEAR FROM NOW()) || '-' || gen_random_uuid()::text,
        period_start, period_end, subscription_record.currency_code, 'draft'
    ) RETURNING id INTO invoice_id;
    
    -- Add subscription items
    FOR item_record IN 
        SELECT si.*, sp.name as plan_name
        FROM subscription_items si
        LEFT JOIN subscription_plans sp ON sp.id = si.price_id
        WHERE si.subscription_id = subscription_id
    LOOP
        -- Calculate prorated amount
        DECLARE
            item_amount DECIMAL(19,4);
        BEGIN
            item_amount := COALESCE(item_record.unit_price, subscription_record.current_price) * item_record.quantity;
            
            INSERT INTO invoice_items (
                invoice_id, subscription_item_id, description,
                quantity, unit_price, amount, currency_code,
                period_start, period_end, item_type
            ) VALUES (
                invoice_id, item_record.id,
                COALESCE(item_record.plan_name, 'Subscription Item'),
                item_record.quantity,
                COALESCE(item_record.unit_price, subscription_record.current_price),
                item_amount, subscription_record.currency_code,
                period_start, period_end, 'subscription'
            );
            
            subtotal := subtotal + item_amount;
        END;
    END LOOP;
    
    -- Update invoice totals
    UPDATE invoices 
    SET subtotal = subtotal,
        total_amount = subtotal, -- Simplified (no tax calculation)
        status = 'open'
    WHERE id = invoice_id;
    
    RETURN invoice_id;
END;
$$ LANGUAGE plpgsql;
```

## Usage-Based Billing

### Usage Tracking

```sql
-- Usage records for metered billing
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_item_id UUID NOT NULL REFERENCES subscription_items(id),
    
    -- Usage details
    quantity DECIMAL(10,3) NOT NULL CHECK (quantity >= 0),
    unit_of_measure VARCHAR(50) NOT NULL, -- requests, GB, users, etc.
    
    -- Timing
    usage_timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    recorded_at TIMESTAMP DEFAULT NOW(),
    
    -- Aggregation period (for pre-aggregated data)
    period_start TIMESTAMP,
    period_end TIMESTAMP,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Indexes
    INDEX idx_subscription_item_timestamp (subscription_item_id, usage_timestamp),
    INDEX idx_period (period_start, period_end) WHERE period_start IS NOT NULL
);

-- Usage summaries (for performance)
CREATE TABLE usage_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_item_id UUID NOT NULL REFERENCES subscription_items(id),
    
    -- Summary period
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    
    -- Aggregated usage
    total_usage DECIMAL(19,4) NOT NULL,
    unit_of_measure VARCHAR(50) NOT NULL,
    
    -- Billing calculation
    billable_usage DECIMAL(19,4) NOT NULL, -- After applying included usage
    unit_price DECIMAL(19,4) NOT NULL,
    total_amount DECIMAL(19,4) NOT NULL,
    
    -- Status
    is_billed BOOLEAN DEFAULT FALSE,
    invoice_item_id UUID REFERENCES invoice_items(id),
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure unique summaries per period
    UNIQUE (subscription_item_id, period_start, period_end)
);

-- Function to aggregate usage for billing
CREATE OR REPLACE FUNCTION aggregate_usage_for_billing(
    subscription_item_id UUID,
    period_start TIMESTAMP,
    period_end TIMESTAMP
) RETURNS DECIMAL(19,4) AS $$
DECLARE
    total_usage DECIMAL(19,4) := 0;
    included_usage DECIMAL(19,4) := 0;
    billable_usage DECIMAL(19,4) := 0;
BEGIN
    -- Get total usage for the period
    SELECT COALESCE(SUM(quantity), 0) INTO total_usage
    FROM usage_records
    WHERE subscription_item_id = aggregate_usage_for_billing.subscription_item_id
      AND usage_timestamp >= period_start
      AND usage_timestamp < period_end;
    
    -- Get included usage from subscription plan
    -- This is simplified - in practice, you'd join with plan details
    included_usage := 0; -- Plans might include free usage
    
    -- Calculate billable usage
    billable_usage := GREATEST(0, total_usage - included_usage);
    
    RETURN billable_usage;
END;
$$ LANGUAGE plpgsql;
```

## Subscription Lifecycle

### Lifecycle Management

```sql
-- Subscription changes (for audit trail)
CREATE TABLE subscription_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- Change details
    change_type change_type NOT NULL,
    from_plan_id UUID REFERENCES subscription_plans(id),
    to_plan_id UUID REFERENCES subscription_plans(id),
    
    -- Pricing changes
    from_price DECIMAL(19,4),
    to_price DECIMAL(19,4),
    
    -- Proration details
    proration_amount DECIMAL(19,4),
    proration_credit_id UUID REFERENCES invoice_items(id),
    
    -- Change timing
    effective_date TIMESTAMP NOT NULL,
    requested_by UUID, -- User who requested the change
    
    -- Metadata
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_subscription_date (subscription_id, effective_date),
    INDEX idx_change_type (change_type, effective_date)
);

CREATE TYPE change_type AS ENUM (
    'created', 'upgraded', 'downgraded', 'cancelled', 
    'reactivated', 'paused', 'resumed', 'price_changed'
);

-- Function to upgrade/downgrade subscription
CREATE OR REPLACE FUNCTION change_subscription_plan(
    subscription_id UUID,
    new_plan_id UUID,
    effective_date TIMESTAMP DEFAULT NOW(),
    prorate BOOLEAN DEFAULT TRUE
) RETURNS BOOLEAN AS $$
DECLARE
    subscription_record RECORD;
    old_plan_record RECORD;  
    new_plan_record RECORD;
    proration_amount DECIMAL(19,4) := 0;
    change_type change_type;
BEGIN
    -- Get current subscription
    SELECT * INTO subscription_record
    FROM subscriptions s
    WHERE s.id = subscription_id AND s.status = 'active';
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Active subscription not found';
    END IF;
    
    -- Get plan details
    SELECT * INTO old_plan_record FROM subscription_plans WHERE id = subscription_record.plan_id;
    SELECT * INTO new_plan_record FROM subscription_plans WHERE id = new_plan_id;
    
    -- Determine change type
    IF new_plan_record.base_price > old_plan_record.base_price THEN
        change_type := 'upgraded';
    ELSIF new_plan_record.base_price < old_plan_record.base_price THEN
        change_type := 'downgraded';
    ELSE
        change_type := 'price_changed';
    END IF;
    
    -- Calculate proration if enabled
    IF prorate THEN
        DECLARE
            days_remaining INTEGER;
            total_days INTEGER;
            unused_amount DECIMAL(19,4);
        BEGIN
            -- Calculate remaining days in current period
            days_remaining := EXTRACT(DAY FROM subscription_record.current_period_end - effective_date);
            total_days := EXTRACT(DAY FROM subscription_record.current_period_end - subscription_record.current_period_start);
            
            -- Calculate unused amount from old plan
            unused_amount := (subscription_record.current_price * days_remaining) / total_days;
            
            -- Calculate prorated amount for new plan
            proration_amount := ((new_plan_record.base_price * days_remaining) / total_days) - unused_amount;
        END;
    END IF;
    
    -- Update subscription
    UPDATE subscriptions
    SET plan_id = new_plan_id,
        current_price = new_plan_record.base_price,
        billing_interval = new_plan_record.billing_interval,
        billing_interval_count = new_plan_record.billing_interval_count,
        updated_at = NOW()
    WHERE id = subscription_id;
    
    -- Record the change
    INSERT INTO subscription_changes (
        subscription_id, change_type, from_plan_id, to_plan_id,
        from_price, to_price, proration_amount, effective_date
    ) VALUES (
        subscription_id, change_type, subscription_record.plan_id, new_plan_id,
        subscription_record.current_price, new_plan_record.base_price,
        proration_amount, effective_date
    );
    
    -- Create proration invoice if needed
    IF proration_amount != 0 THEN
        -- Create immediate invoice for proration
        -- This is simplified - in practice you'd create a proper invoice
        RAISE NOTICE 'Proration amount: %', proration_amount;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## Real-World Examples

### SaaS Platform

```sql
-- Example: Project management SaaS with user-based pricing
CREATE TABLE saas_subscriptions AS (
    SELECT 
        s.*,
        -- Calculate seat-based pricing
        (
            SELECT SUM(si.quantity * COALESCE(si.unit_price, sp.base_price))
            FROM subscription_items si
            LEFT JOIN subscription_plans sp ON sp.id = si.price_id
            WHERE si.subscription_id = s.id
        ) as monthly_recurring_revenue
    FROM subscriptions s
    WHERE s.status IN ('active', 'trialing')
);

-- User seat management
CREATE TABLE subscription_seats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    user_email VARCHAR(255) NOT NULL,
    
    -- Seat details
    role VARCHAR(50) DEFAULT 'member',
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timing
    added_at TIMESTAMP DEFAULT NOW(),
    removed_at TIMESTAMP,
    
    UNIQUE (subscription_id, user_email) WHERE removed_at IS NULL
);

-- Automatically adjust subscription items based on active seats
CREATE OR REPLACE FUNCTION sync_subscription_seats()
RETURNS TRIGGER AS $$
DECLARE
    active_seats INTEGER;
    subscription_item_id UUID;
BEGIN
    -- Count active seats for the subscription
    SELECT COUNT(*) INTO active_seats
    FROM subscription_seats
    WHERE subscription_id = COALESCE(NEW.subscription_id, OLD.subscription_id)
      AND is_active = TRUE
      AND removed_at IS NULL;
    
    -- Update subscription item quantity
    UPDATE subscription_items
    SET quantity = active_seats
    WHERE subscription_id = COALESCE(NEW.subscription_id, OLD.subscription_id)
      AND usage_type = 'licensed';
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sync_seats_trigger
    AFTER INSERT OR UPDATE OR DELETE ON subscription_seats
    FOR EACH ROW EXECUTE FUNCTION sync_subscription_seats();
```

### Media Streaming Service

```sql
-- Example: Streaming service with multiple subscription tiers
CREATE TABLE streaming_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL, -- Basic, Standard, Premium
    
    -- Features
    max_streams INTEGER NOT NULL,
    max_resolution streaming_quality NOT NULL,
    has_downloads BOOLEAN DEFAULT FALSE,
    
    -- Pricing
    monthly_price DECIMAL(19,4) NOT NULL,
    annual_price DECIMAL(19,4) NOT NULL,
    
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TYPE streaming_quality AS ENUM ('sd', 'hd', 'uhd');

-- Track device usage
CREATE TABLE streaming_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- Session details
    device_type VARCHAR(50),
    device_id VARCHAR(255),
    content_id VARCHAR(255),
    
    -- Timing
    started_at TIMESTAMP DEFAULT NOW(),
    ended_at TIMESTAMP,
    duration_seconds INTEGER,
    
    -- Usage tracking
    bytes_streamed BIGINT DEFAULT 0,
    quality streaming_quality
);

-- Monitor concurrent streams
CREATE OR REPLACE FUNCTION check_concurrent_streams()
RETURNS TRIGGER AS $$
DECLARE
    max_streams INTEGER;
    current_streams INTEGER;
BEGIN
    -- Get max streams for subscription
    SELECT sp.max_streams INTO max_streams
    FROM subscriptions s
    JOIN streaming_plans sp ON sp.id = s.plan_id
    WHERE s.id = NEW.subscription_id;
    
    -- Count current active streams
    SELECT COUNT(*) INTO current_streams
    FROM streaming_sessions
    WHERE subscription_id = NEW.subscription_id
      AND ended_at IS NULL;
    
    IF current_streams >= max_streams THEN
        RAISE EXCEPTION 'Maximum concurrent streams exceeded';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_streams_trigger
    BEFORE INSERT ON streaming_sessions
    FOR EACH ROW EXECUTE FUNCTION check_concurrent_streams();
```

## Best Practices

### 1. Financial Accuracy
- **Use precise decimal types**: Never use floating point for money
- **Implement proper rounding**: Follow accounting standards
- **Track all changes**: Maintain complete audit trails
- **Handle proration carefully**: Ensure fair billing for plan changes
- **Validate totals**: Always check that invoice totals are correct

### 2. Performance Optimization
- **Index subscription queries**: Focus on customer_id, status, and dates
- **Partition large tables**: Usage records and invoice items by date
- **Use materialized views**: For complex reporting queries
- **Cache frequently accessed data**: Current subscription status, plan details
- **Batch process usage data**: Aggregate usage periodically

### 3. Data Integrity
- **Use foreign key constraints**: Maintain referential integrity
- **Implement check constraints**: Validate business rules at database level
- **Handle currency consistently**: Store currency with all monetary values
- **Prevent duplicate subscriptions**: One active subscription per customer per plan type
- **Validate billing periods**: Ensure period_end > period_start

### 4. Subscription Management
- **Support flexible billing cycles**: Not just monthly/annual
- **Implement grace periods**: Handle failed payments gracefully
- **Track subscription changes**: Complete history of upgrades/downgrades
- **Handle cancellations properly**: Distinguish between immediate and end-of-period
- **Support subscription pausing**: For seasonal businesses

### 5. Reporting and Analytics
- **Calculate MRR/ARR accurately**: Include all recurring revenue
- **Track churn metrics**: Customer and revenue churn
- **Monitor usage patterns**: Identify upsell opportunities
- **Generate aging reports**: Track overdue payments
- **Implement cohort analysis**: Track customer lifetime value

This comprehensive subscription system design supports complex billing scenarios while maintaining data integrity and performance at scale.
- will user receive freemium? we can create a default plan that is freemium (valid say for 1 month), which cannot be renewed.
- statuses: subscriptions can be renewed, cancelled, upgraded, downgraded etc
- plans are valid for the period (month, year, etc)
- plans have a start and end date
- subscriptions are charged at the end of the month
- if the users did not paid for the month, next month account is temporarily disabled
- if the users downgrade the plan, the features will be missing (disabled), when they enable it back then the features will be added back. additional costs are refunded (?) unless stated not in the agreement
- how to renew the existing subscription? ( without plan changes)
- Do we need to create the basic plan? It will only take up rows in the db.
- How to pause subscription?
- How to generate invoice for subscription?
- What if the plans changed? Add new plans, expire the old one through valid from date, but don’t delete it. The old data may still reference the old plan, but the new ones will have the new plans. What if we want to force upgrade the old plans (deprecation), we can automatically extend them.
- the period (weekly, monthly, yearly) matters if we are going to do deduction/refund when the user changes the subscription plan
- the cost per day, month, year also needs to be defined
- is the cost affected by other rules? such as country, location, roles.
- if the user upgrades his subscription, then he has to pay more for the current elapsed difference in the duration left for the current subscription. What if the user downgrades his subscription? Do we need to refund the subscription?
- If the plans is for individual vs organization, the features could have different business rules, e.g. individual can create 5 items. But if we have organization account, we can probably have a rule that only 5 users can be added, and each of them can only create 1 item.

## Subscription schema

- https://www.nathanhammond.com/the-subscription-library-schema-to-rule-them-all
- https://www.nathanhammond.com/patterns-for-subscription-based-billing
- https://softwareengineering.stackexchange.com/questions/196524/handling-subscriptions-balances-and-pricing-plan-changes

## Subscription plans naming

https://www.paidmembershipspro.com/how-to-name-your-membership-levels-or-subscription-options/


## Sample code in Golang

```go
package main

import (
	"fmt"
	"math"
	"time"
)

type Feature struct {
	ClientCount int
	// The cost of the feature.
	// NOTE: What is the currency of the cost?
	Cost float64
}

type Plan struct {
	ID   string
	Type string

	// The date the plan is introduced.
	ValidFrom time.Time

	// The validity of the plan. If we want to terminate the plan, just set the valid till date.
	ValidTill time.Time

	// The Plan that the current plan superseded. This may happen when we deprecate the old plans and introduce a new one.
	ParentID string

	// Cost Overwrite - either take the cost of the plans, or indicate the overwritten cost here in the Plan.
	// NOTE: For fremium, we probably also need to generate an invoice for the user. That way, we can probably keep track of the costing model for freemium. (or not).
	CostPerDay   float64
	CostPerMonth float64
	CostPerYear  float64
	// The country the plan is available in.
	Country string
	// The currency used for the plan pricing. For conversion, we can create a pricing table for the costs.
	Currency string
}

// The plan features keep track of the features for each plans. Different plans that are in different countries, region, tier may have different plan features available.
// Whenever a new plan is created/deprecated/deleted/updated, the plan features needs to be modified as well.
type PlanFeature struct {
	PlanID   string
	Features []Feature
}

type SubscriptionPlan struct {
	ID             string
	SubscriptionID string
	PlanID         string
	ValidFrom      time.Time
	ValidTill      time.Time
	PeriodType     string // weekly, monthly, annually.
	// The previous subscription that is renewed. Note that the fremium model cannot be renewed.
	ParentID string
	// A boolean to keep track of the paid status. If the previous subscription is not paid, do not extend.
	// Once the user made the payment, update the status.
	IsPaid      bool
	IsRenewable bool

	// NOTE: We might need the following to compute the final cost of the subscription,
	// since different country might have different pricing tables.
	// On second thoughts, probably create the different plans in different countries. The plan tables won't be that much anyway.
	// Country string
	// Currency string
}

// Prorated cost of the subscription.
func (s *SubscriptionPlan) CalculateCost(plans map[string]Plan) float64 {
	days := math.Ceil(s.ValidTill.Sub(s.ValidFrom).Second() / (24 * time.Hour))
	// TODO: Handle if plans does not exist.
	currentPlan := plans[s.PlanID]
	return days * currentPlan.CostPerDay
}

type Invoice struct {
	SubscriptionPlanID string
	Amount             float64
	// The date the invoice is sent.
	SentAt time.Time
	// The date the invoice id paid.
	PaidAt time.Time
	UserID string
}

type User struct {
	ID string
}

type Subscription struct {
	ID     string
	UserID string
	// The date the subscription is made.
	ValidFrom time.Time
	// The date the subscription is terminated.
	ValidTill time.Time
	// A boolean to indicate if the subscription is still active.
	Active bool
	// If true, then the subscription will be renewed automatically.
	AutoRenewed bool
}

func main() {
	// Assuming we already have a user.
	u := User{"1"}
	// And a bunch of plans.
	p0 := Plan{
		ID:        "0",
		PlanType:  "freemium",
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	p1 := Plan{
		ID:        "1",
		PlanType:  "basic",
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	p2 := Plan{
		ID:        "2",
		PlanType:  "premium",
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	// User subscribes to a plan.
	s := Subscription{
		ID:        "1",
		UserID:    u.ID,
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	// The plan is created for the current month.
	sp := SubscriptionPlan{
		SubscriptionID: "1",
		PlanID:         "1",
		ValidFrom:      time.Now(),
		ValidTill:      time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
		IsPaid:         true,
	}
	sp1 := SubscriptionPlan{
		SubscriptionID: "1",
		PlanID:         "1",
		ParentID:       "1",
		ValidFrom:      time.Now(), // start of the month. NOTE: Check the period first, if it's annual, it should be start/end of the year, and the pricing deduction should be based on the difference.
		ValidTill:      time.Now(), // end of the month.
	}
	// To compute the final subscription values when the user upgrade/downgrade/terminate their plan:
	// Get all subscriptions where the valid_from is within the current period (month, year...).
	// Find the difference in days (plan 1 duration, plan 2 duration)
	// Compute the difference.
	// What if the user is attempting to modify the subscription frequently (?). Block them.
	fmt.Println("Hello, playground")
}
```


## Schema

```

party
- id
- subtype enum(person, organization)

// Subscription information for the user. If the plan is basic, it won’t be counted as a subscription to avoid creating redundant roles.
subscription
- id
- party_id
- valid_from // The date the subscription is activated
- valid_till // The date the subscription is expected to end (can be different than deleted at)
- is_active // The subscription status, or just check the date of valid_till
- created_at // The date the subscription is created.
- updated_at
- deleted_at


// Feature type
feature_type 
- name // E.g. country, period (weekly, monthly, yearly), currency
- description

// Feature represents the chosen feature type and it’s corresponding value.
feature
- id
- feature_type_id
- value

feature 
{id: 1, feature_type_id: currency, value: “SGD”},
{id: 2, feature_type_id: period, value: “monthly”}
{id: 3, feature_type_id: country, value: “Singapore”}
{id: 4, feature_type_id: max_clients, value: 20}

// Plan describes the value of the feature. Each plan will have a feature and a designated value. There are only three plans at most, but with different combination of features.
plan
- id
- name // The name of the plan (basic, elite, enterprise)
- description // The description of the plan.
- billing_method_type (auto, manual (?) better naming please)
- cost
- valid_from 
- valid_till // If we are going to deprecate a plan…

plan_feature
- plan_id
- feature_id
- cost
- duration_feature (yearly/monthly)

subscription plan
- subscription_id
- plan_id
- valid_from
- valid_till
- superseded_by (the previous subscription plan)
- // NOTE: This can be part of the feature.
- // country (subscription is different per country)
- // currency (currency is different per country)
- // cost (the cost depends on currency)
- // duration_feature (yearly/monthly)
- // is_renewable (?) can just check the valid_till date
- // status (?)


invoice 
- subscription_plan_id
- paid_amount (probably need this to offset the upgrade)
- amount
- for_date (what month/year is this invoice for?)
```

## Monthly subscription

https://stackoverflow.com/questions/23507200/good-practices-for-designing-monthly-subscription-system-in-database
https://vertabelo.com/blog/creating-a-dwh-part-one-a-subscription-business-data-model/
https://docs.microsoft.com/en-us/sql/reporting-services/lesson-1-creating-a-sample-subscriber-database?view=sql-server-ver15
https://softwareengineering.stackexchange.com/questions/361940/how-should-i-go-about-creating-a-db-schema-for-news-subscription-and-connectin
