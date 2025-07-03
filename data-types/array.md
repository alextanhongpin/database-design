# Database Arrays: Complete Guide

Array data types allow storing multiple values in a single column. While this breaks the First Normal Form (1NF) rule of atomicity, arrays can be practical when used appropriately for specific use cases.

## Table of Contents
- [When to Use Arrays](#when-to-use-arrays)
- [PostgreSQL Array Implementation](#postgresql-array-implementation)
- [MySQL Array Alternative](#mysql-alternative)
- [Array Operations](#array-operations)
- [Indexing and Performance](#indexing-and-performance)
- [Migration Patterns](#migration-patterns)
- [Best Practices](#best-practices)

## When to Use Arrays

### ✅ Good Use Cases

**Tags and Labels**
```sql
-- Repository tags
CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    tags TEXT[] DEFAULT '{}', -- e.g., ['javascript', 'web', 'frontend']
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Configuration Options**
```sql
-- Feature flags
CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY,
    enabled_features TEXT[] DEFAULT '{}', -- ['dark_mode', 'notifications', 'beta_features']
    notification_types TEXT[] DEFAULT '{email,push}'
);
```

**Historical Data**
```sql
-- Price history (when you need the sequence, not individual analysis)
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_history DECIMAL[] DEFAULT '{}', -- Store chronological prices
    stock_levels INTEGER[] DEFAULT '{}'   -- Daily stock snapshots
);
```

### ❌ Anti-Patterns

**Foreign Key References**
```sql
-- ❌ DON'T: Store user IDs as array
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    mentioned_users INTEGER[] -- Bad: references change when users are deleted
);

-- ✅ DO: Use junction table instead
CREATE TABLE post_mentions (
    post_id INTEGER REFERENCES posts(id),
    user_id INTEGER REFERENCES users(id),
    PRIMARY KEY (post_id, user_id)
);
```

**Relational Data**
```sql
-- ❌ DON'T: Store order items as array
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    items JSONB[] -- Bad: items need individual querying, pricing, inventory
);

-- ✅ DO: Separate table for items
CREATE TABLE order_items (
    order_id INTEGER REFERENCES orders(id),
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    price DECIMAL(10,2) NOT NULL
);
```

## PostgreSQL Array Implementation

### Array Types and Syntax

```sql
-- Various array types
CREATE TABLE array_examples (
    id SERIAL PRIMARY KEY,
    
    -- Text arrays
    tags TEXT[],
    categories TEXT[3], -- Fixed size: exactly 3 elements
    
    -- Numeric arrays  
    scores INTEGER[],
    prices DECIMAL(10,2)[],
    
    -- Multi-dimensional arrays
    grid INTEGER[][], -- 2D array
    
    -- Arrays with defaults
    labels TEXT[] DEFAULT '{}',
    flags BOOLEAN[] DEFAULT '{false,false,true}'
);

-- Inserting array data
INSERT INTO array_examples (tags, scores, grid) VALUES 
(
    '{"postgresql", "database", "sql"}',  -- Array literal syntax
    ARRAY[85, 92, 78, 95],              -- ARRAY constructor
    '{{1,2,3},{4,5,6}}'                 -- 2D array
);
```

### Array Querying

```sql
-- Contains operator (@>)
SELECT * FROM posts WHERE tags @> ARRAY['postgresql'];

-- Overlap operator (&&)
SELECT * FROM posts WHERE tags && ARRAY['web', 'frontend'];

-- ANY/ALL for element comparison
SELECT * FROM products WHERE 100 = ANY(price_history);
SELECT * FROM products WHERE price_history[1] > 50;

-- Array length and dimensions
SELECT 
    tags,
    array_length(tags, 1) as tag_count,
    array_dims(tags) as dimensions
FROM posts;

-- Unnesting arrays
SELECT 
    id,
    unnest(tags) as individual_tag
FROM posts;
```

### Array Modification

```sql
-- Append elements
UPDATE posts 
SET tags = array_append(tags, 'new-tag')
WHERE id = 1;

-- Prepend elements
UPDATE posts 
SET tags = array_prepend('featured', tags)
WHERE id = 1;

-- Concatenate arrays
UPDATE posts 
SET tags = tags || ARRAY['extra', 'tags']
WHERE id = 1;

-- Remove elements
UPDATE posts 
SET tags = array_remove(tags, 'old-tag')
WHERE id = 1;

-- Replace elements
UPDATE posts 
SET tags[1] = 'updated-first-tag'
WHERE id = 1;
```

## Array Slicing and Indexing

### Basic Slicing (1-based indexing)

```sql
-- Array indexing (starts from 1, not 0)
SELECT 
    ARRAY[10, 20, 30, 40, 50][1] as first_element,    -- 10
    ARRAY[10, 20, 30, 40, 50][1:3] as first_three,    -- {10,20,30}
    ARRAY[10, 20, 30, 40, 50][2:] as from_second,     -- {20,30,40,50}
    ARRAY[10, 20, 30, 40, 50][:3] as up_to_third;     -- {10,20,30}

-- Practical slicing example: Latest 3 notifications
SELECT 
    user_id,
    (array_agg(notification_data ORDER BY created_at DESC))[1:3] as recent_notifications
FROM notifications 
GROUP BY user_id;
```

### Array Slicing for Updates

```sql
-- Implementing a notification history with size limit
CREATE TABLE user_notifications (
    user_id INTEGER PRIMARY KEY,
    history JSONB[] DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Keep only the latest 10 notifications
INSERT INTO user_notifications (user_id, history) 
VALUES (1, ARRAY['{"id": "new_notification"}'::jsonb])
ON CONFLICT (user_id) DO UPDATE 
SET 
    history = (array_prepend(EXCLUDED.history[1], user_notifications.history))[1:10],
    updated_at = NOW();
```

## MySQL Alternative

MySQL doesn't have native array types, but offers JSON arrays:

```sql
-- MySQL JSON array approach
CREATE TABLE posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    tags JSON DEFAULT ('[]'),
    
    -- Add constraint to ensure it's an array
    CONSTRAINT tags_is_array CHECK (JSON_TYPE(tags) = 'ARRAY')
);

-- Insert JSON array
INSERT INTO posts (title, tags) VALUES 
('My Post', '["mysql", "json", "arrays"]');

-- Query JSON arrays
SELECT * FROM posts 
WHERE JSON_CONTAINS(tags, '"mysql"');

-- Extract array elements
SELECT 
    title,
    JSON_EXTRACT(tags, '$[0]') as first_tag,
    JSON_LENGTH(tags) as tag_count
FROM posts;
```

## Indexing and Performance

### GIN Indexes for PostgreSQL Arrays

```sql
-- Create GIN index for array containment queries
CREATE INDEX idx_posts_tags_gin ON posts USING GIN (tags);

-- This index optimizes queries like:
-- WHERE tags @> ARRAY['postgresql']
-- WHERE tags && ARRAY['web', 'frontend']

-- For element-level queries, consider GIN with specific operators
CREATE INDEX idx_posts_tags_gin_ops ON posts USING GIN (tags array_ops);
```

### Performance Considerations

```sql
-- Efficient: Uses GIN index
SELECT * FROM posts WHERE tags @> ARRAY['postgresql'];

-- Less efficient: Requires full scan
SELECT * FROM posts WHERE 'postgresql' = ANY(tags);

-- Efficient: Array length is computed quickly
SELECT * FROM posts WHERE array_length(tags, 1) > 3;

-- Inefficient: Unnesting in WHERE clause
SELECT DISTINCT post_id FROM (
    SELECT id as post_id, unnest(tags) as tag 
    FROM posts
) t WHERE tag = 'postgresql'; -- Use containment operator instead
```

## Migration Patterns

### From Arrays to Normalized Tables

```sql
-- Step 1: Create normalized table
CREATE TABLE post_tags (
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (post_id, tag)
);

-- Step 2: Migrate data
INSERT INTO post_tags (post_id, tag)
SELECT 
    p.id,
    unnest(p.tags)
FROM posts p
WHERE p.tags IS NOT NULL AND array_length(p.tags, 1) > 0;

-- Step 3: Verify migration
SELECT 
    p.id,
    p.tags as original_tags,
    array_agg(pt.tag ORDER BY pt.tag) as migrated_tags
FROM posts p
LEFT JOIN post_tags pt ON p.id = pt.post_id
GROUP BY p.id, p.tags
HAVING p.tags IS DISTINCT FROM array_agg(pt.tag ORDER BY pt.tag);

-- Step 4: Drop array column (after verification)
ALTER TABLE posts DROP COLUMN tags;
```

### From Normalized Tables to Arrays

```sql
-- Aggregate normalized data back to arrays
UPDATE posts 
SET tags = (
    SELECT COALESCE(array_agg(pt.tag ORDER BY pt.tag), '{}')
    FROM post_tags pt 
    WHERE pt.post_id = posts.id
);
```

## Best Practices

### 1. Use Arrays for Value Objects, Not Entities

```sql
-- ✅ Good: Tags are value objects (no independent lifecycle)
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    tags TEXT[] DEFAULT '{}' -- Tags don't exist independently
);

-- ❌ Bad: Authors are entities (independent lifecycle)
CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author_ids INTEGER[] -- Authors exist independently
);
```

### 2. Enforce Array Constraints

```sql
-- Limit array size
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    categories TEXT[],
    
    -- Ensure reasonable array size
    CONSTRAINT max_categories CHECK (array_length(categories, 1) <= 5),
    
    -- Ensure no null elements
    CONSTRAINT no_null_categories CHECK (NOT (NULL = ANY(categories))),
    
    -- Ensure no empty strings
    CONSTRAINT no_empty_categories CHECK (NOT ('' = ANY(categories)))
);
```

### 3. Validate Array Contents

```sql
-- Create domain for validated arrays
CREATE DOMAIN email_array AS TEXT[]
    CHECK (
        VALUE IS NULL OR (
            array_length(VALUE, 1) <= 10 AND
            NOT (NULL = ANY(VALUE)) AND
            NOT ('' = ANY(VALUE)) AND
            VALUE <@ (SELECT array_agg(email) FROM valid_emails)
        )
    );

CREATE TABLE mailing_lists (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    recipients email_array
);
```

### 4. Consider Array Alternatives

```sql
-- For frequently queried relational data, use junction tables
CREATE TABLE post_categories (
    post_id INTEGER REFERENCES posts(id),
    category_id INTEGER REFERENCES categories(id),
    PRIMARY KEY (post_id, category_id)
);

-- For JSON-like flexible structure, use JSONB
CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY,
    settings JSONB DEFAULT '{}'
);

-- For ordered collections with metadata, use numbered fields
CREATE TABLE survey_responses (
    id SERIAL PRIMARY KEY,
    question_1_answer TEXT,
    question_2_answer TEXT,
    question_3_answer TEXT
);
```

### 5. Monitor Array Performance

```sql
-- Monitor array sizes
SELECT 
    'posts.tags' as table_column,
    avg(array_length(tags, 1)) as avg_array_size,
    max(array_length(tags, 1)) as max_array_size,
    count(*) filter (where array_length(tags, 1) > 10) as large_arrays
FROM posts
WHERE tags IS NOT NULL;

-- Monitor query patterns
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM posts WHERE tags @> ARRAY['postgresql'];
```

## Conclusion

Arrays are powerful for specific use cases but should be used judiciously:

- **Use for**: Tags, labels, simple lists, configuration arrays
- **Avoid for**: Foreign key relationships, frequently queried individual elements
- **Consider alternatives**: JSON for flexible structure, junction tables for relational data
- **Index appropriately**: GIN indexes for containment queries
- **Validate data**: Add constraints to ensure data quality

The key is understanding when arrays simplify your data model versus when they add unnecessary complexity.
