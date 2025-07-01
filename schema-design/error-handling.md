# Database Validation & Error Handling Patterns

Modern applications require robust data validation at the database level to ensure data integrity, security, and meaningful error messages. This guide covers PostgreSQL's powerful domain and constraint features with real-world examples.

## 🎯 Why Database-Level Validation?

### Benefits
- **Single Source of Truth** - Validation rules are centralized
- **Multi-Application Safety** - Works across different clients/languages
- **Performance** - Database-native validation is faster than application logic
- **Data Integrity** - Impossible to bypass validation
- **Better Error Messages** - Customize validation feedback

### When to Use
- ✅ Data format validation (email, phone, etc.)
- ✅ Business rule enforcement (price > 0, age ranges)
- ✅ Cross-column constraints (start_date < end_date)
- ✅ Referential integrity beyond foreign keys
- ❌ Complex business logic requiring external APIs
- ❌ User-specific validation rules
- ❌ Dynamic validation rules that change frequently

## 🏗️ PostgreSQL Domain Types

### Basic Email Validation
```sql
-- Create a reusable email domain type
CREATE DOMAIN email AS TEXT 
CHECK (
    VALUE ~* '^[a-zA-Z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'
);

-- Usage examples
SELECT 'john.doe@company.com'::email; -- ✅ Valid
SELECT 'invalid-email'::email;        -- ❌ Throws constraint error

-- In table definitions
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email email NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Advanced Username Validation
```sql
-- Username with comprehensive rules
CREATE DOMAIN username AS TEXT 
CHECK (
    LENGTH(VALUE) >= 3 AND 
    LENGTH(VALUE) <= 30 AND
    VALUE ~ '^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$' AND
    VALUE !~ '__' AND  -- No consecutive underscores
    VALUE !~ '\.\.' AND -- No consecutive dots
    VALUE !~ '--'      -- No consecutive dashes
);

-- Test the domain
SELECT username('john_doe');     -- ✅ Valid
SELECT username('a');           -- ❌ Too short
SELECT username('user__name');  -- ❌ Consecutive underscores
```

### Password Strength Domain
```sql
-- Password with complexity requirements
CREATE DOMAIN strong_password AS TEXT 
CHECK (
    LENGTH(VALUE) >= 8 AND
    LENGTH(VALUE) <= 128 AND
    VALUE ~ '[A-Z]' AND        -- At least one uppercase
    VALUE ~ '[a-z]' AND        -- At least one lowercase  
    VALUE ~ '[0-9]' AND        -- At least one digit
    VALUE ~ '[!@#$%^&*(),.?":{}|<>]' -- At least one special char
);

-- Usage in auth table
CREATE TABLE user_auth (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    password_hash TEXT NOT NULL, -- Store hash, not the actual password
    password_changed_at TIMESTAMP DEFAULT NOW(),
    
    -- Validate raw password before hashing in application
    CONSTRAINT validate_password_on_reset 
        CHECK (LENGTH(password_hash) > 50) -- Ensure it's hashed
);
```

## 🔧 Safe Domain Migration

### Problem: Updating Constraints Without Downtime
```sql
-- Initial domain (too restrictive)
CREATE DOMAIN product_code AS TEXT 
CHECK (VALUE ~ '^[A-Z]{3}[0-9]{3}$'); -- Only allows ABC123 format

-- Later need to support: ABC123, ABC-123, ABC_123
-- Solution: Add new constraint with NOT VALID
ALTER DOMAIN product_code 
ADD CONSTRAINT product_code_new_format 
CHECK (VALUE ~ '^[A-Z]{3}[-_]?[0-9]{3}$') NOT VALID;

-- Remove old constraint
ALTER DOMAIN product_code DROP CONSTRAINT product_code_check;

-- Rename new constraint (optional)
ALTER DOMAIN product_code 
RENAME CONSTRAINT product_code_new_format TO product_code_check;

-- Now validate existing data if needed
ALTER DOMAIN product_code VALIDATE CONSTRAINT product_code_check;
```

### Zero-Downtime Migration Strategy
```sql
-- Step 1: Add new permissive constraint
ALTER DOMAIN email 
ADD CONSTRAINT email_v2 
CHECK (
    VALUE ~* '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$' OR
    VALUE ~* '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\.[a-zA-Z]{2,}$'
) NOT VALID;

-- Step 2: Clean up data that doesn't meet new standards
UPDATE users 
SET email = LOWER(TRIM(email)) 
WHERE email != LOWER(TRIM(email));

-- Step 3: Validate the constraint
ALTER DOMAIN email VALIDATE CONSTRAINT email_v2;

-- Step 4: Remove old constraint
ALTER DOMAIN email DROP CONSTRAINT email_check;

-- Step 5: Rename for consistency
ALTER DOMAIN email RENAME CONSTRAINT email_v2 TO email_check;
```

## 💬 Custom Error Messages

### Using Comments for Better Errors
```sql
-- Domain with descriptive error message
CREATE DOMAIN phone_number AS TEXT 
CHECK (VALUE ~ '^\+?[1-9]\d{1,14}$');

COMMENT ON DOMAIN phone_number IS 
'Phone number must be in international format (+1234567890) or national format (1234567890)';

-- Check constraint with meaningful name
ALTER DOMAIN phone_number 
ADD CONSTRAINT phone_international_format 
CHECK (VALUE ~ '^\+?[1-9]\d{1,14}$');
```

### Application-Level Error Translation
```sql
-- Create a function to get user-friendly error messages
CREATE OR REPLACE FUNCTION get_constraint_error_message(
    constraint_name TEXT,
    column_value TEXT DEFAULT NULL
) RETURNS TEXT AS $$
BEGIN
    RETURN CASE constraint_name
        WHEN 'email_check' THEN 
            'Please enter a valid email address (e.g., user@example.com)'
        WHEN 'username_check' THEN 
            'Username must be 3-30 characters, start and end with letter/number, no consecutive special characters'
        WHEN 'phone_international_format' THEN 
            'Phone number must be in international format (e.g., +1234567890)'
        WHEN 'strong_password_check' THEN 
            'Password must be 8-128 characters with uppercase, lowercase, number, and special character'
        ELSE 
            'Invalid value: ' || COALESCE(column_value, 'unknown')
    END;
END;
$$ LANGUAGE plpgsql;

-- Usage in application error handling
SELECT get_constraint_error_message('email_check');
```

## 🌍 Real-World Business Domain Examples

### E-Commerce Product Domains
```sql
-- Product SKU with business rules
CREATE DOMAIN product_sku AS TEXT
CHECK (
    LENGTH(VALUE) = 12 AND
    VALUE ~ '^[A-Z]{3}-[A-Z]{2}-[0-9]{6}$' -- Format: CAT-SH-123456
);

-- Price in cents (avoid floating point issues)
CREATE DOMAIN price_cents AS INTEGER
CHECK (VALUE >= 0 AND VALUE <= 99999999); -- Max $999,999.99

-- Discount percentage
CREATE DOMAIN discount_percentage AS DECIMAL(5,2)
CHECK (VALUE >= 0.00 AND VALUE <= 100.00);

-- Product weight in grams
CREATE DOMAIN weight_grams AS INTEGER
CHECK (VALUE > 0 AND VALUE <= 50000); -- Max 50kg

-- Usage in products table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku product_sku NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price_cents price_cents NOT NULL,
    weight_grams weight_grams NOT NULL,
    
    -- Cross-column constraint
    CONSTRAINT sale_price_valid 
        CHECK (
            (sale_price_cents IS NULL) OR 
            (sale_price_cents < price_cents)
        )
);
```

### SaaS Platform Domains
```sql
-- Tenant subdomain validation
CREATE DOMAIN tenant_subdomain AS TEXT
CHECK (
    LENGTH(VALUE) >= 3 AND 
    LENGTH(VALUE) <= 30 AND
    VALUE ~ '^[a-z][a-z0-9-]*[a-z0-9]$' AND
    VALUE !~ '--' AND
    VALUE NOT IN ('www', 'api', 'admin', 'app', 'mail', 'ftp', 'blog')
);

-- API key format
CREATE DOMAIN api_key AS TEXT
CHECK (
    LENGTH(VALUE) = 64 AND
    VALUE ~ '^[a-f0-9]{64}$' -- Hex string
);

-- Plan names
CREATE DOMAIN plan_name AS TEXT
CHECK (VALUE IN ('free', 'starter', 'professional', 'enterprise'));

-- Usage tracking
CREATE TABLE tenant_usage (
    tenant_id UUID REFERENCES tenants(id),
    month_year DATE NOT NULL, -- First day of month
    api_calls INTEGER CHECK (api_calls >= 0),
    storage_bytes BIGINT CHECK (storage_bytes >= 0),
    
    PRIMARY KEY (tenant_id, month_year)
);
```

### Financial Domains
```sql
-- Currency code (ISO 4217)
CREATE DOMAIN currency_code AS CHAR(3)
CHECK (VALUE ~ '^[A-Z]{3}$');

-- Bank account number (simplified)
CREATE DOMAIN bank_account AS TEXT
CHECK (
    LENGTH(VALUE) >= 8 AND 
    LENGTH(VALUE) <= 17 AND
    VALUE ~ '^[0-9]+$'
);

-- Credit card last 4 digits
CREATE DOMAIN card_last_four AS CHAR(4)
CHECK (VALUE ~ '^[0-9]{4}$');

-- Transaction amount in cents
CREATE DOMAIN transaction_amount_cents AS BIGINT
CHECK (VALUE != 0); -- No zero transactions

-- Financial transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    amount_cents transaction_amount_cents NOT NULL,
    currency currency_code NOT NULL DEFAULT 'USD',
    type TEXT CHECK (type IN ('payment', 'refund', 'transfer', 'fee')),
    
    -- Business rule: refunds must be negative
    CONSTRAINT refund_amount_check 
        CHECK (
            (type != 'refund') OR 
            (type = 'refund' AND amount_cents < 0)
        ),
    
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 🔍 Advanced Constraint Patterns

### Multi-Column Business Rules
```sql
-- Event booking with time validation
CREATE TABLE event_bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    user_id UUID NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP NOT NULL,
    max_attendees INTEGER CHECK (max_attendees > 0),
    current_attendees INTEGER DEFAULT 0,
    
    -- Time-based constraints
    CONSTRAINT valid_time_range 
        CHECK (ends_at > starts_at),
    
    CONSTRAINT future_event 
        CHECK (starts_at > NOW()),
    
    CONSTRAINT reasonable_duration 
        CHECK (ends_at - starts_at <= INTERVAL '7 days'),
    
    -- Capacity constraint
    CONSTRAINT under_capacity 
        CHECK (current_attendees <= max_attendees),
    
    -- Business hours (9 AM to 6 PM)
    CONSTRAINT business_hours 
        CHECK (
            EXTRACT(hour FROM starts_at) >= 9 AND
            EXTRACT(hour FROM ends_at) <= 18
        )
);
```

### Conditional Constraints
```sql
-- User profiles with conditional requirements
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY,
    account_type TEXT CHECK (account_type IN ('personal', 'business', 'enterprise')),
    
    -- Personal fields
    first_name TEXT,
    last_name TEXT,
    date_of_birth DATE,
    
    -- Business fields  
    company_name TEXT,
    tax_id TEXT,
    business_email email,
    
    -- Conditional constraints based on account type
    CONSTRAINT personal_required_fields
        CHECK (
            account_type != 'personal' OR (
                first_name IS NOT NULL AND 
                last_name IS NOT NULL AND
                date_of_birth IS NOT NULL
            )
        ),
    
    CONSTRAINT business_required_fields
        CHECK (
            account_type NOT IN ('business', 'enterprise') OR (
                company_name IS NOT NULL AND
                tax_id IS NOT NULL AND
                business_email IS NOT NULL
            )
        ),
    
    CONSTRAINT valid_age
        CHECK (
            date_of_birth IS NULL OR 
            date_of_birth <= CURRENT_DATE - INTERVAL '13 years'
        )
);
```

## 🛠️ Utility Functions

### Constraint Discovery
```sql
-- Get all domain constraints
CREATE OR REPLACE FUNCTION get_domain_constraints()
RETURNS TABLE(
    domain_name TEXT,
    constraint_name TEXT,
    constraint_definition TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.typname::TEXT,
        c.conname::TEXT,
        pg_get_constraintdef(c.oid)::TEXT
    FROM pg_constraint c
    JOIN pg_type t ON c.contypid = t.oid
    WHERE c.contype = 'c'
    ORDER BY t.typname, c.conname;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT * FROM get_domain_constraints();
```

### Validation Testing
```sql
-- Test domain values
CREATE OR REPLACE FUNCTION test_domain_value(
    domain_name TEXT, 
    test_value TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    result BOOLEAN := FALSE;
BEGIN
    EXECUTE format('SELECT %L::%I', test_value, domain_name);
    result := TRUE;
EXCEPTION 
    WHEN check_violation THEN
        result := FALSE;
    WHEN OTHERS THEN
        RAISE EXCEPTION 'Error testing domain %: %', domain_name, SQLERRM;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT test_domain_value('email', 'valid@example.com');    -- Returns true
SELECT test_domain_value('email', 'invalid-email');       -- Returns false
```

## 📊 Performance Considerations

### Index-Friendly Domains
```sql
-- Create domains that work well with indexes
CREATE DOMAIN normalized_email AS TEXT
CHECK (
    VALUE = LOWER(TRIM(VALUE)) AND
    VALUE ~* '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
);

-- Index will work efficiently with normalized values
CREATE INDEX idx_users_email ON users USING btree (email);
```

### Constraint Performance
```sql
-- Efficient regex patterns (anchor at start/end)
CREATE DOMAIN fast_code AS TEXT
CHECK (VALUE ~ '^[A-Z]{2}[0-9]{6}$'); -- Good: anchored

-- Avoid
CREATE DOMAIN slow_code AS TEXT  
CHECK (VALUE ~ '[A-Z]{2}[0-9]{6}'); -- Bad: unanchored, scans entire string
```

## 🚨 Common Pitfalls & Solutions

### Pitfall 1: Overly Restrictive Domains
```sql
-- ❌ Too restrictive - hard to change later
CREATE DOMAIN rigid_phone AS TEXT
CHECK (VALUE ~ '^\+1[0-9]{10}$'); -- Only US numbers

-- ✅ Flexible for international growth
CREATE DOMAIN flexible_phone AS TEXT
CHECK (
    LENGTH(VALUE) BETWEEN 7 AND 15 AND
    VALUE ~ '^[\+]?[0-9\s\-\(\)]+$'
);
```

### Pitfall 2: Complex Business Logic in Domains
```sql
-- ❌ Don't put complex business logic in domains
CREATE DOMAIN bad_business_rule AS TEXT
CHECK (
    -- Complex logic that might change
    VALUE IN (
        SELECT allowed_value 
        FROM business_rules 
        WHERE active = true 
        AND effective_date <= NOW()
    )
);

-- ✅ Keep domains simple, use constraints for business logic
CREATE DOMAIN simple_status AS TEXT
CHECK (VALUE IN ('active', 'inactive', 'pending', 'suspended'));
```

### Pitfall 3: Not Planning for Internationalization
```sql
-- ❌ ASCII-only validation
CREATE DOMAIN ascii_name AS TEXT
CHECK (VALUE ~ '^[a-zA-Z\s]+$');

-- ✅ Unicode-aware validation
CREATE DOMAIN international_name AS TEXT
CHECK (
    LENGTH(VALUE) BETWEEN 1 AND 100 AND
    VALUE ~ '^[\p{L}\p{M}\s''.-]+$' -- Unicode letters, marks, spaces, apostrophes, hyphens, dots
);
```

## 📝 Best Practices Summary

1. **Start Simple** - Begin with basic validation, evolve complexity
2. **Use NOT VALID** - For safe constraint migrations
3. **Document Domains** - Use comments to explain business rules
4. **Test Thoroughly** - Create test cases for edge cases
5. **Plan for Change** - Design domains that can evolve
6. **Performance First** - Consider index impact of constraints
7. **Separate Concerns** - Keep technical validation (format) separate from business rules
8. **Error Messages** - Provide meaningful constraint names and comments
9. **Version Control** - Track domain changes like any other schema change
10. **Monitor Usage** - Track constraint violations in production
COMMENT ON DOMAIN user_password IS 'password must be 8 characters';

CREATE OR REPLACE FUNCTION check_user_password( cmd TEXT ) RETURNS user_password AS $$
DECLARE
  dom text;
  friendly text;
  retval user_password;
BEGIN
  -- attempt to run original command
  select user_password(cmd) INTO retval;
  RETURN retval;
EXCEPTION WHEN check_violation THEN
  -- extract the relevant data type from the exception
  GET STACKED DIAGNOSTICS dom = PG_DATATYPE_NAME;

  -- look for a user comment on that type
  SELECT pg_catalog.obj_description(oid)
    FROM pg_catalog.pg_type
   WHERE typname = dom
    INTO friendly;

  IF friendly IS NULL THEN
    -- if there is no comment, throw original exception
    RAISE;
  ELSE
    -- otherwise throw a revised exception with better message
    RAISE check_violation USING message = friendly;
  END IF;
END;
$$ language plpgsql;
```

Test:
```sql
SELECT check_user_password('12');
```

Output:
```
ERROR:  password must be 8 characters
CONTEXT:  PL/pgSQL function check_user_password(text) line 25 at RAISE
```


## Thoughts on using Custom Domain on Table datatype

- does the language support casting the type to the primitives? If we create the `username` type, does it get converted back to type `text`?
- are there complexity in updating the constraints? (no, but requires proper version control, as well as documentation)
- we don't have to use the `domain` type in tables, we can always just use it for validation by casting the types back, e.g. `select 'text'::username`

## Getting all UDF (user-defined functions)

```sql
SELECT * 
FROM information_schema.routines 
WHERE routine_type='FUNCTION' 
  AND specific_schema='public'
  AND data_type = 'USER-DEFINED'
  AND external_language = 'PLPGSQL';
```

## Better error message with custom domain type

```sql
CREATE OR REPLACE FUNCTION check_username(name TEXT) RETURNS username AS $$
DECLARE
  result username;
BEGIN
  SELECT name::username INTO result;
  RETURN result;
EXCEPTION WHEN check_violation THEN
--	RAISE SQLSTATE '23514'
--	RAISE check_violation
	RAISE
		USING HINT = 'Please check your username for invalid characters',
		MESSAGE=format('Invalid username: %s', name),
		DETAIL=format('name %I contains invalid character', name);

END;
$$ LANGUAGE plpgsql;
```

Test:
```sql
select check_username('johnhleo*');
```

Output:
```sql
ERROR:  Invalid username: johnhleo*
DETAIL:  name "johnhleo*" contains invalid character
HINT:  Please check your username for invalid characters
CONTEXT:  PL/pgSQL function check_username(text) line 10 at RAISE
```
