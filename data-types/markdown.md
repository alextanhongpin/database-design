# Storing Markdown in Databases

Best practices for storing and handling markdown content in database systems.

## Why Store Markdown Instead of HTML?

### Benefits of Markdown Storage

1. **Human Readable**: Easy to read and edit in raw form
2. **Version Control Friendly**: Text-based format works well with diffs
3. **Platform Independent**: Not tied to specific rendering engines  
4. **Lightweight**: Smaller storage footprint than HTML
5. **Secure**: Less risk of XSS attacks compared to raw HTML
6. **Future-Proof**: Simple format that's unlikely to become obsolete

### Comparison: Markdown vs HTML vs Rich Text

| Aspect | Markdown | HTML | Rich Text |
|--------|----------|------|-----------|
| Storage Size | Small | Large | Very Large |
| Human Readable | ✅ Yes | ❌ No | ❌ No |
| Edit Complexity | Simple | Complex | Medium |
| Security Risk | Low | High | Medium |
| Styling Control | Limited | Full | Medium |

## Database Schema Design

### Basic Content Table
```sql
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content_markdown TEXT NOT NULL,
    content_html TEXT, -- Cached rendered HTML
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for full-text search on rendered content
CREATE INDEX idx_articles_content_search 
ON articles USING gin(to_tsvector('english', content_html));
```

### With Version History
```sql
CREATE TABLE content_revisions (
    id SERIAL PRIMARY KEY,
    article_id INT REFERENCES articles(id),
    content_markdown TEXT NOT NULL,
    content_html TEXT,
    revision_number INT NOT NULL,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(article_id, revision_number)
);
```

### Content with Metadata
```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content_markdown TEXT NOT NULL,
    content_html TEXT,
    excerpt TEXT, -- Auto-generated from markdown
    word_count INT, -- Calculated from markdown
    reading_time_minutes INT, -- Estimated reading time
    tags TEXT[], -- PostgreSQL array
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to update metadata on content change
CREATE OR REPLACE FUNCTION update_content_metadata()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate word count (simple approximation)
    NEW.word_count = array_length(string_to_array(NEW.content_markdown, ' '), 1);
    
    -- Estimate reading time (250 words per minute)
    NEW.reading_time_minutes = CEIL(NEW.word_count::FLOAT / 250);
    
    -- Generate excerpt (first 200 characters, cleaned)
    NEW.excerpt = LEFT(regexp_replace(NEW.content_markdown, '[#*`_\[\]()]', '', 'g'), 200);
    
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_content_metadata_trigger
    BEFORE INSERT OR UPDATE OF content_markdown ON posts
    FOR EACH ROW EXECUTE FUNCTION update_content_metadata();
```

## Storage Strategies

### 1. Markdown-Only Storage
```sql
-- Store only markdown, render on-the-fly
CREATE TABLE simple_posts (
    id SERIAL PRIMARY KEY,
    content_markdown TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Pros: Simple, always fresh rendering
-- Cons: Performance cost for each view
```

### 2. Dual Storage (Recommended)
```sql
-- Store both markdown (source) and HTML (cache)
CREATE TABLE cached_posts (
    id SERIAL PRIMARY KEY,
    content_markdown TEXT NOT NULL,
    content_html TEXT,
    html_cache_updated_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Cache invalidation trigger
CREATE OR REPLACE FUNCTION invalidate_html_cache()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.content_markdown IS DISTINCT FROM NEW.content_markdown THEN
        NEW.content_html = NULL;
        NEW.html_cache_updated_at = NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### 3. External Storage
```sql
-- Store large content externally (S3, file system)
CREATE TABLE external_content (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content_markdown_url TEXT, -- S3 URL or file path
    content_html_url TEXT,
    content_hash VARCHAR(64), -- For cache validation
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Performance Considerations

### Indexing Strategies
```sql
-- Full-text search on rendered HTML
CREATE INDEX idx_posts_content_search 
ON posts USING gin(to_tsvector('english', content_html));

-- Prefix search on titles and content
CREATE INDEX idx_posts_title_prefix 
ON posts USING gin(title gin_trgm_ops);

-- Tag-based filtering (PostgreSQL arrays)
CREATE INDEX idx_posts_tags 
ON posts USING gin(tags);
```

### Query Optimization
```sql
-- Efficient content retrieval with caching check
SELECT 
    id, title, content_markdown,
    CASE 
        WHEN content_html IS NOT NULL 
             AND html_cache_updated_at > updated_at 
        THEN content_html
        ELSE NULL -- Need to render
    END as cached_html
FROM posts 
WHERE id = $1;
```

## Security Considerations

### Input Sanitization
```sql
-- Function to clean markdown input
CREATE OR REPLACE FUNCTION sanitize_markdown(input_text TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Remove potentially dangerous HTML tags
    RETURN regexp_replace(
        input_text, 
        '<script[^>]*>.*?</script>', 
        '', 
        'gi'
    );
END;
$$ LANGUAGE plpgsql;

-- Use in application or trigger
INSERT INTO posts (content_markdown) 
VALUES (sanitize_markdown($1));
```

### Safe HTML Rendering
```sql
-- Store allowed HTML tags for rendering
CREATE TABLE markdown_config (
    allowed_tags TEXT[] DEFAULT ARRAY['p', 'h1', 'h2', 'h3', 'strong', 'em', 'ul', 'ol', 'li', 'a', 'code', 'pre'],
    allowed_attributes JSONB DEFAULT '{"a": ["href"], "img": ["src", "alt"]}'::jsonb
);
```

## Application Integration

### Rendering Pipeline
```python
# Example Python approach
def save_post(title, markdown_content):
    # 1. Sanitize markdown
    clean_markdown = sanitize_markdown(markdown_content)
    
    # 2. Render to HTML
    html_content = markdown.markdown(clean_markdown, extensions=['tables', 'fenced_code'])
    
    # 3. Store both versions
    cursor.execute("""
        INSERT INTO posts (title, content_markdown, content_html, html_cache_updated_at)
        VALUES (%s, %s, %s, CURRENT_TIMESTAMP)
    """, (title, clean_markdown, html_content))
```

### Lazy Rendering
```javascript
// Example Node.js approach
async function getPost(id) {
    const post = await db.query('SELECT * FROM posts WHERE id = $1', [id]);
    
    if (!post.content_html || post.html_cache_updated_at < post.updated_at) {
        // Render and cache HTML
        const html = markdownToHtml(post.content_markdown);
        await db.query(
            'UPDATE posts SET content_html = $1, html_cache_updated_at = CURRENT_TIMESTAMP WHERE id = $2',
            [html, id]
        );
        post.content_html = html;
    }
    
    return post;
}
```

## Best Practices

1. **Always store the original markdown** - It's your source of truth
2. **Cache rendered HTML** - For performance in read-heavy applications
3. **Use database constraints** - Ensure content integrity
4. **Implement proper indexing** - For search and filtering
5. **Sanitize input** - Prevent XSS and other security issues
6. **Version control** - Keep history of content changes
7. **Monitor storage growth** - Large text fields can impact performance

## Common Pitfalls

- Storing only HTML (hard to edit later)
- Not caching rendered output (performance issues)
- Inadequate input sanitization (security risks)  
- Missing full-text search indexes (poor search performance)
- Not handling content migrations (when markdown parsers change)

## Related Topics

- [Text Search](../query-patterns/text-search.md) - Full-text search implementation
- [Content Versioning](../schema-design/versioning.md) - Version control patterns
- [Security Patterns](../security/README.md) - Input validation and XSS prevention
- [Performance Optimization](../performance/README.md) - Text storage optimization
