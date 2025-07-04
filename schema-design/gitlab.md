# GitLab Schema Design Patterns

Lessons learned from GitLab's PostgreSQL schema design for large-scale applications.

## 🎯 Overview

GitLab's database schema provides excellent examples of how to design scalable database systems for complex applications. As one of the largest Ruby on Rails applications, GitLab has evolved sophisticated patterns for handling:
- **Multi-tenancy** - Organizations, projects, and namespaces
- **Hierarchical Data** - Nested groups and projects
- **Complex Relationships** - Users, permissions, and project structures
- **Performance at Scale** - Millions of records and complex queries

## 🏗️ Core Architecture Patterns

### Namespace-Based Multi-Tenancy

```sql
-- GitLab's namespace pattern for hierarchical organization
CREATE TABLE namespaces (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Self-referencing for nested groups
    parent_id BIGINT REFERENCES namespaces(id),
    
    -- Namespace types: 'User', 'Group', 'Project'
    type VARCHAR(255) NOT NULL,
    
    -- Materialized path for efficient querying
    traversal_ids BIGINT[],
    
    -- Soft delete pattern
    deleted_at TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(parent_id, name),
    INDEX(parent_id),
    INDEX(type),
    INDEX(path),
    INDEX(traversal_ids) -- GIN index for array operations
);

-- Projects belong to namespaces
CREATE TABLE projects (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Belongs to a namespace (user or group)
    namespace_id BIGINT NOT NULL REFERENCES namespaces(id),
    
    -- Project visibility levels
    visibility_level INTEGER NOT NULL DEFAULT 0, -- 0=private, 10=internal, 20=public
    
    -- Feature flags
    issues_enabled BOOLEAN DEFAULT TRUE,
    merge_requests_enabled BOOLEAN DEFAULT TRUE,
    wiki_enabled BOOLEAN DEFAULT TRUE,
    
    -- Soft delete
    deleted_at TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(namespace_id, name),
    INDEX(namespace_id),
    INDEX(visibility_level),
    INDEX(created_at)
);
```

### Permission System

```sql
-- GitLab's flexible permission model
CREATE TABLE members (
    id BIGSERIAL PRIMARY KEY,
    
    -- Polymorphic source (Project or Namespace)
    source_type VARCHAR(255) NOT NULL,
    source_id BIGINT NOT NULL,
    
    -- User being granted access
    user_id BIGINT NOT NULL REFERENCES users(id),
    
    -- Access level (10=guest, 20=reporter, 30=developer, 40=maintainer, 50=owner)
    access_level INTEGER NOT NULL,
    
    -- Invitation system
    invite_email VARCHAR(255),
    invite_token VARCHAR(255),
    invite_accepted_at TIMESTAMP,
    
    -- Expiration for temporary access
    expires_at DATE,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(source_type, source_id, user_id),
    INDEX(source_type, source_id),
    INDEX(user_id),
    INDEX(invite_token),
    INDEX(expires_at)
);

-- Separate table for project-specific permissions
CREATE TABLE project_authorizations (
    user_id BIGINT NOT NULL REFERENCES users(id),
    project_id BIGINT NOT NULL REFERENCES projects(id),
    access_level INTEGER NOT NULL,
    
    PRIMARY KEY (user_id, project_id),
    INDEX(project_id, access_level)
);
```

### Activity Tracking

```sql
-- GitLab's event tracking system
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    
    -- Event details
    action VARCHAR(255) NOT NULL, -- 'created', 'updated', 'deleted', etc.
    target_type VARCHAR(255), -- 'Issue', 'MergeRequest', 'Project', etc.
    target_id BIGINT,
    
    -- Context
    project_id BIGINT REFERENCES projects(id),
    author_id BIGINT REFERENCES users(id),
    
    -- Event data
    fingerprint VARCHAR(255), -- For deduplication
    data TEXT, -- Serialized event data
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX(project_id, created_at),
    INDEX(author_id, created_at),
    INDEX(target_type, target_id),
    INDEX(action, created_at),
    INDEX(fingerprint)
);

-- Partitioning for performance
CREATE TABLE events_y2024m01 PARTITION OF events
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## 📊 Performance Patterns

### Efficient Counting

```sql
-- GitLab's counter cache pattern
CREATE TABLE project_statistics (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id),
    
    -- Cached counts
    commit_count BIGINT DEFAULT 0,
    storage_size BIGINT DEFAULT 0,
    repository_size BIGINT DEFAULT 0,
    wiki_size BIGINT DEFAULT 0,
    lfs_objects_size BIGINT DEFAULT 0,
    
    -- Issue and MR counts
    issues_count INTEGER DEFAULT 0,
    merge_requests_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(project_id)
);

-- Trigger to maintain counts
CREATE OR REPLACE FUNCTION update_project_statistics()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO project_statistics (project_id, issues_count)
        VALUES (NEW.project_id, 1)
        ON CONFLICT (project_id) DO UPDATE SET
            issues_count = project_statistics.issues_count + 1,
            updated_at = CURRENT_TIMESTAMP;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE project_statistics
        SET issues_count = issues_count - 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE project_id = OLD.project_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
```

### Batch Operations

```sql
-- GitLab's background job pattern
CREATE TABLE background_jobs (
    id BIGSERIAL PRIMARY KEY,
    
    -- Job identification
    job_class VARCHAR(255) NOT NULL,
    job_id VARCHAR(255) NOT NULL,
    
    -- Job data
    arguments JSONB NOT NULL,
    
    -- Scheduling
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    
    -- Status and error handling
    status VARCHAR(255) DEFAULT 'pending',
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    INDEX(job_class, status),
    INDEX(scheduled_at),
    INDEX(status, created_at)
);

-- Example: Bulk project updates
CREATE OR REPLACE FUNCTION schedule_bulk_project_update(
    project_ids BIGINT[],
    update_type VARCHAR(255),
    update_data JSONB
) RETURNS VOID AS $$
BEGIN
    INSERT INTO background_jobs (job_class, job_id, arguments, scheduled_at)
    SELECT 
        'BulkProjectUpdateJob',
        'bulk_update_' || gen_random_uuid(),
        jsonb_build_object(
            'project_ids', project_ids,
            'update_type', update_type,
            'update_data', update_data
        ),
        CURRENT_TIMESTAMP + INTERVAL '1 minute'
    FROM generate_series(1, array_length(project_ids, 1), 1000) as batch_start;
END;
$$ LANGUAGE plpgsql;
```

## 🔒 Security Patterns

### Row-Level Security

```sql
-- GitLab's visibility control
CREATE POLICY project_visibility_policy ON projects
    FOR ALL TO application_user
    USING (
        visibility_level = 20 -- Public
        OR 
        (visibility_level = 10 AND is_internal_user()) -- Internal
        OR
        (visibility_level = 0 AND can_access_project(id, current_user_id())) -- Private
    );

-- Helper function for project access
CREATE OR REPLACE FUNCTION can_access_project(
    project_id BIGINT,
    user_id BIGINT
) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM project_authorizations pa
        WHERE pa.project_id = $1 AND pa.user_id = $2
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Enable RLS
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
```

### Audit Logging

```sql
-- GitLab's audit event system
CREATE TABLE audit_events (
    id BIGSERIAL PRIMARY KEY,
    
    -- Event context
    author_id BIGINT REFERENCES users(id),
    entity_type VARCHAR(255) NOT NULL,
    entity_id BIGINT NOT NULL,
    
    -- Event details
    action VARCHAR(255) NOT NULL,
    details JSONB,
    
    -- IP and user agent for security
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX(author_id, created_at),
    INDEX(entity_type, entity_id),
    INDEX(action, created_at),
    INDEX(ip_address, created_at)
);
```

## 📈 Scalability Patterns

### Database Sharding Preparation

```sql
-- GitLab's namespace-based sharding preparation
CREATE TABLE shard_assignments (
    namespace_id BIGINT PRIMARY KEY REFERENCES namespaces(id),
    shard_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX(shard_name)
);

-- Function to determine shard
CREATE OR REPLACE FUNCTION get_shard_for_namespace(namespace_id BIGINT)
RETURNS VARCHAR(255) AS $$
DECLARE
    shard_name VARCHAR(255);
BEGIN
    SELECT sa.shard_name INTO shard_name
    FROM shard_assignments sa
    WHERE sa.namespace_id = $1;
    
    -- Default shard if not assigned
    RETURN COALESCE(shard_name, 'default');
END;
$$ LANGUAGE plpgsql;
```

### Read Replicas

```sql
-- GitLab's read-only query optimization
CREATE OR REPLACE FUNCTION route_to_replica()
RETURNS BOOLEAN AS $$
BEGIN
    -- Route read-only queries to replica
    RETURN current_setting('transaction_read_only')::boolean;
END;
$$ LANGUAGE plpgsql;

-- Views for replica routing
CREATE VIEW projects_read_only AS
SELECT * FROM projects
WHERE route_to_replica() OR NOT route_to_replica();
```

## 🛠️ Migration Patterns

### Zero-Downtime Migrations

```sql
-- GitLab's approach to large table migrations
-- Step 1: Add new column
ALTER TABLE projects ADD COLUMN new_feature_flag BOOLEAN;

-- Step 2: Populate in batches
CREATE OR REPLACE FUNCTION migrate_project_feature_flags()
RETURNS VOID AS $$
DECLARE
    batch_size INTEGER := 1000;
    last_id BIGINT := 0;
    current_batch INTEGER;
BEGIN
    LOOP
        UPDATE projects 
        SET new_feature_flag = (old_feature_data->>'enabled')::boolean
        WHERE id > last_id 
        AND new_feature_flag IS NULL
        AND id IN (
            SELECT id FROM projects 
            WHERE id > last_id 
            ORDER BY id 
            LIMIT batch_size
        );
        
        GET DIAGNOSTICS current_batch = ROW_COUNT;
        
        IF current_batch = 0 THEN
            EXIT;
        END IF;
        
        SELECT MAX(id) INTO last_id FROM projects WHERE new_feature_flag IS NOT NULL;
        
        -- Pause between batches
        PERFORM pg_sleep(0.1);
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Step 3: Add NOT NULL constraint after population
ALTER TABLE projects ALTER COLUMN new_feature_flag SET NOT NULL;

-- Step 4: Drop old column
ALTER TABLE projects DROP COLUMN old_feature_data;
```

## 🎯 Key Takeaways

### Design Principles

1. **Namespace Everything** - Use hierarchical namespaces for multi-tenancy
2. **Cache Strategically** - Counter caches for expensive aggregations
3. **Partition by Time** - Split large tables by date ranges
4. **Batch Operations** - Background jobs for bulk updates
5. **Plan for Growth** - Design for millions of records from day one

### Performance Lessons

1. **Index Strategy** - Composite indexes for common query patterns
2. **Avoid N+1 Queries** - Eager loading and batch operations
3. **Materialized Views** - For complex aggregations
4. **Connection Pooling** - Manage database connections efficiently
5. **Query Optimization** - Regular analysis of slow queries

### Security Practices

1. **Row-Level Security** - Database-level access control
2. **Audit Everything** - Comprehensive logging of sensitive operations
3. **Principle of Least Privilege** - Minimal necessary permissions
4. **IP Tracking** - Log IP addresses for security events
5. **Token-Based Auth** - Secure API access patterns

## 🔗 Related Resources

- **[GitLab Database Guide](https://docs.gitlab.com/ee/development/database/)** - Official GitLab database documentation
- **[PostgreSQL at Scale](../performance/postgresql-optimization.md)** - Performance optimization techniques
- **[Multi-Tenancy Patterns](../security/multi-tenancy.md)** - Implementing multi-tenant architectures
- **[Background Jobs](../application/background-jobs.md)** - Asynchronous processing patterns

GitLab's schema design demonstrates how to build scalable, secure, and maintainable database systems for complex applications. These patterns are battle-tested at scale and provide excellent examples for similar applications.
