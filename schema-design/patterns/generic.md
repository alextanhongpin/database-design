# Generic & Polymorphic Database Patterns

Polymorphism in databases involves storing different types of related data in flexible structures. This guide covers various approaches to handle polymorphic relationships, type hierarchies, and generic data patterns.

## 🎯 Understanding Polymorphism in Databases

### What is Database Polymorphism?
Polymorphism allows a single interface to represent different types. In databases, this translates to:
- **Type Hierarchies** - Storing different but related entity types
- **Flexible Relationships** - One table relating to multiple other tables
- **Generic Attributes** - Dynamic properties that vary by type
- **Inheritance Patterns** - Sharing common fields across types

### Common Use Cases
- **Content Management** - Articles, videos, images, podcasts
- **E-commerce** - Physical products, digital downloads, subscriptions
- **Social Media** - Posts, comments, reactions, shares
- **Payment Methods** - Credit cards, bank transfers, digital wallets
- **Notifications** - Email, SMS, push, in-app notifications

## 🏗️ Core Polymorphic Patterns

### 1. Single Table Inheritance (STI)

**Best for**: Similar entities with mostly shared attributes

```sql
-- All content types in one table
CREATE TABLE content_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('article', 'video', 'podcast', 'image')),
    title TEXT NOT NULL,
    description TEXT,
    author_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Article-specific fields
    body TEXT,
    word_count INTEGER,
    
    -- Video/Podcast-specific fields
    duration_seconds INTEGER,
    file_url TEXT,
    file_size_bytes BIGINT,
    
    -- Image-specific fields
    width INTEGER,
    height INTEGER,
    alt_text TEXT,
    
    -- Constraints based on type
    CONSTRAINT valid_article CHECK (
        type != 'article' OR (body IS NOT NULL AND word_count > 0)
    ),
    CONSTRAINT valid_media CHECK (
        type NOT IN ('video', 'podcast') OR (file_url IS NOT NULL AND duration_seconds > 0)
    ),
    CONSTRAINT valid_image CHECK (
        type != 'image' OR (width > 0 AND height > 0)
    )
);

-- Indexes for type-specific queries
CREATE INDEX idx_content_type ON content_items(type);
CREATE INDEX idx_content_author_type ON content_items(author_id, type);
```

### 2. Class Table Inheritance (CTI)

**Best for**: Different entities with some shared and many distinct attributes

```sql
-- Base table with common attributes
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('physical', 'digital', 'subscription')),
    name TEXT NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Physical products table
CREATE TABLE physical_products (
    product_id UUID PRIMARY KEY,
    weight_grams INTEGER NOT NULL,
    dimensions_cm INTEGER[3], -- [length, width, height]
    requires_shipping BOOLEAN DEFAULT true,
    inventory_quantity INTEGER DEFAULT 0,
    sku TEXT UNIQUE,
    
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT valid_physical_product CHECK (
        weight_grams > 0 AND 
        array_length(dimensions_cm, 1) = 3
    )
);

-- Digital products table
CREATE TABLE digital_products (
    product_id UUID PRIMARY KEY,
    file_url TEXT NOT NULL,
    file_size_bytes BIGINT,
    download_limit INTEGER,
    license_type TEXT DEFAULT 'single_user',
    platform_compatibility TEXT[],
    
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

-- Subscription products table
CREATE TABLE subscription_products (
    product_id UUID PRIMARY KEY,
    billing_cycle TEXT NOT NULL CHECK (billing_cycle IN ('monthly', 'yearly', 'weekly')),
    trial_period_days INTEGER DEFAULT 0,
    max_users INTEGER,
    features JSONB,
    
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
```

### 3. Polymorphic Association Pattern

**Best for**: When one entity needs to relate to multiple different entity types

```sql
-- Comments that can be on articles, videos, or products
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    author_id UUID NOT NULL,
    
    -- Polymorphic relationship fields
    commentable_type TEXT NOT NULL CHECK (
        commentable_type IN ('article', 'video', 'product', 'event')
    ),
    commentable_id UUID NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure the combination is unique if needed
    UNIQUE(commentable_type, commentable_id, author_id)
);

-- Indexes for polymorphic queries
CREATE INDEX idx_comments_polymorphic ON comments(commentable_type, commentable_id);
CREATE INDEX idx_comments_author ON comments(author_id);

-- Views for type-specific access
CREATE VIEW article_comments AS
SELECT c.*, a.title as article_title
FROM comments c
JOIN articles a ON c.commentable_id = a.id
WHERE c.commentable_type = 'article';

CREATE VIEW product_comments AS
SELECT c.*, p.name as product_name
FROM comments c
JOIN products p ON c.commentable_id = p.id
WHERE c.commentable_type = 'product';
```

## 🔄 Advanced Polymorphic Patterns

### 1. Entity-Attribute-Value (EAV) Pattern

**Best for**: Highly dynamic attributes that vary significantly by type

```sql
-- Entity definitions
CREATE TABLE entity_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    schema_definition JSONB -- JSON Schema for validation
);

-- Attribute definitions
CREATE TABLE attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type_id UUID NOT NULL,
    name TEXT NOT NULL,
    data_type TEXT NOT NULL CHECK (
        data_type IN ('string', 'integer', 'decimal', 'boolean', 'date', 'json')
    ),
    is_required BOOLEAN DEFAULT false,
    validation_rules JSONB,
    
    FOREIGN KEY (entity_type_id) REFERENCES entity_types(id),
    UNIQUE(entity_type_id, name)
);

-- Entities (instances)
CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type_id UUID NOT NULL,
    name TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (entity_type_id) REFERENCES entity_types(id)
);

-- Attribute values (polymorphic values)
CREATE TABLE attribute_values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL,
    attribute_id UUID NOT NULL,
    
    -- Polymorphic value storage
    string_value TEXT,
    integer_value BIGINT,
    decimal_value DECIMAL(15,6),
    boolean_value BOOLEAN,
    date_value TIMESTAMP,
    json_value JSONB,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE,
    FOREIGN KEY (attribute_id) REFERENCES attributes(id),
    UNIQUE(entity_id, attribute_id),
    
    -- Ensure only one value type is set
    CHECK (
        (string_value IS NOT NULL)::int + 
        (integer_value IS NOT NULL)::int + 
        (decimal_value IS NOT NULL)::int + 
        (boolean_value IS NOT NULL)::int + 
        (date_value IS NOT NULL)::int + 
        (json_value IS NOT NULL)::int = 1
    )
);
```

### 2. JSONB-Based Polymorphism

**Best for**: Modern PostgreSQL environments with complex, varied attributes

```sql
-- Products with type-specific attributes in JSONB
CREATE TABLE products_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    base_price DECIMAL(10,2) NOT NULL,
    
    -- All type-specific attributes in JSONB
    attributes JSONB NOT NULL DEFAULT '{}',
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- JSON Schema validation
    CONSTRAINT valid_physical_attributes CHECK (
        type != 'physical' OR (
            attributes ? 'weight' AND 
            attributes ? 'dimensions' AND
            (attributes->>'weight')::numeric > 0
        )
    ),
    CONSTRAINT valid_digital_attributes CHECK (
        type != 'digital' OR (
            attributes ? 'file_url' AND 
            attributes ? 'file_size'
        )
    )
);

-- Indexes for JSONB queries
CREATE INDEX idx_products_type ON products_v2(type);
CREATE INDEX idx_products_attributes_gin ON products_v2 USING gin(attributes);
CREATE INDEX idx_products_weight ON products_v2 USING btree(((attributes->>'weight')::numeric))
    WHERE type = 'physical';

-- Example usage
INSERT INTO products_v2 (type, name, base_price, attributes) VALUES
('physical', 'Laptop', 999.99, '{
    "weight": 2.5,
    "dimensions": {"length": 35, "width": 25, "height": 2},
    "color": "silver",
    "warranty_months": 24
}'),
('digital', 'Software License', 199.99, '{
    "file_url": "https://download.example.com/software",
    "file_size": 157286400,
    "license_type": "perpetual",
    "platforms": ["windows", "mac", "linux"]
}');
```

## 🎭 Type-Safe Polymorphic Queries

### 1. Union Views for Type Safety

```sql
-- Create type-specific views that enforce structure
CREATE VIEW physical_products_view AS
SELECT 
    id,
    name,
    base_price,
    (attributes->>'weight')::numeric as weight,
    (attributes->'dimensions'->>'length')::integer as length,
    (attributes->'dimensions'->>'width')::integer as width,
    (attributes->'dimensions'->>'height')::integer as height,
    attributes->>'color' as color,
    created_at
FROM products_v2 
WHERE type = 'physical';

CREATE VIEW digital_products_view AS
SELECT 
    id,
    name,
    base_price,
    attributes->>'file_url' as file_url,
    (attributes->>'file_size')::bigint as file_size,
    attributes->>'license_type' as license_type,
    attributes->'platforms' as platforms,
    created_at
FROM products_v2 
WHERE type = 'digital';
```

### 2. Polymorphic Query Functions

```sql
-- Function to get typed product data
CREATE OR REPLACE FUNCTION get_product_with_type(product_id UUID)
RETURNS TABLE(
    id UUID,
    type TEXT,
    name TEXT,
    price DECIMAL(10,2),
    type_specific_data JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.type,
        p.name,
        p.base_price,
        CASE p.type
            WHEN 'physical' THEN jsonb_build_object(
                'weight', p.attributes->>'weight',
                'dimensions', p.attributes->'dimensions',
                'shipping_required', true
            )
            WHEN 'digital' THEN jsonb_build_object(
                'file_url', p.attributes->>'file_url',
                'file_size', p.attributes->>'file_size',
                'instant_download', true
            )
            ELSE p.attributes
        END
    FROM products_v2 p
    WHERE p.id = product_id;
END;
$$ LANGUAGE plpgsql;
```

## ⚡ Performance Optimization

### 1. Strategic Indexing for Polymorphic Data

```sql
-- Partial indexes for specific types
CREATE INDEX idx_active_physical_products 
ON products_v2(id) 
WHERE type = 'physical' AND status = 'active';

-- Functional indexes on JSONB attributes
CREATE INDEX idx_physical_weight 
ON products_v2(((attributes->>'weight')::numeric))
WHERE type = 'physical';

-- Composite indexes for common query patterns
CREATE INDEX idx_type_price_created 
ON products_v2(type, base_price, created_at);
```

### 2. Materialized Views for Complex Polymorphic Queries

```sql
-- Materialized view for product analytics
CREATE MATERIALIZED VIEW product_analytics AS
SELECT 
    type,
    COUNT(*) as total_count,
    AVG(base_price) as avg_price,
    CASE type
        WHEN 'physical' THEN AVG((attributes->>'weight')::numeric)
        WHEN 'digital' THEN AVG((attributes->>'file_size')::bigint)
        ELSE NULL
    END as avg_type_metric,
    DATE_TRUNC('month', created_at) as month
FROM products_v2
WHERE status = 'active'
GROUP BY type, DATE_TRUNC('month', created_at);

-- Refresh the materialized view
CREATE OR REPLACE FUNCTION refresh_product_analytics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW product_analytics;
END;
$$ LANGUAGE plpgsql;
```

## ⚠️ Anti-Patterns and Pitfalls

### 1. Over-Normalization
```sql
-- ❌ Too many tables for simple variations
CREATE TABLE products (id UUID, name TEXT, price DECIMAL);
CREATE TABLE product_colors (product_id UUID, color TEXT);
CREATE TABLE product_sizes (product_id UUID, size TEXT);
CREATE TABLE product_materials (product_id UUID, material TEXT);

-- ✅ Use JSONB for simple attributes
CREATE TABLE products (
    id UUID PRIMARY KEY,
    name TEXT,
    price DECIMAL,
    attributes JSONB -- Store color, size, material here
);
```

### 2. Excessive STI (Single Table Inheritance)
```sql
-- ❌ Too many NULL columns
CREATE TABLE content (
    id UUID,
    type TEXT,
    title TEXT,
    -- Article fields (NULLs for other types)
    body TEXT,
    word_count INTEGER,
    -- Video fields (NULLs for other types)  
    duration INTEGER,
    resolution TEXT,
    -- Image fields (NULLs for other types)
    width INTEGER,
    height INTEGER
    -- ... 20+ type-specific columns
);

-- ✅ Use Class Table Inheritance or JSONB
```

### 3. Ignoring Referential Integrity in Polymorphic Associations

```sql
-- ❌ No enforcement of referential integrity
CREATE TABLE comments (
    id UUID,
    commentable_type TEXT,
    commentable_id UUID -- Could reference non-existent records
);

-- ✅ Add integrity checks
CREATE OR REPLACE FUNCTION validate_polymorphic_reference()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.commentable_type = 'article' THEN
        IF NOT EXISTS(SELECT 1 FROM articles WHERE id = NEW.commentable_id) THEN
            RAISE EXCEPTION 'Invalid article reference: %', NEW.commentable_id;
        END IF;
    ELSIF NEW.commentable_type = 'product' THEN
        IF NOT EXISTS(SELECT 1 FROM products WHERE id = NEW.commentable_id) THEN
            RAISE EXCEPTION 'Invalid product reference: %', NEW.commentable_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER validate_comment_reference
    BEFORE INSERT OR UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION validate_polymorphic_reference();
```

## 🎯 Choosing the Right Pattern

| Pattern | Best For | Pros | Cons |
|---------|----------|------|------|
| **Single Table Inheritance** | Similar entities, mostly shared attributes | Simple queries, good performance | Many NULL columns, rigid schema |
| **Class Table Inheritance** | Related entities, many distinct attributes | Clean separation, no NULLs | Complex joins, more tables |
| **Polymorphic Association** | One-to-many across types | Flexible relationships | Referential integrity challenges |
| **EAV Pattern** | Highly dynamic attributes | Maximum flexibility | Complex queries, performance issues |
| **JSONB Polymorphism** | PostgreSQL, semi-structured data | JSON flexibility with SQL power | PostgreSQL-specific, less structured |

## 🔗 Best Practices

1. **Start Simple** - Begin with STI if entities are similar
2. **Enforce Constraints** - Use CHECK constraints and triggers for validation
3. **Index Strategically** - Create partial and functional indexes for common patterns
4. **Use Views** - Abstract complex polymorphic queries behind views
5. **Document Types** - Maintain clear documentation of type schemas
6. **Validate JSON** - Use JSON Schema validation for JSONB approaches
7. **Consider Performance** - Profile queries and optimize for your access patterns
8. **Plan Migration Paths** - Design for evolution between patterns

## 🔗 References

- [PostgreSQL Inheritance](https://www.postgresql.org/docs/current/ddl-inherit.html)
- [JSONB in PostgreSQL](https://www.postgresql.org/docs/current/datatype-json.html)
- [Polymorphic Associations in Rails](https://guides.rubyonrails.org/association_basics.html#polymorphic-associations)
- [EAV Database Design Pattern](https://en.wikipedia.org/wiki/Entity%E2%80%93attribute%E2%80%93value_model)
