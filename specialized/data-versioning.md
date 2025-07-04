# Data Versioning in Database Design

Data versioning is crucial for maintaining historical context and supporting analytics across different business logic changes. This guide covers strategies for versioning data structures, business rules, and feature rollouts.

## Why Data Versioning Matters

Business logic changes frequently, but these changes are often not recorded in the database. This creates challenges for:
- Historical analytics and reporting
- A/B testing analysis
- Feature rollback scenarios
- Compliance and audit requirements

## Core Versioning Strategies

### 1. Schema Versioning

Track changes to data structures over time:

```sql
-- Version tracking table
CREATE TABLE schema_versions (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    migration_sql TEXT,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by VARCHAR(100)
);

-- Example: User table evolution
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    
    -- Version tracking
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    schema_version VARCHAR(50) NOT NULL DEFAULT '1.0',
    
    -- V1.0 fields
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    
    -- V2.0 fields (added later)
    username VARCHAR(50),
    display_name VARCHAR(100),
    
    -- V3.0 fields (social login support)
    google_id VARCHAR(100),
    facebook_id VARCHAR(100),
    github_id VARCHAR(100),
    
    UNIQUE(email),
    UNIQUE(username),
    UNIQUE(google_id),
    UNIQUE(facebook_id),
    UNIQUE(github_id)
);
```

### 2. Feature Flag Versioning

Track feature availability and user segments:

```sql
-- Feature flags table
CREATE TABLE feature_flags (
    id SERIAL PRIMARY KEY,
    flag_name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enabled_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    
    -- Rollout configuration
    rollout_percentage DECIMAL(5,2) DEFAULT 0.0,
    target_groups JSONB,
    
    -- Version tracking
    version VARCHAR(50) NOT NULL DEFAULT '1.0'
);

-- User feature access tracking
CREATE TABLE user_feature_access (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    feature_flag_id INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    
    -- Context
    user_segment VARCHAR(50),
    experiment_group VARCHAR(50),
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (feature_flag_id) REFERENCES feature_flags(id),
    
    UNIQUE(user_id, feature_flag_id, granted_at)
);
```

### 3. Business Rule Versioning

Track changes to business logic and calculations:

```sql
-- Business rules configuration
CREATE TABLE business_rules (
    id SERIAL PRIMARY KEY,
    rule_name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    rule_config JSONB NOT NULL,
    
    -- Validity period
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ NOT NULL DEFAULT 'infinity',
    
    -- Metadata
    created_by VARCHAR(100),
    description TEXT,
    
    UNIQUE(rule_name, version),
    
    -- Prevent overlapping rules
    EXCLUDE USING gist (
        rule_name WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    )
);

-- Example: Pricing rules
INSERT INTO business_rules (rule_name, version, rule_config, description)
VALUES 
    ('shipping_cost', '1.0', '{"base_cost": 5.00, "per_kg": 2.00}', 'Initial shipping calculation'),
    ('shipping_cost', '2.0', '{"base_cost": 4.00, "per_kg": 1.50, "free_threshold": 50.00}', 'Added free shipping threshold');

-- Apply business rules to transactions
CREATE TABLE order_calculations (
    id SERIAL PRIMARY KEY,
    order_id UUID NOT NULL,
    rule_name VARCHAR(100) NOT NULL,
    rule_version VARCHAR(50) NOT NULL,
    calculation_result JSONB NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (order_id) REFERENCES orders(id)
);
```

## User Segmentation and Cohort Analysis

### Track User Registration Cohorts

```sql
-- Enhanced user table with cohort tracking
CREATE TABLE users_versioned (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    
    -- Registration context
    registration_date DATE NOT NULL,
    registration_source VARCHAR(50), -- 'email', 'google', 'facebook', 'github'
    registration_version VARCHAR(50) NOT NULL, -- App version when registered
    
    -- Feature availability at registration
    available_features JSONB,
    
    -- Cohort identification
    cohort_month VARCHAR(7) GENERATED ALWAYS AS (
        TO_CHAR(registration_date, 'YYYY-MM')
    ) STORED,
    
    cohort_week VARCHAR(10) GENERATED ALWAYS AS (
        TO_CHAR(registration_date, 'YYYY-"W"WW')
    ) STORED,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Track feature introduction timeline
CREATE TABLE feature_timeline (
    id SERIAL PRIMARY KEY,
    feature_name VARCHAR(100) NOT NULL,
    introduced_at TIMESTAMPTZ NOT NULL,
    deprecated_at TIMESTAMPTZ,
    description TEXT,
    
    -- Impact metrics
    affected_users_count INTEGER,
    rollout_duration INTERVAL
);
```

### Cohort Analysis Queries

```sql
-- Analyze user behavior by registration cohort
WITH user_cohorts AS (
    SELECT 
        cohort_month,
        COUNT(*) as cohort_size,
        COUNT(CASE WHEN registration_source = 'google' THEN 1 END) as google_users,
        COUNT(CASE WHEN registration_source = 'facebook' THEN 1 END) as facebook_users,
        COUNT(CASE WHEN registration_source = 'email' THEN 1 END) as email_users
    FROM users_versioned
    GROUP BY cohort_month
)
SELECT 
    cohort_month,
    cohort_size,
    google_users,
    facebook_users,
    email_users,
    ROUND(google_users::DECIMAL / cohort_size * 100, 2) as google_percentage,
    ROUND(facebook_users::DECIMAL / cohort_size * 100, 2) as facebook_percentage,
    ROUND(email_users::DECIMAL / cohort_size * 100, 2) as email_percentage
FROM user_cohorts
ORDER BY cohort_month;
```

## A/B Testing and Experiment Versioning

### Experiment Tracking

```sql
-- A/B test experiments
CREATE TABLE experiments (
    id SERIAL PRIMARY KEY,
    experiment_name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    
    -- Experiment timeline
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    
    -- Experiment configuration
    traffic_allocation DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    variants JSONB NOT NULL,
    
    -- Success metrics
    primary_metric VARCHAR(100),
    secondary_metrics JSONB,
    
    -- Status
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'running', 'completed', 'paused')),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User experiment assignments
CREATE TABLE user_experiment_assignments (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    experiment_id INTEGER NOT NULL,
    variant_name VARCHAR(50) NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Context
    user_segment VARCHAR(50),
    assignment_method VARCHAR(50), -- 'random', 'targeted', 'manual'
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (experiment_id) REFERENCES experiments(id),
    
    UNIQUE(user_id, experiment_id)
);

-- Track experiment results
CREATE TABLE experiment_metrics (
    id SERIAL PRIMARY KEY,
    experiment_id INTEGER NOT NULL,
    variant_name VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(15,6) NOT NULL,
    measurement_date DATE NOT NULL,
    
    -- Additional context
    sample_size INTEGER,
    confidence_interval JSONB,
    
    FOREIGN KEY (experiment_id) REFERENCES experiments(id),
    
    UNIQUE(experiment_id, variant_name, metric_name, measurement_date)
);
```

## Configuration Versioning

### Application Configuration History

```sql
-- Configuration versions
CREATE TABLE app_configurations (
    id SERIAL PRIMARY KEY,
    config_name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    config_data JSONB NOT NULL,
    
    -- Deployment info
    deployed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deployed_by VARCHAR(100),
    deployment_environment VARCHAR(50), -- 'development', 'staging', 'production'
    
    -- Validation
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,
    validation_errors JSONB,
    
    -- Rollback info
    previous_version VARCHAR(50),
    rollback_reason TEXT,
    
    UNIQUE(config_name, version, deployment_environment)
);

-- Configuration change audit
CREATE TABLE config_changes (
    id SERIAL PRIMARY KEY,
    config_id INTEGER NOT NULL,
    change_type VARCHAR(20) NOT NULL, -- 'create', 'update', 'delete', 'rollback'
    field_path VARCHAR(200),
    old_value JSONB,
    new_value JSONB,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by VARCHAR(100),
    change_reason TEXT,
    
    FOREIGN KEY (config_id) REFERENCES app_configurations(id)
);
```

## API Version Tracking

### API Usage and Versioning

```sql
-- API versions
CREATE TABLE api_versions (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    release_date TIMESTAMPTZ NOT NULL,
    deprecation_date TIMESTAMPTZ,
    sunset_date TIMESTAMPTZ,
    
    -- Version info
    major_version INTEGER,
    minor_version INTEGER,
    patch_version INTEGER,
    
    -- Documentation
    changelog TEXT,
    breaking_changes JSONB,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'sunset'))
);

-- API usage tracking
CREATE TABLE api_usage_log (
    id SERIAL PRIMARY KEY,
    user_id UUID,
    api_version VARCHAR(50) NOT NULL,
    endpoint VARCHAR(200) NOT NULL,
    method VARCHAR(10) NOT NULL,
    
    -- Request details
    request_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    response_status INTEGER,
    response_time_ms INTEGER,
    
    -- Client info
    client_version VARCHAR(50),
    user_agent TEXT,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (api_version) REFERENCES api_versions(version)
);
```

## Migration and Rollback Strategies

### Data Migration Versioning

```sql
-- Migration tracking
CREATE TABLE data_migrations (
    id SERIAL PRIMARY KEY,
    migration_name VARCHAR(100) NOT NULL UNIQUE,
    version VARCHAR(50) NOT NULL,
    
    -- Migration details
    migration_type VARCHAR(50), -- 'schema', 'data', 'cleanup'
    up_sql TEXT,
    down_sql TEXT,
    
    -- Execution info
    executed_at TIMESTAMPTZ,
    execution_duration INTERVAL,
    records_affected INTEGER,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'rolled_back')),
    error_message TEXT,
    
    -- Dependencies
    depends_on VARCHAR(100)[],
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Rollback procedures
CREATE OR REPLACE FUNCTION rollback_migration(p_migration_name VARCHAR(100))
RETURNS BOOLEAN AS $$
DECLARE
    migration_record RECORD;
BEGIN
    -- Get migration details
    SELECT * INTO migration_record 
    FROM data_migrations 
    WHERE migration_name = p_migration_name 
      AND status = 'completed';
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Migration % not found or not completed', p_migration_name;
    END IF;
    
    -- Execute rollback
    EXECUTE migration_record.down_sql;
    
    -- Update status
    UPDATE data_migrations 
    SET status = 'rolled_back',
        executed_at = NOW()
    WHERE migration_name = p_migration_name;
    
    RETURN TRUE;
EXCEPTION
    WHEN OTHERS THEN
        -- Log error
        UPDATE data_migrations 
        SET status = 'failed',
            error_message = SQLERRM
        WHERE migration_name = p_migration_name;
        
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql;
```

## Analytics and Reporting

### Version-Aware Analytics

```sql
-- User behavior by app version
CREATE VIEW user_behavior_by_version AS
SELECT 
    uv.registration_version as app_version,
    COUNT(*) as user_count,
    AVG(EXTRACT(EPOCH FROM NOW() - uv.created_at)/86400) as avg_user_age_days,
    
    -- Feature adoption
    COUNT(CASE WHEN uv.available_features ? 'social_login' THEN 1 END) as social_login_users,
    COUNT(CASE WHEN uv.available_features ? 'advanced_search' THEN 1 END) as advanced_search_users,
    
    -- Engagement metrics
    AVG(engagement_score) as avg_engagement
FROM users_versioned uv
LEFT JOIN user_engagement_metrics uem ON uv.id = uem.user_id
GROUP BY uv.registration_version;

-- Feature impact analysis
CREATE VIEW feature_impact_analysis AS
SELECT 
    ft.feature_name,
    ft.introduced_at,
    
    -- Before/after user counts
    COUNT(CASE WHEN u.created_at < ft.introduced_at THEN 1 END) as users_before,
    COUNT(CASE WHEN u.created_at >= ft.introduced_at THEN 1 END) as users_after,
    
    -- Adoption metrics
    COUNT(CASE WHEN ufa.enabled AND u.created_at >= ft.introduced_at THEN 1 END) as feature_adopters,
    
    -- Calculate adoption rate
    ROUND(
        COUNT(CASE WHEN ufa.enabled AND u.created_at >= ft.introduced_at THEN 1 END)::DECIMAL / 
        NULLIF(COUNT(CASE WHEN u.created_at >= ft.introduced_at THEN 1 END), 0) * 100, 2
    ) as adoption_rate_percent
    
FROM feature_timeline ft
LEFT JOIN users u ON true
LEFT JOIN feature_flags ff ON ft.feature_name = ff.flag_name
LEFT JOIN user_feature_access ufa ON ff.id = ufa.feature_flag_id AND u.id = ufa.user_id
GROUP BY ft.feature_name, ft.introduced_at;
```

## Best Practices

1. **Version everything** - Schema, configuration, business rules, and API changes
2. **Maintain backward compatibility** - Support multiple versions during transitions
3. **Document changes** - Keep detailed changelogs and migration notes
4. **Test rollbacks** - Ensure you can revert changes safely
5. **Monitor impact** - Track metrics before and after changes
6. **Use feature flags** - Enable gradual rollouts and easy rollbacks
7. **Segment users** - Analyze impact on different user cohorts
8. **Archive old versions** - Keep historical data but optimize storage

## Common Pitfalls

- Not tracking when features were introduced
- Mixing users from different feature sets in analytics
- Losing historical context during schema changes
- Poor rollback procedures
- Not versioning configuration changes
- Inadequate testing of version compatibility
- Missing impact analysis on existing users

## Related Patterns

- [Feature Flags](../patterns/feature-flags.md)
- [Audit Logging](../patterns/audit-logging.md)
- [Temporal Data](temporal.md)
- [A/B Testing](../patterns/ab-testing.md)
