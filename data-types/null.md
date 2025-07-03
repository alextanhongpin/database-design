
# Null Values: Complete Guide

Understanding when to use NULL values versus NOT NULL constraints is crucial for database integrity, application reliability, and query performance. This guide provides comprehensive guidance on null handling strategies.

## Table of Contents
- [Philosophy: Avoiding Nulls](#philosophy-avoiding-nulls)
- [When Nulls Are Appropriate](#when-nulls-are-appropriate)
- [Null Handling Patterns](#null-handling-patterns)
- [Application Integration](#application-integration)
- [Performance Implications](#performance-implications)
- [Migration Strategies](#migration-strategies)
- [Best Practices](#best-practices)

## Philosophy: Avoiding Nulls

### The Case Against Nulls

**Strongly Typed Language Compatibility**
```go
// Go example: Nulls create complexity
type User struct {
    ID       int
    Email    string          // Simple
    Name     *string         // Nullable - requires nil checks
    Age      sql.NullInt64   // Nullable - cumbersome API
    Bio      sql.NullString  // Nullable - verbose handling
}

// Every access requires null checking
func (u *User) GetDisplayName() string {
    if u.Name != nil {
        return *u.Name
    }
    return "Anonymous"
}
```

**SQL Query Complexity**
```sql
-- Nulls complicate queries
SELECT * FROM users 
WHERE name = 'John'     -- Misses NULL names
   OR name IS NULL;     -- Need explicit null handling

-- Three-valued logic confusion
SELECT * FROM users WHERE NOT (name = 'John'); -- Excludes NULLs unexpectedly
```

**Application Logic Complexity**
```sql
-- Simple case without nulls
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Complex case with nulls
CREATE TABLE users_with_nulls (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT,                    -- Nullable
    phone TEXT,                   -- Nullable  
    bio TEXT,                     -- Nullable
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Recommended Default Strategy

```sql
-- Prefer NOT NULL with meaningful defaults
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    
    -- Instead of nullable strings, use empty strings
    display_name TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    
    -- Instead of nullable numbers, use zero or specific defaults
    login_count INTEGER NOT NULL DEFAULT 0,
    reputation_score INTEGER NOT NULL DEFAULT 0,
    
    -- Instead of nullable booleans, use explicit defaults
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    marketing_emails BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## When Nulls Are Appropriate

### 1. Soft Deletion Pattern

```sql
-- NULL for "not deleted", timestamp for "deleted at"
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,  -- NULL = active, timestamp = deleted
    
    -- Partial unique index for active posts
    CONSTRAINT unique_active_title 
    EXCLUDE (title WITH =) WHERE (deleted_at IS NULL)
);

-- Query active posts
SELECT * FROM posts WHERE deleted_at IS NULL;

-- Query deleted posts
SELECT * FROM posts WHERE deleted_at IS NOT NULL;
```

### 2. Optional Foreign Key References

```sql
-- When relationship is truly optional
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    shipping_address_id INTEGER NOT NULL REFERENCES addresses(id),
    
    -- Optional: only set if different from customer's billing address
    billing_address_id INTEGER REFERENCES addresses(id),
    
    -- Optional: only set if order is part of a promotion
    promotion_id INTEGER REFERENCES promotions(id),
    
    total_amount INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 3. Unique Constraints with Optional Values

```sql
-- Scenario: Username is optional but must be unique when provided
CREATE TABLE user_accounts (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    
    -- Username is optional, but when provided must be unique
    username TEXT,
    
    -- Partial unique index: only enforce uniqueness for non-null usernames
    CONSTRAINT unique_username UNIQUE (username)
);

-- This allows multiple users with NULL username
INSERT INTO user_accounts (email, username) VALUES
('user1@example.com', 'john_doe'),    -- OK: unique username
('user2@example.com', NULL),          -- OK: null username
('user3@example.com', NULL);          -- OK: multiple null usernames

-- But prevents duplicate non-null usernames
-- INSERT INTO user_accounts (email, username) VALUES
-- ('user4@example.com', 'john_doe'); -- ERROR: duplicate username
```

### 4. Progressive Data Collection

```sql
-- User registration: collect data progressively
CREATE TABLE user_onboarding (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    
    -- Collected in step 2 (optional)
    first_name TEXT,
    last_name TEXT,
    
    -- Collected in step 3 (optional)
    phone_number TEXT,
    birth_date DATE,
    
    -- Collected in step 4 (optional)
    company_name TEXT,
    job_title TEXT,
    
    -- Onboarding progress tracking
    onboarding_step INTEGER NOT NULL DEFAULT 1,
    profile_completed BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Validate profile completion
CREATE OR REPLACE FUNCTION validate_profile_completion()
RETURNS TRIGGER AS $$
BEGIN
    NEW.profile_completed := (
        NEW.first_name IS NOT NULL AND
        NEW.last_name IS NOT NULL AND
        NEW.phone_number IS NOT NULL AND
        NEW.birth_date IS NOT NULL
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_profile_completion
    BEFORE INSERT OR UPDATE ON user_onboarding
    FOR EACH ROW EXECUTE FUNCTION validate_profile_completion();
```

## Null Handling Patterns

### 1. COALESCE for Default Values

```sql
-- Provide fallback values for nulls
SELECT 
    id,
    email,
    COALESCE(display_name, email) as display_name,
    COALESCE(bio, 'No bio provided') as bio,
    COALESCE(login_count, 0) as login_count
FROM user_profiles;

-- Multi-level fallbacks
SELECT 
    COALESCE(preferred_name, first_name, username, 'Anonymous') as name
FROM users;
```

### 2. NULLIF for Conditional Nulls

```sql
-- Convert empty strings back to nulls if needed
UPDATE user_profiles 
SET bio = NULLIF(TRIM(bio), '')
WHERE bio IS NOT NULL;

-- Convert zero values to nulls for optional numeric fields
SELECT 
    id,
    name,
    NULLIF(optional_score, 0) as score  -- Shows NULL instead of 0
FROM competitions;
```

### 3. Conditional Aggregation with Nulls

```sql
-- Count non-null values
SELECT 
    COUNT(*) as total_users,
    COUNT(phone_number) as users_with_phone,
    COUNT(bio) as users_with_bio,
    COUNT(*) - COUNT(phone_number) as users_without_phone
FROM user_profiles;

-- Conditional aggregation
SELECT 
    AVG(rating) as avg_rating,                    -- Excludes nulls automatically
    AVG(COALESCE(rating, 0)) as avg_rating_with_zeros,
    COUNT(CASE WHEN rating >= 4 THEN 1 END) as high_ratings
FROM product_reviews;
```

### 4. Null-Safe Comparisons

```sql
-- Use IS DISTINCT FROM for null-safe equality
SELECT * FROM users 
WHERE old_email IS DISTINCT FROM new_email;  -- Handles nulls correctly

-- Traditional comparison misses null cases
SELECT * FROM users 
WHERE old_email != new_email;  -- Misses cases where either is null

-- Null-safe string comparison function
CREATE OR REPLACE FUNCTION null_safe_equals(a TEXT, b TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN (a IS NOT DISTINCT FROM b);
END;
$$ LANGUAGE plpgsql;
```

## Application Integration

### 1. Handling Nulls in Application Code

```go
// Go: Use pointers for optional fields
type UserProfile struct {
    ID          int       `json:"id"`
    Email       string    `json:"email"`
    DisplayName *string   `json:"display_name,omitempty"`
    Bio         *string   `json:"bio,omitempty"`
    Phone       *string   `json:"phone,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// Helper function for safe string dereferencing
func SafeString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}

// Helper function for creating string pointers
func StringPtr(s string) *string {
    if s == "" {
        return nil
    }
    return &s
}

// Usage in handlers
func (u *UserProfile) GetDisplayName() string {
    if u.DisplayName != nil {
        return *u.DisplayName
    }
    return u.Email // Fallback to email
}
```

### 2. JSON Serialization with Nulls

```json
// Client representation
{
    "id": 123,
    "email": "user@example.com",
    "display_name": null,      // Explicitly null
    "bio": "Software developer",
    "phone": null
}
```

```go
// Go: Custom JSON marshaling for null handling
func (u *UserProfile) MarshalJSON() ([]byte, error) {
    type Alias UserProfile
    return json.Marshal(&struct {
        DisplayName string `json:"display_name"`
        Bio         string `json:"bio"`
        Phone       string `json:"phone"`
        *Alias
    }{
        DisplayName: SafeString(u.DisplayName),
        Bio:         SafeString(u.Bio),
        Phone:       SafeString(u.Phone),
        Alias:       (*Alias)(u),
    })
}
```

### 3. Database Query Builders

```go
// Building dynamic queries with optional filters
func BuildUserQuery(filters UserFilters) (string, []interface{}) {
    query := "SELECT * FROM users WHERE 1=1"
    args := []interface{}{}
    argIndex := 1
    
    if filters.Email != nil {
        query += fmt.Sprintf(" AND email = $%d", argIndex)
        args = append(args, *filters.Email)
        argIndex++
    }
    
    if filters.HasBio != nil {
        if *filters.HasBio {
            query += " AND bio IS NOT NULL AND bio != ''"
        } else {
            query += " AND (bio IS NULL OR bio = '')"
        }
    }
    
    return query, args
}
```

## Performance Implications

### 1. Index Behavior with Nulls

```sql
-- Regular indexes don't include NULL values
CREATE INDEX idx_users_phone ON users (phone_number);

-- This query cannot use the index efficiently
SELECT * FROM users WHERE phone_number IS NULL;

-- Create partial index for null checks
CREATE INDEX idx_users_no_phone ON users (id) WHERE phone_number IS NULL;

-- Composite indexes with nulls
CREATE INDEX idx_users_name_phone ON users (last_name, phone_number);
-- Entries with NULL phone_number are not indexed

-- Force inclusion of nulls in composite index
CREATE INDEX idx_users_name_phone_nulls ON users (last_name, COALESCE(phone_number, ''));
```

### 2. Query Performance with Nulls

```sql
-- Inefficient: null checks can't use indexes
SELECT * FROM large_table WHERE optional_field IS NULL;

-- More efficient: use partial indexes
CREATE INDEX idx_large_table_no_optional 
ON large_table (id) WHERE optional_field IS NULL;

-- Efficient: positive conditions
SELECT * FROM large_table WHERE required_field = 'value';

-- Consider materialized columns for complex null logic
ALTER TABLE user_profiles 
ADD COLUMN has_complete_profile BOOLEAN 
GENERATED ALWAYS AS (
    first_name IS NOT NULL AND 
    last_name IS NOT NULL AND 
    phone_number IS NOT NULL
) STORED;

CREATE INDEX idx_complete_profiles ON user_profiles (has_complete_profile);
```

### 3. Aggregation Performance

```sql
-- COUNT(*) vs COUNT(column) performance difference
EXPLAIN (ANALYZE, BUFFERS) 
SELECT COUNT(*) FROM large_table;                    -- Faster

EXPLAIN (ANALYZE, BUFFERS) 
SELECT COUNT(nullable_column) FROM large_table;      -- Slower (must check nulls)

-- Use FILTER for conditional counts (PostgreSQL)
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE phone_number IS NOT NULL) as with_phone,
    COUNT(*) FILTER (WHERE phone_number IS NULL) as without_phone
FROM users;
```

## Migration Strategies

### 1. Adding NOT NULL Constraints

```sql
-- Safe migration: add column with default, then make it required
-- Step 1: Add nullable column with default
ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';

-- Step 2: Populate existing rows
UPDATE users SET status = 'active' WHERE status IS NULL;

-- Step 3: Add NOT NULL constraint
ALTER TABLE users ALTER COLUMN status SET NOT NULL;

-- Step 4: Remove default if no longer needed
ALTER TABLE users ALTER COLUMN status DROP DEFAULT;
```

### 2. Removing NULL Values

```sql
-- Migration: convert nulls to meaningful defaults
-- Step 1: Update null values
UPDATE user_profiles 
SET 
    display_name = COALESCE(display_name, email),
    bio = COALESCE(bio, ''),
    login_count = COALESCE(login_count, 0)
WHERE display_name IS NULL 
   OR bio IS NULL 
   OR login_count IS NULL;

-- Step 2: Add NOT NULL constraints
ALTER TABLE user_profiles 
    ALTER COLUMN display_name SET NOT NULL,
    ALTER COLUMN bio SET NOT NULL,
    ALTER COLUMN login_count SET NOT NULL;
```

### 3. Splitting Tables to Avoid Nulls

```sql
-- Before: table with many optional fields
CREATE TABLE user_profiles_before (
    user_id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    company_name TEXT,
    job_title TEXT,
    phone_number TEXT,
    linkedin_url TEXT
);

-- After: split into required and optional data
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_personal_info (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    phone_number TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_professional_info (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    company_name TEXT NOT NULL,
    job_title TEXT NOT NULL,
    linkedin_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Best Practices

### 1. Design Guidelines

```sql
-- ✅ Good: Explicit about nullability and defaults
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    
    -- Required fields: NOT NULL with appropriate defaults
    status TEXT NOT NULL DEFAULT 'pending',
    total_amount INTEGER NOT NULL,
    currency_code CHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Optional relationships: nullable foreign keys
    coupon_id INTEGER REFERENCES coupons(id),
    
    -- Soft deletion: nullable timestamp
    cancelled_at TIMESTAMPTZ,
    
    -- Audit fields: always required
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 2. Documentation Standards

```sql
-- Document null semantics clearly
COMMENT ON COLUMN orders.coupon_id IS 
'Optional coupon applied to order. NULL indicates no coupon used.';

COMMENT ON COLUMN orders.cancelled_at IS 
'Timestamp when order was cancelled. NULL indicates active order.';

COMMENT ON COLUMN user_profiles.phone_number IS 
'User phone number. NULL allowed during progressive onboarding.';
```

### 3. Validation Strategies

```sql
-- Add check constraints to enforce business rules
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    
    -- Optional fields with business rules
    discount_percentage DECIMAL(5,2),
    sale_end_date TIMESTAMPTZ,
    
    -- Ensure discount logic is consistent
    CONSTRAINT valid_discount CHECK (
        (discount_percentage IS NULL AND sale_end_date IS NULL) OR
        (discount_percentage IS NOT NULL AND sale_end_date IS NOT NULL AND
         discount_percentage > 0 AND discount_percentage < 100)
    )
);
```

### 4. Application Layer Patterns

```go
// Define clear semantics for optional vs required fields
type CreateUserRequest struct {
    Email       string  `json:"email" validate:"required,email"`
    Password    string  `json:"password" validate:"required,min=8"`
    
    // Optional during creation
    FirstName   *string `json:"first_name,omitempty"`
    LastName    *string `json:"last_name,omitempty"`
    PhoneNumber *string `json:"phone_number,omitempty"`
}

type User struct {
    ID          int       `json:"id"`
    Email       string    `json:"email"`
    
    // Use pointers to distinguish between not provided and empty
    FirstName   *string   `json:"first_name"`
    LastName    *string   `json:"last_name"`
    PhoneNumber *string   `json:"phone_number"`
    
    CreatedAt   time.Time `json:"created_at"`
}

// Helper methods for null handling
func (u *User) HasCompleteName() bool {
    return u.FirstName != nil && u.LastName != nil
}

func (u *User) GetDisplayName() string {
    if u.HasCompleteName() {
        return fmt.Sprintf("%s %s", *u.FirstName, *u.LastName)
    }
    return u.Email
}
```

## Conclusion

### When to Use NULL:
- ✅ Soft deletion timestamps (`deleted_at`)
- ✅ Optional foreign key relationships
- ✅ Progressive data collection scenarios
- ✅ Unique constraints on optional fields

### When to Avoid NULL:
- ❌ Required business data
- ❌ Fields with meaningful defaults
- ❌ Boolean flags (use explicit true/false)
- ❌ Counter fields (use 0 instead)

### Key Principles:
1. **Default to NOT NULL** with meaningful defaults
2. **Use NULL sparingly** and only when it represents a genuine "unknown" or "not applicable" state
3. **Document null semantics** clearly in schema and code
4. **Consider alternatives** like separate tables or default values
5. **Test null handling** thoroughly in application code
6. **Use partial indexes** for performance with nullable columns

The goal is to create schemas that are both semantically correct and practical to work with in application code.




