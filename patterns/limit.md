# Limiting & Quota Patterns

Database-level limiting patterns help enforce business rules, prevent abuse, and maintain data integrity. This guide covers various approaches to implementing limits, quotas, and constraints.

## 🎯 Core Limiting Patterns

### 1. LIMIT with TIES

**Best for**: Returning top N records including ties

```sql
-- Standard LIMIT - stops at exactly N rows
SELECT name, salary, department
FROM employees
ORDER BY salary DESC
LIMIT 3;

-- LIMIT WITH TIES - includes tied values
SELECT name, salary, department
FROM employees
ORDER BY salary DESC
FETCH FIRST 3 ROWS WITH TIES;

-- PostgreSQL equivalent with window functions
SELECT name, salary, department
FROM (
    SELECT 
        name, 
        salary, 
        department,
        DENSE_RANK() OVER (ORDER BY salary DESC) as rank
    FROM employees
) ranked
WHERE rank <= 3;
```

### 2. Row Limits Per Category

**Best for**: Limiting entries per user/category/group

```sql
-- Limit users to 3 posts per day
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Constraint function to limit posts per day
CREATE OR REPLACE FUNCTION check_daily_post_limit()
RETURNS TRIGGER AS $$
BEGIN
    IF (
        SELECT COUNT(*) 
        FROM posts 
        WHERE user_id = NEW.user_id 
        AND DATE(created_at) = DATE(NEW.created_at)
    ) >= 3 THEN
        RAISE EXCEPTION 'User has reached daily post limit of 3';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER limit_daily_posts
    BEFORE INSERT ON posts
    FOR EACH ROW
    EXECUTE FUNCTION check_daily_post_limit();
```

### 3. Quota-Based Limiting

**Best for**: Resource usage quotas and consumption tracking

```sql
-- User quotas table
CREATE TABLE user_quotas (
    user_id UUID PRIMARY KEY,
    storage_bytes_limit BIGINT NOT NULL DEFAULT 1073741824, -- 1GB
    storage_bytes_used BIGINT NOT NULL DEFAULT 0,
    api_calls_limit INTEGER NOT NULL DEFAULT 1000,
    api_calls_used INTEGER NOT NULL DEFAULT 0,
    reset_date DATE NOT NULL DEFAULT CURRENT_DATE + INTERVAL '1 month',
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    
    -- Ensure usage doesn't exceed limits
    CONSTRAINT chk_storage_within_limit 
        CHECK (storage_bytes_used <= storage_bytes_limit),
    CONSTRAINT chk_api_calls_within_limit 
        CHECK (api_calls_used <= api_calls_limit)
);

-- Function to check and update quota
CREATE OR REPLACE FUNCTION consume_api_quota(
    p_user_id UUID,
    p_calls_to_consume INTEGER DEFAULT 1
) RETURNS BOOLEAN AS $$
DECLARE
    quota_available BOOLEAN := FALSE;
BEGIN
    UPDATE user_quotas
    SET api_calls_used = api_calls_used + p_calls_to_consume
    WHERE user_id = p_user_id
    AND api_calls_used + p_calls_to_consume <= api_calls_limit
    AND reset_date >= CURRENT_DATE;
    
    quota_available := FOUND;
    
    -- Reset quota if past reset date
    IF NOT quota_available THEN
        UPDATE user_quotas
        SET 
            api_calls_used = p_calls_to_consume,
            reset_date = CURRENT_DATE + INTERVAL '1 month'
        WHERE user_id = p_user_id
        AND reset_date < CURRENT_DATE
        AND p_calls_to_consume <= api_calls_limit;
        
        quota_available := FOUND;
    END IF;
    
    RETURN quota_available;
END;
$$ LANGUAGE plpgsql;
```

## 🎫 Coupon & Redemption Patterns

### 1. Basic Coupon Limiting

```sql
-- Coupons with redemption limits
CREATE TABLE coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT UNIQUE NOT NULL,
    discount_percent DECIMAL(5,2) NOT NULL,
    max_redemptions INTEGER NOT NULL CHECK (max_redemptions > 0),
    current_redemptions INTEGER NOT NULL DEFAULT 0,
    valid_from TIMESTAMP DEFAULT NOW(),
    valid_until TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT true,
    
    -- Ensure redemptions don't exceed limit
    CONSTRAINT chk_redemptions_within_limit 
        CHECK (current_redemptions <= max_redemptions)
);

-- User redemptions tracking
CREATE TABLE user_coupon_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    coupon_id UUID NOT NULL,
    redeemed_at TIMESTAMP DEFAULT NOW(),
    order_id UUID,
    
    -- Each user can only redeem each coupon once
    UNIQUE (user_id, coupon_id),
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (coupon_id) REFERENCES coupons(id)
);

-- Thread-safe coupon redemption function
CREATE OR REPLACE FUNCTION redeem_coupon(
    p_user_id UUID,
    p_coupon_code TEXT,
    p_order_id UUID DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    coupon_record RECORD;
    redemption_successful BOOLEAN := FALSE;
BEGIN
    -- Lock and validate coupon
    SELECT id, max_redemptions, current_redemptions, valid_until, is_active
    INTO coupon_record
    FROM coupons
    WHERE code = p_coupon_code
    AND is_active = true
    AND valid_from <= NOW()
    AND valid_until >= NOW()
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RETURN FALSE; -- Coupon not found or expired
    END IF;
    
    -- Check if user already redeemed this coupon
    IF EXISTS (
        SELECT 1 FROM user_coupon_redemptions 
        WHERE user_id = p_user_id AND coupon_id = coupon_record.id
    ) THEN
        RETURN FALSE; -- Already redeemed
    END IF;
    
    -- Check redemption limit
    IF coupon_record.current_redemptions >= coupon_record.max_redemptions THEN
        RETURN FALSE; -- Limit reached
    END IF;
    
    -- Redeem coupon
    INSERT INTO user_coupon_redemptions (user_id, coupon_id, order_id)
    VALUES (p_user_id, coupon_record.id, p_order_id);
    
    UPDATE coupons
    SET current_redemptions = current_redemptions + 1
    WHERE id = coupon_record.id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### 2. Advanced Coupon Patterns

```sql
-- Multi-tier coupon system
CREATE TABLE coupon_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    max_redemptions_per_user INTEGER NOT NULL DEFAULT 1,
    max_total_redemptions INTEGER NOT NULL,
    current_total_redemptions INTEGER NOT NULL DEFAULT 0,
    tier_priority INTEGER NOT NULL DEFAULT 1,
    
    CONSTRAINT chk_total_redemptions_limit 
        CHECK (current_total_redemptions <= max_total_redemptions)
);

CREATE TABLE tiered_coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_id UUID NOT NULL,
    code TEXT UNIQUE NOT NULL,
    discount_amount DECIMAL(10,2),
    discount_percent DECIMAL(5,2),
    minimum_order_amount DECIMAL(10,2) DEFAULT 0,
    valid_from TIMESTAMP DEFAULT NOW(),
    valid_until TIMESTAMP NOT NULL,
    
    FOREIGN KEY (tier_id) REFERENCES coupon_tiers(id),
    
    -- Either amount or percent discount, not both
    CONSTRAINT chk_discount_type 
        CHECK ((discount_amount IS NULL) != (discount_percent IS NULL))
);

-- User redemptions with tier tracking
CREATE TABLE user_tier_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tier_id UUID NOT NULL,
    coupon_id UUID NOT NULL,
    redemption_count INTEGER NOT NULL DEFAULT 1,
    last_redeemed_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (tier_id) REFERENCES coupon_tiers(id),
    FOREIGN KEY (coupon_id) REFERENCES tiered_coupons(id),
    
    UNIQUE (user_id, tier_id)
);
```

## 🔄 Rate Limiting Patterns

### 1. Time-Window Rate Limiting

```sql
-- API rate limiting table
CREATE TABLE api_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    endpoint TEXT NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 1,
    window_start TIMESTAMP NOT NULL DEFAULT NOW(),
    window_duration INTERVAL NOT NULL DEFAULT '1 hour',
    max_requests INTEGER NOT NULL DEFAULT 100,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    
    -- Composite index for efficient lookups
    UNIQUE (user_id, endpoint, window_start)
);

-- Rate limiting function
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_user_id UUID,
    p_endpoint TEXT,
    p_max_requests INTEGER DEFAULT 100,
    p_window_duration INTERVAL DEFAULT '1 hour'
) RETURNS BOOLEAN AS $$
DECLARE
    current_window TIMESTAMP;
    current_count INTEGER;
    within_limit BOOLEAN := FALSE;
BEGIN
    -- Calculate current window start
    current_window := date_trunc('hour', NOW());
    
    -- Upsert rate limit record
    INSERT INTO api_rate_limits (
        user_id, endpoint, request_count, window_start, 
        window_duration, max_requests
    )
    VALUES (
        p_user_id, p_endpoint, 1, current_window,
        p_window_duration, p_max_requests
    )
    ON CONFLICT (user_id, endpoint, window_start)
    DO UPDATE SET 
        request_count = api_rate_limits.request_count + 1
    RETURNING request_count INTO current_count;
    
    within_limit := current_count <= p_max_requests;
    
    RETURN within_limit;
END;
$$ LANGUAGE plpgsql;
```

### 2. Sliding Window Rate Limiting

```sql
-- More granular rate limiting with sliding window
CREATE TABLE request_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    endpoint TEXT NOT NULL,
    request_timestamp TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Sliding window rate limit check
CREATE OR REPLACE FUNCTION check_sliding_rate_limit(
    p_user_id UUID,
    p_endpoint TEXT,
    p_max_requests INTEGER DEFAULT 100,
    p_window_duration INTERVAL DEFAULT '1 hour'
) RETURNS BOOLEAN AS $$
DECLARE
    request_count INTEGER;
    window_start TIMESTAMP;
BEGIN
    window_start := NOW() - p_window_duration;
    
    -- Count requests in sliding window
    SELECT COUNT(*)
    INTO request_count
    FROM request_logs
    WHERE user_id = p_user_id
    AND endpoint = p_endpoint
    AND request_timestamp >= window_start;
    
    -- Log current request
    INSERT INTO request_logs (user_id, endpoint)
    VALUES (p_user_id, p_endpoint);
    
    -- Clean up old requests periodically
    DELETE FROM request_logs
    WHERE request_timestamp < NOW() - INTERVAL '24 hours';
    
    RETURN request_count < p_max_requests;
END;
$$ LANGUAGE plpgsql;
```

## 🏗️ Advanced Limiting Patterns

### 1. Generic Counter System

```sql
-- Flexible counter system for various limits
CREATE TABLE counters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL, -- 'user', 'organization', 'api_key', etc.
    entity_id UUID NOT NULL,
    counter_type TEXT NOT NULL, -- 'posts', 'api_calls', 'storage', etc.
    
    current_value BIGINT NOT NULL DEFAULT 0,
    limit_value BIGINT NOT NULL DEFAULT 0, -- 0 = no limit
    reset_period INTERVAL, -- NULL = no reset
    last_reset TIMESTAMP DEFAULT NOW(),
    
    metadata JSONB DEFAULT '{}',
    
    -- Composite key for entity + counter type
    UNIQUE (entity_type, entity_id, counter_type),
    
    -- Ensure current value doesn't exceed limit (when limit > 0)
    CONSTRAINT chk_within_limit 
        CHECK (limit_value = 0 OR current_value <= limit_value)
);

-- Generic counter increment function
CREATE OR REPLACE FUNCTION increment_counter(
    p_entity_type TEXT,
    p_entity_id UUID,
    p_counter_type TEXT,
    p_increment BIGINT DEFAULT 1,
    p_limit BIGINT DEFAULT 0,
    p_reset_period INTERVAL DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    counter_record RECORD;
    increment_successful BOOLEAN := FALSE;
BEGIN
    -- Lock and get current counter
    SELECT current_value, limit_value, last_reset, reset_period
    INTO counter_record
    FROM counters
    WHERE entity_type = p_entity_type
    AND entity_id = p_entity_id
    AND counter_type = p_counter_type
    FOR UPDATE;
    
    -- Create counter if doesn't exist
    IF NOT FOUND THEN
        INSERT INTO counters (
            entity_type, entity_id, counter_type, 
            current_value, limit_value, reset_period
        )
        VALUES (
            p_entity_type, p_entity_id, p_counter_type,
            p_increment, p_limit, p_reset_period
        );
        RETURN TRUE;
    END IF;
    
    -- Check if reset needed
    IF counter_record.reset_period IS NOT NULL 
    AND counter_record.last_reset + counter_record.reset_period <= NOW() THEN
        UPDATE counters
        SET 
            current_value = p_increment,
            last_reset = NOW()
        WHERE entity_type = p_entity_type
        AND entity_id = p_entity_id
        AND counter_type = p_counter_type;
        RETURN TRUE;
    END IF;
    
    -- Check limit
    IF counter_record.limit_value > 0 
    AND counter_record.current_value + p_increment > counter_record.limit_value THEN
        RETURN FALSE; -- Would exceed limit
    END IF;
    
    -- Increment counter
    UPDATE counters
    SET current_value = current_value + p_increment
    WHERE entity_type = p_entity_type
    AND entity_id = p_entity_id
    AND counter_type = p_counter_type;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### 2. Hierarchical Limits

```sql
-- Organization-level limits that cascade to users
CREATE TABLE organization_limits (
    organization_id UUID PRIMARY KEY,
    max_users INTEGER NOT NULL DEFAULT 100,
    max_projects INTEGER NOT NULL DEFAULT 50,
    max_storage_gb INTEGER NOT NULL DEFAULT 100,
    
    current_users INTEGER NOT NULL DEFAULT 0,
    current_projects INTEGER NOT NULL DEFAULT 0,
    current_storage_gb INTEGER NOT NULL DEFAULT 0,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
    
    CONSTRAINT chk_users_limit CHECK (current_users <= max_users),
    CONSTRAINT chk_projects_limit CHECK (current_projects <= max_projects),
    CONSTRAINT chk_storage_limit CHECK (current_storage_gb <= max_storage_gb)
);

-- User-level limits within organization context
CREATE TABLE user_limits (
    user_id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    max_projects INTEGER NOT NULL DEFAULT 10,
    max_storage_gb INTEGER NOT NULL DEFAULT 5,
    
    current_projects INTEGER NOT NULL DEFAULT 0,
    current_storage_gb INTEGER NOT NULL DEFAULT 0,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
    
    CONSTRAINT chk_user_projects_limit CHECK (current_projects <= max_projects),
    CONSTRAINT chk_user_storage_limit CHECK (current_storage_gb <= max_storage_gb)
);

-- Function to check hierarchical limits
CREATE OR REPLACE FUNCTION can_create_project(
    p_user_id UUID,
    p_organization_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    org_limit RECORD;
    user_limit RECORD;
BEGIN
    -- Check organization limits
    SELECT max_projects, current_projects
    INTO org_limit
    FROM organization_limits
    WHERE organization_id = p_organization_id;
    
    IF org_limit.current_projects >= org_limit.max_projects THEN
        RETURN FALSE; -- Organization limit reached
    END IF;
    
    -- Check user limits
    SELECT max_projects, current_projects
    INTO user_limit
    FROM user_limits
    WHERE user_id = p_user_id;
    
    IF user_limit.current_projects >= user_limit.max_projects THEN
        RETURN FALSE; -- User limit reached
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Monitoring & Analytics

### 1. Limit Usage Monitoring

```sql
-- View for monitoring limit usage
CREATE VIEW limit_usage_summary AS
SELECT 
    entity_type,
    counter_type,
    COUNT(*) as total_entities,
    AVG(current_value::FLOAT / NULLIF(limit_value, 0) * 100) as avg_usage_percent,
    COUNT(*) FILTER (
        WHERE limit_value > 0 AND current_value::FLOAT / limit_value > 0.8
    ) as near_limit_count,
    COUNT(*) FILTER (
        WHERE limit_value > 0 AND current_value >= limit_value
    ) as at_limit_count
FROM counters
WHERE limit_value > 0
GROUP BY entity_type, counter_type;

-- Alert for entities approaching limits
CREATE OR REPLACE FUNCTION alert_limit_warnings()
RETURNS TABLE(
    entity_type TEXT,
    entity_id UUID,
    counter_type TEXT,
    usage_percent NUMERIC,
    current_value BIGINT,
    limit_value BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c.entity_type,
        c.entity_id,
        c.counter_type,
        ROUND((c.current_value::NUMERIC / c.limit_value * 100), 2) as usage_percent,
        c.current_value,
        c.limit_value
    FROM counters c
    WHERE c.limit_value > 0
    AND c.current_value::FLOAT / c.limit_value > 0.8
    ORDER BY usage_percent DESC;
END;
$$ LANGUAGE plpgsql;
```

## ⚠️ Best Practices

1. **Use Appropriate Locking** - FOR UPDATE for critical counters
2. **Handle Race Conditions** - Use transactions for multi-step operations
3. **Set Reasonable Limits** - Balance security with usability
4. **Monitor Limit Usage** - Alert before limits are reached
5. **Plan for Resets** - Implement time-based limit resets
6. **Graceful Degradation** - Provide meaningful error messages
7. **Audit Limit Changes** - Log when limits are modified
8. **Consider Caching** - Cache frequently checked limits
9. **Batch Operations** - Use efficient bulk limit checking
10. **Document Business Rules** - Clearly explain limit rationale

## 🔗 References

- [PostgreSQL Row Locking](https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS)
- [FETCH FIRST WITH TIES](https://www.postgresql.org/docs/current/sql-select.html#SQL-LIMIT)
- [Database Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)
- [Rate Limiting Patterns](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)
