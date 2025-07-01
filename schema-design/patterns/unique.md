# Unique Constraints & Index Patterns

Unique constraints are fundamental to data integrity, ensuring no duplicate values exist for specified columns. This guide covers unique constraint patterns, limitations, and advanced techniques for maintaining data uniqueness.

## 🎯 Core Unique Constraint Patterns

### 1. Single Column Uniqueness

```sql
-- Basic unique constraint
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Alternative syntax with named constraints
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    username TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT uk_users_email UNIQUE (email),
    CONSTRAINT uk_users_username UNIQUE (username)
);
```

### 2. Composite Unique Constraints

```sql
-- Multiple columns must be unique together
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    assigned_at TIMESTAMP DEFAULT NOW(),
    
    -- User can have same role in different organizations
    UNIQUE (user_id, role_id, organization_id),
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

-- Friend relationships (prevent duplicate friendships)
CREATE TABLE friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id_1 UUID NOT NULL,
    user_id_2 UUID NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure friendship uniqueness regardless of order
    CONSTRAINT uk_friendship UNIQUE (
        LEAST(user_id_1, user_id_2), 
        GREATEST(user_id_1, user_id_2)
    ),
    
    -- Prevent self-friendship
    CONSTRAINT chk_no_self_friendship CHECK (user_id_1 != user_id_2)
);
```

## 🔧 Advanced Unique Patterns

### 1. Conditional Uniqueness (Partial Unique Indexes)

```sql
-- Only active users need unique usernames
CREATE TABLE users_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Partial unique index - only enforce uniqueness for active users
CREATE UNIQUE INDEX uk_active_users_username 
ON users_v2 (username) 
WHERE status = 'active';

CREATE UNIQUE INDEX uk_active_users_email 
ON users_v2 (email) 
WHERE status = 'active';

-- Allow multiple users with same username if not active
INSERT INTO users_v2 (username, email, status) VALUES 
('john', 'john1@example.com', 'active'),
('john', 'john2@example.com', 'deleted'), -- ✅ Allowed
('john', 'john3@example.com', 'suspended'); -- ✅ Allowed
```

### 2. Time-Based Unique Constraints

```sql
-- Only one active session per user at a time
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    session_token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Partial unique index for active sessions
CREATE UNIQUE INDEX uk_active_user_session 
ON user_sessions (user_id) 
WHERE is_active = true;

-- Or time-based uniqueness
CREATE UNIQUE INDEX uk_current_user_session 
ON user_sessions (user_id) 
WHERE expires_at > NOW();
```

### 3. Hierarchical Uniqueness

```sql
-- Category names must be unique within their parent
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    parent_id UUID,
    slug TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    
    -- Name unique within same parent (NULL parent = root level)
    UNIQUE (name, parent_id),
    
    -- Slug globally unique for URL generation
    UNIQUE (slug),
    
    FOREIGN KEY (parent_id) REFERENCES categories(id),
    
    -- Prevent deep nesting
    CONSTRAINT chk_max_level CHECK (level <= 5)
);

-- Function to generate unique slugs
CREATE OR REPLACE FUNCTION generate_unique_slug(base_name TEXT)
RETURNS TEXT AS $$
DECLARE
    base_slug TEXT;
    final_slug TEXT;
    counter INTEGER := 1;
BEGIN
    -- Convert name to slug format
    base_slug := lower(regexp_replace(base_name, '[^a-zA-Z0-9]+', '-', 'g'));
    base_slug := trim(both '-' from base_slug);
    
    final_slug := base_slug;
    
    -- Find unique slug
    WHILE EXISTS(SELECT 1 FROM categories WHERE slug = final_slug) LOOP
        final_slug := base_slug || '-' || counter;
        counter := counter + 1;
    END LOOP;
    
    RETURN final_slug;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Database-Specific Unique Behaviors

### 1. PostgreSQL: NULLS NOT DISTINCT (v15+)

```sql
-- PostgreSQL 15+ allows only one NULL value in unique constraint
CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT,
    phone TEXT,
    
    -- Before PG15: multiple NULLs allowed
    -- PG15+: only one NULL allowed with NULLS NOT DISTINCT
    UNIQUE NULLS NOT DISTINCT (email),
    UNIQUE NULLS NOT DISTINCT (phone)
);

-- Before PostgreSQL 15, use partial indexes for single NULL
CREATE UNIQUE INDEX uk_single_null_email 
ON contacts (email) 
WHERE email IS NOT NULL;

CREATE UNIQUE INDEX uk_single_null_constraint 
ON contacts ((1)) 
WHERE email IS NULL;
```

### 2. MySQL: Unique Index Length Limits

```sql
-- MySQL has a 3072 character limit for unique indexes
CREATE TABLE articles (
    id INT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    
    -- For long text fields, use hash for uniqueness
    content_hash CHAR(64) AS (SHA2(content, 256)) STORED,
    
    UNIQUE KEY uk_content_hash (content_hash),
    KEY idx_title (title)
);

-- Prefix indexes for long VARCHAR fields
CREATE TABLE urls (
    id INT PRIMARY KEY AUTO_INCREMENT,
    full_url TEXT NOT NULL,
    
    -- Use first 100 characters + hash for practical uniqueness
    url_prefix VARCHAR(100) AS (LEFT(full_url, 100)) STORED,
    url_hash CHAR(64) AS (SHA2(full_url, 256)) STORED,
    
    UNIQUE KEY uk_url_unique (url_prefix, url_hash)
);
```

## 🛡️ Handling Unique Constraint Violations

### 1. Graceful Violation Handling

```sql
-- Function to handle unique violations gracefully
CREATE OR REPLACE FUNCTION upsert_user(
    p_email TEXT,
    p_name TEXT,
    p_username TEXT
) RETURNS UUID AS $$
DECLARE
    user_id UUID;
    attempt_count INTEGER := 0;
    final_username TEXT;
BEGIN
    -- Try to insert with original username
    final_username := p_username;
    
    LOOP
        BEGIN
            INSERT INTO users (email, name, username)
            VALUES (p_email, p_name, final_username)
            RETURNING id INTO user_id;
            
            EXIT; -- Success, exit loop
            
        EXCEPTION 
            WHEN unique_violation THEN
                -- Check which constraint was violated
                GET STACKED DIAGNOSTICS constraint_name = CONSTRAINT_NAME;
                
                IF constraint_name = 'uk_users_email' THEN
                    -- Email already exists, return existing user
                    SELECT id INTO user_id FROM users WHERE email = p_email;
                    EXIT;
                    
                ELSIF constraint_name = 'uk_users_username' THEN
                    -- Username taken, try with suffix
                    attempt_count := attempt_count + 1;
                    final_username := p_username || '_' || attempt_count;
                    
                    IF attempt_count > 100 THEN
                        RAISE EXCEPTION 'Could not generate unique username after % attempts', attempt_count;
                    END IF;
                    
                ELSE
                    RAISE; -- Re-raise unexpected constraint violations
                END IF;
        END;
    END LOOP;
    
    RETURN user_id;
END;
$$ LANGUAGE plpgsql;
```

### 2. Application-Level Unique Validation

```sql
-- Pre-check function to validate uniqueness before insert
CREATE OR REPLACE FUNCTION check_user_uniqueness(
    p_email TEXT,
    p_username TEXT,
    p_exclude_user_id UUID DEFAULT NULL
) RETURNS TABLE(
    email_available BOOLEAN,
    username_available BOOLEAN,
    suggested_username TEXT
) AS $$
DECLARE
    email_exists BOOLEAN;
    username_exists BOOLEAN;
    counter INTEGER := 1;
    suggested TEXT;
BEGIN
    -- Check email availability
    SELECT EXISTS(
        SELECT 1 FROM users 
        WHERE email = p_email 
        AND (p_exclude_user_id IS NULL OR id != p_exclude_user_id)
    ) INTO email_exists;
    
    -- Check username availability
    SELECT EXISTS(
        SELECT 1 FROM users 
        WHERE username = p_username 
        AND (p_exclude_user_id IS NULL OR id != p_exclude_user_id)
    ) INTO username_exists;
    
    -- Generate suggested username if needed
    suggested := p_username;
    WHILE username_exists LOOP
        suggested := p_username || counter;
        SELECT EXISTS(
            SELECT 1 FROM users 
            WHERE username = suggested 
            AND (p_exclude_user_id IS NULL OR id != p_exclude_user_id)
        ) INTO username_exists;
        counter := counter + 1;
    END LOOP;
    
    RETURN QUERY SELECT 
        NOT email_exists,
        p_username = suggested,
        suggested;
END;
$$ LANGUAGE plpgsql;
```

## 🚀 Performance Optimization

### 1. Unique Index Optimization

```sql
-- Use expression indexes for case-insensitive uniqueness
CREATE UNIQUE INDEX uk_users_email_ci 
ON users (LOWER(email));

CREATE UNIQUE INDEX uk_users_username_ci 
ON users (LOWER(username));

-- Partial indexes for better performance
CREATE UNIQUE INDEX uk_published_articles_slug 
ON articles (slug) 
WHERE status = 'published';

-- Composite indexes with selective ordering
CREATE UNIQUE INDEX uk_user_organization_role 
ON user_roles (user_id, organization_id, role_id);
-- Put most selective column first for better performance
```

### 2. Monitoring Unique Constraint Performance

```sql
-- View to monitor unique constraint violations
CREATE VIEW unique_constraint_violations AS
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch,
    idx_blks_read,
    idx_blks_hit
FROM pg_stat_user_indexes 
WHERE indexname LIKE '%unique%' OR indexname LIKE '%uk_%'
ORDER BY idx_tup_read DESC;

-- Function to find duplicate values before adding unique constraint
CREATE OR REPLACE FUNCTION find_duplicates(
    table_name TEXT,
    column_names TEXT[]
) RETURNS TABLE(
    duplicate_values TEXT,
    count BIGINT
) AS $$
DECLARE
    sql_query TEXT;
    col_list TEXT;
BEGIN
    col_list := array_to_string(column_names, ', ');
    
    sql_query := format('
        SELECT %s::TEXT as duplicate_values, COUNT(*) as count
        FROM %I 
        GROUP BY %s 
        HAVING COUNT(*) > 1 
        ORDER BY COUNT(*) DESC',
        col_list, table_name, col_list
    );
    
    RETURN QUERY EXECUTE sql_query;
END;
$$ LANGUAGE plpgsql;

-- Usage: SELECT * FROM find_duplicates('users', ARRAY['email']);
```

## ⚠️ Common Pitfalls

### 1. Forgotten NULL Behavior
```sql
-- ❌ Multiple NULLs are allowed in most databases
CREATE TABLE contacts (
    email TEXT UNIQUE -- Allows multiple NULL emails
);

-- ✅ Handle NULLs explicitly
CREATE TABLE contacts (
    email TEXT,
    UNIQUE (email) -- Explicitly decide NULL behavior
);

-- Or use NOT NULL if required
CREATE TABLE contacts (
    email TEXT UNIQUE NOT NULL
);
```

### 2. Case-Sensitivity Issues
```sql
-- ❌ 'John@Example.com' and 'john@example.com' are different
CREATE TABLE users (
    email TEXT UNIQUE
);

-- ✅ Use expression index for case-insensitive uniqueness
CREATE UNIQUE INDEX uk_users_email_ci ON users (LOWER(email));
```

### 3. Composite Key Ordering
```sql
-- ❌ Less efficient for single-column queries
CREATE UNIQUE INDEX uk_inefficient (less_selective_col, more_selective_col);

-- ✅ Most selective column first
CREATE UNIQUE INDEX uk_efficient (more_selective_col, less_selective_col);
```

## 🎯 Best Practices

1. **Name Constraints Explicitly** - Use descriptive names like `uk_users_email`
2. **Consider Case Sensitivity** - Use expression indexes for case-insensitive uniqueness
3. **Handle NULLs Deliberately** - Decide whether NULLs should be unique or not
4. **Use Partial Indexes** - For conditional uniqueness requirements
5. **Order Composite Keys** - Put most selective columns first
6. **Validate Before Insert** - Check uniqueness in application when possible
7. **Monitor Performance** - Track unique constraint performance metrics
8. **Plan for Growth** - Consider uniqueness scalability as data grows
9. **Document Business Rules** - Clearly explain uniqueness requirements
10. **Test Edge Cases** - Unicode, special characters, length limits

## 📊 Performance Comparison

| Pattern | Use Case | Performance | Complexity |
|---------|----------|-------------|------------|
| Single Column | Simple uniqueness | ⭐⭐⭐⭐⭐ | ⭐ |
| Composite | Multi-column uniqueness | ⭐⭐⭐⭐ | ⭐⭐ |
| Partial Index | Conditional uniqueness | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Expression Index | Case-insensitive | ⭐⭐⭐⭐ | ⭐⭐ |
| Hash-based | Long text uniqueness | ⭐⭐⭐ | ⭐⭐⭐⭐ |

## 🔗 References

- [PostgreSQL Unique Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-UNIQUE-CONSTRAINTS)
- [MySQL Unique Indexes](https://dev.mysql.com/doc/refman/8.0/en/create-index.html#create-index-unique)
- [PostgreSQL 15 NULLS NOT DISTINCT](https://www.postgresql.org/docs/15/sql-createindex.html)
- [Index Length Limits](https://dev.mysql.com/doc/refman/8.0/en/innodb-limits.html)
