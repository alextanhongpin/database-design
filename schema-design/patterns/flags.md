# Database Flags: Boolean and Bitwise Patterns

Flags are columns that represent binary or multi-value states in database records. This guide covers various approaches to implementing flags, from simple boolean columns to complex bitwise operations for efficient multi-flag storage.

## Table of Contents
- [Boolean Flags](#boolean-flags)
- [Bitwise Flags](#bitwise-flags)
- [Enum-Based Flags](#enum-based-flags)
- [JSON-Based Flags](#json-based-flags)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)
- [Real-World Examples](#real-world-examples)

## Boolean Flags

### Simple Boolean Columns

The most straightforward approach for binary states:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    
    -- Boolean flags
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    is_premium BOOLEAN DEFAULT FALSE,
    email_notifications BOOLEAN DEFAULT TRUE,
    sms_notifications BOOLEAN DEFAULT FALSE,
    marketing_consent BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient querying
CREATE INDEX idx_users_active_verified ON users (is_active, is_verified);
CREATE INDEX idx_users_premium ON users (is_premium) WHERE is_premium = TRUE;
```

### Boolean Flags with Metadata

Track when flags were changed and by whom:

```sql
CREATE TABLE user_flags (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    flag_name TEXT NOT NULL,
    flag_value BOOLEAN NOT NULL,
    changed_by INTEGER REFERENCES users(id),
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    reason TEXT,
    
    UNIQUE(user_id, flag_name)
);

-- Function to update flags with audit trail
CREATE OR REPLACE FUNCTION set_user_flag(
    p_user_id INTEGER,
    p_flag_name TEXT,
    p_flag_value BOOLEAN,
    p_changed_by INTEGER DEFAULT NULL,
    p_reason TEXT DEFAULT NULL
) RETURNS VOID AS $$
BEGIN
    INSERT INTO user_flags (user_id, flag_name, flag_value, changed_by, reason)
    VALUES (p_user_id, p_flag_name, p_flag_value, p_changed_by, p_reason)
    ON CONFLICT (user_id, flag_name) 
    DO UPDATE SET 
        flag_value = EXCLUDED.flag_value,
        changed_by = EXCLUDED.changed_by,
        changed_at = NOW(),
        reason = EXCLUDED.reason;
END;
$$ LANGUAGE plpgsql;
```

## Bitwise Flags

### Single Integer Bitwise Flags

Efficient storage for multiple boolean flags in a single integer:

```sql
CREATE TABLE user_permissions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    
    -- Bitwise flags stored as integer
    permissions INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Define permission constants (powers of 2)
-- 1 = READ (2^0)
-- 2 = WRITE (2^1) 
-- 4 = DELETE (2^2)
-- 8 = ADMIN (2^3)
-- 16 = EXPORT (2^4)
-- 32 = IMPORT (2^5)

-- Functions for bitwise operations
CREATE OR REPLACE FUNCTION has_permission(flags INTEGER, permission INTEGER)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN (flags & permission) = permission;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION add_permission(flags INTEGER, permission INTEGER)
RETURNS INTEGER AS $$
BEGIN
    RETURN flags | permission;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION remove_permission(flags INTEGER, permission INTEGER)
RETURNS INTEGER AS $$
BEGIN
    RETURN flags & ~permission;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Usage examples
INSERT INTO user_permissions (user_id, permissions) VALUES
(1, 1 | 2),      -- READ + WRITE
(2, 1 | 2 | 4),  -- READ + WRITE + DELETE
(3, 1 | 8);      -- READ + ADMIN

-- Queries
SELECT u.name, 
       has_permission(up.permissions, 1) AS can_read,
       has_permission(up.permissions, 2) AS can_write,
       has_permission(up.permissions, 4) AS can_delete,
       has_permission(up.permissions, 8) AS is_admin
FROM users u
JOIN user_permissions up ON u.id = up.user_id;

-- Find users with specific permissions
SELECT u.name FROM users u
JOIN user_permissions up ON u.id = up.user_id
WHERE has_permission(up.permissions, 8); -- Find admins

-- Find users with multiple permissions
SELECT u.name FROM users u
JOIN user_permissions up ON u.id = up.user_id
WHERE (up.permissions & (1 | 2)) = (1 | 2); -- Has both READ and WRITE
```

### Multiple Bitwise Columns

For complex permission systems, separate different types of flags:

```sql
CREATE TABLE user_access_control (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    
    -- Different categories of permissions
    content_permissions INTEGER DEFAULT 0,  -- Read, Write, Delete, Publish
    admin_permissions INTEGER DEFAULT 0,    -- User mgmt, System config
    api_permissions INTEGER DEFAULT 0,      -- API access levels
    feature_flags INTEGER DEFAULT 0,        -- Feature toggles
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create a view for human-readable permissions
CREATE VIEW user_permissions_view AS
SELECT 
    u.id,
    u.name,
    u.email,
    
    -- Content permissions
    has_permission(uac.content_permissions, 1) AS can_read_content,
    has_permission(uac.content_permissions, 2) AS can_write_content,
    has_permission(uac.content_permissions, 4) AS can_delete_content,
    has_permission(uac.content_permissions, 8) AS can_publish_content,
    
    -- Admin permissions
    has_permission(uac.admin_permissions, 1) AS can_manage_users,
    has_permission(uac.admin_permissions, 2) AS can_configure_system,
    has_permission(uac.admin_permissions, 4) AS can_view_analytics,
    
    -- Feature flags
    has_permission(uac.feature_flags, 1) AS beta_features_enabled,
    has_permission(uac.feature_flags, 2) AS advanced_ui_enabled,
    has_permission(uac.feature_flags, 4) AS experimental_features_enabled
    
FROM users u
LEFT JOIN user_access_control uac ON u.id = uac.user_id;
```

## Enum-Based Flags

### Using PostgreSQL ENUMs

For flags with predefined values:

```sql
-- Create enum types
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'deleted');
CREATE TYPE subscription_type AS ENUM ('free', 'basic', 'premium', 'enterprise');
CREATE TYPE notification_preference AS ENUM ('all', 'important', 'none');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    
    -- Enum flags
    status user_status DEFAULT 'active',
    subscription subscription_type DEFAULT 'free',
    email_notifications notification_preference DEFAULT 'all',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Queries with enum flags
SELECT COUNT(*) FROM users WHERE status = 'active';
SELECT COUNT(*) FROM users WHERE subscription IN ('premium', 'enterprise');

-- Index on enum columns
CREATE INDEX idx_users_status ON users (status);
CREATE INDEX idx_users_subscription ON users (subscription);
```

### Enum Arrays for Multi-Select Flags

```sql
CREATE TYPE user_role AS ENUM ('admin', 'moderator', 'editor', 'viewer');
CREATE TYPE feature_flag AS ENUM ('beta_ui', 'advanced_search', 'api_access', 'bulk_operations');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    
    -- Array of enums for multi-select
    roles user_role[] DEFAULT '{}',
    enabled_features feature_flag[] DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert users with multiple roles/features
INSERT INTO users (email, name, roles, enabled_features) VALUES
('admin@example.com', 'Admin User', 
 ARRAY['admin', 'moderator']::user_role[], 
 ARRAY['beta_ui', 'api_access']::feature_flag[]),
('editor@example.com', 'Editor User',
 ARRAY['editor']::user_role[],
 ARRAY['advanced_search']::feature_flag[]);

-- Queries with array enums
SELECT * FROM users WHERE 'admin' = ANY(roles);
SELECT * FROM users WHERE roles @> ARRAY['admin']::user_role[];
SELECT * FROM users WHERE enabled_features && ARRAY['beta_ui']::feature_flag[];

-- Index for array queries
CREATE INDEX idx_users_roles ON users USING GIN (roles);
CREATE INDEX idx_users_features ON users USING GIN (enabled_features);
```

## JSON-Based Flags

### JSONB for Complex Flag Structures

```sql
CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    
    -- JSONB for complex nested flags
    preferences JSONB DEFAULT '{}',
    feature_flags JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert complex flag structures
INSERT INTO user_preferences (user_id, preferences, feature_flags) VALUES
(1, '{
    "notifications": {
        "email": true,
        "sms": false,
        "push": true,
        "frequency": "daily"
    },
    "privacy": {
        "profile_public": false,
        "show_activity": true,
        "allow_messages": true
    },
    "display": {
        "theme": "dark",
        "language": "en",
        "timezone": "UTC"
    }
}', '{
    "beta_features": true,
    "advanced_ui": false,
    "api_access": true,
    "experimental": {
        "new_dashboard": true,
        "ai_suggestions": false
    }
}');

-- Queries with JSONB
SELECT * FROM user_preferences 
WHERE preferences->>'notifications'->>'email' = 'true';

SELECT * FROM user_preferences 
WHERE feature_flags->>'beta_features' = 'true';

SELECT * FROM user_preferences 
WHERE feature_flags->'experimental'->>'new_dashboard' = 'true';

-- Index JSONB paths
CREATE INDEX idx_user_prefs_email_notifications 
ON user_preferences USING BTREE ((preferences->'notifications'->>'email'));

CREATE INDEX idx_user_prefs_beta_features 
ON user_preferences USING BTREE ((feature_flags->>'beta_features'));

-- GIN index for complex queries
CREATE INDEX idx_user_prefs_feature_flags 
ON user_preferences USING GIN (feature_flags);
```

## Performance Considerations

### Indexing Strategies

```sql
-- Partial indexes for boolean flags
CREATE INDEX idx_users_active ON users (id) WHERE is_active = TRUE;
CREATE INDEX idx_users_premium ON users (id) WHERE is_premium = TRUE;

-- Composite indexes for common queries
CREATE INDEX idx_users_status_subscription ON users (is_active, is_verified, is_premium);

-- Functional indexes for bitwise operations
CREATE INDEX idx_permissions_read ON user_permissions (user_id) 
WHERE has_permission(permissions, 1);

-- JSONB GIN indexes for complex flag queries
CREATE INDEX idx_preferences_notifications 
ON user_preferences USING GIN ((preferences->'notifications'));
```

### Query Optimization

```sql
-- Efficient bitwise queries
EXPLAIN (ANALYZE, BUFFERS) 
SELECT u.name FROM users u
JOIN user_permissions up ON u.id = up.user_id
WHERE (up.permissions & 8) = 8; -- Check admin permission

-- Use EXISTS for better performance with complex flag conditions
SELECT u.name FROM users u
WHERE EXISTS (
    SELECT 1 FROM user_flags uf 
    WHERE uf.user_id = u.id 
    AND uf.flag_name = 'is_premium' 
    AND uf.flag_value = TRUE
);
```

## Best Practices

### 1. Flag Naming Conventions

```sql
-- Good: Clear, consistent naming
is_active BOOLEAN
is_verified BOOLEAN
has_premium_access BOOLEAN
email_notifications_enabled BOOLEAN

-- Bad: Ambiguous naming
active INTEGER  -- What does 0 vs 1 mean?
status BOOLEAN  -- Status of what?
flag1 BOOLEAN   -- What is flag1?
```

### 2. Default Values and NULL Handling

```sql
-- Good: Explicit defaults, avoid NULL for flags
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    
    -- Explicit defaults
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE NOT NULL,
    newsletter_subscribed BOOLEAN DEFAULT FALSE NOT NULL,
    
    -- Use NULL only when "unknown" is a valid state
    age_verified BOOLEAN DEFAULT NULL -- NULL = not yet verified
);
```

### 3. Flag Migration and Versioning

```sql
-- Version your flag changes
CREATE TABLE flag_schema_versions (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);

-- Migration function example
CREATE OR REPLACE FUNCTION migrate_flags_v2()
RETURNS VOID AS $$
BEGIN
    -- Add new columns with proper defaults
    ALTER TABLE users ADD COLUMN gdpr_consent BOOLEAN DEFAULT FALSE;
    ALTER TABLE users ADD COLUMN data_processing_consent BOOLEAN DEFAULT FALSE;
    
    -- Migrate existing data
    UPDATE users SET 
        gdpr_consent = CASE 
            WHEN created_at > '2018-05-25' THEN marketing_consent 
            ELSE FALSE 
        END;
    
    -- Log migration
    INSERT INTO flag_schema_versions (version, description) 
    VALUES (2, 'Added GDPR consent flags');
END;
$$ LANGUAGE plpgsql;
```

### 4. Flag Validation

```sql
-- Add constraints to ensure flag consistency
ALTER TABLE users ADD CONSTRAINT check_verified_requires_active 
CHECK (NOT is_verified OR is_active);

ALTER TABLE users ADD CONSTRAINT check_premium_requires_verified 
CHECK (NOT is_premium OR is_verified);

-- Trigger for complex flag validation
CREATE OR REPLACE FUNCTION validate_user_flags()
RETURNS TRIGGER AS $$
BEGIN
    -- Custom validation logic
    IF NEW.is_premium AND NOT NEW.is_verified THEN
        RAISE EXCEPTION 'Premium users must be verified';
    END IF;
    
    IF NEW.is_admin AND NOT NEW.is_active THEN
        RAISE EXCEPTION 'Admin users must be active';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_user_flags
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION validate_user_flags();
```

## Real-World Examples

### 1. E-commerce Product Flags

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    
    -- Product status flags
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_digital BOOLEAN DEFAULT FALSE,
    requires_shipping BOOLEAN DEFAULT TRUE,
    age_restricted BOOLEAN DEFAULT FALSE,
    
    -- Availability flags
    in_stock BOOLEAN DEFAULT TRUE,
    backorder_allowed BOOLEAN DEFAULT FALSE,
    preorder_enabled BOOLEAN DEFAULT FALSE,
    
    -- Marketing flags
    on_sale BOOLEAN DEFAULT FALSE,
    new_arrival BOOLEAN DEFAULT FALSE,
    bestseller BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Complex query with multiple flags
SELECT * FROM products 
WHERE is_active = TRUE 
  AND in_stock = TRUE 
  AND (on_sale = TRUE OR is_featured = TRUE)
  AND NOT age_restricted;
```

### 2. Content Management System

```sql
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    
    -- Publication flags
    is_published BOOLEAN DEFAULT FALSE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    
    -- Access control
    is_premium_content BOOLEAN DEFAULT FALSE,
    requires_login BOOLEAN DEFAULT FALSE,
    
    -- SEO and social
    noindex BOOLEAN DEFAULT FALSE,
    nofollow BOOLEAN DEFAULT FALSE,
    
    -- Moderation
    is_moderated BOOLEAN DEFAULT FALSE,
    flagged_content BOOLEAN DEFAULT FALSE,
    
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- View for public articles
CREATE VIEW public_articles AS
SELECT * FROM articles 
WHERE is_published = TRUE 
  AND NOT is_archived 
  AND NOT flagged_content;
```

### 3. User Account Management

```sql
CREATE TABLE user_accounts (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    
    -- Account status
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    is_suspended BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    -- Security flags
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    password_reset_required BOOLEAN DEFAULT FALSE,
    suspicious_activity_detected BOOLEAN DEFAULT FALSE,
    
    -- Preferences
    email_notifications BOOLEAN DEFAULT TRUE,
    sms_notifications BOOLEAN DEFAULT FALSE,
    marketing_emails BOOLEAN DEFAULT FALSE,
    
    -- Compliance
    gdpr_consent BOOLEAN DEFAULT FALSE,
    terms_accepted BOOLEAN DEFAULT FALSE,
    privacy_policy_accepted BOOLEAN DEFAULT FALSE,
    
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Business logic view
CREATE VIEW active_users AS
SELECT * FROM user_accounts 
WHERE is_active = TRUE 
  AND is_verified = TRUE 
  AND NOT is_suspended 
  AND NOT is_deleted;
```

This comprehensive guide provides various approaches to implementing flags in database design, from simple boolean columns to complex bitwise operations and JSON-based storage. Choose the approach that best fits your use case, considering factors like query performance, storage efficiency, and maintenance complexity.
