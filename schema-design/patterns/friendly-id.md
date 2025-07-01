# Friendly ID & Slug Patterns

Friendly IDs provide human-readable, URL-safe identifiers that improve user experience and SEO while maintaining database performance. This guide covers implementation strategies, slug generation, and best practices.

## 🎯 What are Friendly IDs?

### Definition & Benefits
- **Human-readable** - `github.com/user/repo` vs `github.com/12345/67890`
- **URL-friendly** - Safe for web URLs without encoding
- **SEO-optimized** - Descriptive URLs improve search rankings
- **Memorable** - Easier for users to remember and share
- **Security** - Avoid exposing internal numeric IDs

### Common Use Cases
- **User profiles** - `/users/john-doe` vs `/users/12345`
- **Blog posts** - `/posts/how-to-build-apis` vs `/posts/987`
- **Products** - `/products/iphone-15-pro` vs `/products/456`
- **Organizations** - `/orgs/my-company` vs `/orgs/789`
- **API endpoints** - More intuitive and stable URLs

## 🏗️ Implementation Patterns

### 1. Dual Column Approach (Recommended)

```sql
-- Best practice: UUID primary key + friendly slug
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    user_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Indexes for both lookup patterns
CREATE INDEX idx_posts_slug ON posts(slug);
CREATE INDEX idx_posts_user_created ON posts(user_id, created_at DESC);

-- Function to generate slug from title
CREATE OR REPLACE FUNCTION generate_slug(title TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN lower(
        regexp_replace(
            regexp_replace(
                regexp_replace(title, '[^\w\s-]', '', 'g'), -- Remove special chars
                '\s+', '-', 'g'                            -- Spaces to hyphens
            ),
            '-+', '-', 'g'                                 -- Multiple hyphens to single
        )
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Auto-generate slug on insert
CREATE OR REPLACE FUNCTION auto_generate_slug()
RETURNS TRIGGER AS $$
DECLARE
    base_slug TEXT;
    final_slug TEXT;
    counter INTEGER := 1;
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        base_slug := generate_slug(NEW.title);
        final_slug := base_slug;
        
        -- Ensure uniqueness by appending counter if needed
        WHILE EXISTS(SELECT 1 FROM posts WHERE slug = final_slug AND id != COALESCE(NEW.id, '00000000-0000-0000-0000-000000000000'::UUID)) LOOP
            final_slug := base_slug || '-' || counter;
            counter := counter + 1;
        END LOOP;
        
        NEW.slug := final_slug;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auto_generate_slug
    BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION auto_generate_slug();
```

### 2. Advanced Slug Generation

```sql
-- More sophisticated slug generation with customization
CREATE OR REPLACE FUNCTION generate_advanced_slug(
    input_text TEXT,
    max_length INTEGER DEFAULT 50,
    preserve_case BOOLEAN DEFAULT FALSE
) RETURNS TEXT AS $$
DECLARE
    clean_text TEXT;
    words TEXT[];
    result_slug TEXT := '';
    word TEXT;
BEGIN
    -- Handle null/empty input
    IF input_text IS NULL OR trim(input_text) = '' THEN
        RETURN 'untitled';
    END IF;
    
    -- Clean and normalize text
    clean_text := trim(input_text);
    
    -- Convert to lowercase unless preserving case
    IF NOT preserve_case THEN
        clean_text := lower(clean_text);
    END IF;
    
    -- Remove special characters, keep letters, numbers, spaces, hyphens
    clean_text := regexp_replace(clean_text, '[^\w\s-]', '', 'g');
    
    -- Split into words and rebuild with hyphens
    words := string_to_array(clean_text, ' ');
    
    FOREACH word IN ARRAY words LOOP
        word := trim(word);
        IF word != '' THEN
            IF result_slug != '' THEN
                result_slug := result_slug || '-';
            END IF;
            result_slug := result_slug || word;
            
            -- Stop if we're approaching max length
            IF length(result_slug) >= max_length THEN
                EXIT;
            END IF;
        END IF;
    END LOOP;
    
    -- Trim to max length at word boundary
    IF length(result_slug) > max_length THEN
        result_slug := left(result_slug, max_length);
        -- Trim at last hyphen to avoid cutting words
        IF position('-' in reverse(result_slug)) > 0 THEN
            result_slug := left(result_slug, max_length - position('-' in reverse(result_slug)));
        END IF;
    END IF;
    
    -- Clean up multiple hyphens and trim
    result_slug := regexp_replace(result_slug, '-+', '-', 'g');
    result_slug := trim(result_slug, '-');
    
    -- Fallback for edge cases
    IF result_slug = '' THEN
        result_slug := 'item';
    END IF;
    
    RETURN result_slug;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

### 3. Hierarchical Friendly IDs

```sql
-- Organizations with hierarchical slugs
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    parent_id UUID,
    
    FOREIGN KEY (parent_id) REFERENCES organizations(id)
);

-- Projects within organizations
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    organization_id UUID NOT NULL,
    
    -- Slug unique within organization
    UNIQUE (organization_id, slug),
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

-- Function to get full hierarchical path
CREATE OR REPLACE FUNCTION get_project_path(project_id UUID)
RETURNS TEXT AS $$
DECLARE
    org_slug TEXT;
    proj_slug TEXT;
BEGIN
    SELECT o.slug, p.slug
    INTO org_slug, proj_slug
    FROM projects p
    JOIN organizations o ON p.organization_id = o.id
    WHERE p.id = project_id;
    
    RETURN org_slug || '/' || proj_slug;
END;
$$ LANGUAGE plpgsql;

-- View for easy path access
CREATE VIEW project_paths AS
SELECT 
    p.id,
    p.name,
    p.slug,
    o.slug as org_slug,
    o.slug || '/' || p.slug as full_path
FROM projects p
JOIN organizations o ON p.organization_id = o.id;
```

## 🔍 Lookup & Resolution Patterns

### 1. Flexible ID Resolution

```sql
-- Function that can resolve both UUID and slug
CREATE OR REPLACE FUNCTION resolve_post_id(identifier TEXT)
RETURNS UUID AS $$
DECLARE
    post_id UUID;
BEGIN
    -- Try to parse as UUID first
    BEGIN
        post_id := identifier::UUID;
        -- Verify UUID exists
        IF EXISTS(SELECT 1 FROM posts WHERE id = post_id) THEN
            RETURN post_id;
        END IF;
    EXCEPTION WHEN invalid_text_representation THEN
        -- Not a valid UUID, continue to slug lookup
    END;
    
    -- Look up by slug
    SELECT id INTO post_id
    FROM posts
    WHERE slug = identifier;
    
    IF post_id IS NULL THEN
        RAISE EXCEPTION 'Post not found: %', identifier;
    END IF;
    
    RETURN post_id;
END;
$$ LANGUAGE plpgsql;

-- Usage examples
SELECT * FROM posts WHERE id = resolve_post_id('how-to-build-apis');
SELECT * FROM posts WHERE id = resolve_post_id('123e4567-e89b-12d3-a456-426614174000');
```

### 2. Anti-Pattern Protection

```sql
-- Prevent purely numeric slugs (security issue)
CREATE OR REPLACE FUNCTION validate_slug()
RETURNS TRIGGER AS $$
BEGIN
    -- Reject purely numeric slugs
    IF NEW.slug ~ '^\d+$' THEN
        RAISE EXCEPTION 'Slug cannot be purely numeric: %', NEW.slug;
    END IF;
    
    -- Reject reserved words
    IF NEW.slug IN ('admin', 'api', 'www', 'mail', 'ftp', 'localhost', 'new', 'edit', 'delete') THEN
        RAISE EXCEPTION 'Slug cannot be a reserved word: %', NEW.slug;
    END IF;
    
    -- Minimum length check
    IF length(NEW.slug) < 3 THEN
        RAISE EXCEPTION 'Slug must be at least 3 characters: %', NEW.slug;
    END IF;
    
    -- Maximum length check
    IF length(NEW.slug) > 50 THEN
        RAISE EXCEPTION 'Slug cannot exceed 50 characters: %', NEW.slug;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER validate_post_slug
    BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION validate_slug();
```

## 🎭 Alternative Approaches

### 1. Hash-Based Short IDs

```sql
-- Short, opaque but URL-friendly IDs
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION generate_short_id(length INTEGER DEFAULT 8)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(ceil(length * 3.0/4.0)::INTEGER), 'base64')
           -- Remove URL-unsafe characters
           REPLACE 'base64', '+', '-')
           REPLACE 'base64', '/', '_')
           -- Trim to desired length
           LEFT(length);
END;
$$ LANGUAGE plpgsql;

-- Table with short IDs
CREATE TABLE short_urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id TEXT UNIQUE NOT NULL DEFAULT generate_short_id(8),
    long_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Example: abc123xy -> much shorter than full UUID
```

### 2. Nanoid Implementation

```sql
-- Nanoid-style ID generation (URL-safe, shorter than UUID)
CREATE OR REPLACE FUNCTION generate_nanoid(size INTEGER DEFAULT 21)
RETURNS TEXT AS $$
DECLARE
    alphabet TEXT := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-';
    result TEXT := '';
    i INTEGER;
    random_byte INTEGER;
BEGIN
    FOR i IN 1..size LOOP
        random_byte := (random() * 63)::INTEGER + 1;
        result := result || substr(alphabet, random_byte, 1);
    END LOOP;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Usage in table
CREATE TABLE events (
    id TEXT PRIMARY KEY DEFAULT generate_nanoid(),
    name TEXT NOT NULL,
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 🚀 Performance Optimization

### 1. Efficient Lookups

```sql
-- Covering index for common queries
CREATE UNIQUE INDEX idx_posts_slug_covering 
ON posts(slug) 
INCLUDE (id, title, created_at, user_id);

-- Partial indexes for active content
CREATE UNIQUE INDEX idx_posts_active_slug 
ON posts(slug) 
WHERE status = 'published';

-- Function-based index for case-insensitive lookups
CREATE UNIQUE INDEX idx_posts_slug_ci 
ON posts(lower(slug));
```

### 2. Bulk Slug Operations

```sql
-- Batch slug generation for existing records
CREATE OR REPLACE FUNCTION regenerate_slugs_batch(
    table_name TEXT,
    title_column TEXT DEFAULT 'title',
    slug_column TEXT DEFAULT 'slug',
    batch_size INTEGER DEFAULT 1000
) RETURNS INTEGER AS $$
DECLARE
    total_updated INTEGER := 0;
    batch_count INTEGER;
    sql_query TEXT;
BEGIN
    LOOP
        sql_query := format('
            UPDATE %I 
            SET %I = generate_advanced_slug(%I)
            WHERE %I IS NULL OR %I = ''''
            AND id IN (
                SELECT id FROM %I 
                WHERE %I IS NULL OR %I = ''''
                LIMIT %s
            )',
            table_name, slug_column, title_column,
            slug_column, slug_column,
            table_name, slug_column, slug_column,
            batch_size
        );
        
        EXECUTE sql_query;
        GET DIAGNOSTICS batch_count = ROW_COUNT;
        
        total_updated := total_updated + batch_count;
        
        EXIT WHEN batch_count = 0;
        
        -- Small delay to avoid overwhelming the database
        PERFORM pg_sleep(0.1);
    END LOOP;
    
    RETURN total_updated;
END;
$$ LANGUAGE plpgsql;
```

## 🔒 Security Considerations

### 1. Slug Enumeration Prevention

```sql
-- Add randomness to prevent easy enumeration
CREATE OR REPLACE FUNCTION generate_secure_slug(base_text TEXT)
RETURNS TEXT AS $$
DECLARE
    base_slug TEXT;
    random_suffix TEXT;
BEGIN
    base_slug := generate_advanced_slug(base_text, 40);
    
    -- Add short random suffix for uniqueness and security
    random_suffix := generate_short_id(4);
    
    RETURN base_slug || '-' || random_suffix;
END;
$$ LANGUAGE plpgsql;

-- Usage
INSERT INTO posts (title, slug) 
VALUES ('My Blog Post', generate_secure_slug('My Blog Post'));
-- Results in: my-blog-post-x7k9
```

### 2. Rate Limiting Slug Generation

```sql
-- Prevent abuse of slug generation endpoints
CREATE TABLE slug_generation_limits (
    ip_address INET NOT NULL,
    user_id UUID,
    generation_count INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (ip_address, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID))
);

CREATE OR REPLACE FUNCTION check_slug_generation_limit(
    p_ip_address INET,
    p_user_id UUID DEFAULT NULL,
    p_max_generations INTEGER DEFAULT 100
) RETURNS BOOLEAN AS $$
DECLARE
    current_count INTEGER;
BEGIN
    -- Clean old records
    DELETE FROM slug_generation_limits 
    WHERE window_start < NOW() - INTERVAL '1 hour';
    
    -- Check current usage
    SELECT generation_count INTO current_count
    FROM slug_generation_limits
    WHERE ip_address = p_ip_address
    AND user_id IS NOT DISTINCT FROM p_user_id;
    
    IF current_count IS NULL THEN
        -- First generation in this window
        INSERT INTO slug_generation_limits (ip_address, user_id)
        VALUES (p_ip_address, p_user_id);
        RETURN TRUE;
    END IF;
    
    IF current_count >= p_max_generations THEN
        RETURN FALSE; -- Rate limit exceeded
    END IF;
    
    -- Increment counter
    UPDATE slug_generation_limits
    SET generation_count = generation_count + 1
    WHERE ip_address = p_ip_address
    AND user_id IS NOT DISTINCT FROM p_user_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Monitoring & Analytics

### 1. Slug Usage Analytics

```sql
-- Track slug performance and usage
CREATE TABLE slug_analytics (
    slug TEXT NOT NULL,
    table_name TEXT NOT NULL,
    access_count BIGINT DEFAULT 1,
    last_accessed TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (slug, table_name)
);

-- Function to record slug access
CREATE OR REPLACE FUNCTION record_slug_access(
    p_slug TEXT,
    p_table_name TEXT
) RETURNS VOID AS $$
BEGIN
    INSERT INTO slug_analytics (slug, table_name)
    VALUES (p_slug, p_table_name)
    ON CONFLICT (slug, table_name)
    DO UPDATE SET
        access_count = slug_analytics.access_count + 1,
        last_accessed = NOW();
END;
$$ LANGUAGE plpgsql;

-- View for popular slugs
CREATE VIEW popular_slugs AS
SELECT 
    table_name,
    slug,
    access_count,
    last_accessed,
    RANK() OVER (PARTITION BY table_name ORDER BY access_count DESC) as popularity_rank
FROM slug_analytics
ORDER BY table_name, access_count DESC;
```

## ⚠️ Common Pitfalls

### 1. Purely Numeric Slugs
```sql
-- ❌ Security risk - can be confused with internal IDs
INSERT INTO posts (slug, title) VALUES ('12345', 'Some Title');

-- ✅ Always validate against purely numeric slugs
-- See validation function above
```

### 2. Case Sensitivity Issues
```sql
-- ❌ Case-sensitive lookup might fail
SELECT * FROM posts WHERE slug = 'My-Blog-Post'; -- Won't match 'my-blog-post'

-- ✅ Use consistent case (lowercase) and case-insensitive indexes
CREATE UNIQUE INDEX idx_posts_slug_ci ON posts(lower(slug));
```

### 3. Missing Uniqueness Handling
```sql
-- ❌ Duplicate slug errors
INSERT INTO posts (slug, title) VALUES ('same-title', 'Same Title');
INSERT INTO posts (slug, title) VALUES ('same-title', 'Same Title'); -- ERROR!

-- ✅ Use auto-incrementing suffix pattern (see implementation above)
```

## 🎯 Best Practices

1. **Keep UUIDs as Primary Keys** - Don't lose the benefits of UUIDs
2. **Auto-generate Slugs** - Use triggers or application logic consistently
3. **Validate Slug Format** - Prevent security issues and conflicts
4. **Handle Duplicates Gracefully** - Auto-append counters or random suffixes
5. **Index Strategically** - Cover both UUID and slug lookups
6. **Consider Case Sensitivity** - Use consistent casing rules
7. **Implement Rate Limiting** - Prevent abuse of slug generation
8. **Monitor Usage** - Track popular slugs and access patterns
9. **Plan for Changes** - Consider slug history and redirects
10. **Test Edge Cases** - Unicode, special characters, very long titles

## 📊 Pattern Comparison

| Pattern | URL Friendliness | Performance | Security | Complexity |
|---------|------------------|-------------|----------|------------|
| **UUID + Slug** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Slug Only** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Hash IDs** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ |
| **Short IDs** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Numeric IDs** | ⭐ | ⭐⭐⭐⭐⭐ | ⭐ | ⭐ |

## 🔗 References

- [Rails FriendlyId Gem](https://github.com/norman/friendly_id)
- [Google API Design - Identifiers](https://cloud.google.com/blog/products/api-management/api-design-choosing-between-names-and-identifiers-in-urls)
- [URL Slug Best Practices](https://developers.google.com/search/docs/crawling-indexing/url-structure)
- [NanoID Specification](https://github.com/ai/nanoid)




