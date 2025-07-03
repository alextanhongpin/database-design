# Database Design Patterns

A comprehensive collection of battle-tested database design patterns for building robust, scalable applications. Each pattern includes real-world examples, implementation details, and trade-off analysis.

## 🎯 Pattern Categories

### 🏗️ Foundational Patterns
Essential patterns that form the building blocks of robust database design.

| Pattern | Use Case | Complexity | Examples |
|---------|----------|------------|----------|
| **[Primary Keys & Identity](../data-types/id.md)** | Unique record identification | ⭐ | UUID vs Auto-increment, Composite keys |
| **[Foreign Key Strategies](foreign-key.md)** | Relationship integrity | ⭐ | When to use, when to avoid, soft references |
| **[Soft Delete](soft-delete-schema.md)** | Data retention without removal | ⭐⭐ | User accounts, orders, audit trails |
| **[Timestamps & Audit](../data-types/date.md)** | Change tracking | ⭐ | created_at, updated_at, deleted_at |

### 🔄 State Management Patterns
Patterns for modeling complex business processes and state transitions.

| Pattern | Use Case | Complexity | Examples |
|---------|----------|------------|----------|
| **[State Machines](state-machine.md)** | Complex workflows | ⭐⭐⭐ | Order processing, content approval |
| **[Status Patterns](status.md)** | Simple state tracking | ⭐⭐ | Published/draft, active/inactive |
| **[Workflow Systems](workflow.md)** | Multi-step processes | ⭐⭐⭐⭐ | Job applications, loan approvals |
| **[Event Sourcing](event-sourcing.md)** | Complete audit trail | ⭐⭐⭐⭐ | Financial transactions, system logs |

### 🔗 Relationship Patterns
Patterns for modeling complex relationships between entities.

| Pattern | Use Case | Complexity | Examples |
|---------|----------|------------|----------|
| **[Polymorphic Associations](polymorphic.md)** | Flexible relationships | ⭐⭐⭐ | Comments on multiple entities, tags |
| **[Inheritance](inheritance.md)** | Type hierarchies | ⭐⭐⭐ | User types, product categories |
| **[Many-to-Many](many-to-many.md)** | Complex associations | ⭐⭐ | User roles, product categories |
| **[Self-Referencing](friendship.md)** | Hierarchical data | ⭐⭐⭐ | Organization charts, social networks |

### 📊 Data Integrity Patterns
Patterns for ensuring data quality and business rule enforcement.

| Pattern | Use Case | Complexity | Examples |
|---------|----------|------------|----------|
| **[Constraints](constraint.md)** | Business rule enforcement | ⭐⭐ | Price validation, date ranges |
| **[Custom Domains](error-handling.md)** | Type safety | ⭐⭐ | Email validation, currency codes |
| **[Reference Tables](reference.md)** | Controlled vocabularies | ⭐ | Countries, status types, categories |
| **[Business Rules](business-rule.md)** | Complex validations | ⭐⭐⭐ | Multi-column constraints, conditional logic |

### 🚀 Performance Patterns
Patterns for optimizing query performance and handling scale.

| Pattern | Use Case | Complexity | Examples |
|---------|----------|------------|----------|
| **[Pagination](keyset-pagination.md)** | Large result sets | ⭐⭐ | Cursor-based, offset-based pagination |
| **[Counter Caches](counter_cache.md)** | Denormalization | ⭐⭐ | Post counts, follower counts |
| **[Partitioning](partition.md)** | Large table scaling | ⭐⭐⭐⭐ | Time-based, hash-based partitioning |
| **[Indexing Strategies](../performance/indexing.md)** | Query optimization | ⭐⭐⭐ | Composite indexes, partial indexes |

### 🏢 Enterprise Patterns
Advanced patterns for complex business scenarios.

| Pattern | Use Case | Complexity | Examples |
|---------|----------|------------|----------|
| **[Multi-Tenancy](../security/row-level-security.md)** | Isolated data access | ⭐⭐⭐⭐ | SaaS platforms, enterprise apps |
| **[Temporal Data](../specialized/README.md)** | Time-based modeling | ⭐⭐⭐⭐ | Price histories, employee records |
| **[Data Versioning](revision.md)** | Change tracking | ⭐⭐⭐ | Document versions, configuration changes |
| **[Approval Workflows](approval.md)** | Multi-stage processes | ⭐⭐⭐ | Content moderation, financial approvals |

## 🌍 Real-World Pattern Applications

### E-Commerce Platform
Combining multiple patterns for a complete solution:

```sql
-- Product catalog with inheritance
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Identity pattern
    status TEXT NOT NULL DEFAULT 'draft',          -- Status pattern
    created_at TIMESTAMP DEFAULT NOW(),            -- Audit pattern
    deleted_at TIMESTAMP                           -- Soft delete pattern
);

-- Order processing with state machine
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled')
    ),
    -- State machine transition tracking
    status_changed_at TIMESTAMP DEFAULT NOW(),
    status_changed_by UUID
);

-- Reviews with polymorphic associations
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reviewable_type TEXT NOT NULL,  -- 'product' or 'service'
    reviewable_id UUID NOT NULL,
    rating INTEGER CHECK (rating BETWEEN 1 AND 5),
    -- Polymorphic constraint
    CHECK ((reviewable_type, reviewable_id) IN (
        SELECT 'product', id FROM products UNION ALL
        SELECT 'service', id FROM services
    ))
);
```

### Social Media Application
```sql
-- User relationships with self-referencing
CREATE TABLE friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id UUID NOT NULL REFERENCES users(id),
    addressee_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'accepted', 'blocked')
    ),
    -- Prevent self-friendship and duplicates
    CHECK (requester_id != addressee_id),
    UNIQUE (requester_id, addressee_id)
);

-- Activity feed with event sourcing
CREATE TABLE activity_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_type TEXT NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### SaaS Multi-Tenant Platform
```sql
-- Tenant isolation with row-level security
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active'
);

-- All tenant data inherits tenant_id
CREATE TABLE tenant_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email TEXT NOT NULL,
    -- Unique within tenant
    UNIQUE (tenant_id, email)
);

-- Row-level security policy
ALTER TABLE tenant_users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_users
FOR ALL TO application_role
USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

## 📚 Pattern Selection Guide

### By Use Case

#### Content Management
- **Core**: Soft Delete, Status Patterns, Timestamps
- **Advanced**: Approval Workflows, Data Versioning, Polymorphic Associations (comments)

#### E-Commerce
- **Core**: State Machines (orders), Reference Tables (categories), Constraints (pricing)
- **Advanced**: Temporal Data (price history), Counter Caches (inventory), Partitioning (analytics)

#### Social Networks  
- **Core**: Self-Referencing (friendships), Many-to-Many (groups), Soft Delete
- **Advanced**: Event Sourcing (activity feeds), Polymorphic Associations (notifications)

#### SaaS Platforms
- **Core**: Multi-Tenancy, Reference Tables (plans), Constraints (limits)
- **Advanced**: Temporal Data (usage tracking), Approval Workflows (upgrades)

### By Scale

#### Small Scale (<10K records)
Focus on simplicity and correctness:
- ✅ Basic constraints and foreign keys
- ✅ Simple status columns
- ✅ Standard indexing
- ❌ Complex partitioning
- ❌ Heavy denormalization

#### Medium Scale (10K-1M records)
Balance performance and maintainability:
- ✅ Strategic denormalization (counter caches)
- ✅ Proper indexing strategies
- ✅ Query optimization
- ⚠️ Consider partitioning for time-series data
- ❌ Premature micro-optimizations

#### Large Scale (1M+ records)
Performance becomes critical:
- ✅ Partitioning strategies
- ✅ Read replicas and caching
- ✅ Materialized views
- ✅ Event sourcing for audit trails
- ⚠️ Consider CQRS patterns

## 🔧 Implementation Guidelines

### 1. Start Simple
```sql
-- ❌ Don't start with complex patterns
CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    status user_status_type NOT NULL,
    profile_data JSONB,
    audit_log audit_entry[],
    -- Too complex for initial implementation
);

-- ✅ Start with basics, evolve
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 2. Add Constraints Gradually
```sql
-- Phase 1: Basic structure
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    total_cents INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);

-- Phase 2: Add business rules
ALTER TABLE orders ADD CONSTRAINT positive_total 
CHECK (total_cents > 0);

ALTER TABLE orders ADD CONSTRAINT valid_status
CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered'));

-- Phase 3: Add complex constraints
ALTER TABLE orders ADD CONSTRAINT status_progression
CHECK (
    (status = 'pending') OR
    (status = 'confirmed' AND confirmed_at IS NOT NULL) OR
    (status = 'shipped' AND shipped_at IS NOT NULL)
);
```

### 3. Monitor and Optimize
```sql
-- Add monitoring for pattern effectiveness
CREATE VIEW pattern_health AS
SELECT 
    'soft_delete' as pattern,
    COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_records,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) as deleted_records,
    ROUND(
        COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) * 100.0 / COUNT(*), 
        2
    ) as deletion_rate
FROM users;
```

## 🚨 Anti-Patterns to Avoid

### 1. The God Table
```sql
-- ❌ Don't put everything in one table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    -- User data
    email TEXT, first_name TEXT, last_name TEXT,
    -- Address data  
    street TEXT, city TEXT, state TEXT, zip TEXT,
    -- Preferences
    theme TEXT, language TEXT, timezone TEXT,
    -- Billing
    card_number TEXT, expiry TEXT, cvv TEXT,
    -- And 50 more columns...
);

-- ✅ Separate concerns
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    first_name TEXT, last_name TEXT
);

CREATE TABLE user_addresses (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    type TEXT, street TEXT, city TEXT
);
```

### 2. EAV (Entity-Attribute-Value)
```sql
-- ❌ Avoid EAV when possible
CREATE TABLE entity_attributes (
    entity_id UUID,
    attribute_name TEXT,
    attribute_value TEXT
); -- Nightmare to query and maintain

-- ✅ Use JSONB for flexible attributes
CREATE TABLE products (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    category_id UUID NOT NULL,
    attributes JSONB, -- Structured but flexible
    
    -- Index for common queries
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_product_attributes ON products USING gin(attributes);
```

### 3. Boolean Trap
```sql
-- ❌ Multiple boolean flags create confusion
CREATE TABLE posts (
    id UUID PRIMARY KEY,
    is_published BOOLEAN DEFAULT FALSE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE
    -- What if is_published=true AND is_deleted=true?
);

-- ✅ Use explicit status
CREATE TABLE posts (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (
        status IN ('draft', 'published', 'featured', 'archived', 'deleted')
    ),
    deleted_at TIMESTAMP -- Soft delete if needed
);
```

## 📖 Further Reading

### Books
- **"The Data Model Resource Book"** by Len Silverston - Industry-standard patterns
- **"Database Design for Mere Mortals"** by Michael Hernandez - Fundamental principles
- **"SQL Antipatterns"** by Bill Karwin - What not to do

### Articles & References
- [PostgreSQL Documentation](https://www.postgresql.org/docs/) - Official patterns and best practices
- [Use The Index, Luke](https://use-the-index-luke.com/) - Indexing strategies
- [Database Design Patterns](https://databasepatterns.org/) - Community patterns

### Tools
- **[pgTAP](https://pgtap.org/)** - Unit testing for database patterns
- **[Migra](https://pypi.org/project/migra/)** - Schema diffing and migration
- **[pgMustard](https://www.pgmustard.com/)** - Query performance analysis

## 🤝 Contributing

### Pattern Template
When adding new patterns, use this structure:

```markdown
# Pattern Name

## Problem
What business problem does this solve?

## Solution
How does the pattern address the problem?

## Implementation
SQL examples with real-world context

## Trade-offs
Pros and cons of this approach

## When to Use
Specific scenarios where this pattern applies

## Alternatives
Other approaches and when to consider them

## Examples
Real-world applications of the pattern
```

### Quality Standards
- ✅ Production-tested examples
- ✅ Clear business context
- ✅ Performance considerations
- ✅ Migration strategies
- ✅ Testing approaches

Remember: **Patterns are tools, not rules**. Choose the right pattern for your specific context, scale, and requirements. Start simple and evolve your design as your understanding of the domain deepens.
