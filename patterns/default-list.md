# Default Value & Fallback Patterns

Managing default values and fallback scenarios is crucial for robust database design. This guide covers patterns for handling defaults, option lists, cascading fallbacks, and configuration management.

## 🎯 Core Default Value Patterns

### 1. Simple Default with Override

**Use case**: System-wide defaults that users can override

```sql
-- User preferences with system defaults
CREATE TABLE user_preferences (
    user_id UUID PRIMARY KEY,
    theme TEXT DEFAULT 'light' CHECK (theme IN ('light', 'dark', 'auto')),
    language TEXT DEFAULT 'en' CHECK (language ~ '^[a-z]{2}$'),
    notifications_enabled BOOLEAN DEFAULT true,
    timezone TEXT DEFAULT 'UTC',
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Get preference with fallback to default
CREATE OR REPLACE FUNCTION get_user_preference(
    p_user_id UUID,
    p_preference TEXT,
    p_default TEXT DEFAULT NULL
) RETURNS TEXT AS $$
DECLARE
    result TEXT;
    sql_query TEXT;
BEGIN
    sql_query := format('SELECT %I FROM user_preferences WHERE user_id = $1', p_preference);
    EXECUTE sql_query INTO result USING p_user_id;
    
    -- Return user preference, or column default, or provided default
    RETURN COALESCE(result, p_default);
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT get_user_preference('user-uuid', 'theme', 'light') as user_theme;
```

### 2. Separate Defaults Table Pattern

**Use case**: Configurable system defaults across multiple entities

```sql
-- System defaults configuration
CREATE TABLE system_defaults (
    category TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    value_type TEXT DEFAULT 'string' CHECK (value_type IN ('string', 'number', 'boolean', 'json')),
    description TEXT,
    updated_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (category, key)
);

-- Insert system defaults
INSERT INTO system_defaults (category, key, value, value_type, description) VALUES
('user', 'theme', 'light', 'string', 'Default theme for new users'),
('user', 'notifications_enabled', 'true', 'boolean', 'Enable notifications by default'),
('order', 'currency', 'USD', 'string', 'Default currency for orders'),
('order', 'tax_rate', '0.10', 'number', 'Default tax rate percentage');

-- User settings with fallback to system defaults
CREATE OR REPLACE FUNCTION get_effective_setting(
    p_user_id UUID,
    p_category TEXT,
    p_key TEXT
) RETURNS TEXT AS $$
DECLARE
    user_value TEXT;
    default_value TEXT;
BEGIN
    -- Try to get user-specific value
    EXECUTE format('
        SELECT value FROM user_settings 
        WHERE user_id = $1 AND category = $2 AND key = $3
    ') INTO user_value USING p_user_id, p_category, p_key;
    
    IF user_value IS NOT NULL THEN
        RETURN user_value;
    END IF;
    
    -- Fall back to system default
    SELECT value INTO default_value
    FROM system_defaults
    WHERE category = p_category AND key = p_key;
    
    RETURN default_value;
END;
$$ LANGUAGE plpgsql;
```

## 🏗️ Advanced Default Patterns

### 1. Hierarchical Defaults

**Use case**: Organization → Team → User preference hierarchy

```sql
-- Multi-level configuration hierarchy
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    settings JSONB DEFAULT '{}'
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    name TEXT NOT NULL,
    settings JSONB DEFAULT '{}',
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    name TEXT NOT NULL,
    settings JSONB DEFAULT '{}',
    
    FOREIGN KEY (team_id) REFERENCES teams(id)
);

-- Function to resolve settings with hierarchy
CREATE OR REPLACE FUNCTION get_hierarchical_setting(
    p_user_id UUID,
    p_setting_key TEXT,
    p_system_default TEXT DEFAULT NULL
) RETURNS TEXT AS $$
DECLARE
    user_value TEXT;
    team_value TEXT;
    org_value TEXT;
    system_value TEXT;
BEGIN
    -- Check user settings first
    SELECT settings->>p_setting_key INTO user_value
    FROM users WHERE id = p_user_id;
    
    IF user_value IS NOT NULL THEN
        RETURN user_value;
    END IF;
    
    -- Check team settings
    SELECT t.settings->>p_setting_key INTO team_value
    FROM users u
    JOIN teams t ON u.team_id = t.id
    WHERE u.id = p_user_id;
    
    IF team_value IS NOT NULL THEN
        RETURN team_value;
    END IF;
    
    -- Check organization settings
    SELECT o.settings->>p_setting_key INTO org_value
    FROM users u
    JOIN teams t ON u.team_id = t.id
    JOIN organizations o ON t.organization_id = o.id
    WHERE u.id = p_user_id;
    
    IF org_value IS NOT NULL THEN
        RETURN org_value;
    END IF;
    
    -- Check system defaults
    SELECT value INTO system_value
    FROM system_defaults
    WHERE category = 'global' AND key = p_setting_key;
    
    -- Return system default or provided fallback
    RETURN COALESCE(system_value, p_system_default);
END;
$$ LANGUAGE plpgsql;
```

### 2. Time-Based Defaults

**Use case**: Defaults that change based on time periods or business rules

```sql
-- Time-sensitive default configurations
CREATE TABLE default_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    configuration JSONB NOT NULL,
    valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP,
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    
    -- Ensure no overlapping active configurations with same name
    EXCLUDE USING gist (
        name WITH =,
        tsrange(valid_from, valid_until, '[)') WITH &&
    ) WHERE (is_active = true)
);

-- Function to get current default configuration
CREATE OR REPLACE FUNCTION get_current_default(
    p_config_name TEXT,
    p_at_time TIMESTAMP DEFAULT NOW()
) RETURNS JSONB AS $$
DECLARE
    config JSONB;
BEGIN
    SELECT configuration INTO config
    FROM default_configurations
    WHERE name = p_config_name
    AND is_active = true
    AND valid_from <= p_at_time
    AND (valid_until IS NULL OR valid_until > p_at_time)
    ORDER BY priority DESC, valid_from DESC
    LIMIT 1;
    
    RETURN COALESCE(config, '{}'::JSONB);
END;
$$ LANGUAGE plpgsql;

-- Example usage for shipping rates
INSERT INTO default_configurations (name, configuration, valid_from, valid_until) VALUES
('shipping_rates', '{"standard": 5.99, "express": 12.99}', '2024-01-01', '2024-06-30'),
('shipping_rates', '{"standard": 6.99, "express": 14.99}', '2024-07-01', NULL);
```

### 3. Context-Aware Defaults

**Use case**: Defaults that vary based on user context, location, or other factors

```sql
-- Context-aware default rules
CREATE TABLE default_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    context_conditions JSONB NOT NULL, -- JSON conditions to match
    default_values JSONB NOT NULL,     -- Default values to apply
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert context-aware rules
INSERT INTO default_rules (name, context_conditions, default_values, priority) VALUES
('payment_defaults', '{"country": "USA"}', '{"currency": "USD", "tax_rate": 0.08}', 10),
('payment_defaults', '{"country": "GBR"}', '{"currency": "GBP", "tax_rate": 0.20}', 10),
('payment_defaults', '{"user_type": "premium"}', '{"shipping": "free", "priority_support": true}', 20),
('payment_defaults', '{}', '{"currency": "USD", "tax_rate": 0.00}', 0); -- Fallback

-- Function to resolve context-aware defaults
CREATE OR REPLACE FUNCTION get_context_defaults(
    p_rule_name TEXT,
    p_context JSONB
) RETURNS JSONB AS $$
DECLARE
    rule_record RECORD;
    result JSONB := '{}'::JSONB;
BEGIN
    -- Get matching rules ordered by priority
    FOR rule_record IN
        SELECT default_values, priority
        FROM default_rules
        WHERE name = p_rule_name
        AND is_active = true
        AND (
            context_conditions = '{}'::JSONB OR  -- Match-all fallback
            p_context @> context_conditions      -- Context contains all required conditions
        )
        ORDER BY priority DESC, created_at ASC
    LOOP
        -- Merge defaults (higher priority rules override lower ones)
        result := rule_record.default_values || result;
    END LOOP;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT get_context_defaults('payment_defaults', '{"country": "USA", "user_type": "premium"}'::JSONB);
-- Returns: {"currency": "USD", "tax_rate": 0.08, "shipping": "free", "priority_support": true}
```

## 🎭 Option Lists with Defaults

### 1. Enumerated Options with Default

```sql
-- Product categories with default selection
CREATE TABLE product_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    is_default BOOLEAN DEFAULT false,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true
);

-- Ensure only one default category
CREATE UNIQUE INDEX idx_single_default_category 
ON product_categories(is_default) 
WHERE is_default = true;

-- Function to get default category
CREATE OR REPLACE FUNCTION get_default_category()
RETURNS UUID AS $$
DECLARE
    default_id UUID;
BEGIN
    SELECT id INTO default_id
    FROM product_categories
    WHERE is_default = true AND is_active = true;
    
    -- If no explicit default, return first active category
    IF default_id IS NULL THEN
        SELECT id INTO default_id
        FROM product_categories
        WHERE is_active = true
        ORDER BY sort_order, name
        LIMIT 1;
    END IF;
    
    RETURN default_id;
END;
$$ LANGUAGE plpgsql;
```

### 2. Dynamic Option Lists

```sql
-- Flexible option management system
CREATE TABLE option_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_name TEXT UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true
);

CREATE TABLE option_list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id UUID NOT NULL,
    value TEXT NOT NULL,
    display_text TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    
    FOREIGN KEY (list_id) REFERENCES option_lists(id) ON DELETE CASCADE,
    UNIQUE (list_id, value)
);

-- Ensure only one default per list
CREATE UNIQUE INDEX idx_single_default_per_list 
ON option_list_items(list_id, is_default) 
WHERE is_default = true;

-- Function to get options with default marked
CREATE OR REPLACE FUNCTION get_option_list_with_default(p_list_name TEXT)
RETURNS TABLE(
    value TEXT,
    display_text TEXT,
    is_default BOOLEAN,
    sort_order INTEGER,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        oli.value,
        oli.display_text,
        oli.is_default,
        oli.sort_order,
        oli.metadata
    FROM option_list_items oli
    JOIN option_lists ol ON oli.list_id = ol.id
    WHERE ol.list_name = p_list_name
    AND ol.is_active = true
    AND oli.is_active = true
    ORDER BY oli.sort_order, oli.display_text;
END;
$$ LANGUAGE plpgsql;
```

## 🔄 Cascading Defaults & Inheritance

### 1. Template-Based Defaults

```sql
-- Template system for default configurations
CREATE TABLE configuration_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    parent_template_id UUID,
    configuration JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    
    FOREIGN KEY (parent_template_id) REFERENCES configuration_templates(id)
);

-- Function to resolve template with inheritance
CREATE OR REPLACE FUNCTION resolve_template_config(p_template_id UUID)
RETURNS JSONB AS $$
DECLARE
    final_config JSONB := '{}'::JSONB;
    current_template RECORD;
    template_chain UUID[];
    template_id UUID;
BEGIN
    -- Build inheritance chain (prevent cycles)
    current_template.id := p_template_id;
    template_chain := ARRAY[p_template_id];
    
    WHILE current_template.id IS NOT NULL LOOP
        SELECT parent_template_id, configuration 
        INTO current_template
        FROM configuration_templates
        WHERE id = current_template.id AND is_active = true;
        
        IF current_template.parent_template_id IS NOT NULL THEN
            -- Check for cycles
            IF current_template.parent_template_id = ANY(template_chain) THEN
                RAISE EXCEPTION 'Circular reference detected in template inheritance';
            END IF;
            template_chain := array_append(template_chain, current_template.parent_template_id);
        END IF;
        
        -- Merge configuration (child overrides parent)
        final_config := current_template.configuration || final_config;
        
        current_template.id := current_template.parent_template_id;
    END LOOP;
    
    RETURN final_config;
END;
$$ LANGUAGE plpgsql;
```

### 2. Multi-Source Default Resolution

```sql
-- Comprehensive default resolution with multiple sources
CREATE OR REPLACE FUNCTION resolve_multi_source_default(
    p_entity_type TEXT,
    p_entity_id UUID,
    p_setting_key TEXT,
    p_context JSONB DEFAULT '{}'::JSONB
) RETURNS TEXT AS $$
DECLARE
    result TEXT;
    temp_result TEXT;
    resolution_order TEXT[] := ARRAY[
        'entity_override',
        'context_rule',
        'entity_template',
        'global_default',
        'system_fallback'
    ];
    source TEXT;
BEGIN
    FOREACH source IN ARRAY resolution_order LOOP
        CASE source
            WHEN 'entity_override' THEN
                -- Check entity-specific override
                EXECUTE format('
                    SELECT settings->>%L 
                    FROM %I_settings 
                    WHERE entity_id = %L
                ', p_setting_key, p_entity_type, p_entity_id)
                INTO temp_result;
                
            WHEN 'context_rule' THEN
                -- Check context-aware rules
                SELECT (get_context_defaults(p_entity_type || '_defaults', p_context))->p_setting_key
                INTO temp_result;
                
            WHEN 'entity_template' THEN
                -- Check template defaults
                EXECUTE format('
                    SELECT (resolve_template_config(template_id))->>%L
                    FROM %I_templates et
                    JOIN %I e ON et.id = e.template_id
                    WHERE e.id = %L
                ', p_setting_key, p_entity_type, p_entity_type, p_entity_id)
                INTO temp_result;
                
            WHEN 'global_default' THEN
                -- Check global defaults
                SELECT value INTO temp_result
                FROM system_defaults
                WHERE category = p_entity_type AND key = p_setting_key;
                
            WHEN 'system_fallback' THEN
                -- System-wide fallback
                temp_result := 'default_value';
        END CASE;
        
        IF temp_result IS NOT NULL AND temp_result != '' THEN
            RETURN temp_result;
        END IF;
    END LOOP;
    
    RETURN NULL; -- No default found
END;
$$ LANGUAGE plpgsql;
```

## 📊 Monitoring & Management

### 1. Default Usage Analytics

```sql
-- Track which defaults are being used
CREATE TABLE default_usage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category TEXT NOT NULL,
    setting_key TEXT NOT NULL,
    default_source TEXT NOT NULL, -- 'user', 'template', 'system', etc.
    usage_count BIGINT DEFAULT 1,
    last_used TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (category, setting_key, default_source)
);

-- Function to record default usage
CREATE OR REPLACE FUNCTION record_default_usage(
    p_category TEXT,
    p_setting_key TEXT,
    p_source TEXT
) RETURNS VOID AS $$
BEGIN
    INSERT INTO default_usage_stats (category, setting_key, default_source)
    VALUES (p_category, p_setting_key, p_source)
    ON CONFLICT (category, setting_key, default_source)
    DO UPDATE SET
        usage_count = default_usage_stats.usage_count + 1,
        last_used = NOW();
END;
$$ LANGUAGE plpgsql;

-- View for default usage analysis
CREATE VIEW default_usage_summary AS
SELECT 
    category,
    setting_key,
    default_source,
    usage_count,
    last_used,
    RANK() OVER (PARTITION BY category, setting_key ORDER BY usage_count DESC) as usage_rank
FROM default_usage_stats
ORDER BY category, setting_key, usage_count DESC;
```

## ⚠️ Best Practices

1. **Document Default Logic** - Clearly explain default resolution order
2. **Avoid Deep Inheritance** - Keep template/inheritance chains shallow
3. **Use Constraints** - Ensure only one default per group when needed
4. **Test Fallback Paths** - Verify all default scenarios work correctly
5. **Monitor Usage** - Track which defaults are actually used
6. **Plan for Changes** - Consider impact when changing defaults
7. **Validate Defaults** - Ensure default values meet business rules
8. **Cache Frequently Used** - Cache resolved defaults for performance
9. **Version Defaults** - Track changes to default configurations
10. **Handle NULLs Properly** - Distinguish between NULL and empty defaults

## 🔗 References

- [PostgreSQL Default Values](https://www.postgresql.org/docs/current/ddl-default.html)
- [JSON/JSONB in PostgreSQL](https://www.postgresql.org/docs/current/datatype-json.html)
- [Configuration Management Patterns](https://martinfowler.com/articles/feature-toggles.html)
