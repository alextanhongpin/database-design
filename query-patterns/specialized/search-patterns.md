# Search & Text Query Patterns

Modern applications require sophisticated search capabilities. This guide covers full-text search, fuzzy matching, trigram indexing, and advanced text search patterns in PostgreSQL.

## 🎯 Core Search Patterns

### 1. Full-Text Search (PostgreSQL)

**Best for**: Natural language search across text content

```sql
-- Basic full-text search setup
CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    author_id UUID NOT NULL,
    published_at TIMESTAMP DEFAULT NOW(),
    
    -- Generated tsvector column for search
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(content, '')), 'B')
    ) STORED
);

-- Efficient GIN index for full-text search
CREATE INDEX idx_articles_search ON articles USING gin(search_vector);

-- Basic search queries
-- Simple search
SELECT id, title, ts_rank(search_vector, query) as rank
FROM articles, plainto_tsquery('english', 'database design') as query
WHERE search_vector @@ query
ORDER BY rank DESC;

-- Phrase search
SELECT id, title
FROM articles
WHERE search_vector @@ phraseto_tsquery('english', 'database design patterns');

-- Prefix search (useful for autocomplete)
SELECT id, title
FROM articles
WHERE search_vector @@ to_tsquery('english', 'datab:*');
```

### 2. Advanced Full-Text Search

```sql
-- Multi-language search support
CREATE TABLE multilingual_content (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title_en TEXT,
    content_en TEXT,
    title_es TEXT,
    content_es TEXT,
    
    -- Language-specific search vectors
    search_vector_en tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title_en, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(content_en, '')), 'B')
    ) STORED,
    search_vector_es tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('spanish', coalesce(title_es, '')), 'A') ||
        setweight(to_tsvector('spanish', coalesce(content_es, '')), 'B')
    ) STORED
);

CREATE INDEX idx_content_search_en ON multilingual_content USING gin(search_vector_en);
CREATE INDEX idx_content_search_es ON multilingual_content USING gin(search_vector_es);

-- Search function with language support
CREATE OR REPLACE FUNCTION search_content(
    p_query TEXT,
    p_language TEXT DEFAULT 'english',
    p_limit INTEGER DEFAULT 20
) RETURNS TABLE(
    id UUID,
    title TEXT,
    content TEXT,
    rank REAL
) AS $$
DECLARE
    search_config TEXT;
    search_column TEXT;
BEGIN
    -- Determine search configuration and column
    CASE p_language
        WHEN 'spanish' THEN
            search_config := 'spanish';
            search_column := 'search_vector_es';
        ELSE
            search_config := 'english';
            search_column := 'search_vector_en';
    END CASE;
    
    RETURN QUERY EXECUTE format('
        SELECT 
            mc.id,
            CASE WHEN %L = ''english'' THEN title_en ELSE title_es END,
            CASE WHEN %L = ''english'' THEN content_en ELSE content_es END,
            ts_rank(%I, query) as rank
        FROM multilingual_content mc,
             plainto_tsquery(%L, %L) as query
        WHERE %I @@ query
        ORDER BY rank DESC
        LIMIT %s',
        p_language, p_language, search_column, search_config, p_query, search_column, p_limit
    );
END;
$$ LANGUAGE plpgsql;
```

### 3. Trigram Search for Fuzzy Matching

**Best for**: Typo-tolerant search, partial matches, and similarity

```sql
-- Enable trigram extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram index for similarity search
CREATE INDEX idx_products_name_trigram ON products USING gin(name gin_trgm_ops);
CREATE INDEX idx_users_email_trigram ON users USING gin(email gin_trgm_ops);

-- Fuzzy search with similarity scoring
SELECT 
    id,
    name,
    similarity(name, 'iPhone') as similarity_score
FROM products
WHERE similarity(name, 'iPhone') > 0.3
ORDER BY similarity_score DESC;

-- Combined ILIKE and trigram search
SELECT id, name
FROM products
WHERE name ILIKE '%phone%'
   OR similarity(name, 'phone') > 0.3
ORDER BY 
    CASE WHEN name ILIKE '%phone%' THEN 1 ELSE 0 END DESC,
    similarity(name, 'phone') DESC;

-- Advanced trigram search with multiple terms
CREATE OR REPLACE FUNCTION fuzzy_search_products(
    p_query TEXT,
    p_similarity_threshold REAL DEFAULT 0.3
) RETURNS TABLE(
    id UUID,
    name TEXT,
    similarity_score REAL,
    match_type TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH search_results AS (
        SELECT 
            p.id,
            p.name,
            similarity(p.name, p_query) as sim_score,
            CASE 
                WHEN p.name ILIKE '%' || p_query || '%' THEN 'exact_substring'
                WHEN similarity(p.name, p_query) > 0.6 THEN 'high_similarity'
                WHEN similarity(p.name, p_query) > p_similarity_threshold THEN 'medium_similarity'
                ELSE 'low_similarity'
            END as match_type
        FROM products p
        WHERE p.name ILIKE '%' || p_query || '%'
           OR similarity(p.name, p_query) > p_similarity_threshold
    )
    SELECT 
        sr.id,
        sr.name,
        sr.sim_score,
        sr.match_type
    FROM search_results sr
    ORDER BY 
        CASE sr.match_type
            WHEN 'exact_substring' THEN 1
            WHEN 'high_similarity' THEN 2
            WHEN 'medium_similarity' THEN 3
            ELSE 4
        END,
        sr.sim_score DESC;
END;
$$ LANGUAGE plpgsql;
```

## 🚀 Advanced Search Techniques

### 1. Hybrid Search (Full-Text + Trigram)

```sql
-- Combined search strategy for best results
CREATE OR REPLACE FUNCTION hybrid_search(
    p_query TEXT,
    p_limit INTEGER DEFAULT 20
) RETURNS TABLE(
    id UUID,
    title TEXT,
    content_snippet TEXT,
    search_rank REAL,
    search_type TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH fulltext_results AS (
        SELECT 
            a.id,
            a.title,
            ts_headline('english', left(a.content, 200), query) as snippet,
            ts_rank(a.search_vector, query) as rank,
            'fulltext' as type
        FROM articles a,
             plainto_tsquery('english', p_query) as query
        WHERE a.search_vector @@ query
    ),
    trigram_results AS (
        SELECT 
            a.id,
            a.title,
            left(a.content, 200) as snippet,
            (similarity(a.title, p_query) + similarity(a.content, p_query)) / 2 as rank,
            'trigram' as type
        FROM articles a
        WHERE similarity(a.title, p_query) > 0.3
           OR similarity(a.content, p_query) > 0.2
    ),
    combined_results AS (
        SELECT * FROM fulltext_results
        UNION ALL
        SELECT * FROM trigram_results
    )
    SELECT DISTINCT ON (cr.id)
        cr.id,
        cr.title,
        cr.snippet,
        cr.rank,
        cr.type
    FROM combined_results cr
    ORDER BY cr.id, cr.rank DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

### 2. Faceted Search

```sql
-- Search with filters and facets
CREATE TABLE products_search (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    category_id UUID NOT NULL,
    brand_id UUID NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    tags TEXT[],
    attributes JSONB DEFAULT '{}',
    
    -- Search vector including tags and attributes
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(array_to_string(tags, ' '), '')), 'C')
    ) STORED
);

-- Indexes for faceted search
CREATE INDEX idx_products_search_vector ON products_search USING gin(search_vector);
CREATE INDEX idx_products_category ON products_search(category_id);
CREATE INDEX idx_products_brand ON products_search(brand_id);
CREATE INDEX idx_products_price ON products_search(price);
CREATE INDEX idx_products_tags ON products_search USING gin(tags);
CREATE INDEX idx_products_attributes ON products_search USING gin(attributes);

-- Faceted search function
CREATE OR REPLACE FUNCTION faceted_search(
    p_query TEXT DEFAULT NULL,
    p_category_ids UUID[] DEFAULT NULL,
    p_brand_ids UUID[] DEFAULT NULL,
    p_min_price DECIMAL DEFAULT NULL,
    p_max_price DECIMAL DEFAULT NULL,
    p_tags TEXT[] DEFAULT NULL,
    p_attributes JSONB DEFAULT NULL,
    p_limit INTEGER DEFAULT 20,
    p_offset INTEGER DEFAULT 0
) RETURNS TABLE(
    id UUID,
    name TEXT,
    description TEXT,
    price DECIMAL,
    rank REAL,
    total_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    WITH filtered_products AS (
        SELECT 
            ps.id,
            ps.name,
            ps.description,
            ps.price,
            CASE 
                WHEN p_query IS NOT NULL 
                THEN ts_rank(ps.search_vector, plainto_tsquery('english', p_query))
                ELSE 1.0
            END as search_rank
        FROM products_search ps
        WHERE 1=1
            AND (p_query IS NULL OR ps.search_vector @@ plainto_tsquery('english', p_query))
            AND (p_category_ids IS NULL OR ps.category_id = ANY(p_category_ids))
            AND (p_brand_ids IS NULL OR ps.brand_id = ANY(p_brand_ids))
            AND (p_min_price IS NULL OR ps.price >= p_min_price)
            AND (p_max_price IS NULL OR ps.price <= p_max_price)
            AND (p_tags IS NULL OR ps.tags && p_tags)
            AND (p_attributes IS NULL OR ps.attributes @> p_attributes)
    ),
    counted_results AS (
        SELECT *, COUNT(*) OVER() as total_count
        FROM filtered_products
        ORDER BY search_rank DESC, name
        LIMIT p_limit OFFSET p_offset
    )
    SELECT 
        cr.id,
        cr.name,
        cr.description,
        cr.price,
        cr.search_rank,
        cr.total_count
    FROM counted_results cr;
END;
$$ LANGUAGE plpgsql;
```

### 3. Geographic Search

```sql
-- Location-based search with PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    address TEXT,
    location GEOMETRY(POINT, 4326),
    category TEXT,
    
    -- Text search vector
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(address, '')), 'C')
    ) STORED
);

-- Spatial and text indexes
CREATE INDEX idx_locations_search ON locations USING gin(search_vector);
CREATE INDEX idx_locations_spatial ON locations USING gist(location);
CREATE INDEX idx_locations_category ON locations(category);

-- Geographic search function
CREATE OR REPLACE FUNCTION search_nearby_locations(
    p_query TEXT DEFAULT NULL,
    p_latitude DECIMAL DEFAULT NULL,
    p_longitude DECIMAL DEFAULT NULL,
    p_radius_km DECIMAL DEFAULT 10,
    p_category TEXT DEFAULT NULL,
    p_limit INTEGER DEFAULT 20
) RETURNS TABLE(
    id UUID,
    name TEXT,
    address TEXT,
    category TEXT,
    distance_km DECIMAL,
    search_rank REAL
) AS $$
DECLARE
    search_point GEOMETRY;
BEGIN
    -- Create search point if coordinates provided
    IF p_latitude IS NOT NULL AND p_longitude IS NOT NULL THEN
        search_point := ST_SetSRID(ST_MakePoint(p_longitude, p_latitude), 4326);
    END IF;
    
    RETURN QUERY
    SELECT 
        l.id,
        l.name,
        l.address,
        l.category,
        CASE 
            WHEN search_point IS NOT NULL 
            THEN ROUND(ST_Distance(l.location, search_point) * 111.32, 2) -- Convert to km
            ELSE NULL 
        END as distance_km,
        CASE 
            WHEN p_query IS NOT NULL 
            THEN ts_rank(l.search_vector, plainto_tsquery('english', p_query))
            ELSE 1.0
        END as search_rank
    FROM locations l
    WHERE 1=1
        AND (p_query IS NULL OR l.search_vector @@ plainto_tsquery('english', p_query))
        AND (p_category IS NULL OR l.category = p_category)
        AND (search_point IS NULL OR ST_DWithin(l.location, search_point, p_radius_km / 111.32))
    ORDER BY 
        CASE WHEN search_point IS NOT NULL THEN ST_Distance(l.location, search_point) END,
        search_rank DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Search Performance Optimization

### 1. Search Index Management

```sql
-- Monitor search index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE indexname LIKE '%search%' OR indexname LIKE '%gin%'
ORDER BY idx_scan DESC;

-- Optimize tsvector storage and updates
-- Use trigger-based updates for better control
CREATE OR REPLACE FUNCTION update_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.content, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.tags::TEXT, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_search_vector
    BEFORE INSERT OR UPDATE ON articles
    FOR EACH ROW
    EXECUTE FUNCTION update_search_vector();
```

### 2. Search Result Caching

```sql
-- Cache popular search results
CREATE TABLE search_cache (
    query_hash CHAR(64) PRIMARY KEY,
    query_text TEXT NOT NULL,
    filters JSONB DEFAULT '{}',
    results JSONB NOT NULL,
    result_count INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '1 hour'
);

CREATE INDEX idx_search_cache_expires ON search_cache(expires_at);

-- Function to get or cache search results
CREATE OR REPLACE FUNCTION cached_search(
    p_query TEXT,
    p_filters JSONB DEFAULT '{}',
    p_limit INTEGER DEFAULT 20
) RETURNS JSONB AS $$
DECLARE
    query_hash TEXT;
    cached_result JSONB;
    fresh_results JSONB;
BEGIN
    -- Generate cache key
    query_hash := encode(sha256((p_query || p_filters::TEXT)::BYTEA), 'hex');
    
    -- Check cache
    SELECT results INTO cached_result
    FROM search_cache
    WHERE query_hash = query_hash
    AND expires_at > NOW();
    
    IF cached_result IS NOT NULL THEN
        RETURN cached_result;
    END IF;
    
    -- Generate fresh results (simplified example)
    SELECT jsonb_agg(
        jsonb_build_object(
            'id', id,
            'title', title,
            'rank', rank
        )
    ) INTO fresh_results
    FROM (
        SELECT 
            a.id,
            a.title,
            ts_rank(a.search_vector, plainto_tsquery('english', p_query)) as rank
        FROM articles a
        WHERE a.search_vector @@ plainto_tsquery('english', p_query)
        ORDER BY rank DESC
        LIMIT p_limit
    ) search_results;
    
    -- Cache results
    INSERT INTO search_cache (query_hash, query_text, filters, results, result_count)
    VALUES (
        query_hash, 
        p_query, 
        p_filters, 
        fresh_results, 
        jsonb_array_length(fresh_results)
    )
    ON CONFLICT (query_hash) DO UPDATE SET
        results = EXCLUDED.results,
        result_count = EXCLUDED.result_count,
        created_at = NOW(),
        expires_at = NOW() + INTERVAL '1 hour';
    
    RETURN fresh_results;
END;
$$ LANGUAGE plpgsql;
```

## 🎯 Search Analytics & Monitoring

### 1. Search Query Analytics

```sql
-- Track search queries and performance
CREATE TABLE search_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_text TEXT NOT NULL,
    user_id UUID,
    result_count INTEGER NOT NULL,
    execution_time_ms INTEGER,
    clicked_result_id UUID,
    search_filters JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_search_analytics_query ON search_analytics(query_text);
CREATE INDEX idx_search_analytics_created ON search_analytics(created_at);

-- Popular search queries
CREATE VIEW popular_searches AS
SELECT 
    query_text,
    COUNT(*) as search_count,
    AVG(result_count) as avg_results,
    AVG(execution_time_ms) as avg_execution_time,
    COUNT(clicked_result_id) * 100.0 / COUNT(*) as click_through_rate
FROM search_analytics
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY query_text
HAVING COUNT(*) >= 5
ORDER BY search_count DESC;

-- Zero-result searches (need attention)
CREATE VIEW zero_result_searches AS
SELECT 
    query_text,
    COUNT(*) as search_count,
    MAX(created_at) as last_searched
FROM search_analytics
WHERE result_count = 0
AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY query_text
HAVING COUNT(*) >= 3
ORDER BY search_count DESC;
```

## ⚠️ Best Practices

1. **Choose Right Search Type** - Full-text for content, trigram for fuzzy matching
2. **Use Appropriate Indexes** - GIN for tsvector, trigram indexes for similarity
3. **Optimize Vector Generation** - Use stored generated columns or triggers
4. **Cache Popular Searches** - Implement result caching for performance
5. **Monitor Search Performance** - Track query times and index usage
6. **Handle Multiple Languages** - Use appropriate text search configurations
7. **Combine Search Methods** - Hybrid approaches often work best
8. **Limit Result Sets** - Always paginate search results
9. **Track Analytics** - Monitor what users search for and optimize accordingly
10. **Handle Edge Cases** - Empty queries, special characters, very long queries

## 📊 Search Method Comparison

| Method | Best For | Performance | Accuracy | Complexity |
|--------|----------|-------------|----------|------------|
| **Full-Text Search** | Content search | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Trigram/Similarity** | Fuzzy matching | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ |
| **ILIKE Patterns** | Simple patterns | ⭐⭐ | ⭐⭐⭐ | ⭐ |
| **Regex Matching** | Complex patterns | ⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Hybrid Approach** | Best results | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

## 🔗 References

- [PostgreSQL Full-Text Search](https://www.postgresql.org/docs/current/textsearch.html)
- [pg_trgm Extension](https://www.postgresql.org/docs/current/pgtrgm.html)
- [PostGIS for Geographic Search](https://postgis.net/)
- [Search Optimization Guide](https://use-the-index-luke.com/sql/where-clause/searching)
