# Data Presentation Layer in Database Design

Strategies for implementing presentation-specific data storage that separates UI concerns from core business logic while maintaining clean architecture principles.

## Overview

In clean architecture, there's typically a separation between presentation layer, domain layer, and repository layer. This guide explores how to extend this separation into the database layer itself through presentation-specific tables and materialized views.

## Presentation vs Domain Data Separation

### Why Separate Presentation Data?

Presentation data often includes:
- **UI-specific computed values** - Machine learning rankings, search scores
- **Display toggles and flags** - Show/hide controls, highlighting flags  
- **Expensive computed aggregations** - Sorting scores, recommendation weights
- **Feature-specific metadata** - A/B testing flags, personalization data

### Problems with Mixed Storage

Storing presentation data in domain tables creates several issues:

```sql
-- Anti-pattern: Mixing domain and presentation concerns
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,        -- Domain data
    price DECIMAL(10,2) NOT NULL,      -- Domain data
    category_id INT NOT NULL,          -- Domain data
    
    -- Presentation-specific columns (problematic)
    ui_show_flag BOOLEAN DEFAULT true,
    ui_highlight_flag BOOLEAN DEFAULT false,
    ml_ranking_score DECIMAL(8,4),
    search_boost_factor DECIMAL(4,2),
    last_ui_update TIMESTAMP
);
```

**Issues:**
- **Schema bloat** - Core tables become cluttered with UI-specific columns
- **Migration complexity** - UI changes require core table migrations
- **Performance impact** - Extra columns increase row size and query cost
- **Coupling** - UI changes affect core business logic tables
- **Null pollution** - Older records have NULL values for new UI columns

## Presentation Layer Database Patterns

### 1. Presentation Extension Tables

Create separate tables that extend domain entities with presentation-specific data.

```sql
-- Core domain table (clean)
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    category_id INT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Presentation extension table
CREATE TABLE product_presentation (
    product_id INT PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
    
    -- UI display controls
    show_in_catalog BOOLEAN DEFAULT true,
    highlight_flag BOOLEAN DEFAULT false,
    featured_until TIMESTAMP NULL,
    
    -- Computed display values
    ml_ranking_score DECIMAL(8,4) DEFAULT 0,
    search_boost_factor DECIMAL(4,2) DEFAULT 1.0,
    popularity_score INT DEFAULT 0,
    
    -- UI-specific metadata
    custom_sort_order INT,
    promotion_badge VARCHAR(50),
    ui_theme VARCHAR(20) DEFAULT 'default',
    
    -- Tracking
    last_ml_update TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient presentation queries
CREATE INDEX idx_product_presentation_catalog 
ON product_presentation (show_in_catalog, ml_ranking_score DESC);

CREATE INDEX idx_product_presentation_featured 
ON product_presentation (featured_until) 
WHERE featured_until IS NOT NULL;
```

### 2. Materialized Views for Computed Presentation Data

Use materialized views for expensive computations that drive UI display.

```sql
-- Materialized view for product catalog display
CREATE MATERIALIZED VIEW product_catalog_view AS
SELECT 
    p.id,
    p.name,
    p.price,
    p.category_id,
    c.name as category_name,
    
    -- Computed display scores
    COALESCE(pp.ml_ranking_score, 0) * 
    COALESCE(pp.search_boost_factor, 1.0) as display_score,
    
    -- Availability computed logic
    CASE 
        WHEN p.status = 'active' 
             AND COALESCE(pp.show_in_catalog, true) = true
             AND i.quantity > 0 
        THEN true
        ELSE false
    END as available_for_display,
    
    -- Featured status
    CASE 
        WHEN pp.featured_until > CURRENT_TIMESTAMP 
        THEN true 
        ELSE false 
    END as is_featured,
    
    -- Computed sorting helpers
    CASE WHEN i.quantity = 0 THEN 1 ELSE 0 END as out_of_stock_sort,
    p.created_at as newness_sort,
    p.price as price_sort,
    
    -- UI metadata
    COALESCE(pp.promotion_badge, '') as badge,
    COALESCE(pp.ui_theme, 'default') as theme
    
FROM products p
LEFT JOIN product_presentation pp ON p.id = pp.product_id
LEFT JOIN categories c ON p.category_id = c.id  
LEFT JOIN inventory i ON p.id = i.product_id
WHERE p.status = 'active';

-- Indexes on materialized view
CREATE INDEX idx_product_catalog_display_score 
ON product_catalog_view (available_for_display, display_score DESC);

CREATE INDEX idx_product_catalog_category 
ON product_catalog_view (category_id, display_score DESC);

CREATE INDEX idx_product_catalog_featured 
ON product_catalog_view (is_featured, display_score DESC);
```

### 3. Event-Driven Presentation Updates

Keep presentation data synchronized through event-driven updates.

```sql
-- Function to update presentation scores
CREATE OR REPLACE FUNCTION update_product_presentation_scores()
RETURNS TRIGGER AS $$
BEGIN
    -- Update ML ranking when product data changes
    INSERT INTO product_presentation (product_id, last_ml_update)
    VALUES (NEW.id, CURRENT_TIMESTAMP)
    ON CONFLICT (product_id) 
    DO UPDATE SET last_ml_update = CURRENT_TIMESTAMP;
    
    -- Trigger materialized view refresh
    -- (In practice, this would be handled by application logic)
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on product changes
CREATE TRIGGER trigger_product_presentation_update
    AFTER INSERT OR UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_product_presentation_scores();
```

## Advanced Presentation Patterns

### 1. User-Specific Presentation Data

Store personalized presentation data per user.

```sql
-- User-specific product presentation
CREATE TABLE user_product_presentation (
    user_id INT NOT NULL REFERENCES users(id),
    product_id INT NOT NULL REFERENCES products(id),
    
    -- Personalized display data
    personal_ranking_score DECIMAL(8,4),
    recommendation_weight DECIMAL(6,3),
    last_viewed TIMESTAMP,
    view_count INT DEFAULT 0,
    is_favorited BOOLEAN DEFAULT false,
    is_hidden BOOLEAN DEFAULT false,
    
    -- A/B testing
    experiment_variant VARCHAR(50),
    experiment_exposure_time TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (user_id, product_id)
);

-- Efficient queries for personalized catalogs
CREATE INDEX idx_user_product_presentation_ranking 
ON user_product_presentation (user_id, personal_ranking_score DESC);

CREATE INDEX idx_user_product_presentation_recent 
ON user_product_presentation (user_id, last_viewed DESC);
```

### 2. Feature Toggle Management

Manage feature-specific presentation data.

```sql
-- Feature toggles for products
CREATE TABLE product_feature_toggles (
    product_id INT NOT NULL REFERENCES products(id),
    feature_name VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN DEFAULT false,
    
    -- Feature-specific metadata
    feature_config JSONB,
    
    -- Scheduling
    enabled_from TIMESTAMP,
    enabled_until TIMESTAMP,
    
    -- Tracking
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (product_id, feature_name)
);

-- Query products with specific features enabled
CREATE INDEX idx_product_feature_toggles_enabled
ON product_feature_toggles (feature_name, is_enabled, enabled_from, enabled_until);

-- Example: Get products with recommendation feature enabled
SELECT p.*, pft.feature_config
FROM products p
JOIN product_feature_toggles pft ON p.id = pft.product_id
WHERE pft.feature_name = 'ml_recommendations'
AND pft.is_enabled = true
AND (pft.enabled_from IS NULL OR pft.enabled_from <= CURRENT_TIMESTAMP)
AND (pft.enabled_until IS NULL OR pft.enabled_until > CURRENT_TIMESTAMP);
```

### 3. Search-Optimized Presentation Tables

Create search-optimized tables for text search and filtering.

```sql
-- Search-optimized product presentation
CREATE TABLE product_search_presentation (
    product_id INT PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
    
    -- Searchable text (pre-processed)
    search_text TEXT NOT NULL,
    search_vector tsvector,
    
    -- Faceting data
    facet_brand VARCHAR(100),
    facet_tags TEXT[],
    facet_price_range VARCHAR(20),
    facet_availability VARCHAR(20),
    
    -- Search ranking factors
    base_search_score DECIMAL(8,4) DEFAULT 1.0,
    popularity_factor DECIMAL(6,3) DEFAULT 1.0,
    recency_factor DECIMAL(6,3) DEFAULT 1.0,
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Full-text search index
CREATE INDEX idx_product_search_vector 
ON product_search_presentation USING gin(search_vector);

-- Faceting indexes
CREATE INDEX idx_product_search_brand 
ON product_search_presentation (facet_brand);

CREATE INDEX idx_product_search_tags 
ON product_search_presentation USING gin(facet_tags);

-- Function to maintain search data
CREATE OR REPLACE FUNCTION update_product_search_data()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO product_search_presentation (
        product_id, 
        search_text, 
        search_vector,
        facet_brand,
        facet_price_range
    )
    VALUES (
        NEW.id,
        NEW.name || ' ' || COALESCE(NEW.description, ''),
        to_tsvector('english', NEW.name || ' ' || COALESCE(NEW.description, '')),
        -- Extract brand from product name or use category
        SPLIT_PART(NEW.name, ' ', 1),
        CASE 
            WHEN NEW.price < 25 THEN 'under-25'
            WHEN NEW.price < 100 THEN '25-100'
            WHEN NEW.price < 500 THEN '100-500'
            ELSE 'over-500'
        END
    )
    ON CONFLICT (product_id) DO UPDATE SET
        search_text = EXCLUDED.search_text,
        search_vector = EXCLUDED.search_vector,
        facet_brand = EXCLUDED.facet_brand,
        facet_price_range = EXCLUDED.facet_price_range,
        updated_at = CURRENT_TIMESTAMP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

## Implementation Strategies

### 1. Application Layer Integration

```python
# Python example: Service layer handling presentation data
class ProductCatalogService:
    def __init__(self, product_repo, presentation_repo):
        self.product_repo = product_repo
        self.presentation_repo = presentation_repo
    
    def get_catalog_page(self, user_id, page=1, sort_by='relevance'):
        # Get base product data
        products = self.product_repo.get_active_products(page=page)
        
        # Enhance with presentation data
        for product in products:
            # Get general presentation data
            presentation = self.presentation_repo.get_product_presentation(
                product.id
            )
            product.display_score = presentation.ml_ranking_score
            product.is_featured = presentation.featured_until > datetime.now()
            
            # Get user-specific presentation data
            user_presentation = self.presentation_repo.get_user_product_presentation(
                user_id, product.id
            )
            if user_presentation:
                product.personal_score = user_presentation.personal_ranking_score
                product.is_favorited = user_presentation.is_favorited
        
        # Sort by presentation criteria
        if sort_by == 'relevance':
            products.sort(key=lambda p: p.display_score, reverse=True)
        elif sort_by == 'personal':
            products.sort(key=lambda p: getattr(p, 'personal_score', 0), reverse=True)
        
        return products
```

### 2. Batch Updates for ML Scores

```sql
-- Stored procedure for batch ML score updates
CREATE OR REPLACE FUNCTION update_ml_ranking_scores(score_data JSONB)
RETURNS void AS $$
DECLARE
    score_record RECORD;
BEGIN
    -- Expect score_data as: [{"product_id": 123, "score": 0.85}, ...]
    FOR score_record IN 
        SELECT (value->>'product_id')::INT as product_id,
               (value->>'score')::DECIMAL as score
        FROM jsonb_array_elements(score_data)
    LOOP
        INSERT INTO product_presentation (product_id, ml_ranking_score, last_ml_update)
        VALUES (score_record.product_id, score_record.score, CURRENT_TIMESTAMP)
        ON CONFLICT (product_id) DO UPDATE SET
            ml_ranking_score = EXCLUDED.ml_ranking_score,
            last_ml_update = EXCLUDED.last_ml_update;
    END LOOP;
    
    -- Refresh materialized view
    REFRESH MATERIALIZED VIEW CONCURRENTLY product_catalog_view;
END;
$$ LANGUAGE plpgsql;

-- Usage example
SELECT update_ml_ranking_scores('[
    {"product_id": 123, "score": 0.85},
    {"product_id": 124, "score": 0.92},
    {"product_id": 125, "score": 0.78}
]'::JSONB);
```

## Alternative Storage Solutions

### 1. Separate Database for Presentation Data

For high-scale applications, consider dedicated presentation databases.

```yaml
# Example architecture
Primary Database (PostgreSQL):
  - Core business entities
  - Transactional data
  - ACID compliance critical

Presentation Database (e.g., MongoDB, Cassandra):
  - Computed display data
  - User personalization
  - Search indexes
  - Feature flags
```

### 2. Cache Layer Integration

```python
# Example: Redis cache for presentation data
class PresentationCacheService:
    def __init__(self, redis_client):
        self.redis = redis_client
    
    def get_product_display_data(self, product_id, user_id=None):
        cache_key = f"product:display:{product_id}"
        if user_id:
            cache_key += f":user:{user_id}"
        
        cached_data = self.redis.get(cache_key)
        if cached_data:
            return json.loads(cached_data)
        
        # Fetch from database and cache
        display_data = self._fetch_from_db(product_id, user_id)
        self.redis.setex(
            cache_key, 
            3600,  # 1 hour TTL
            json.dumps(display_data)
        )
        return display_data
```

## Performance Considerations

### 1. Materialized View Refresh Strategies

```sql
-- Incremental refresh for large catalogs
CREATE OR REPLACE FUNCTION refresh_catalog_incremental()
RETURNS void AS $$
BEGIN
    -- Only refresh products updated in last hour
    DELETE FROM product_catalog_view 
    WHERE id IN (
        SELECT p.id FROM products p 
        WHERE p.updated_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    );
    
    INSERT INTO product_catalog_view 
    SELECT * FROM product_catalog_view_source 
    WHERE id IN (
        SELECT p.id FROM products p 
        WHERE p.updated_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    );
END;
$$ LANGUAGE plpgsql;
```

### 2. Partitioning for User-Specific Data

```sql
-- Partition user presentation data by user ID hash
CREATE TABLE user_product_presentation (
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    -- ... other columns
) PARTITION BY HASH (user_id);

-- Create partitions
CREATE TABLE user_product_presentation_0 PARTITION OF user_product_presentation
FOR VALUES WITH (modulus 4, remainder 0);

CREATE TABLE user_product_presentation_1 PARTITION OF user_product_presentation
FOR VALUES WITH (modulus 4, remainder 1);

-- ... more partitions
```

## Best Practices

### 1. Design Principles
- **Separation of Concerns** - Keep domain and presentation data separate
- **Eventual Consistency** - Accept some lag in presentation data updates
- **Fail Gracefully** - Handle missing presentation data gracefully
- **Cache Aggressively** - Presentation data is ideal for caching

### 2. Maintenance
- **Regular Cleanup** - Remove stale presentation data
- **Monitor Performance** - Track materialized view refresh times
- **Version Schema** - Use migrations for presentation schema changes
- **Test Thoroughly** - Ensure presentation updates don't break core functionality

### 3. Scaling Considerations
- **Read Replicas** - Use separate replicas for presentation queries
- **Horizontal Scaling** - Consider sharding user-specific presentation data
- **CDN Integration** - Cache computed presentation data at the edge
- **Async Processing** - Update presentation data asynchronously

## Related Topics

- [Materialized Views](../query-patterns/materialized-views.md) - Advanced view patterns
- [Caching Strategies](../performance/caching.md) - Data caching approaches
- [Search Optimization](../query-patterns/text-search.md) - Full-text search implementation
- [Personalization Patterns](../schema-design/personalization.md) - User-specific data design
- [Feature Flags](../schema-design/feature-flags.md) - Feature toggle implementation
