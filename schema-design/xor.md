# Exclusive Constraint Patterns (XOR)

Implementing mutually exclusive constraints to ensure that only one of multiple conditions can be true at a time.

## 🎯 Overview

XOR (exclusive OR) constraints are useful when:
- **Polymorphic Associations** - Entity belongs to one of several types
- **Conditional Fields** - Fields required based on other field values
- **State Dependencies** - Different fields valid for different states
- **Type-Specific Data** - Additional data based on entity type

## 🔧 Basic XOR Patterns

### Conditional Field Requirements

```sql
-- Order table with type-dependent fields
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('market', 'limit', 'stop')),
    
    -- Conditional fields based on order type
    limit_price DECIMAL(13, 4),
    stop_price DECIMAL(13, 4),
    
    -- Constraints for type-specific fields
    CONSTRAINT limit_order_price CHECK (
        (type = 'limit' AND limit_price IS NOT NULL) OR 
        (type != 'limit' AND limit_price IS NULL)
    ),
    
    CONSTRAINT stop_order_price CHECK (
        (type = 'stop' AND stop_price IS NOT NULL) OR 
        (type != 'stop' AND stop_price IS NULL)
    ),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Test the constraints
INSERT INTO orders (type, limit_price) VALUES ('market', 12.3);  -- ❌ Fails
INSERT INTO orders (type, limit_price) VALUES ('market', NULL);   -- ✅ Success
INSERT INTO orders (type, limit_price) VALUES ('limit', NULL);    -- ❌ Fails
INSERT INTO orders (type, limit_price) VALUES ('limit', 13.4);    -- ✅ Success
```

### Alternative CASE WHEN Syntax

```sql
-- More readable constraint using CASE WHEN
CREATE TABLE orders_v2 (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('market', 'limit', 'stop')),
    limit_price DECIMAL(13, 4),
    stop_price DECIMAL(13, 4),
    
    CONSTRAINT conditional_fields CHECK (
        CASE 
            WHEN type = 'limit' THEN limit_price IS NOT NULL
            WHEN type = 'stop' THEN stop_price IS NOT NULL
            ELSE limit_price IS NULL AND stop_price IS NULL
        END
    )
);
```

## 🔗 Polymorphic Association Patterns

### Simple XOR Foreign Keys

```sql
-- Supporting tables
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

-- Polymorphic ownership - can belong to user OR organization
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    
    -- XOR foreign keys
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    
    -- Exactly one must be set
    CONSTRAINT owner_xor CHECK (
        (user_id IS NULL) != (organization_id IS NULL)
    ),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Test data
INSERT INTO users (name) VALUES ('John Doe');
INSERT INTO organizations (name) VALUES ('Acme Corp');

-- Valid inserts
INSERT INTO projects (name, user_id) 
SELECT 'Personal Project', id FROM users WHERE name = 'John Doe';

INSERT INTO projects (name, organization_id) 
SELECT 'Company Project', id FROM organizations WHERE name = 'Acme Corp';

-- Invalid inserts
INSERT INTO projects (name) VALUES ('Orphaned Project');  -- ❌ Both NULL
INSERT INTO projects (name, user_id, organization_id) 
SELECT 'Conflicted Project', u.id, o.id 
FROM users u, organizations o 
WHERE u.name = 'John Doe' AND o.name = 'Acme Corp';  -- ❌ Both set
```

### Multi-Way XOR Constraints

```sql
-- Support for multiple entity types
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    
    -- Can comment on posts, photos, or videos
    post_id UUID REFERENCES posts(id),
    photo_id UUID REFERENCES photos(id),
    video_id UUID REFERENCES videos(id),
    
    -- Exactly one target must be specified
    CONSTRAINT comment_target_xor CHECK (
        (post_id IS NOT NULL)::int + 
        (photo_id IS NOT NULL)::int + 
        (video_id IS NOT NULL)::int = 1
    ),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🏗️ Advanced XOR Patterns

### State-Dependent Fields

```sql
-- User profile with different required fields based on account type
CREATE TABLE user_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    account_type TEXT NOT NULL CHECK (account_type IN ('individual', 'business', 'nonprofit')),
    
    -- Individual fields
    first_name TEXT,
    last_name TEXT,
    date_of_birth DATE,
    
    -- Business fields
    company_name TEXT,
    tax_id TEXT,
    business_type TEXT,
    
    -- Nonprofit fields
    organization_name TEXT,
    nonprofit_id TEXT,
    mission_statement TEXT,
    
    -- State-dependent constraints
    CONSTRAINT individual_fields CHECK (
        account_type != 'individual' OR (
            first_name IS NOT NULL AND 
            last_name IS NOT NULL AND 
            date_of_birth IS NOT NULL AND
            company_name IS NULL AND 
            tax_id IS NULL AND 
            organization_name IS NULL
        )
    ),
    
    CONSTRAINT business_fields CHECK (
        account_type != 'business' OR (
            company_name IS NOT NULL AND 
            tax_id IS NOT NULL AND 
            first_name IS NULL AND 
            last_name IS NULL AND 
            organization_name IS NULL
        )
    ),
    
    CONSTRAINT nonprofit_fields CHECK (
        account_type != 'nonprofit' OR (
            organization_name IS NOT NULL AND 
            nonprofit_id IS NOT NULL AND 
            first_name IS NULL AND 
            last_name IS NULL AND 
            company_name IS NULL
        )
    )
);
```

### Payment Method XOR

```sql
-- Payment table with mutually exclusive payment methods
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount DECIMAL(10,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Payment method details (mutually exclusive)
    credit_card_token TEXT,
    bank_account_id UUID REFERENCES bank_accounts(id),
    crypto_wallet_address TEXT,
    paypal_email TEXT,
    
    -- Exactly one payment method required
    CONSTRAINT payment_method_xor CHECK (
        (credit_card_token IS NOT NULL)::int +
        (bank_account_id IS NOT NULL)::int +
        (crypto_wallet_address IS NOT NULL)::int +
        (paypal_email IS NOT NULL)::int = 1
    ),
    
    -- Additional validation per payment type
    CONSTRAINT crypto_address_format CHECK (
        crypto_wallet_address IS NULL OR 
        crypto_wallet_address ~ '^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$'
    ),
    
    CONSTRAINT paypal_email_format CHECK (
        paypal_email IS NULL OR 
        paypal_email ~ '^[^@]+@[^@]+\.[^@]+$'
    ),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🔍 Validation and Helper Functions

### Check Both Columns Null/Not Null

```sql
-- Utility functions for common XOR scenarios
CREATE TABLE relationship_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    
    -- Either both start/end dates or neither
    relationship_start DATE,
    relationship_end DATE,
    
    -- Both must be null or both must be not null
    CONSTRAINT relationship_dates CHECK (
        (ROW(relationship_start, relationship_end) IS NULL) OR
        (ROW(relationship_start, relationship_end) IS NOT NULL)
    ),
    
    -- If both are set, start must be before end
    CONSTRAINT relationship_date_order CHECK (
        relationship_start IS NULL OR 
        relationship_end IS NULL OR 
        relationship_start < relationship_end
    )
);
```

### Helper Functions for Complex XOR

```sql
-- Function to validate XOR constraints
CREATE OR REPLACE FUNCTION validate_exactly_one_not_null(VARIADIC fields ANYELEMENT[])
RETURNS BOOLEAN AS $$
DECLARE
    non_null_count INTEGER := 0;
    field ANYELEMENT;
BEGIN
    FOREACH field IN ARRAY fields LOOP
        IF field IS NOT NULL THEN
            non_null_count := non_null_count + 1;
        END IF;
    END LOOP;
    
    RETURN non_null_count = 1;
END;
$$ LANGUAGE plpgsql;

-- Usage in constraint
CREATE TABLE flexible_xor (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_a TEXT,
    field_b TEXT,
    field_c TEXT,
    
    CONSTRAINT exactly_one_field CHECK (
        validate_exactly_one_not_null(field_a, field_b, field_c)
    )
);
```

## 📊 Performance Considerations

### Indexing XOR Columns

```sql
-- Create partial indexes for XOR scenarios
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message TEXT NOT NULL,
    
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    
    CONSTRAINT recipient_xor CHECK (
        (user_id IS NULL) != (organization_id IS NULL)
    ),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Separate indexes for each case
CREATE INDEX idx_notifications_user 
ON notifications (user_id, created_at) 
WHERE user_id IS NOT NULL;

CREATE INDEX idx_notifications_org 
ON notifications (organization_id, created_at) 
WHERE organization_id IS NOT NULL;

-- Query patterns
-- Find user notifications
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM notifications 
WHERE user_id = 'some-uuid' 
ORDER BY created_at DESC;

-- Find organization notifications  
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM notifications 
WHERE organization_id = 'some-uuid' 
ORDER BY created_at DESC;
```

### Views for Polymorphic Queries

```sql
-- Unified view for polymorphic associations
CREATE VIEW project_owners AS
SELECT 
    p.id as project_id,
    p.name as project_name,
    CASE 
        WHEN p.user_id IS NOT NULL THEN 'user'
        WHEN p.organization_id IS NOT NULL THEN 'organization'
    END as owner_type,
    COALESCE(p.user_id, p.organization_id) as owner_id,
    COALESCE(u.name, o.name) as owner_name,
    p.created_at
FROM projects p
LEFT JOIN users u ON p.user_id = u.id
LEFT JOIN organizations o ON p.organization_id = o.id;

-- Query all projects with owner information
SELECT * FROM project_owners 
WHERE owner_type = 'user'
ORDER BY created_at DESC;
```

## 🎯 Best Practices

### Design Guidelines

1. **Keep It Simple** - Start with simple XOR, add complexity as needed
2. **Clear Naming** - Use descriptive constraint names
3. **Document Logic** - Comment complex XOR constraints
4. **Test Thoroughly** - Verify all valid and invalid combinations
5. **Consider Performance** - Index appropriately for query patterns

### Common Patterns

```sql
-- Pattern 1: Type-dependent fields
CONSTRAINT type_dependent_field CHECK (
    (type = 'specific_type' AND required_field IS NOT NULL) OR
    (type != 'specific_type' AND required_field IS NULL)
)

-- Pattern 2: Exactly one of many
CONSTRAINT exactly_one CHECK (
    (field1 IS NOT NULL)::int + 
    (field2 IS NOT NULL)::int + 
    (field3 IS NOT NULL)::int = 1
)

-- Pattern 3: All or none
CONSTRAINT all_or_none CHECK (
    (ROW(field1, field2, field3) IS NULL) OR
    (field1 IS NOT NULL AND field2 IS NOT NULL AND field3 IS NOT NULL)
)
```

### Error Handling

```sql
-- Create custom error messages for XOR violations
CREATE OR REPLACE FUNCTION validate_payment_method()
RETURNS TRIGGER AS $$
BEGIN
    IF (NEW.credit_card_token IS NOT NULL)::int +
       (NEW.bank_account_id IS NOT NULL)::int +
       (NEW.crypto_wallet_address IS NOT NULL)::int +
       (NEW.paypal_email IS NOT NULL)::int != 1 THEN
        RAISE EXCEPTION 'Exactly one payment method must be specified'
            USING HINT = 'Choose credit card, bank account, crypto wallet, or PayPal';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER payment_method_validation
    BEFORE INSERT OR UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION validate_payment_method();
```

## 🔗 Related Patterns

- **[Polymorphic Associations](polymorphic.md)** - Flexible entity relationships
- **[State Machines](state-machine.md)** - State-dependent field validation
- **[Constraint Patterns](constraint-patterns.md)** - Advanced constraint techniques
- **[Business Rules](business-rule.md)** - Complex validation logic

XOR constraints are powerful tools for ensuring data integrity in complex scenarios where fields have conditional requirements or mutually exclusive relationships.
