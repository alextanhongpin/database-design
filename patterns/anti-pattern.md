# Database Anti-Patterns: Data vs Query

> **Golden Rule**: Never mix data with query logic in your data model

## The Problem

One of the most common mistakes in database design is mixing raw data with computed/derived data in the same table. This leads to data inconsistency, complex maintenance, and poor performance.

## ❌ Anti-Pattern: Storing Computed Values

### Bad Example: Author Stats Mixed with Raw Data
```sql
-- DON'T DO THIS
CREATE TABLE authors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    bio TEXT,
    
    -- Anti-pattern: computed fields mixed with raw data
    books_count INTEGER DEFAULT 0,
    total_reviews INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0.0,
    last_book_published_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- The nightmare of keeping computed fields in sync
CREATE OR REPLACE FUNCTION update_author_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- This gets complex and error-prone quickly
    UPDATE authors SET 
        books_count = (SELECT COUNT(*) FROM books WHERE author_id = NEW.author_id),
        total_reviews = (SELECT COUNT(*) FROM reviews r 
                        JOIN books b ON r.book_id = b.id 
                        WHERE b.author_id = NEW.author_id),
        average_rating = (SELECT AVG(rating) FROM reviews r 
                         JOIN books b ON r.book_id = b.id 
                         WHERE b.author_id = NEW.author_id)
    WHERE id = NEW.author_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### Problems with This Approach
1. **Data Inconsistency** - Computed fields can become stale
2. **Complex Triggers** - Maintenance nightmare as business rules change
3. **Performance Issues** - Every insert/update triggers expensive calculations
4. **Race Conditions** - Concurrent updates can cause incorrect counts
5. **Testing Complexity** - Hard to ensure data integrity in tests

## ✅ Better Patterns

### Pattern 1: Real-time Computation with Views
```sql
-- Clean separation: raw data only
CREATE TABLE authors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    bio TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES authors(id),
    title TEXT NOT NULL,
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES books(id),
    user_id UUID NOT NULL,
    rating INTEGER CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Computed data through views (always accurate)
CREATE VIEW authors_with_stats AS
SELECT 
    a.*,
    COALESCE(stats.books_count, 0) AS books_count,
    COALESCE(stats.total_reviews, 0) AS total_reviews,
    COALESCE(stats.average_rating, 0) AS average_rating,
    stats.last_book_published_at
FROM authors a
LEFT JOIN (
    SELECT 
        b.author_id,
        COUNT(b.id) AS books_count,
        COUNT(r.id) AS total_reviews,
        ROUND(AVG(r.rating)::numeric, 2) AS average_rating,
        MAX(b.published_at) AS last_book_published_at
    FROM books b
    LEFT JOIN reviews r ON r.book_id = b.id
    GROUP BY b.author_id
) stats ON stats.author_id = a.id;
```

### Pattern 2: Materialized Views for Performance
```sql
-- For high-traffic scenarios where real-time computation is too slow
CREATE MATERIALIZED VIEW authors_stats_cache AS
SELECT 
    a.id,
    a.name,
    a.email,
    COUNT(DISTINCT b.id) AS books_count,
    COUNT(r.id) AS total_reviews,
    ROUND(AVG(r.rating)::numeric, 2) AS average_rating,
    MAX(b.published_at) AS last_book_published_at,
    NOW() AS last_updated
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
LEFT JOIN reviews r ON r.book_id = b.id
GROUP BY a.id, a.name, a.email;

-- Refresh strategy (choose based on your needs)
-- Option 1: Scheduled refresh
-- SELECT cron.schedule('refresh-author-stats', '*/15 * * * *', 'REFRESH MATERIALIZED VIEW authors_stats_cache;');

-- Option 2: Event-driven refresh
CREATE OR REPLACE FUNCTION refresh_author_stats()
RETURNS TRIGGER AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY authors_stats_cache;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Only refresh when data actually changes
CREATE TRIGGER refresh_author_stats_on_book_change
    AFTER INSERT OR UPDATE OR DELETE ON books
    FOR EACH STATEMENT EXECUTE FUNCTION refresh_author_stats();
```

### Pattern 3: Separate Stats Table for Complex Scenarios
```sql
-- When you need more control and better performance
CREATE TABLE author_stats (
    author_id UUID PRIMARY KEY REFERENCES authors(id) ON DELETE CASCADE,
    books_count INTEGER DEFAULT 0,
    total_reviews INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0.0,
    last_book_published_at TIMESTAMP,
    last_updated TIMESTAMP DEFAULT NOW(),
    
    -- Add constraints to ensure data quality
    CHECK (books_count >= 0),
    CHECK (total_reviews >= 0),
    CHECK (average_rating >= 0 AND average_rating <= 5)
);

-- Function to recalculate stats for a specific author
CREATE OR REPLACE FUNCTION recalculate_author_stats(author_uuid UUID)
RETURNS VOID AS $$
BEGIN
    INSERT INTO author_stats (
        author_id, books_count, total_reviews, 
        average_rating, last_book_published_at
    )
    SELECT 
        author_uuid,
        COUNT(DISTINCT b.id),
        COUNT(r.id),
        COALESCE(ROUND(AVG(r.rating)::numeric, 2), 0),
        MAX(b.published_at)
    FROM books b
    LEFT JOIN reviews r ON r.book_id = b.id
    WHERE b.author_id = author_uuid
    ON CONFLICT (author_id) 
    DO UPDATE SET
        books_count = EXCLUDED.books_count,
        total_reviews = EXCLUDED.total_reviews,
        average_rating = EXCLUDED.average_rating,
        last_book_published_at = EXCLUDED.last_book_published_at,
        last_updated = NOW();
END;
$$ LANGUAGE plpgsql;

-- Efficient trigger that only updates affected authors
CREATE OR REPLACE FUNCTION update_author_stats_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- Handle different trigger events
    IF TG_OP = 'DELETE' THEN
        PERFORM recalculate_author_stats(OLD.author_id);
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        PERFORM recalculate_author_stats(NEW.author_id);
        IF NEW.author_id != OLD.author_id THEN
            PERFORM recalculate_author_stats(OLD.author_id);
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        PERFORM recalculate_author_stats(NEW.author_id);
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER books_stats_trigger
    AFTER INSERT OR UPDATE OR DELETE ON books
    FOR EACH ROW EXECUTE FUNCTION update_author_stats_trigger();
```

## 🎯 Real-World Examples

### E-Commerce: Product Statistics
```sql
-- ❌ Anti-pattern: Mixing product data with computed stats
CREATE TABLE products_bad (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    -- Don't do this:
    total_sales INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2),
    stock_level INTEGER DEFAULT 0
);

-- ✅ Better: Separate concerns
CREATE TABLE products (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    base_price_cents INTEGER NOT NULL, -- Store as cents to avoid float issues
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE inventory (
    product_id UUID PRIMARY KEY REFERENCES products(id),
    quantity_available INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    last_restocked_at TIMESTAMP,
    
    CHECK (quantity_available >= 0),
    CHECK (reserved_quantity >= 0)
);

-- Computed view for product display
CREATE VIEW products_with_stats AS
SELECT 
    p.*,
    COALESCE(i.quantity_available, 0) AS stock_level,
    COALESCE(sales.total_quantity, 0) AS total_sales,
    COALESCE(reviews.avg_rating, 0) AS average_rating,
    COALESCE(reviews.review_count, 0) AS review_count
FROM products p
LEFT JOIN inventory i ON i.product_id = p.id
LEFT JOIN (
    SELECT 
        product_id,
        SUM(quantity) AS total_quantity
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE o.status = 'completed'
    GROUP BY product_id
) sales ON sales.product_id = p.id
LEFT JOIN (
    SELECT 
        product_id,
        ROUND(AVG(rating)::numeric, 2) AS avg_rating,
        COUNT(*) AS review_count
    FROM product_reviews
    WHERE deleted_at IS NULL
    GROUP BY product_id
) reviews ON reviews.product_id = p.id;
```

## 🔧 Migration Strategy

If you're already stuck with computed fields in your main tables:

### Step 1: Create Clean Tables
```sql
-- Create new clean table structure
CREATE TABLE authors_clean (LIKE authors);
ALTER TABLE authors_clean 
    DROP COLUMN books_count,
    DROP COLUMN total_reviews,
    DROP COLUMN average_rating;

-- Create separate stats table
CREATE TABLE author_stats (
    author_id UUID PRIMARY KEY REFERENCES authors_clean(id),
    books_count INTEGER DEFAULT 0,
    total_reviews INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0.0,
    computed_at TIMESTAMP DEFAULT NOW()
);
```

### Step 2: Migrate Data
```sql
-- Copy clean data
INSERT INTO authors_clean 
SELECT id, name, email, bio, created_at, updated_at 
FROM authors;

-- Recompute and store stats separately
INSERT INTO author_stats (author_id, books_count, total_reviews, average_rating)
SELECT 
    a.id,
    COUNT(DISTINCT b.id),
    COUNT(r.id),
    COALESCE(AVG(r.rating), 0)
FROM authors_clean a
LEFT JOIN books b ON b.author_id = a.id
LEFT JOIN reviews r ON r.book_id = b.id
GROUP BY a.id;
```

### Step 3: Update Application Code
```sql
-- Replace direct table queries with views
CREATE VIEW authors_with_stats AS
SELECT 
    a.*,
    COALESCE(s.books_count, 0) AS books_count,
    COALESCE(s.total_reviews, 0) AS total_reviews,
    COALESCE(s.average_rating, 0) AS average_rating
FROM authors_clean a
LEFT JOIN author_stats s ON s.author_id = a.id;
```

## 📊 Performance Comparison

| Approach | Read Performance | Write Performance | Data Consistency | Maintenance |
|----------|------------------|-------------------|------------------|-------------|
| Computed Fields | Fast | Slow (triggers) | Poor | High |
| Real-time Views | Medium | Fast | Perfect | Low |
| Materialized Views | Fast | Fast | Good | Medium |
| Stats Tables | Fast | Medium | Good | Medium |

## 💡 Best Practices

1. **Keep Raw Data Pure** - Never pollute source tables with computed values
2. **Use Views for Simple Cases** - When real-time accuracy is needed
3. **Materialized Views for Scale** - When performance matters more than real-time accuracy
4. **Stats Tables for Complex Logic** - When you need fine-grained control
5. **Plan for Evolution** - Business rules change; computed fields make changes harder
6. **Monitor Performance** - Profile your queries and choose the right pattern
7. **Test Data Integrity** - Ensure computed values match source data

## 🚨 Warning Signs You're Mixing Data and Query

- Multiple triggers updating the same computed fields
- `UPDATE` statements with complex subqueries in triggers
- Data inconsistencies between computed and actual values
- Slow write operations due to stats calculations
- Difficulty adding new computed fields
- Complex rollback scenarios for failed transactions

Remember: **Your database should store facts, not opinions**. Let your queries compute the opinions from the facts.
 
