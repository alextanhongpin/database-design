# Payment System Database Design

## Table of Contents
- [Overview](#overview)
- [Core Payment Tables](#core-payment-tables)
- [Payment Methods](#payment-methods)
- [Transaction Management](#transaction-management)
- [Subscription Billing](#subscription-billing)
- [E-commerce Integration](#e-commerce-integration)
- [Financial Compliance](#financial-compliance)
- [Advanced Patterns](#advanced-patterns)
- [Performance & Security](#performance--security)
- [Best Practices](#best-practices)

## Overview

Payment systems require careful design to handle money safely, maintain audit trails, support multiple payment methods, and comply with financial regulations. This guide covers common patterns for building robust payment databases.

### Key Requirements
- **ACID Compliance**: All financial transactions must be atomic
- **Audit Trail**: Complete history of all payment activities
- **Idempotency**: Prevent duplicate charges
- **Multi-currency**: Support international payments
- **PCI Compliance**: Secure handling of payment data
- **Reconciliation**: Match payments with external systems

## Core Payment Tables

### Basic Payment Schema

```sql
-- Payment accounts (users, merchants, system accounts)
CREATE TABLE payment_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_type account_type NOT NULL,
    user_id UUID REFERENCES users(id),
    merchant_id UUID REFERENCES merchants(id),
    
    -- Account details
    account_number VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Status and metadata
    status account_status DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure only one type of owner
    CHECK (
        (user_id IS NOT NULL AND merchant_id IS NULL) OR
        (user_id IS NULL AND merchant_id IS NOT NULL) OR
        (user_id IS NULL AND merchant_id IS NULL AND account_type = 'system')
    )
);

-- Account types
CREATE TYPE account_type AS ENUM ('user', 'merchant', 'system', 'escrow');
CREATE TYPE account_status AS ENUM ('active', 'suspended', 'closed');

-- Payment transactions (double-entry bookkeeping)
CREATE TABLE payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Transaction identification
    external_id VARCHAR(255) UNIQUE, -- For idempotency
    transaction_type transaction_type NOT NULL,
    reference_type VARCHAR(50), -- order, subscription, refund, etc.
    reference_id UUID,
    
    -- Financial details
    gross_amount DECIMAL(19,4) NOT NULL CHECK (gross_amount > 0),
    fee_amount DECIMAL(19,4) DEFAULT 0 CHECK (fee_amount >= 0),
    net_amount DECIMAL(19,4) GENERATED ALWAYS AS (gross_amount - fee_amount) STORED,
    currency_code CHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Status and timing
    status transaction_status DEFAULT 'pending',
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Idempotency and audit
    created_by UUID NOT NULL,
    notes TEXT,
    
    -- Indexes
    INDEX idx_external_id (external_id),
    INDEX idx_reference (reference_type, reference_id),
    INDEX idx_status_created (status, created_at)
);

-- Transaction types
CREATE TYPE transaction_type AS ENUM (
    'payment', 'refund', 'chargeback', 'fee', 
    'payout', 'adjustment', 'transfer'
);

CREATE TYPE transaction_status AS ENUM (
    'pending', 'processing', 'completed', 'failed', 
    'cancelled', 'disputed', 'settled'
);

-- Ledger entries (double-entry accounting)
CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES payment_transactions(id),
    account_id UUID NOT NULL REFERENCES payment_accounts(id),
    
    -- Entry details
    entry_type entry_type NOT NULL,
    amount DECIMAL(19,4) NOT NULL CHECK (amount != 0),
    currency_code CHAR(3) NOT NULL,
    
    -- Balance tracking
    running_balance DECIMAL(19,4),
    
    -- Metadata
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_transaction (transaction_id),
    INDEX idx_account_created (account_id, created_at),
    INDEX idx_account_type (account_id, entry_type)
);

CREATE TYPE entry_type AS ENUM ('debit', 'credit');

-- Trigger to maintain running balance
CREATE OR REPLACE FUNCTION update_running_balance()
RETURNS TRIGGER AS $$
DECLARE
    prev_balance DECIMAL(19,4) := 0;
BEGIN
    -- Get previous balance for this account
    SELECT COALESCE(running_balance, 0) INTO prev_balance
    FROM ledger_entries
    WHERE account_id = NEW.account_id
      AND created_at < NEW.created_at
    ORDER BY created_at DESC, id DESC
    LIMIT 1;
    
    -- Calculate new running balance
    IF NEW.entry_type = 'credit' THEN
        NEW.running_balance := prev_balance + NEW.amount;
    ELSE
        NEW.running_balance := prev_balance - NEW.amount;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ledger_balance_trigger
    BEFORE INSERT ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION update_running_balance();
```

## Payment Methods

### Payment Method Management

```sql
-- Customer payment methods
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Method details
    method_type payment_method_type NOT NULL,
    provider VARCHAR(50) NOT NULL, -- stripe, paypal, braintree, etc.
    
    -- Tokenized payment data (PCI compliant)
    provider_token VARCHAR(255) NOT NULL,
    
    -- Display information (safe to store)
    display_name VARCHAR(255),
    last_four VARCHAR(10),
    brand VARCHAR(50), -- visa, mastercard, paypal, etc.
    expiry_month INTEGER,
    expiry_year INTEGER,
    
    -- Billing address
    billing_address_id UUID REFERENCES addresses(id),
    
    -- Status and metadata
    is_default BOOLEAN DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    status method_status DEFAULT 'active',
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    UNIQUE (user_id, provider_token),
    INDEX idx_user_default (user_id, is_default) WHERE is_default = TRUE,
    INDEX idx_user_active (user_id, status) WHERE status = 'active'
);

CREATE TYPE payment_method_type AS ENUM (
    'credit_card', 'debit_card', 'bank_account', 
    'digital_wallet', 'crypto', 'buy_now_pay_later'
);

CREATE TYPE method_status AS ENUM ('active', 'expired', 'disabled');

-- Ensure only one default payment method per user
CREATE UNIQUE INDEX idx_user_single_default 
ON payment_methods (user_id) 
WHERE is_default = TRUE;
```

### Payment Processing

```sql
-- Payment attempts and retries
CREATE TABLE payment_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES payment_transactions(id),
    payment_method_id UUID NOT NULL REFERENCES payment_methods(id),
    
    -- Attempt details
    attempt_number INTEGER NOT NULL DEFAULT 1,
    amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    -- Provider details
    provider VARCHAR(50) NOT NULL,
    provider_transaction_id VARCHAR(255),
    provider_response JSONB,
    
    -- Status and timing
    status attempt_status NOT NULL,
    attempted_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    -- Error handling
    error_code VARCHAR(100),
    error_message TEXT,
    is_retryable BOOLEAN DEFAULT FALSE,
    
    -- Indexes
    INDEX idx_transaction_attempt (transaction_id, attempt_number),
    INDEX idx_payment_method (payment_method_id),
    INDEX idx_provider_transaction (provider, provider_transaction_id)
);

CREATE TYPE attempt_status AS ENUM (
    'pending', 'processing', 'succeeded', 'failed', 'cancelled'
);

-- Payment processing function
CREATE OR REPLACE FUNCTION process_payment(
    user_id UUID,
    payment_method_id UUID,
    amount DECIMAL(19,4),
    currency_code CHAR(3),
    reference_type VARCHAR(50),
    reference_id UUID,
    external_id VARCHAR(255) DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    transaction_id UUID;
    user_account_id UUID;
    merchant_account_id UUID;
    fee_amount DECIMAL(19,4) := amount * 0.029 + 0.30; -- Example fee structure
BEGIN
    -- Generate external_id if not provided
    IF external_id IS NULL THEN
        external_id := 'txn_' || gen_random_uuid()::text;
    END IF;
    
    -- Check for duplicate transaction
    IF EXISTS (SELECT 1 FROM payment_transactions WHERE external_id = external_id) THEN
        RAISE EXCEPTION 'Duplicate transaction: %', external_id;
    END IF;
    
    -- Get account IDs
    SELECT id INTO user_account_id 
    FROM payment_accounts 
    WHERE user_id = process_payment.user_id AND account_type = 'user';
    
    SELECT id INTO merchant_account_id 
    FROM payment_accounts 
    WHERE account_type = 'merchant' AND status = 'active' 
    LIMIT 1;
    
    -- Create transaction
    INSERT INTO payment_transactions (
        external_id, transaction_type, reference_type, reference_id,
        gross_amount, fee_amount, currency_code, status, created_by
    ) VALUES (
        external_id, 'payment', reference_type, reference_id,
        amount, fee_amount, currency_code, 'pending', user_id
    ) RETURNING id INTO transaction_id;
    
    -- Create ledger entries (double-entry)
    INSERT INTO ledger_entries (transaction_id, account_id, entry_type, amount, currency_code, description)
    VALUES 
        (transaction_id, user_account_id, 'credit', amount, currency_code, 'Payment received'),
        (transaction_id, merchant_account_id, 'debit', amount - fee_amount, currency_code, 'Payment processed'),
        (transaction_id, (SELECT id FROM payment_accounts WHERE account_type = 'system' LIMIT 1), 'debit', fee_amount, currency_code, 'Processing fee');
    
    -- Create payment attempt
    INSERT INTO payment_attempts (
        transaction_id, payment_method_id, amount, currency_code, 
        provider, status
    ) VALUES (
        transaction_id, payment_method_id, amount, currency_code,
        (SELECT provider FROM payment_methods WHERE id = payment_method_id),
        'pending'
    );
    
    RETURN transaction_id;
END;
$$ LANGUAGE plpgsql;
```

## Transaction Management

### Refunds and Reversals

```sql
-- Refund management
CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_transaction_id UUID NOT NULL REFERENCES payment_transactions(id),
    refund_transaction_id UUID REFERENCES payment_transactions(id),
    
    -- Refund details
    refund_amount DECIMAL(19,4) NOT NULL CHECK (refund_amount > 0),
    refund_reason refund_reason,
    
    -- Status and timing
    status refund_status DEFAULT 'pending',
    requested_by UUID NOT NULL,
    requested_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    
    -- Metadata
    notes TEXT,
    admin_notes TEXT,
    
    -- Constraints
    CHECK (refund_amount <= (
        SELECT gross_amount FROM payment_transactions 
        WHERE id = original_transaction_id
    ))
);

CREATE TYPE refund_reason AS ENUM (
    'customer_request', 'fraud', 'duplicate', 'error', 
    'chargeback', 'quality_issue', 'cancelled_order'
);

CREATE TYPE refund_status AS ENUM (
    'pending', 'approved', 'processing', 'completed', 
    'failed', 'rejected'
);

-- Function to process refunds
CREATE OR REPLACE FUNCTION process_refund(
    original_transaction_id UUID,
    refund_amount DECIMAL(19,4),
    reason refund_reason,
    requested_by UUID,
    notes TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    refund_id UUID;
    refund_transaction_id UUID;
    original_amount DECIMAL(19,4);
    currency_code CHAR(3);
    user_account_id UUID;
    merchant_account_id UUID;
BEGIN
    -- Get original transaction details
    SELECT gross_amount, pt.currency_code INTO original_amount, currency_code
    FROM payment_transactions pt
    WHERE pt.id = original_transaction_id;
    
    -- Validate refund amount
    IF refund_amount > original_amount THEN
        RAISE EXCEPTION 'Refund amount cannot exceed original payment amount';
    END IF;
    
    -- Create refund record
    INSERT INTO refunds (
        original_transaction_id, refund_amount, refund_reason,
        requested_by, notes
    ) VALUES (
        original_transaction_id, refund_amount, reason,
        requested_by, notes
    ) RETURNING id INTO refund_id;
    
    -- Create refund transaction
    INSERT INTO payment_transactions (
        external_id, transaction_type, reference_type, reference_id,
        gross_amount, currency_code, status, created_by
    ) VALUES (
        'refund_' || refund_id::text, 'refund', 'refund', refund_id,
        refund_amount, currency_code, 'pending', requested_by
    ) RETURNING id INTO refund_transaction_id;
    
    -- Update refund with transaction ID
    UPDATE refunds SET refund_transaction_id = refund_transaction_id
    WHERE id = refund_id;
    
    -- Create ledger entries
    SELECT id INTO user_account_id FROM payment_accounts 
    WHERE user_id = (SELECT created_by FROM payment_transactions WHERE id = original_transaction_id);
    
    SELECT id INTO merchant_account_id FROM payment_accounts 
    WHERE account_type = 'merchant' LIMIT 1;
    
    INSERT INTO ledger_entries (transaction_id, account_id, entry_type, amount, currency_code, description)
    VALUES 
        (refund_transaction_id, user_account_id, 'debit', refund_amount, currency_code, 'Refund processed'),
        (refund_transaction_id, merchant_account_id, 'credit', refund_amount, currency_code, 'Refund issued');
    
    RETURN refund_id;
END;
$$ LANGUAGE plpgsql;
```

## Subscription Billing

### Subscription Management

```sql
-- Subscription plans
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Plan details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    billing_cycle billing_cycle NOT NULL,
    
    -- Pricing
    base_price DECIMAL(19,4) NOT NULL CHECK (base_price >= 0),
    setup_fee DECIMAL(19,4) DEFAULT 0 CHECK (setup_fee >= 0),
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Features and limits
    features JSONB,
    usage_limits JSONB,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_active_plans (is_active, billing_cycle)
);

CREATE TYPE billing_cycle AS ENUM ('monthly', 'quarterly', 'annually', 'weekly');

-- User subscriptions
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    payment_method_id UUID REFERENCES payment_methods(id),
    
    -- Billing details
    current_price DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    billing_cycle billing_cycle NOT NULL,
    
    -- Subscription period
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    current_period_start TIMESTAMP NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMP NOT NULL,
    next_billing_date TIMESTAMP NOT NULL,
    
    -- Status
    status subscription_status DEFAULT 'active',
    cancelled_at TIMESTAMP,
    ends_at TIMESTAMP,
    
    -- Trial period
    trial_start TIMESTAMP,
    trial_end TIMESTAMP,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_user_active (user_id, status),
    INDEX idx_billing_date (next_billing_date, status)
);

CREATE TYPE subscription_status AS ENUM (
    'trialing', 'active', 'past_due', 'cancelled', 'unpaid', 'paused'
);

-- Subscription billing history
CREATE TABLE subscription_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- Invoice details
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    billing_period_start TIMESTAMP NOT NULL,
    billing_period_end TIMESTAMP NOT NULL,
    
    -- Amounts
    subtotal DECIMAL(19,4) NOT NULL,
    tax_amount DECIMAL(19,4) DEFAULT 0,
    discount_amount DECIMAL(19,4) DEFAULT 0,
    total_amount DECIMAL(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    -- Payment
    payment_transaction_id UUID REFERENCES payment_transactions(id),
    payment_attempted_at TIMESTAMP,
    paid_at TIMESTAMP,
    
    -- Status
    status invoice_status DEFAULT 'draft',
    due_date TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_subscription (subscription_id, created_at),
    INDEX idx_status_due (status, due_date)
);

CREATE TYPE invoice_status AS ENUM (
    'draft', 'open', 'paid', 'void', 'uncollectible'
);
```

## E-commerce Integration

### Order Payment Integration

```sql
-- Orders table (simplified)
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    
    -- Order amounts
    subtotal DECIMAL(19,4) NOT NULL,
    tax_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    shipping_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    total_amount DECIMAL(19,4) GENERATED ALWAYS AS (
        subtotal + tax_amount + shipping_amount - discount_amount
    ) STORED,
    currency_code CHAR(3) DEFAULT 'USD',
    
    -- Payment
    payment_status payment_status DEFAULT 'pending',
    payment_method_id UUID REFERENCES payment_methods(id),
    
    -- Status
    order_status order_status DEFAULT 'pending',
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TYPE payment_status AS ENUM (
    'pending', 'authorized', 'captured', 'partially_refunded', 
    'refunded', 'failed', 'cancelled'
);

CREATE TYPE order_status AS ENUM (
    'pending', 'confirmed', 'processing', 'shipped', 
    'delivered', 'cancelled', 'returned'
);

-- Payment authorization and capture
CREATE TABLE payment_authorizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    payment_method_id UUID NOT NULL REFERENCES payment_methods(id),
    
    -- Authorization details
    authorized_amount DECIMAL(19,4) NOT NULL,
    captured_amount DECIMAL(19,4) DEFAULT 0,
    remaining_amount DECIMAL(19,4) GENERATED ALWAYS AS (
        authorized_amount - captured_amount
    ) STORED,
    
    currency_code CHAR(3) NOT NULL,
    
    -- Provider details
    provider VARCHAR(50) NOT NULL,
    provider_auth_id VARCHAR(255),
    
    -- Status and timing
    status authorization_status DEFAULT 'pending',
    authorized_at TIMESTAMP,
    expires_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_order (order_id),
    INDEX idx_expiry (expires_at, status)
);

CREATE TYPE authorization_status AS ENUM (
    'pending', 'authorized', 'expired', 'cancelled', 'captured'
);

-- Capture payments when order ships
CREATE OR REPLACE FUNCTION capture_payment(
    auth_id UUID,
    capture_amount DECIMAL(19,4)
) RETURNS UUID AS $$
DECLARE
    transaction_id UUID;
    auth_record RECORD;
BEGIN
    -- Get authorization details
    SELECT * INTO auth_record
    FROM payment_authorizations
    WHERE id = auth_id AND status = 'authorized';
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Invalid or expired authorization';
    END IF;
    
    IF capture_amount > auth_record.remaining_amount THEN
        RAISE EXCEPTION 'Capture amount exceeds remaining authorized amount';
    END IF;
    
    -- Create payment transaction
    INSERT INTO payment_transactions (
        external_id, transaction_type, reference_type, reference_id,
        gross_amount, currency_code, status, created_by
    ) VALUES (
        'capture_' || auth_id::text || '_' || extract(epoch from now())::text,
        'payment', 'order', auth_record.order_id,
        capture_amount, auth_record.currency_code, 'completed',
        (SELECT user_id FROM orders WHERE id = auth_record.order_id)
    ) RETURNING id INTO transaction_id;
    
    -- Update authorization
    UPDATE payment_authorizations
    SET captured_amount = captured_amount + capture_amount,
        status = CASE 
            WHEN captured_amount + capture_amount >= authorized_amount THEN 'captured'
            ELSE 'authorized'
        END
    WHERE id = auth_id;
    
    -- Update order payment status
    UPDATE orders
    SET payment_status = 'captured'
    WHERE id = auth_record.order_id;
    
    RETURN transaction_id;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Security and Compliance
- **Never store sensitive payment data**: Use tokenization
- **Implement PCI DSS compliance**: Follow payment card industry standards
- **Use HTTPS everywhere**: Encrypt all payment communications
- **Log all activities**: Maintain comprehensive audit trails
- **Implement fraud detection**: Monitor for suspicious patterns

### 2. Data Integrity
- **Use double-entry bookkeeping**: Every transaction affects two accounts
- **Implement idempotency**: Prevent duplicate charges
- **Validate all amounts**: Use appropriate precision for currency
- **Maintain referential integrity**: Use foreign keys properly
- **Handle currency conversion**: Store original and converted amounts

### 3. Performance Optimization
- **Index payment queries**: Focus on status, user, and date ranges
- **Partition large tables**: By date or account for better performance
- **Use read replicas**: For reporting and analytics
- **Cache frequently accessed data**: Payment methods, balances
- **Archive old transactions**: Move historical data to separate tables

### 4. Error Handling
- **Implement retry logic**: For failed payment attempts
- **Handle timeouts gracefully**: Set appropriate timeouts for external APIs
- **Provide clear error messages**: Help users understand payment failures
- **Log all errors**: For debugging and monitoring
- **Implement circuit breakers**: Prevent cascading failures

### 5. Monitoring and Alerting
- **Track payment success rates**: Monitor for declining performance
- **Alert on failed payments**: Immediate notification for critical failures
- **Monitor for fraud**: Unusual patterns or high-risk transactions
- **Track reconciliation**: Ensure all payments match external records
- **Monitor compliance**: Regular audits of payment processes

This payment system design provides a robust foundation for handling various payment scenarios while maintaining security, compliance, and scalability requirements.
