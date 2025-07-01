# Ranking and Scoring Database Design

A comprehensive guide to implementing ranking systems, scoring algorithms, and leaderboards in database applications.

## Table of Contents

1. [Overview](#overview)
2. [Core Ranking Concepts](#core-ranking-concepts)
3. [Ranking Functions](#ranking-functions)
4. [Scoring Systems](#scoring-systems)
5. [Real-World Examples](#real-world-examples)
6. [Advanced Ranking Patterns](#advanced-ranking-patterns)
7. [Performance Optimization](#performance-optimization)
8. [Leaderboard Implementation](#leaderboard-implementation)
9. [Time-Based Rankings](#time-based-rankings)
10. [Best Practices](#best-practices)

## Overview

Ranking systems are essential for applications that need to order, score, and compare entities based on various criteria. This pattern is crucial for search results, product listings, user leaderboards, content recommendations, and competitive applications.

### Key Use Cases
- **Product Rankings**: E-commerce product sorting by popularity, rating, price
- **Content Rankings**: Blog posts, articles, social media content
- **User Leaderboards**: Gaming scores, activity rankings, reputation systems
- **Search Results**: Relevance scoring and result ordering
- **Recommendation Systems**: Content and product recommendations

## Core Ranking Concepts

### Basic Ranking Schema

```sql
-- Generic ranking table for different entities
CREATE TABLE entity_rankings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL, -- 'product', 'user', 'post', etc.
    entity_id UUID NOT NULL,
    
    -- Ranking metrics
    total_score DECIMAL(10,4) NOT NULL DEFAULT 0,
    rank_position INTEGER,
    rank_percentile DECIMAL(5,4),
    
    -- Component scores
    quality_score DECIMAL(8,4) DEFAULT 0,
    popularity_score DECIMAL(8,4) DEFAULT 0,
    recency_score DECIMAL(8,4) DEFAULT 0,
    engagement_score DECIMAL(8,4) DEFAULT 0,
    
    -- Ranking metadata
    ranking_category VARCHAR(100),
    ranking_date DATE NOT NULL DEFAULT CURRENT_DATE,
    calculation_method VARCHAR(50) DEFAULT 'weighted_sum',
    
    -- Audit trail
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(entity_type, entity_id, ranking_category, ranking_date),
    INDEX idx_entity_rankings_type_rank (entity_type, ranking_category, rank_position),
    INDEX idx_entity_rankings_score (entity_type, total_score DESC),
    INDEX idx_entity_rankings_date (ranking_date, entity_type)
);

-- Ranking weights configuration
CREATE TABLE ranking_weights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    ranking_category VARCHAR(100) NOT NULL,
    
    -- Weight configuration
    quality_weight DECIMAL(4,3) DEFAULT 0,
    popularity_weight DECIMAL(4,3) DEFAULT 0,
    recency_weight DECIMAL(4,3) DEFAULT 0,
    engagement_weight DECIMAL(4,3) DEFAULT 0,
    
    -- Metadata
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(entity_type, ranking_category),
    CONSTRAINT weights_sum_valid CHECK (
        quality_weight + popularity_weight + recency_weight + engagement_weight = 1.0
    )
);

-- Historical ranking data for trend analysis
CREATE TABLE ranking_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    ranking_category VARCHAR(100),
    
    -- Historical data
    rank_position INTEGER NOT NULL,
    total_score DECIMAL(10,4) NOT NULL,
    rank_change INTEGER, -- Change from previous period
    
    -- Time period
    period_date DATE NOT NULL,
    period_type VARCHAR(20) DEFAULT 'daily', -- daily, weekly, monthly
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_ranking_history_entity (entity_type, entity_id, period_date),
    INDEX idx_ranking_history_period (period_date, entity_type, ranking_category)
);
```

## Ranking Functions

### PostgreSQL Window Functions for Ranking

```sql
-- Comprehensive ranking function using multiple criteria
CREATE OR REPLACE FUNCTION calculate_product_rankings(
    p_category VARCHAR DEFAULT NULL,
    p_date DATE DEFAULT CURRENT_DATE
) RETURNS TABLE(
    product_id UUID,
    product_name VARCHAR,
    total_score DECIMAL,
    quality_rank INTEGER,
    popularity_rank INTEGER,
    overall_rank INTEGER,
    rank_change INTEGER
) AS $$
BEGIN
    RETURN QUERY
    WITH product_metrics AS (
        SELECT 
            p.id,
            p.name,
            
            -- Quality metrics
            COALESCE(AVG(r.rating), 0) as avg_rating,
            COUNT(r.id) as review_count,
            
            -- Popularity metrics
            COALESCE(SUM(CASE WHEN s.created_at >= p_date - INTERVAL '30 days' THEN 1 ELSE 0 END), 0) as recent_sales,
            COALESCE(SUM(CASE WHEN pv.viewed_at >= p_date - INTERVAL '7 days' THEN 1 ELSE 0 END), 0) as recent_views,
            
            -- Recency score
            EXTRACT(EPOCH FROM (p_date - p.created_at)) / 86400.0 as days_old,
            
            -- Engagement metrics
            COALESCE(COUNT(DISTINCT c.user_id), 0) as unique_commenters
            
        FROM products p
        LEFT JOIN reviews r ON r.product_id = p.id AND r.deleted_at IS NULL
        LEFT JOIN sales s ON s.product_id = p.id
        LEFT JOIN product_views pv ON pv.product_id = p.id
        LEFT JOIN comments c ON c.product_id = p.id AND c.created_at >= p_date - INTERVAL '30 days'
        WHERE (p_category IS NULL OR p.category = p_category)
          AND p.deleted_at IS NULL
        GROUP BY p.id, p.name, p.created_at
    ),
    scored_products AS (
        SELECT 
            pm.*,
            
            -- Normalized scores (0-1 range)
            CASE 
                WHEN MAX(pm.avg_rating) OVER() > 0 
                THEN pm.avg_rating / MAX(pm.avg_rating) OVER() 
                ELSE 0 
            END as quality_score,
            
            CASE 
                WHEN MAX(pm.recent_sales + pm.recent_views) OVER() > 0 
                THEN (pm.recent_sales + pm.recent_views) / MAX(pm.recent_sales + pm.recent_views) OVER() 
                ELSE 0 
            END as popularity_score,
            
            -- Recency score (newer products get higher scores)
            CASE 
                WHEN MAX(pm.days_old) OVER() > 0 
                THEN 1.0 - (pm.days_old / MAX(pm.days_old) OVER()) 
                ELSE 1.0 
            END as recency_score,
            
            CASE 
                WHEN MAX(pm.unique_commenters) OVER() > 0 
                THEN pm.unique_commenters / MAX(pm.unique_commenters) OVER() 
                ELSE 0 
            END as engagement_score
            
        FROM product_metrics pm
    ),
    weighted_scores AS (
        SELECT 
            sp.*,
            
            -- Calculate weighted total score
            (sp.quality_score * 0.4 + 
             sp.popularity_score * 0.3 + 
             sp.recency_score * 0.2 + 
             sp.engagement_score * 0.1) as total_score
             
        FROM scored_products sp
    ),
    ranked_products AS (
        SELECT 
            ws.*,
            
            -- Different ranking methods
            DENSE_RANK() OVER (ORDER BY ws.quality_score DESC) as quality_rank,
            DENSE_RANK() OVER (ORDER BY ws.popularity_score DESC) as popularity_rank,
            DENSE_RANK() OVER (ORDER BY ws.total_score DESC) as overall_rank,
            ROW_NUMBER() OVER (ORDER BY ws.total_score DESC) as row_rank
            
        FROM weighted_scores ws
    )
    SELECT 
        rp.id,
        rp.name,
        ROUND(rp.total_score, 4),
        rp.quality_rank::INTEGER,
        rp.popularity_rank::INTEGER,
        rp.overall_rank::INTEGER,
        
        -- Calculate rank change from previous period
        COALESCE(
            rp.overall_rank - LAG(rp.overall_rank) OVER (
                PARTITION BY rp.id 
                ORDER BY p_date
            ), 
            0
        )::INTEGER as rank_change
        
    FROM ranked_products rp
    ORDER BY rp.overall_rank;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate Wilson Score for better rating rankings
CREATE OR REPLACE FUNCTION wilson_score(positive INTEGER, total INTEGER, confidence DECIMAL DEFAULT 0.95)
RETURNS DECIMAL AS $$
DECLARE
    z DECIMAL;
    phat DECIMAL;
    result DECIMAL;
BEGIN
    IF total = 0 THEN
        RETURN 0;
    END IF;
    
    -- Z-score for 95% confidence interval
    z := CASE 
        WHEN confidence = 0.95 THEN 1.96
        WHEN confidence = 0.90 THEN 1.645
        WHEN confidence = 0.99 THEN 2.576
        ELSE 1.96
    END;
    
    phat := positive::DECIMAL / total::DECIMAL;
    
    result := (phat + z*z/(2*total) - z * sqrt((phat*(1-phat)+z*z/(4*total))/total))/(1+z*z/total);
    
    RETURN GREATEST(0, result);
END;
$$ LANGUAGE plpgsql;

-- Advanced ranking with Wilson Score
CREATE OR REPLACE FUNCTION calculate_wilson_rankings()
RETURNS TABLE(
    product_id UUID,
    product_name VARCHAR,
    wilson_score DECIMAL,
    rank_position INTEGER,
    total_reviews INTEGER,
    avg_rating DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    WITH product_ratings AS (
        SELECT 
            p.id,
            p.name,
            COUNT(r.id) as total_reviews,
            AVG(r.rating) as avg_rating,
            
            -- Convert 5-star ratings to positive/negative
            SUM(CASE WHEN r.rating >= 4 THEN 1 ELSE 0 END) as positive_reviews,
            COUNT(r.id) as total_reviews_count
            
        FROM products p
        LEFT JOIN reviews r ON r.product_id = p.id AND r.deleted_at IS NULL
        WHERE p.deleted_at IS NULL
        GROUP BY p.id, p.name
        HAVING COUNT(r.id) >= 5 -- Minimum reviews threshold
    )
    SELECT 
        pr.id,
        pr.name,
        wilson_score(pr.positive_reviews, pr.total_reviews_count)::DECIMAL(8,6),
        DENSE_RANK() OVER (ORDER BY wilson_score(pr.positive_reviews, pr.total_reviews_count) DESC)::INTEGER,
        pr.total_reviews_count::INTEGER,
        ROUND(pr.avg_rating, 2)::DECIMAL(3,2)
    FROM product_ratings pr
    ORDER BY wilson_score(pr.positive_reviews, pr.total_reviews_count) DESC;
END;
$$ LANGUAGE plpgsql;
```

## Scoring Systems

### Multi-Criteria Decision Analysis (MCDA)

```sql
-- MCDA scoring system for complex rankings
CREATE TABLE scoring_criteria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    criteria_name VARCHAR(100) NOT NULL,
    criteria_description TEXT,
    
    -- Scoring configuration
    weight DECIMAL(5,4) NOT NULL,
    min_value DECIMAL(12,4) DEFAULT 0,
    max_value DECIMAL(12,4) DEFAULT 100,
    higher_is_better BOOLEAN DEFAULT TRUE,
    
    -- Scoring function
    scoring_function VARCHAR(50) DEFAULT 'linear', -- linear, logarithmic, exponential
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(entity_type, criteria_name)
);

-- Entity scores for each criteria
CREATE TABLE entity_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    criteria_id UUID NOT NULL REFERENCES scoring_criteria(id),
    
    -- Score data
    raw_value DECIMAL(12,4) NOT NULL,
    normalized_score DECIMAL(8,6) NOT NULL, -- 0-1 range
    weighted_score DECIMAL(8,6) NOT NULL,
    
    -- Metadata
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    calculation_method VARCHAR(100),
    
    UNIQUE(entity_type, entity_id, criteria_id),
    INDEX idx_entity_scores_entity (entity_type, entity_id),
    INDEX idx_entity_scores_criteria (criteria_id, normalized_score DESC)
);

-- Function to calculate MCDA scores
CREATE OR REPLACE FUNCTION calculate_mcda_scores(
    p_entity_type VARCHAR,
    p_recalculate BOOLEAN DEFAULT FALSE
) RETURNS INTEGER AS $$
DECLARE
    v_criteria RECORD;
    v_entity RECORD;
    v_min_val DECIMAL;
    v_max_val DECIMAL;
    v_normalized_score DECIMAL;
    v_processed_count INTEGER := 0;
BEGIN
    -- Process each criteria
    FOR v_criteria IN 
        SELECT * FROM scoring_criteria 
        WHERE entity_type = p_entity_type AND is_active = TRUE
    LOOP
        -- Get min/max values for normalization
        EXECUTE format('
            SELECT MIN(%I) as min_val, MAX(%I) as max_val 
            FROM %I 
            WHERE deleted_at IS NULL',
            v_criteria.criteria_name, v_criteria.criteria_name, p_entity_type || 's'
        ) INTO v_min_val, v_max_val;
        
        -- Update criteria bounds
        UPDATE scoring_criteria 
        SET min_value = v_min_val, max_value = v_max_val
        WHERE id = v_criteria.id;
        
        -- Calculate scores for all entities
        FOR v_entity IN 
            EXECUTE format('SELECT id, %I as raw_value FROM %I WHERE deleted_at IS NULL', 
                v_criteria.criteria_name, p_entity_type || 's')
        LOOP
            -- Normalize score (0-1 range)
            IF v_max_val > v_min_val THEN
                IF v_criteria.higher_is_better THEN
                    v_normalized_score := (v_entity.raw_value - v_min_val) / (v_max_val - v_min_val);
                ELSE
                    v_normalized_score := (v_max_val - v_entity.raw_value) / (v_max_val - v_min_val);
                END IF;
            ELSE
                v_normalized_score := 0.5; -- Default for identical values
            END IF;
            
            -- Apply scoring function
            v_normalized_score := CASE v_criteria.scoring_function
                WHEN 'logarithmic' THEN LOG(1 + v_normalized_score * 9) / LOG(10) -- log10(1 + 9x)
                WHEN 'exponential' THEN (EXP(v_normalized_score) - 1) / (EXP(1) - 1)
                ELSE v_normalized_score -- linear
            END;
            
            -- Store the score
            INSERT INTO entity_scores (
                entity_type, entity_id, criteria_id, 
                raw_value, normalized_score, weighted_score,
                calculation_method
            ) VALUES (
                p_entity_type, v_entity.id, v_criteria.id,
                v_entity.raw_value, v_normalized_score, 
                v_normalized_score * v_criteria.weight,
                'mcda_' || v_criteria.scoring_function
            )
            ON CONFLICT (entity_type, entity_id, criteria_id) 
            DO UPDATE SET
                raw_value = EXCLUDED.raw_value,
                normalized_score = EXCLUDED.normalized_score,
                weighted_score = EXCLUDED.weighted_score,
                calculated_at = NOW(),
                calculation_method = EXCLUDED.calculation_method;
                
            v_processed_count := v_processed_count + 1;
        END LOOP;
    END LOOP;
    
    RETURN v_processed_count;
END;
$$ LANGUAGE plpgsql;

-- View for final MCDA rankings
CREATE VIEW mcda_rankings AS
SELECT 
    es.entity_type,
    es.entity_id,
    SUM(es.weighted_score) as total_score,
    DENSE_RANK() OVER (
        PARTITION BY es.entity_type 
        ORDER BY SUM(es.weighted_score) DESC
    ) as rank_position,
    COUNT(es.criteria_id) as criteria_count,
    AVG(es.normalized_score) as avg_normalized_score,
    MAX(es.calculated_at) as last_calculated
FROM entity_scores es
JOIN scoring_criteria sc ON sc.id = es.criteria_id AND sc.is_active = TRUE
GROUP BY es.entity_type, es.entity_id
ORDER BY es.entity_type, total_score DESC;
```

## Real-World Examples

### 1. E-commerce Product Ranking

```sql
-- Product ranking for e-commerce
CREATE TABLE product_metrics (
    product_id UUID PRIMARY KEY REFERENCES products(id),
    
    -- Sales metrics
    total_sales INTEGER DEFAULT 0,
    sales_last_30d INTEGER DEFAULT 0,
    revenue_last_30d DECIMAL(12,2) DEFAULT 0,
    
    -- Review metrics
    total_reviews INTEGER DEFAULT 0,
    avg_rating DECIMAL(3,2) DEFAULT 0,
    reviews_last_30d INTEGER DEFAULT 0,
    
    -- Engagement metrics
    total_views INTEGER DEFAULT 0,
    views_last_7d INTEGER DEFAULT 0,
    wishlist_adds INTEGER DEFAULT 0,
    cart_additions INTEGER DEFAULT 0,
    
    -- Inventory and pricing
    current_stock INTEGER DEFAULT 0,
    price_competitiveness_score DECIMAL(4,2) DEFAULT 0,
    
    -- Derived metrics
    conversion_rate DECIMAL(5,4) DEFAULT 0,
    return_rate DECIMAL(5,4) DEFAULT 0,
    
    last_updated TIMESTAMPTZ DEFAULT NOW()
);

-- Function to update product metrics
CREATE OR REPLACE FUNCTION update_product_metrics(p_product_id UUID DEFAULT NULL)
RETURNS INTEGER AS $$
DECLARE
    v_product_id UUID;
    v_count INTEGER := 0;
BEGIN
    FOR v_product_id IN 
        SELECT id FROM products 
        WHERE (p_product_id IS NULL OR id = p_product_id)
          AND deleted_at IS NULL
    LOOP
        INSERT INTO product_metrics (product_id) 
        VALUES (v_product_id)
        ON CONFLICT (product_id) DO NOTHING;
        
        UPDATE product_metrics pm SET
            -- Sales metrics
            total_sales = (
                SELECT COUNT(*) FROM order_items oi 
                JOIN orders o ON o.id = oi.order_id 
                WHERE oi.product_id = v_product_id AND o.status = 'completed'
            ),
            sales_last_30d = (
                SELECT COUNT(*) FROM order_items oi 
                JOIN orders o ON o.id = oi.order_id 
                WHERE oi.product_id = v_product_id 
                  AND o.status = 'completed'
                  AND o.created_at >= NOW() - INTERVAL '30 days'
            ),
            revenue_last_30d = (
                SELECT COALESCE(SUM(oi.price * oi.quantity), 0) 
                FROM order_items oi 
                JOIN orders o ON o.id = oi.order_id 
                WHERE oi.product_id = v_product_id 
                  AND o.status = 'completed'
                  AND o.created_at >= NOW() - INTERVAL '30 days'
            ),
            
            -- Review metrics
            total_reviews = (
                SELECT COUNT(*) FROM reviews 
                WHERE product_id = v_product_id AND deleted_at IS NULL
            ),
            avg_rating = (
                SELECT COALESCE(AVG(rating), 0) FROM reviews 
                WHERE product_id = v_product_id AND deleted_at IS NULL
            ),
            reviews_last_30d = (
                SELECT COUNT(*) FROM reviews 
                WHERE product_id = v_product_id 
                  AND deleted_at IS NULL
                  AND created_at >= NOW() - INTERVAL '30 days'
            ),
            
            -- Engagement metrics
            total_views = (
                SELECT COUNT(*) FROM product_views 
                WHERE product_id = v_product_id
            ),
            views_last_7d = (
                SELECT COUNT(*) FROM product_views 
                WHERE product_id = v_product_id 
                  AND viewed_at >= NOW() - INTERVAL '7 days'
            ),
            
            -- Conversion rate
            conversion_rate = CASE 
                WHEN (SELECT COUNT(*) FROM product_views WHERE product_id = v_product_id) > 0
                THEN (
                    SELECT COUNT(DISTINCT oi.order_id)::DECIMAL 
                    FROM order_items oi 
                    JOIN orders o ON o.id = oi.order_id 
                    WHERE oi.product_id = v_product_id AND o.status = 'completed'
                ) / (
                    SELECT COUNT(DISTINCT session_id) FROM product_views 
                    WHERE product_id = v_product_id
                )
                ELSE 0
            END,
            
            last_updated = NOW()
        WHERE product_id = v_product_id;
        
        v_count := v_count + 1;
    END LOOP;
    
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- E-commerce ranking algorithm
CREATE OR REPLACE FUNCTION calculate_ecommerce_ranking()
RETURNS TABLE(
    product_id UUID,
    product_name VARCHAR,
    category VARCHAR,
    total_score DECIMAL,
    rank_position INTEGER,
    sales_score DECIMAL,
    rating_score DECIMAL,
    engagement_score DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    WITH product_scores AS (
        SELECT 
            p.id,
            p.name,
            p.category,
            pm.total_sales,
            pm.sales_last_30d,
            pm.avg_rating,
            pm.total_reviews,
            pm.views_last_7d,
            pm.conversion_rate,
            
            -- Wilson score for rating reliability
            wilson_score(
                (pm.avg_rating::INTEGER * pm.total_reviews)::INTEGER, 
                (5 * pm.total_reviews)::INTEGER
            ) as wilson_rating_score,
            
            -- Engagement score
            (pm.views_last_7d::DECIMAL * pm.conversion_rate * 100) as engagement_raw_score
            
        FROM products p
        JOIN product_metrics pm ON pm.product_id = p.id
        WHERE p.deleted_at IS NULL
    ),
    normalized_scores AS (
        SELECT 
            ps.*,
            
            -- Normalize scores to 0-1 range
            CASE 
                WHEN MAX(ps.sales_last_30d) OVER() > 0 
                THEN ps.sales_last_30d::DECIMAL / MAX(ps.sales_last_30d) OVER()
                ELSE 0 
            END as sales_norm_score,
            
            ps.wilson_rating_score as rating_norm_score,
            
            CASE 
                WHEN MAX(ps.engagement_raw_score) OVER() > 0 
                THEN ps.engagement_raw_score / MAX(ps.engagement_raw_score) OVER()
                ELSE 0 
            END as engagement_norm_score
            
        FROM product_scores ps
    ),
    final_scores AS (
        SELECT 
            ns.*,
            
            -- Weighted final score
            (ns.sales_norm_score * 0.4 + 
             ns.rating_norm_score * 0.35 + 
             ns.engagement_norm_score * 0.25) as final_score
             
        FROM normalized_scores ns
    )
    SELECT 
        fs.id,
        fs.name,
        fs.category,
        ROUND(fs.final_score, 4)::DECIMAL,
        DENSE_RANK() OVER (ORDER BY fs.final_score DESC)::INTEGER,
        ROUND(fs.sales_norm_score, 4)::DECIMAL,
        ROUND(fs.rating_norm_score, 4)::DECIMAL,
        ROUND(fs.engagement_norm_score, 4)::DECIMAL
    FROM final_scores fs
    ORDER BY fs.final_score DESC;
END;
$$ LANGUAGE plpgsql;
```

### 2. Content Ranking System

```sql
-- Content ranking for blogs, articles, social media
CREATE TABLE content_metrics (
    content_id UUID PRIMARY KEY,
    content_type VARCHAR(50) NOT NULL, -- 'article', 'post', 'video'
    
    -- Engagement metrics
    views_total INTEGER DEFAULT 0,
    views_last_24h INTEGER DEFAULT 0,
    views_last_7d INTEGER DEFAULT 0,
    unique_viewers INTEGER DEFAULT 0,
    
    -- Social metrics
    likes_count INTEGER DEFAULT 0,
    comments_count INTEGER DEFAULT 0,
    shares_count INTEGER DEFAULT 0,
    bookmarks_count INTEGER DEFAULT 0,
    
    -- Quality metrics
    read_completion_rate DECIMAL(5,4) DEFAULT 0,
    avg_time_spent_seconds INTEGER DEFAULT 0,
    bounce_rate DECIMAL(5,4) DEFAULT 0,
    
    -- Freshness
    published_at TIMESTAMPTZ,
    last_updated_at TIMESTAMPTZ,
    
    -- Derived scores
    viral_coefficient DECIMAL(8,4) DEFAULT 0,
    engagement_rate DECIMAL(6,4) DEFAULT 0,
    quality_score DECIMAL(6,4) DEFAULT 0,
    
    last_calculated TIMESTAMPTZ DEFAULT NOW()
);

-- Trending content algorithm
CREATE OR REPLACE FUNCTION calculate_trending_score(
    p_views_24h INTEGER,
    p_views_7d INTEGER,
    p_engagement_count INTEGER,
    p_age_hours INTEGER
) RETURNS DECIMAL AS $$
DECLARE
    v_time_decay DECIMAL;
    v_engagement_boost DECIMAL;
    v_trending_score DECIMAL;
BEGIN
    -- Time decay factor (content gets less trendy over time)
    v_time_decay := EXP(-p_age_hours::DECIMAL / 24.0); -- Exponential decay
    
    -- Engagement boost
    v_engagement_boost := 1.0 + (p_engagement_count::DECIMAL / 100.0);
    
    -- Calculate trending score
    v_trending_score := (
        p_views_24h::DECIMAL * 2.0 + -- Recent views weighted more
        p_views_7d::DECIMAL * 0.5
    ) * v_time_decay * v_engagement_boost;
    
    RETURN v_trending_score;
END;
$$ LANGUAGE plpgsql;

-- Content ranking with multiple algorithms
CREATE OR REPLACE FUNCTION rank_content(
    p_algorithm VARCHAR DEFAULT 'balanced',
    p_content_type VARCHAR DEFAULT NULL,
    p_limit INTEGER DEFAULT 100
) RETURNS TABLE(
    content_id UUID,
    title VARCHAR,
    author VARCHAR,
    score DECIMAL,
    rank_position INTEGER,
    algorithm_used VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    EXECUTE format('
    WITH content_data AS (
        SELECT 
            c.id,
            c.title,
            u.username as author,
            cm.views_total,
            cm.views_last_24h,
            cm.views_last_7d,
            cm.likes_count + cm.comments_count + cm.shares_count as total_engagement,
            cm.read_completion_rate,
            EXTRACT(EPOCH FROM (NOW() - c.created_at)) / 3600 as age_hours,
            c.created_at
        FROM content c
        JOIN users u ON u.id = c.author_id
        LEFT JOIN content_metrics cm ON cm.content_id = c.id
        WHERE c.deleted_at IS NULL
          AND c.status = ''published''
          AND ($1 IS NULL OR c.content_type = $1)
    ),
    scored_content AS (
        SELECT 
            cd.*,
            CASE $2
                WHEN ''trending'' THEN
                    calculate_trending_score(
                        cd.views_last_24h, 
                        cd.views_last_7d, 
                        cd.total_engagement, 
                        cd.age_hours::INTEGER
                    )
                WHEN ''quality'' THEN
                    cd.read_completion_rate * 100 + 
                    (cd.total_engagement::DECIMAL / GREATEST(cd.views_total, 1)) * 50
                WHEN ''popular'' THEN
                    cd.views_total::DECIMAL + cd.total_engagement * 2
                ELSE -- balanced
                    (calculate_trending_score(cd.views_last_24h, cd.views_last_7d, cd.total_engagement, cd.age_hours::INTEGER) * 0.4 +
                     (cd.read_completion_rate * 100) * 0.3 +
                     (cd.views_total::DECIMAL / 1000.0) * 0.3)
            END as final_score
        FROM content_data cd
    )
    SELECT 
        sc.id,
        sc.title,
        sc.author,
        ROUND(sc.final_score, 2)::DECIMAL,
        ROW_NUMBER() OVER (ORDER BY sc.final_score DESC)::INTEGER,
        $2::VARCHAR
    FROM scored_content sc
    ORDER BY sc.final_score DESC
    LIMIT $3', 
    p_content_type, p_algorithm, p_limit);
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Performance Optimization

```sql
-- Materialized view for expensive ranking calculations
CREATE MATERIALIZED VIEW product_rankings_mv AS
SELECT * FROM calculate_ecommerce_ranking();

-- Index on materialized view
CREATE INDEX idx_product_rankings_mv_rank ON product_rankings_mv(rank_position);
CREATE INDEX idx_product_rankings_mv_category ON product_rankings_mv(category, rank_position);

-- Refresh strategy
CREATE OR REPLACE FUNCTION refresh_ranking_materialized_views()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY product_rankings_mv;
    -- Add other materialized views here
END;
$$ LANGUAGE plpgsql;

-- Schedule regular refreshes (pseudo-code for cron job)
-- SELECT cron.schedule('refresh_rankings', '0 */6 * * *', 'SELECT refresh_ranking_materialized_views();');
```

### 2. Ranking Stability and Fairness

```sql
-- Anti-gaming measures and stability
CREATE TABLE ranking_anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    anomaly_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Anomaly details
    metric_name VARCHAR(100),
    expected_value DECIMAL(12,4),
    actual_value DECIMAL(12,4),
    deviation_percentage DECIMAL(6,2),
    
    -- Resolution
    resolved_at TIMESTAMPTZ,
    resolution_action VARCHAR(200),
    
    INDEX idx_ranking_anomalies_entity (entity_type, entity_id),
    INDEX idx_ranking_anomalies_type (anomaly_type, detected_at)
);

-- Function to detect ranking anomalies
CREATE OR REPLACE FUNCTION detect_ranking_anomalies()
RETURNS INTEGER AS $$
DECLARE
    v_anomaly_count INTEGER := 0;
BEGIN
    -- Detect sudden rank changes
    INSERT INTO ranking_anomalies (
        entity_type, entity_id, anomaly_type, severity,
        metric_name, expected_value, actual_value, deviation_percentage
    )
    SELECT 
        'product',
        rh.entity_id,
        'sudden_rank_change',
        CASE 
            WHEN ABS(rh.rank_change) > 100 THEN 'high'
            WHEN ABS(rh.rank_change) > 50 THEN 'medium'
            ELSE 'low'
        END,
        'rank_position',
        rh.rank_position - rh.rank_change,
        rh.rank_position,
        (ABS(rh.rank_change)::DECIMAL / GREATEST(rh.rank_position - rh.rank_change, 1)) * 100
    FROM ranking_history rh
    WHERE rh.period_date = CURRENT_DATE - INTERVAL '1 day'
      AND ABS(rh.rank_change) > 25 -- Threshold for significant change
      AND NOT EXISTS (
          SELECT 1 FROM ranking_anomalies ra 
          WHERE ra.entity_id = rh.entity_id 
            AND ra.detected_at >= CURRENT_DATE - INTERVAL '1 day'
            AND ra.anomaly_type = 'sudden_rank_change'
      );
    
    GET DIAGNOSTICS v_anomaly_count = ROW_COUNT;
    
    RETURN v_anomaly_count;
END;
$$ LANGUAGE plpgsql;
```

This comprehensive ranking system provides a solid foundation for implementing sophisticated scoring and ranking mechanisms across various domains, with built-in performance optimization, anomaly detection, and scalability considerations.
      end
    ) AS negative
  FROM
    product_item_reviews pir
  WHERE pir.created_at < {{endDate}}::date AND
  pi.id = pir.item_id AND
  pir.deleted_at IS NULL
GROUP BY
  pir.item_id
) tmp ON true
  WHERE id IN (
    SELECT product_items.id FROM product_items 
    JOIN product_item_categories ON (product_items.category_id = product_item_categories.id)
    WHERE {{category}}
  )
) tmp
WHERE deleted_at IS NULL
ORDER BY total_rank, total_review_count_rank, score, recent_review_count_rank
```

## Ranking

Strategy for ranking multiple columns:

1. rank each columns individually
2. create a column to store the score of sum of each column rank (the lower the better)
3. sort by the score, then each column in the desired order

| Price ($) | Distance (KM) | Rating (0-5) |
| - | - | - |
| 10 | 5 | 1 |
| 10 | 20 | 3 |
| 100 | 20 | 5 |

When ranking each column:

| Price ($) | Distance (KM) | Rating (0-5) | Score |
| - | - | - | - |
| 1 | 1 | 3 | 5 |
| 1 | 2 | 2 | 5 |
| 100 | 2 | 1 | 5 |

Sort by the score (sum of rank) is the same, sort by the column. The disadavantage of this method is that you have to recompute the scores by taking into account all rows.


## ranking with computed columns


Sometimes we just want to sort a product by rank.

We can just create another view that ranks the table, find the IDs after pagination, and make another query to fetch the full rows.
