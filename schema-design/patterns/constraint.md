# Database Constraints: Comprehensive Guide

Database constraints are essential for maintaining data integrity and enforcing business rules at the database level. This guide covers various types of constraints, their practical applications, and real-world implementation patterns.

## Table of Contents
- [Check Constraints](#check-constraints)
- [Multi-Column Constraints](#multi-column-constraints) 
- [Advanced Constraint Patterns](#advanced-constraint-patterns)
- [Constraint Management](#constraint-management)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Check Constraints

### Basic Check Constraints

Check constraints validate data before it's inserted or updated. Here's a comprehensive example:
```sql
-- E-commerce order status constraint
CREATE TABLE orders (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    status TEXT NOT NULL CHECK (
        status IN ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled', 'refunded')
    ),
    total_amount DECIMAL(10,2) CHECK (total_amount > 0),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User registration with email validation
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL CHECK (email ~* '^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    age INTEGER CHECK (age >= 13 AND age <= 120),
    status TEXT CHECK (status IN ('active', 'inactive', 'suspended'))
);

-- Product inventory constraint
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) CHECK (price >= 0),
    stock_quantity INTEGER CHECK (stock_quantity >= 0),
    discount_percentage DECIMAL(5,2) CHECK (discount_percentage >= 0 AND discount_percentage <= 100)
);
```

### Managing Check Constraints

#### Finding Existing Constraints
```sql
-- Query to find all check constraints in your database
SELECT 
    pgc.conname AS constraint_name,
    ccu.table_schema AS table_schema,
    ccu.table_name,
    ccu.column_name,
    pg_get_constraintdef(pgc.oid) AS constraint_definition
FROM pg_constraint pgc
JOIN pg_namespace nsp ON nsp.oid = pgc.connamespace
JOIN pg_class cls ON pgc.conrelid = cls.oid
LEFT JOIN information_schema.constraint_column_usage ccu
    ON pgc.conname = ccu.constraint_name
    AND nsp.nspname = ccu.constraint_schema
WHERE contype = 'c'
ORDER BY pgc.conname;
```

#### Updating Check Constraints

Since PostgreSQL doesn't support modifying constraints directly, you must drop and recreate them:

```sql
-- Example: Adding a new order status 'processing'
ALTER TABLE orders
DROP CONSTRAINT IF EXISTS orders_status_check,
ADD CONSTRAINT orders_status_check CHECK (
    status IN ('pending', 'processing', 'confirmed', 'shipped', 'delivered', 'cancelled', 'refunded')
);

-- Using a transaction for safety
BEGIN;
    ALTER TABLE orders DROP CONSTRAINT orders_status_check;
    ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (
        status IN ('pending', 'processing', 'confirmed', 'shipped', 'delivered', 'cancelled', 'refunded')
    );
COMMIT;
```

### Constraint Evolution Challenges

**Problem**: You cannot remove constraint values if existing data violates the new constraint.

```sql
-- Setup example table
CREATE TABLE product_types (
    id SERIAL PRIMARY KEY,
    type TEXT CHECK (type IN ('electronics', 'clothing', 'books'))
);

INSERT INTO product_types (type) VALUES ('electronics'), ('clothing'), ('books');

-- This will fail if 'books' records exist
ALTER TABLE product_types
DROP CONSTRAINT product_types_type_check,
ADD CONSTRAINT product_types_type_check CHECK (type IN ('electronics', 'clothing'));
-- ERROR: check constraint is violated by some row

-- Solutions:
-- 1. Migrate existing data first
UPDATE product_types SET type = 'other' WHERE type = 'books';

-- 2. Use a more flexible approach with lookup tables
CREATE TABLE product_categories (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category_id INTEGER REFERENCES product_categories(id)
);

-- This allows dynamic category management without schema changes
```

## Multi-Column Constraints

### All-or-None Patterns


**Use Case**: Multiple columns must be set together or not at all.

```sql
-- E-commerce: Product with price must have currency
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2),
    currency CHAR(3),
    
    -- Both price and currency must exist together
    CONSTRAINT price_with_currency CHECK ((price IS NULL) = (currency IS NULL))
);

-- Shipping address: All address fields required together  
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_email TEXT NOT NULL,
    
    -- Shipping address fields (all or none)
    shipping_street TEXT,
    shipping_city TEXT,
    shipping_state TEXT,
    shipping_zip TEXT,
    shipping_country TEXT,
    
    CONSTRAINT complete_shipping_address CHECK (
        (shipping_street, shipping_city, shipping_state, shipping_zip, shipping_country) IS NULL OR
        (shipping_street, shipping_city, shipping_state, shipping_zip, shipping_country) IS NOT NULL
    )
);

-- Alternative using modulo approach for multiple columns
CREATE TABLE payment_methods (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    
    -- Credit card details (all 4 fields required together)
    card_number TEXT,
    card_expiry TEXT,
    card_cvv TEXT,
    card_holder_name TEXT,
    
    CONSTRAINT complete_card_details CHECK (
        num_nonnulls(card_number, card_expiry, card_cvv, card_holder_name) % 4 = 0
    )
);
```

### XOR Patterns (Exclusive OR)

**Use Case**: Exactly one of multiple columns must be set (exclusive choice).

```sql
-- Document approval: Either internal OR external approval, never both
CREATE TABLE document_approvals (
    id SERIAL PRIMARY KEY,
    document_id INTEGER NOT NULL,
    
    internal_approved_at TIMESTAMPTZ,
    internal_approved_by INTEGER,
    external_approved_at TIMESTAMPTZ,
    external_approved_by TEXT,
    
    remarks TEXT,
    
    -- Exactly one approval type must be set
    CONSTRAINT exclusive_approval CHECK (
        (internal_approved_at IS NOT NULL)::INTEGER + 
        (external_approved_at IS NOT NULL)::INTEGER = 1
    ),
    
    -- If internal approval is set, internal approver must be set
    CONSTRAINT internal_approval_complete CHECK (
        (internal_approved_at IS NULL) = (internal_approved_by IS NULL)
    ),
    
    -- If external approval is set, external approver must be set  
    CONSTRAINT external_approval_complete CHECK (
        (external_approved_at IS NULL) = (external_approved_by IS NULL)
    )
);

-- Social media polymorphic likes: Like exactly one type of content
CREATE TABLE content_likes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    
    -- Polymorphic associations - exactly one must be set
    post_id INTEGER,
    comment_id INTEGER,
    photo_id INTEGER,
    video_id INTEGER,
    story_id INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Exactly one content type must be liked
    CONSTRAINT like_one_content_type CHECK (
        num_nonnulls(post_id, comment_id, photo_id, video_id, story_id) = 1
    ),
    
    -- Prevent duplicate likes for same user/content
    UNIQUE(user_id, post_id),
    UNIQUE(user_id, comment_id),
    UNIQUE(user_id, photo_id),
    UNIQUE(user_id, video_id),
    UNIQUE(user_id, story_id)
);

-- Payment methods: One payment type per transaction
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    
    -- Payment method details (exactly one must be used)
    credit_card_id INTEGER,
    paypal_account TEXT,
    bank_transfer_ref TEXT,
    gift_card_code TEXT,
    cryptocurrency_tx TEXT,
    
    processed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Exactly one payment method must be used
    CONSTRAINT one_payment_method CHECK (
        num_nonnulls(credit_card_id, paypal_account, bank_transfer_ref, 
                    gift_card_code, cryptocurrency_tx) = 1
    )
);
```

## Advanced Constraint Patterns

### Temporal Constraints

```sql
-- Event scheduling: Event must end after it starts
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    
    -- Temporal validation
    CONSTRAINT valid_event_duration CHECK (end_time > start_time),
    CONSTRAINT reasonable_duration CHECK (end_time - start_time <= INTERVAL '7 days'),
    CONSTRAINT future_events CHECK (start_time > NOW() - INTERVAL '1 hour') -- Allow some flexibility
);

-- Employee employment periods
CREATE TABLE employment_history (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    position TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    
    -- Employment period validation
    CONSTRAINT valid_employment_period CHECK (
        end_date IS NULL OR end_date >= start_date
    ),
    CONSTRAINT reasonable_employment CHECK (
        end_date IS NULL OR (end_date - start_date) <= INTERVAL '50 years'
    )
);
```

### Range and Boundary Constraints

```sql
-- Product ratings and reviews
CREATE TABLE product_reviews (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review_text TEXT,
    
    -- Review length constraints
    CONSTRAINT meaningful_review CHECK (
        review_text IS NULL OR LENGTH(review_text) >= 10
    ),
    CONSTRAINT reasonable_review_length CHECK (
        review_text IS NULL OR LENGTH(review_text) <= 5000
    ),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Subscription tiers with pricing
CREATE TABLE subscription_plans (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    monthly_price DECIMAL(10,2) NOT NULL CHECK (monthly_price >= 0),
    yearly_price DECIMAL(10,2) NOT NULL CHECK (yearly_price >= 0),
    max_users INTEGER CHECK (max_users > 0),
    max_storage_gb INTEGER CHECK (max_storage_gb > 0),
    
    -- Yearly pricing should offer discount
    CONSTRAINT yearly_discount CHECK (yearly_price < monthly_price * 12),
    
    -- Reasonable limits
    CONSTRAINT reasonable_user_limit CHECK (max_users <= 100000),
    CONSTRAINT reasonable_storage_limit CHECK (max_storage_gb <= 100000)
);
```

### Business Logic Constraints

```sql
-- Order workflow constraints
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN (
        'draft', 'pending', 'confirmed', 'processing', 
        'shipped', 'delivered', 'cancelled', 'refunded'
    )),
    total_amount DECIMAL(10,2) NOT NULL CHECK (total_amount > 0),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    -- Business logic constraints
    CONSTRAINT confirmed_orders_have_confirmation_time CHECK (
        (status IN ('draft', 'pending', 'cancelled')) OR confirmed_at IS NOT NULL
    ),
    CONSTRAINT shipped_orders_are_confirmed CHECK (
        shipped_at IS NULL OR confirmed_at IS NOT NULL
    ),
    CONSTRAINT delivered_orders_are_shipped CHECK (
        delivered_at IS NULL OR shipped_at IS NOT NULL
    ),
    CONSTRAINT cancelled_orders_not_delivered CHECK (
        cancelled_at IS NULL OR delivered_at IS NULL
    ),
    
    -- Temporal ordering
    CONSTRAINT logical_order_flow CHECK (
        (confirmed_at IS NULL OR confirmed_at >= created_at) AND
        (shipped_at IS NULL OR shipped_at >= confirmed_at) AND
        (delivered_at IS NULL OR delivered_at >= shipped_at)
    )
);
```

## Row-Level Constraints

### Using Exclusion Constraints

```sql
-- Prevent overlapping bookings
CREATE TABLE room_bookings (
    id SERIAL PRIMARY KEY,
    room_id INTEGER NOT NULL,
    guest_name TEXT NOT NULL,
    check_in DATE NOT NULL,
    check_out DATE NOT NULL,
    
    CONSTRAINT valid_booking_period CHECK (check_out > check_in),
    
    -- Prevent overlapping bookings for the same room
    EXCLUDE USING gist (
        room_id WITH =,
        daterange(check_in, check_out, '[]') WITH &&
    )
);

-- Employee shift scheduling without conflicts
CREATE TABLE employee_shifts (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    shift_start TIMESTAMPTZ NOT NULL,
    shift_end TIMESTAMPTZ NOT NULL,
    
    CONSTRAINT valid_shift_duration CHECK (shift_end > shift_start),
    CONSTRAINT reasonable_shift_length CHECK (
        shift_end - shift_start <= INTERVAL '12 hours'
    ),
    
    -- Prevent overlapping shifts for the same employee
    EXCLUDE USING gist (
        employee_id WITH =,
        tstzrange(shift_start, shift_end, '[]') WITH &&
    )
);
```

### Partial Unique Constraints

```sql
-- Active subscriptions: Only one active subscription per user per plan type
CREATE TABLE user_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    plan_type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'cancelled', 'expired')),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    
    -- Only one active subscription per user per plan type
    UNIQUE (user_id, plan_type) WHERE status = 'active'
);

-- User emails: Only one primary email per user
CREATE TABLE user_emails (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    email TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    
    -- Only one primary email per user
    UNIQUE (user_id) WHERE is_primary = TRUE,
    
    -- Email must be unique across all users
    UNIQUE (email)
);
```

## Constraint Management with Triggers

For complex business rules that can't be expressed with simple constraints, use triggers:

```sql
-- Create a function to validate complex business rules
CREATE OR REPLACE FUNCTION validate_order_transition()
RETURNS TRIGGER AS $$
BEGIN
    -- Prevent status regression (can't go back to earlier states)
    IF OLD.status = 'delivered' AND NEW.status NOT IN ('delivered', 'refunded') THEN
        RAISE EXCEPTION 'Cannot change status from delivered to %', NEW.status;
    END IF;
    
    -- Prevent cancellation after shipping
    IF OLD.status IN ('shipped', 'delivered') AND NEW.status = 'cancelled' THEN
        RAISE EXCEPTION 'Cannot cancel order that has been shipped';
    END IF;
    
    -- Validate amount changes
    IF OLD.status = 'confirmed' AND NEW.total_amount != OLD.total_amount THEN
        RAISE EXCEPTION 'Cannot modify amount after order confirmation';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
CREATE TRIGGER order_transition_validation
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION validate_order_transition();
```

### Trigger-Based Row Limits

```sql
-- Limit number of active sessions per user
CREATE OR REPLACE FUNCTION limit_active_sessions()
RETURNS TRIGGER AS $$
DECLARE
    session_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO session_count
    FROM user_sessions
    WHERE user_id = NEW.user_id AND status = 'active';
    
    IF session_count >= 5 THEN
        RAISE EXCEPTION 'User cannot have more than 5 active sessions';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER limit_user_sessions
    BEFORE INSERT ON user_sessions
    FOR EACH ROW
    EXECUTE FUNCTION limit_active_sessions();
```

### Conditional Triggers

Triggers can be enabled/disabled based on application needs:

```sql
-- Disable trigger temporarily for data migration
ALTER TABLE orders DISABLE TRIGGER order_transition_validation;

-- Perform data migration
UPDATE orders SET status = 'migrated' WHERE legacy_flag = true;

-- Re-enable trigger
ALTER TABLE orders ENABLE TRIGGER order_transition_validation;
```

## Performance Considerations

### Indexing Strategy for Constraints

```sql
-- Index columns used in CHECK constraints for better performance
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_active_status ON orders(status) WHERE status IN ('pending', 'confirmed', 'processing');

-- Partial indexes for constraint validation
CREATE INDEX idx_active_subscriptions ON user_subscriptions(user_id, plan_type) 
WHERE status = 'active';

-- Composite indexes for multi-column constraints
CREATE INDEX idx_product_price_currency ON products(price, currency)
WHERE price IS NOT NULL;
```

### Constraint Validation Performance

```sql
-- Use EXPLAIN to analyze constraint validation performance
EXPLAIN ANALYZE INSERT INTO orders (customer_id, status, total_amount)
VALUES (1, 'pending', 99.99);

-- Consider constraint validation order
-- Put most selective constraints first
ALTER TABLE products ADD CONSTRAINT valid_product_data CHECK (
    price > 0 AND                    -- Most common validation failure
    stock_quantity >= 0 AND          -- Second most common
    discount_percentage BETWEEN 0 AND 100  -- Least common
);
```

## Best Practices

### 1. Constraint Naming Convention

```sql
-- Use descriptive names that indicate the constraint purpose
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    status TEXT NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    
    CONSTRAINT orders_status_valid CHECK (status IN ('pending', 'confirmed', 'shipped')),
    CONSTRAINT orders_amount_positive CHECK (total_amount > 0),
    CONSTRAINT orders_amount_reasonable CHECK (total_amount <= 1000000)
);
```

### 2. Layered Validation

```sql
-- Database constraints for data integrity (must never be violated)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL CHECK (email LIKE '%@%'),  -- Basic format
    age INTEGER CHECK (age >= 0 AND age <= 150),   -- Physical constraints
    
    UNIQUE(email)
);

-- Application-level validation for business rules (can be bypassed for admin operations)
-- - Complex email regex validation
-- - Age restrictions based on service terms
-- - Custom business rules that might change
```

### 3. Constraint Documentation

```sql
-- Document complex constraints with comments
CREATE TABLE financial_transactions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    transaction_type TEXT NOT NULL,
    
    -- Withdrawal amounts must be negative, deposits positive
    CONSTRAINT valid_transaction_amount CHECK (
        (transaction_type = 'withdrawal' AND amount < 0) OR
        (transaction_type = 'deposit' AND amount > 0)
    ),
    
    -- Daily transaction limits for security
    CONSTRAINT reasonable_daily_amount CHECK (
        ABS(amount) <= 50000.00  -- $50,000 daily limit
    )
);

COMMENT ON CONSTRAINT valid_transaction_amount ON financial_transactions 
IS 'Ensures withdrawal amounts are negative and deposit amounts are positive';
```

### 4. Constraint Testing

```sql
-- Test constraint validation with edge cases
DO $$
BEGIN
    -- Test valid data
    INSERT INTO products (name, price, currency) VALUES ('Test Product', 19.99, 'USD');
    
    -- Test constraint violations
    BEGIN
        INSERT INTO products (name, price, currency) VALUES ('Invalid Product', 19.99, NULL);
        RAISE EXCEPTION 'Expected constraint violation did not occur';
    EXCEPTION WHEN check_violation THEN
        RAISE NOTICE 'Constraint correctly prevented invalid data';
    END;
    
    -- Cleanup
    DELETE FROM products WHERE name LIKE 'Test Product%';
END $$;
```

### 5. Migration Strategy

```sql
-- Safe constraint addition with validation
-- Step 1: Add constraint as NOT VALID (doesn't check existing data)
ALTER TABLE orders ADD CONSTRAINT orders_amount_positive 
CHECK (total_amount > 0) NOT VALID;

-- Step 2: Validate constraint on existing data
ALTER TABLE orders VALIDATE CONSTRAINT orders_amount_positive;

-- Step 3: If validation fails, fix data and retry
UPDATE orders SET total_amount = 0.01 WHERE total_amount <= 0;
ALTER TABLE orders VALIDATE CONSTRAINT orders_amount_positive;
```

## Common Anti-Patterns to Avoid

### 1. Over-constraining

```sql
-- DON'T: Too many restrictive constraints
CREATE TABLE users (
    email TEXT CHECK (email ~* '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'),
    phone TEXT CHECK (phone ~ '^\+?[1-9]\d{1,14}$'),
    age INTEGER CHECK (age BETWEEN 13 AND 120),
    country TEXT CHECK (country IN ('US', 'CA', 'UK', 'AU')), -- Too restrictive
    zip_code TEXT CHECK (zip_code ~ '^\d{5}(-\d{4})?$')        -- US-only format
);

-- DO: Flexible constraints with room for growth
CREATE TABLE users (
    email TEXT CHECK (email LIKE '%@%'),  -- Basic validation
    age INTEGER CHECK (age >= 0),         -- Reasonable bounds
    country TEXT,                         -- Validate in application
    zip_code TEXT                         -- Validate in application
);
```

### 2. Poor Error Messages

```sql
-- DON'T: Generic constraint names
ALTER TABLE orders ADD CHECK (total_amount > 0);

-- DO: Descriptive constraint names
ALTER TABLE orders ADD CONSTRAINT orders_total_amount_must_be_positive 
CHECK (total_amount > 0);
```

### 3. Ignoring Performance Impact

```sql
-- DON'T: Complex constraints on frequently updated columns
ALTER TABLE page_views ADD CONSTRAINT valid_view_time 
CHECK (view_time >= created_at AND view_time <= created_at + INTERVAL '1 hour');

-- DO: Move complex validation to application or use triggers selectively
-- Keep constraints simple for high-frequency operations
```

## Conclusion

Database constraints are essential for maintaining data integrity, but they must be used thoughtfully. The key is to find the right balance between data protection and system flexibility. Use constraints for:

- **Data integrity**: Prevent invalid data from entering the database
- **Business invariants**: Enforce rules that must never be violated
- **Performance**: Guide query optimization with proper indexing

Avoid constraints for:
- **Frequently changing business rules**: Use application validation instead
- **Complex logic**: Consider triggers or application-level validation
- **Performance-critical operations**: Validate in application when possible

Remember to test constraints thoroughly, document complex rules, and plan for constraint evolution as your application grows.
