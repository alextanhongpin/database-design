# Database Session Management Patterns

Session management is critical for maintaining user context, implementing security policies, and tracking application state across database connections. This guide covers various approaches to session handling in PostgreSQL.

## Table of Contents
- [Session Context Management](#session-context-management)
- [User Context Passing](#user-context-passing)
- [Row-Level Security Integration](#row-level-security-integration)
- [Session Storage Patterns](#session-storage-patterns)
- [Connection Pooling Considerations](#connection-pooling-considerations)
- [Performance Optimization](#performance-optimization)
- [Security Best Practices](#security-best-practices)
- [Real-World Examples](#real-world-examples)

## Session Context Management

### Local Session Variables

PostgreSQL supports session-local variables that persist within a transaction:

```sql
-- Basic session variable usage
BEGIN;
    -- Set variables local to this transaction
    SET LOCAL my.user_id = '12345';
    SET LOCAL my.organization_id = '67890';
    SET LOCAL my.role = 'admin';
    
    -- Access variables within the same transaction
    SELECT 
        current_setting('my.user_id') AS user_id,
        current_setting('my.organization_id') AS org_id,
        current_setting('my.role') AS user_role;
        
    -- Use in queries with RLS
    SELECT * FROM sensitive_data 
    WHERE organization_id = current_setting('my.organization_id')::INTEGER;
COMMIT;

-- Variables are automatically cleared after transaction
SELECT current_setting('my.user_id', true); -- Returns NULL
```

### Session Configuration Functions

```sql
-- Create helper functions for session management
CREATE OR REPLACE FUNCTION set_current_user_context(
    p_user_id INTEGER,
    p_organization_id INTEGER DEFAULT NULL,
    p_role TEXT DEFAULT 'user',
    p_permissions TEXT[] DEFAULT '{}'
) RETURNS VOID AS $$
BEGIN
    -- Set user context variables
    PERFORM set_config('app.user_id', p_user_id::TEXT, true);
    PERFORM set_config('app.organization_id', COALESCE(p_organization_id::TEXT, ''), true);
    PERFORM set_config('app.role', p_role, true);
    PERFORM set_config('app.permissions', array_to_string(p_permissions, ','), true);
    
    -- Set timestamp for session tracking
    PERFORM set_config('app.session_start', NOW()::TEXT, true);
END;
$$ LANGUAGE plpgsql;

-- Get current user context
CREATE OR REPLACE FUNCTION get_current_user_context()
RETURNS TABLE(
    user_id INTEGER,
    organization_id INTEGER,
    role TEXT,
    permissions TEXT[],
    session_start TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY SELECT 
        NULLIF(current_setting('app.user_id', true), '')::INTEGER,
        NULLIF(current_setting('app.organization_id', true), '')::INTEGER,
        current_setting('app.role', true),
        string_to_array(current_setting('app.permissions', true), ','),
        current_setting('app.session_start', true)::TIMESTAMPTZ;
END;
$$ LANGUAGE plpgsql;

-- Usage example
SELECT set_current_user_context(123, 456, 'admin', ARRAY['read', 'write', 'delete']);
SELECT * FROM get_current_user_context();
```

### Advanced Context Management

```sql
-- Comprehensive session context system
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL,
    organization_id INTEGER,
    session_token TEXT UNIQUE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'
);

-- Function to initialize session context from token
CREATE OR REPLACE FUNCTION initialize_session_context(p_session_token TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    session_info RECORD;
BEGIN
    -- Validate and get session information
    SELECT 
        us.user_id,
        us.organization_id,
        u.email,
        u.role,
        us.metadata
    INTO session_info
    FROM user_sessions us
    JOIN users u ON us.user_id = u.id
    WHERE us.session_token = p_session_token
      AND us.is_active = TRUE
      AND us.expires_at > NOW();
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Set session context
    PERFORM set_config('app.user_id', session_info.user_id::TEXT, true);
    PERFORM set_config('app.organization_id', COALESCE(session_info.organization_id::TEXT, ''), true);
    PERFORM set_config('app.user_email', session_info.email, true);
    PERFORM set_config('app.user_role', session_info.role, true);
    PERFORM set_config('app.session_token', p_session_token, true);
    
    -- Update last activity
    UPDATE user_sessions 
    SET last_activity = NOW() 
    WHERE session_token = p_session_token;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## User Context Passing

### Application-Level Context

```sql
-- Create a context management system for web applications
CREATE OR REPLACE FUNCTION set_request_context(
    p_user_id INTEGER,
    p_request_id TEXT DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_feature_flags JSONB DEFAULT '{}'
) RETURNS VOID AS $$
BEGIN
    -- Core user context
    PERFORM set_config('ctx.user_id', p_user_id::TEXT, true);
    PERFORM set_config('ctx.request_id', COALESCE(p_request_id, gen_random_uuid()::TEXT), true);
    
    -- Request metadata
    PERFORM set_config('ctx.ip_address', COALESCE(p_ip_address::TEXT, ''), true);
    PERFORM set_config('ctx.user_agent', COALESCE(p_user_agent, ''), true);
    PERFORM set_config('ctx.request_time', NOW()::TEXT, true);
    
    -- Feature flags for this user/request
    PERFORM set_config('ctx.feature_flags', COALESCE(p_feature_flags::TEXT, '{}'), true);
END;
$$ LANGUAGE plpgsql;

-- Function to get current request context
CREATE OR REPLACE FUNCTION get_request_context()
RETURNS JSONB AS $$
BEGIN
    RETURN jsonb_build_object(
        'user_id', NULLIF(current_setting('ctx.user_id', true), '')::INTEGER,
        'request_id', current_setting('ctx.request_id', true),
        'ip_address', NULLIF(current_setting('ctx.ip_address', true), '')::INET,
        'user_agent', current_setting('ctx.user_agent', true),
        'request_time', current_setting('ctx.request_time', true)::TIMESTAMPTZ,
        'feature_flags', current_setting('ctx.feature_flags', true)::JSONB
    );
END;
$$ LANGUAGE plpgsql;
```

### Multi-Tenant Context

```sql
-- Multi-tenant context management
CREATE OR REPLACE FUNCTION set_tenant_context(
    p_tenant_id INTEGER,
    p_user_id INTEGER,
    p_permissions JSONB DEFAULT '{}'
) RETURNS VOID AS $$
DECLARE
    tenant_info RECORD;
BEGIN
    -- Validate tenant and user relationship
    SELECT t.id, t.name, t.settings, tu.role
    INTO tenant_info
    FROM tenants t
    JOIN tenant_users tu ON t.id = tu.tenant_id
    WHERE t.id = p_tenant_id 
      AND tu.user_id = p_user_id
      AND t.is_active = TRUE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'User % does not have access to tenant %', p_user_id, p_tenant_id;
    END IF;
    
    -- Set tenant context
    PERFORM set_config('tenant.id', tenant_info.id::TEXT, true);
    PERFORM set_config('tenant.name', tenant_info.name, true);
    PERFORM set_config('tenant.user_id', p_user_id::TEXT, true);
    PERFORM set_config('tenant.user_role', tenant_info.role, true);
    PERFORM set_config('tenant.settings', tenant_info.settings::TEXT, true);
    PERFORM set_config('tenant.permissions', p_permissions::TEXT, true);
END;
$$ LANGUAGE plpgsql;

-- Helper function to get current tenant ID
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS INTEGER AS $$
BEGIN
    RETURN NULLIF(current_setting('tenant.id', true), '')::INTEGER;
END;
$$ LANGUAGE plpgsql;

-- Helper function to get current user ID
CREATE OR REPLACE FUNCTION current_user_id()
RETURNS INTEGER AS $$
BEGIN
    RETURN NULLIF(current_setting('tenant.user_id', true), '')::INTEGER;
END;
$$ LANGUAGE plpgsql;
```

## Row-Level Security Integration

### RLS with Session Context

```sql
-- Enable RLS on sensitive tables
CREATE TABLE customer_data (
    id SERIAL PRIMARY KEY,
    customer_name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    organization_id INTEGER NOT NULL,
    sensitive_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE customer_data ENABLE ROW LEVEL SECURITY;

-- Create RLS policies using session context
CREATE POLICY customer_data_org_isolation ON customer_data
    FOR ALL TO application_role
    USING (organization_id = current_setting('app.organization_id')::INTEGER);

-- Create policy for admin access
CREATE POLICY customer_data_admin_access ON customer_data
    FOR ALL TO application_role
    USING (
        current_setting('app.role', true) = 'super_admin' OR
        (current_setting('app.role', true) = 'admin' AND 
         organization_id = current_setting('app.organization_id')::INTEGER)
    );

-- Example usage with context
BEGIN;
    -- Set user context
    SELECT set_current_user_context(123, 456, 'admin');
    
    -- This query will only return data for organization 456
    SELECT * FROM customer_data;
COMMIT;
```

### Dynamic RLS Policies

```sql
-- Table with flexible access control
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    owner_id INTEGER NOT NULL,
    organization_id INTEGER NOT NULL,
    access_level TEXT DEFAULT 'private' CHECK (access_level IN ('private', 'team', 'organization', 'public')),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

-- Complex RLS policy using session context
CREATE POLICY document_access_policy ON documents
    FOR SELECT TO application_role
    USING (
        -- Owner can always access
        owner_id = current_setting('app.user_id')::INTEGER
        OR
        -- Organization members can access organization-level docs
        (access_level = 'organization' AND 
         organization_id = current_setting('app.organization_id')::INTEGER)
        OR
        -- Team members can access team-level docs (requires team context)
        (access_level = 'team' AND
         organization_id = current_setting('app.organization_id')::INTEGER AND
         EXISTS (SELECT 1 FROM team_members tm 
                WHERE tm.user_id = current_setting('app.user_id')::INTEGER
                  AND tm.team_id = ANY(string_to_array(current_setting('app.team_ids', true), ',')::INTEGER[])))
        OR
        -- Public documents
        access_level = 'public'
        OR
        -- Super admin access
        current_setting('app.role', true) = 'super_admin'
    );
```

## Session Storage Patterns

### Database Session Store

```sql
-- Comprehensive session storage system
CREATE TABLE application_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_key TEXT UNIQUE NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id),
    organization_id INTEGER REFERENCES organizations(id),
    
    -- Session metadata
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_accessed TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- Session data
    session_data JSONB DEFAULT '{}',
    
    -- Status tracking
    is_active BOOLEAN DEFAULT TRUE,
    terminated_at TIMESTAMPTZ,
    termination_reason TEXT
);

-- Indexes for performance
CREATE INDEX idx_sessions_key ON application_sessions (session_key) WHERE is_active = TRUE;
CREATE INDEX idx_sessions_user ON application_sessions (user_id, is_active);
CREATE INDEX idx_sessions_expiry ON application_sessions (expires_at) WHERE is_active = TRUE;

-- Session management functions
CREATE OR REPLACE FUNCTION create_session(
    p_user_id INTEGER,
    p_organization_id INTEGER DEFAULT NULL,
    p_session_duration INTERVAL DEFAULT '24 hours',
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_session_data JSONB DEFAULT '{}'
) RETURNS TEXT AS $$
DECLARE
    session_key TEXT;
    session_id UUID;
BEGIN
    -- Generate secure session key
    session_key := encode(gen_random_bytes(32), 'base64');
    
    -- Create session record
    INSERT INTO application_sessions (
        session_key, user_id, organization_id, ip_address, 
        user_agent, expires_at, session_data
    ) VALUES (
        session_key, p_user_id, p_organization_id, p_ip_address,
        p_user_agent, NOW() + p_session_duration, p_session_data
    ) RETURNING id INTO session_id;
    
    -- Log session creation
    INSERT INTO session_events (session_id, event_type, ip_address, created_at)
    VALUES (session_id, 'created', p_ip_address, NOW());
    
    RETURN session_key;
END;
$$ LANGUAGE plpgsql;

-- Validate and refresh session
CREATE OR REPLACE FUNCTION validate_session(p_session_key TEXT)
RETURNS TABLE(
    session_id UUID,
    user_id INTEGER,
    organization_id INTEGER,
    session_data JSONB,
    is_valid BOOLEAN
) AS $$
DECLARE
    session_record RECORD;
BEGIN
    -- Get session info
    SELECT s.id, s.user_id, s.organization_id, s.session_data, s.expires_at > NOW() AS valid
    INTO session_record
    FROM application_sessions s
    WHERE s.session_key = p_session_key AND s.is_active = TRUE;
    
    IF FOUND AND session_record.valid THEN
        -- Update last accessed
        UPDATE application_sessions 
        SET last_accessed = NOW() 
        WHERE session_key = p_session_key;
        
        -- Return session info
        RETURN QUERY SELECT 
            session_record.id,
            session_record.user_id,
            session_record.organization_id,
            session_record.session_data,
            TRUE;
    ELSE
        -- Invalid or expired session
        RETURN QUERY SELECT NULL::UUID, NULL::INTEGER, NULL::INTEGER, NULL::JSONB, FALSE;
    END IF;
END;
$$ LANGUAGE plpgsql;
```

### Session Cleanup and Maintenance

```sql
-- Cleanup expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    cleaned_count INTEGER;
BEGIN
    -- Mark expired sessions as inactive
    UPDATE application_sessions 
    SET 
        is_active = FALSE,
        terminated_at = NOW(),
        termination_reason = 'expired'
    WHERE is_active = TRUE 
      AND expires_at < NOW();
    
    GET DIAGNOSTICS cleaned_count = ROW_COUNT;
    
    -- Log cleanup
    INSERT INTO maintenance_log (operation, affected_rows, executed_at)
    VALUES ('session_cleanup', cleaned_count, NOW());
    
    RETURN cleaned_count;
END;
$$ LANGUAGE plpgsql;

-- Revoke user sessions
CREATE OR REPLACE FUNCTION revoke_user_sessions(
    p_user_id INTEGER,
    p_reason TEXT DEFAULT 'user_logout'
) RETURNS INTEGER AS $$
DECLARE
    revoked_count INTEGER;
BEGIN
    UPDATE application_sessions 
    SET 
        is_active = FALSE,
        terminated_at = NOW(),
        termination_reason = p_reason
    WHERE user_id = p_user_id 
      AND is_active = TRUE;
    
    GET DIAGNOSTICS revoked_count = ROW_COUNT;
    
    -- Log session revocation
    INSERT INTO session_events (user_id, event_type, metadata, created_at)
    VALUES (p_user_id, 'sessions_revoked', 
            jsonb_build_object('count', revoked_count, 'reason', p_reason), NOW());
    
    RETURN revoked_count;
END;
$$ LANGUAGE plpgsql;
```

## Connection Pooling Considerations

### Connection Pool Session Management

```sql
-- Handle session context with connection pooling
CREATE OR REPLACE FUNCTION reset_session_context()
RETURNS VOID AS $$
BEGIN
    -- Clear all custom settings
    PERFORM set_config(name, NULL, false)
    FROM pg_settings 
    WHERE name LIKE 'app.%' OR name LIKE 'ctx.%' OR name LIKE 'tenant.%';
    
    -- Reset to safe defaults
    RESET ROLE;
    SET session_authorization DEFAULT;
END;
$$ LANGUAGE plpgsql;

-- Initialize connection for new request
CREATE OR REPLACE FUNCTION initialize_connection_context(
    p_session_token TEXT,
    p_request_id TEXT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    session_valid BOOLEAN;
    session_info RECORD;
BEGIN
    -- First, reset any existing context
    PERFORM reset_session_context();
    
    -- Validate session and get context
    SELECT * INTO session_info 
    FROM validate_session(p_session_token);
    
    IF NOT session_info.is_valid THEN
        RETURN FALSE;
    END IF;
    
    -- Set session context
    PERFORM set_current_user_context(
        session_info.user_id,
        session_info.organization_id
    );
    
    -- Set request context
    PERFORM set_config('ctx.request_id', 
                      COALESCE(p_request_id, gen_random_uuid()::TEXT), true);
    PERFORM set_config('ctx.session_id', session_info.session_id::TEXT, true);
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### Transaction-Level Context

```sql
-- Wrapper for transactional operations with context
CREATE OR REPLACE FUNCTION execute_with_context(
    p_session_token TEXT,
    p_operation TEXT,
    p_parameters JSONB DEFAULT '{}'
) RETURNS JSONB AS $$
DECLARE
    result JSONB;
    context_valid BOOLEAN;
BEGIN
    -- Start transaction and set context
    context_valid := initialize_connection_context(p_session_token);
    
    IF NOT context_valid THEN
        RAISE EXCEPTION 'Invalid session token';
    END IF;
    
    -- Execute the operation based on type
    CASE p_operation
        WHEN 'get_user_data' THEN
            SELECT jsonb_agg(row_to_json(u)) INTO result
            FROM users u WHERE id = current_user_id();
            
        WHEN 'get_organization_stats' THEN
            SELECT jsonb_build_object(
                'total_users', COUNT(*),
                'active_users', COUNT(*) FILTER (WHERE last_login > NOW() - INTERVAL '30 days')
            ) INTO result
            FROM users WHERE organization_id = current_tenant_id();
            
        ELSE
            RAISE EXCEPTION 'Unknown operation: %', p_operation;
    END CASE;
    
    RETURN COALESCE(result, '{}'::jsonb);
    
EXCEPTION WHEN OTHERS THEN
    -- Ensure context is cleaned up on error
    PERFORM reset_session_context();
    RAISE;
END;
$$ LANGUAGE plpgsql;
```

## Performance Optimization

### Session Context Caching

```sql
-- Cache frequently accessed session context
CREATE TABLE session_context_cache (
    session_key TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    organization_id INTEGER,
    permissions JSONB,
    cached_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Function to get cached context
CREATE OR REPLACE FUNCTION get_cached_context(p_session_key TEXT)
RETURNS TABLE(
    user_id INTEGER,
    organization_id INTEGER,
    permissions JSONB,
    cache_hit BOOLEAN
) AS $$
DECLARE
    cached_data RECORD;
BEGIN
    -- Try to get from cache first
    SELECT c.user_id, c.organization_id, c.permissions
    INTO cached_data
    FROM session_context_cache c
    WHERE c.session_key = p_session_key
      AND c.expires_at > NOW();
    
    IF FOUND THEN
        -- Cache hit
        RETURN QUERY SELECT cached_data.user_id, cached_data.organization_id, 
                           cached_data.permissions, TRUE;
    ELSE
        -- Cache miss - get from main session table
        SELECT s.user_id, s.organization_id, s.session_data->'permissions'
        INTO cached_data
        FROM application_sessions s
        WHERE s.session_key = p_session_key
          AND s.is_active = TRUE
          AND s.expires_at > NOW();
        
        IF FOUND THEN
            -- Update cache
            INSERT INTO session_context_cache (session_key, user_id, organization_id, permissions, expires_at)
            VALUES (p_session_key, cached_data.user_id, cached_data.organization_id, 
                   cached_data.permissions, NOW() + INTERVAL '5 minutes')
            ON CONFLICT (session_key) 
            DO UPDATE SET 
                user_id = EXCLUDED.user_id,
                organization_id = EXCLUDED.organization_id,
                permissions = EXCLUDED.permissions,
                cached_at = NOW(),
                expires_at = EXCLUDED.expires_at;
            
            RETURN QUERY SELECT cached_data.user_id, cached_data.organization_id, 
                               cached_data.permissions, FALSE;
        ELSE
            -- Session not found
            RETURN QUERY SELECT NULL::INTEGER, NULL::INTEGER, NULL::JSONB, FALSE;
        END IF;
    END IF;
END;
$$ LANGUAGE plpgsql;
```

## Security Best Practices

### Secure Session Handling

```sql
-- Secure session management with audit trail
CREATE TABLE session_security_events (
    id SERIAL PRIMARY KEY,
    session_id UUID REFERENCES application_sessions(id),
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to detect suspicious session activity
CREATE OR REPLACE FUNCTION check_session_security(
    p_session_key TEXT,
    p_current_ip INET,
    p_current_user_agent TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    session_info RECORD;
    ip_changed BOOLEAN := FALSE;
    agent_changed BOOLEAN := FALSE;
    suspicious BOOLEAN := FALSE;
BEGIN
    -- Get current session info
    SELECT s.id, s.ip_address, s.user_agent, s.user_id
    INTO session_info
    FROM application_sessions s
    WHERE s.session_key = p_session_key AND s.is_active = TRUE;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Check for IP address changes
    IF session_info.ip_address IS NOT NULL AND session_info.ip_address != p_current_ip THEN
        ip_changed := TRUE;
        
        INSERT INTO session_security_events (session_id, event_type, severity, ip_address, details)
        VALUES (session_info.id, 'ip_change', 'medium', p_current_ip,
                jsonb_build_object('old_ip', session_info.ip_address, 'new_ip', p_current_ip));
    END IF;
    
    -- Check for user agent changes (less critical)
    IF session_info.user_agent IS NOT NULL AND session_info.user_agent != p_current_user_agent THEN
        agent_changed := TRUE;
        
        INSERT INTO session_security_events (session_id, event_type, severity, ip_address, details)
        VALUES (session_info.id, 'user_agent_change', 'low', p_current_ip,
                jsonb_build_object('old_agent', session_info.user_agent, 'new_agent', p_current_user_agent));
    END IF;
    
    -- Determine if session should be terminated
    suspicious := ip_changed; -- Could add more sophisticated logic
    
    IF suspicious THEN
        -- Terminate suspicious session
        UPDATE application_sessions 
        SET 
            is_active = FALSE,
            terminated_at = NOW(),
            termination_reason = 'security_violation'
        WHERE session_key = p_session_key;
        
        INSERT INTO session_security_events (session_id, event_type, severity, ip_address, details)
        VALUES (session_info.id, 'session_terminated', 'high', p_current_ip,
                jsonb_build_object('reason', 'suspicious_activity'));
        
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## Real-World Examples

### Web Application Session Pattern

```sql
-- Complete web application session system
CREATE OR REPLACE FUNCTION web_app_authenticate(
    p_session_token TEXT,
    p_ip_address INET,
    p_user_agent TEXT
) RETURNS JSONB AS $$
DECLARE
    auth_result JSONB;
    session_data RECORD;
    security_check_passed BOOLEAN;
BEGIN
    -- Validate session security
    security_check_passed := check_session_security(p_session_token, p_ip_address, p_user_agent);
    
    IF NOT security_check_passed THEN
        RETURN jsonb_build_object(
            'authenticated', FALSE,
            'reason', 'security_violation'
        );
    END IF;
    
    -- Get session context with cache
    SELECT * INTO session_data 
    FROM get_cached_context(p_session_token);
    
    IF session_data.user_id IS NULL THEN
        RETURN jsonb_build_object(
            'authenticated', FALSE,
            'reason', 'invalid_session'
        );
    END IF;
    
    -- Set application context
    PERFORM set_current_user_context(
        session_data.user_id,
        session_data.organization_id,
        'user',
        ARRAY(SELECT jsonb_array_elements_text(session_data.permissions->'roles'))
    );
    
    -- Build response
    RETURN jsonb_build_object(
        'authenticated', TRUE,
        'user_id', session_data.user_id,
        'organization_id', session_data.organization_id,
        'permissions', session_data.permissions,
        'cache_hit', session_data.cache_hit
    );
END;
$$ LANGUAGE plpgsql;
```

### Microservices Session Context

```sql
-- Session context for microservices architecture
CREATE OR REPLACE FUNCTION microservice_context_from_jwt(
    p_jwt_payload JSONB
) RETURNS BOOLEAN AS $$
BEGIN
    -- Extract and validate JWT claims
    IF NOT (p_jwt_payload ? 'user_id' AND p_jwt_payload ? 'exp') THEN
        RETURN FALSE;
    END IF;
    
    -- Check expiration
    IF (p_jwt_payload->>'exp')::INTEGER < EXTRACT(EPOCH FROM NOW()) THEN
        RETURN FALSE;
    END IF;
    
    -- Set microservice context
    PERFORM set_config('jwt.user_id', p_jwt_payload->>'user_id', true);
    PERFORM set_config('jwt.organization_id', COALESCE(p_jwt_payload->>'org_id', ''), true);
    PERFORM set_config('jwt.role', COALESCE(p_jwt_payload->>'role', 'user'), true);
    PERFORM set_config('jwt.permissions', COALESCE(p_jwt_payload->'permissions', '[]')::TEXT, true);
    PERFORM set_config('jwt.service', COALESCE(p_jwt_payload->>'service', 'unknown'), true);
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Helper function for microservice queries
CREATE OR REPLACE FUNCTION current_jwt_user_id()
RETURNS INTEGER AS $$
BEGIN
    RETURN NULLIF(current_setting('jwt.user_id', true), '')::INTEGER;
END;
$$ LANGUAGE plpgsql STABLE;
```

This comprehensive session management guide provides robust patterns for handling user context, implementing security policies, and maintaining session state across different application architectures. The examples demonstrate both simple local variable usage and complex enterprise-grade session management systems.

Key takeaways:
- Use PostgreSQL's session-local variables for temporary context within transactions
- Implement proper session storage and validation for web applications
- Consider connection pooling implications when designing session context
- Always include security checks and audit trails for production systems
- Cache frequently accessed session data for performance optimization
