# Polymorphic Associations in Database Design

Polymorphic associations allow a single table to reference multiple different types of entities. While this pattern can be powerful, it requires careful design to maintain data integrity and performance.

## 🎯 When to Use Polymorphic Associations

### Common Use Cases
- **Comments System** - Comments on posts, videos, photos, products
- **Activity Feeds** - User actions on different entity types
- **Tagging System** - Tags applied to various content types
- **Notifications** - Alerts about different types of events
- **File Attachments** - Documents attached to multiple entity types

### ⚠️ Warning: Consider Alternatives First

Before implementing polymorphic associations, consider simpler alternatives:
- **Separate tables** for each relationship (comments_on_posts, comments_on_videos)
- **Union views** to combine similar data
- **JSONB columns** for flexible, unstructured references

## 🏗️ Implementation Patterns

### Pattern 1: Constrained Polymorphic Associations

**Best for**: Known, limited set of target types with referential integrity

```sql
-- Target entities
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    duration_seconds INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Comments with constrained polymorphism
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    
    -- Polymorphic columns
    commentable_type TEXT NOT NULL CHECK (
        commentable_type IN ('post', 'video', 'product')
    ),
    commentable_id UUID NOT NULL,
    
    -- Foreign key references (only one will be used)
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    video_id UUID REFERENCES videos(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure exactly one foreign key is set based on type
    CONSTRAINT valid_polymorphic_reference CHECK (
        CASE commentable_type
            WHEN 'post' THEN post_id IS NOT NULL AND video_id IS NULL AND product_id IS NULL
            WHEN 'video' THEN video_id IS NOT NULL AND post_id IS NULL AND product_id IS NULL  
            WHEN 'product' THEN product_id IS NOT NULL AND post_id IS NULL AND video_id IS NULL
        END
    ),
    
    -- Ensure ID consistency
    CONSTRAINT consistent_polymorphic_id CHECK (
        CASE commentable_type
            WHEN 'post' THEN commentable_id = post_id
            WHEN 'video' THEN commentable_id = video_id
            WHEN 'product' THEN commentable_id = product_id
        END
    )
);

-- Partial unique indexes for performance
CREATE UNIQUE INDEX idx_comments_post_unique 
ON comments (post_id, user_id) 
WHERE commentable_type = 'post';

CREATE UNIQUE INDEX idx_comments_video_unique 
ON comments (video_id, user_id) 
WHERE commentable_type = 'video';

CREATE UNIQUE INDEX idx_comments_product_unique 
ON comments (product_id, user_id) 
WHERE commentable_type = 'product';

-- Indexes for queries
CREATE INDEX idx_comments_polymorphic ON comments (commentable_type, commentable_id);
CREATE INDEX idx_comments_user ON comments (user_id, created_at);
```

### Pattern 2: Simple Polymorphic (Less Safe)

**Best for**: Rapid prototyping, flexible schemas where integrity is managed in application

```sql
-- Simple polymorphic without foreign keys
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    
    -- Polymorphic reference
    likeable_type TEXT NOT NULL,
    likeable_id UUID NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Prevent duplicate likes
    UNIQUE (user_id, likeable_type, likeable_id)
);

-- Query helper function
CREATE OR REPLACE FUNCTION get_like_count(
    entity_type TEXT,
    entity_id UUID
) RETURNS INTEGER AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)
        FROM likes
        WHERE likeable_type = entity_type 
        AND likeable_id = entity_id
    );
END;
$$ LANGUAGE plpgsql;

-- Usage
SELECT get_like_count('post', 'post-uuid-here');
```

### Pattern 3: Class Table Inheritance

**Best for**: Shared attributes with type-specific attributes

```sql
-- Base table for shared attributes
CREATE TABLE content_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_type TEXT NOT NULL CHECK (
        content_type IN ('article', 'video', 'podcast')
    ),
    title TEXT NOT NULL,
    description TEXT,
    author_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure referential integrity
    UNIQUE (id, content_type)
);

-- Type-specific tables
CREATE TABLE articles (
    id UUID PRIMARY KEY,
    content_type TEXT NOT NULL DEFAULT 'article' CHECK (content_type = 'article'),
    
    -- Article-specific fields
    content TEXT NOT NULL,
    word_count INTEGER,
    reading_time_minutes INTEGER,
    
    -- Link to base table
    FOREIGN KEY (id, content_type) REFERENCES content_items (id, content_type)
);

CREATE TABLE videos (
    id UUID PRIMARY KEY,
    content_type TEXT NOT NULL DEFAULT 'video' CHECK (content_type = 'video'),
    
    -- Video-specific fields
    url TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    thumbnail_url TEXT,
    
    -- Link to base table  
    FOREIGN KEY (id, content_type) REFERENCES content_items (id, content_type)
);

CREATE TABLE podcasts (
    id UUID PRIMARY KEY,
    content_type TEXT NOT NULL DEFAULT 'podcast' CHECK (content_type = 'podcast'),
    
    -- Podcast-specific fields
    audio_url TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    episode_number INTEGER,
    
    -- Link to base table
    FOREIGN KEY (id, content_type) REFERENCES content_items (id, content_type)
);

-- Comments can now reference the base table
CREATE TABLE content_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_item_id UUID NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    comment TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- View to get all content with type-specific data
CREATE VIEW all_content AS
SELECT 
    ci.id, ci.content_type, ci.title, ci.description, ci.author_id, ci.created_at,
    a.content, a.word_count, a.reading_time_minutes,
    v.url as video_url, v.duration_seconds as video_duration, v.thumbnail_url,
    p.audio_url, p.duration_seconds as podcast_duration, p.episode_number
FROM content_items ci
LEFT JOIN articles a ON a.id = ci.id
LEFT JOIN videos v ON v.id = ci.id  
LEFT JOIN podcasts p ON p.id = ci.id;
```

### Pattern 4: JSON-Based Polymorphism

**Best for**: Highly flexible schemas, document-like data

```sql
-- Flexible polymorphic using JSONB
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    
    -- Event-specific data in JSONB
    event_data JSONB NOT NULL,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure required fields per event type
    CONSTRAINT valid_event_data CHECK (
        CASE event_type
            WHEN 'post_created' THEN event_data ? 'post_id'
            WHEN 'comment_added' THEN event_data ? 'comment_id' AND event_data ? 'post_id'
            WHEN 'user_followed' THEN event_data ? 'followed_user_id'
            WHEN 'purchase_made' THEN event_data ? 'order_id' AND event_data ? 'total_cents'
            ELSE true
        END
    )
);

-- GIN index for JSONB queries
CREATE INDEX idx_events_data ON events USING gin(event_data);

-- Query examples
-- Find all post creation events
SELECT * FROM events 
WHERE event_type = 'post_created' 
AND event_data->>'post_id' = 'specific-post-uuid';

-- Find all events related to a specific post
SELECT * FROM events 
WHERE event_data->>'post_id' = 'specific-post-uuid';

-- Aggregate events by type
SELECT 
    event_type,
    COUNT(*) as event_count,
    MIN(created_at) as first_event,
    MAX(created_at) as last_event
FROM events 
GROUP BY event_type
ORDER BY event_count DESC;
```

## 🌍 Real-World Examples

### Multi-Content Commenting System
```sql
-- Production-ready commenting system for multiple content types
CREATE TABLE content_types (
    type_name TEXT PRIMARY KEY,
    table_name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO content_types (type_name, table_name, display_name) VALUES
('blog_post', 'blog_posts', 'Blog Post'),
('product', 'products', 'Product'),
('video', 'videos', 'Video'),
('course', 'courses', 'Course');

CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    
    -- Polymorphic reference
    content_type TEXT NOT NULL REFERENCES content_types(type_name),
    content_id UUID NOT NULL,
    
    -- Comment content
    body TEXT NOT NULL CHECK (LENGTH(body) BETWEEN 1 AND 2000),
    
    -- Moderation
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'approved', 'rejected', 'flagged')
    ),
    moderated_by UUID,
    moderated_at TIMESTAMP,
    
    -- Hierarchy for nested comments
    parent_comment_id UUID REFERENCES comments(id),
    thread_depth INTEGER DEFAULT 0 CHECK (thread_depth >= 0 AND thread_depth <= 5),
    
    -- Engagement metrics
    upvotes INTEGER DEFAULT 0 CHECK (upvotes >= 0),
    downvotes INTEGER DEFAULT 0 CHECK (downvotes >= 0),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Prevent self-replies at root level
    CHECK (parent_comment_id IS NULL OR parent_comment_id != id)
);

-- Function to validate content exists
CREATE OR REPLACE FUNCTION validate_content_exists()
RETURNS TRIGGER AS $$
DECLARE
    content_exists BOOLEAN := FALSE;
    query_text TEXT;
BEGIN
    -- Dynamically check if content exists
    SELECT table_name INTO query_text 
    FROM content_types 
    WHERE type_name = NEW.content_type;
    
    IF query_text IS NULL THEN
        RAISE EXCEPTION 'Invalid content type: %', NEW.content_type;
    END IF;
    
    EXECUTE format('SELECT EXISTS(SELECT 1 FROM %I WHERE id = $1)', query_text)
    INTO content_exists
    USING NEW.content_id;
    
    IF NOT content_exists THEN
        RAISE EXCEPTION 'Content does not exist: % with id %', NEW.content_type, NEW.content_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER validate_comment_content
    BEFORE INSERT OR UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION validate_content_exists();

-- Optimized indexes
CREATE INDEX idx_comments_content ON comments (content_type, content_id, status);
CREATE INDEX idx_comments_user ON comments (user_id, created_at);
CREATE INDEX idx_comments_moderation ON comments (status, created_at) WHERE status = 'pending';
CREATE INDEX idx_comments_thread ON comments (parent_comment_id, thread_depth);
```

### Activity Feed System
```sql
-- Scalable activity feed with polymorphic events
CREATE TABLE activity_types (
    type_code TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    icon_name TEXT,
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO activity_types (type_code, display_name, description) VALUES
('post_created', 'Posted', 'User created a new post'),
('post_liked', 'Liked Post', 'User liked a post'),
('user_followed', 'Followed User', 'User followed another user'),
('comment_added', 'Commented', 'User commented on content'),
('achievement_earned', 'Achievement', 'User earned an achievement');

CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Who performed the activity
    actor_id UUID NOT NULL,
    
    -- What type of activity
    activity_type TEXT NOT NULL REFERENCES activity_types(type_code),
    
    -- What was acted upon (polymorphic)
    target_type TEXT,
    target_id UUID,
    
    -- Additional context (optional secondary target)
    secondary_target_type TEXT,
    secondary_target_id UUID,
    
    -- Activity metadata
    activity_data JSONB DEFAULT '{}',
    
    -- Privacy and visibility
    visibility TEXT NOT NULL DEFAULT 'public' CHECK (
        visibility IN ('public', 'friends', 'private')
    ),
    
    -- Aggregation support
    aggregation_key TEXT, -- For grouping similar activities
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Indexes for common queries
    CONSTRAINT valid_target CHECK (
        (target_type IS NULL) = (target_id IS NULL)
    )
);

-- Function to generate aggregation keys
CREATE OR REPLACE FUNCTION generate_activity_aggregation_key()
RETURNS TRIGGER AS $$
BEGIN
    -- Create aggregation keys for activities that can be grouped
    NEW.aggregation_key := CASE NEW.activity_type
        WHEN 'post_liked' THEN 
            format('likes:%s:%s', NEW.target_type, NEW.target_id)
        WHEN 'user_followed' THEN 
            format('follows:%s', NEW.actor_id)
        ELSE 
            NULL
    END;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER activity_aggregation_trigger
    BEFORE INSERT ON activities
    FOR EACH ROW EXECUTE FUNCTION generate_activity_aggregation_key();

-- View for activity feed with aggregation
CREATE VIEW activity_feed AS
SELECT 
    CASE 
        WHEN COUNT(*) = 1 THEN MIN(id)
        ELSE NULL
    END as activity_id,
    
    activity_type,
    target_type,
    target_id,
    
    -- Aggregated actors
    CASE 
        WHEN COUNT(*) = 1 THEN ARRAY[MIN(actor_id)]
        ELSE ARRAY_AGG(actor_id ORDER BY created_at DESC)
    END as actor_ids,
    
    COUNT(*) as activity_count,
    MAX(created_at) as latest_at,
    MIN(created_at) as earliest_at,
    
    -- Sample activity data
    (ARRAY_AGG(activity_data ORDER BY created_at DESC))[1] as sample_data
    
FROM activities 
WHERE visibility = 'public'
AND created_at > NOW() - INTERVAL '30 days'
GROUP BY 
    COALESCE(aggregation_key, id::TEXT),
    activity_type,
    target_type, 
    target_id
ORDER BY MAX(created_at) DESC;

-- Partitioning for scale
CREATE TABLE activities_2025_01 PARTITION OF activities
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Indexes for performance
CREATE INDEX idx_activities_actor_time ON activities (actor_id, created_at DESC);
CREATE INDEX idx_activities_target ON activities (target_type, target_id, created_at);
CREATE INDEX idx_activities_aggregation ON activities (aggregation_key, created_at) 
WHERE aggregation_key IS NOT NULL;
```

### Notification System
```sql
-- Flexible notification system with polymorphic triggers
CREATE TABLE notification_types (
    type_code TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    template TEXT NOT NULL, -- Message template with placeholders
    is_email BOOLEAN DEFAULT FALSE,
    is_push BOOLEAN DEFAULT TRUE,
    is_in_app BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO notification_types (type_code, display_name, template) VALUES
('comment_reply', 'Comment Reply', '{actor_name} replied to your comment on {target_title}'),
('post_liked', 'Post Liked', '{actor_name} liked your post "{target_title}"'),
('user_followed', 'New Follower', '{actor_name} started following you'),
('order_shipped', 'Order Shipped', 'Your order #{target_id} has been shipped');

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Who gets the notification
    recipient_id UUID NOT NULL,
    
    -- Notification details
    notification_type TEXT NOT NULL REFERENCES notification_types(type_code),
    
    -- Who triggered it (can be system/null)
    actor_id UUID,
    
    -- What it's about (polymorphic)
    target_type TEXT,
    target_id UUID,
    
    -- Notification content (generated from template)
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    
    -- Delivery tracking
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    
    -- Channel delivery status
    email_sent_at TIMESTAMP,
    push_sent_at TIMESTAMP,
    
    -- Metadata
    data JSONB DEFAULT '{}',
    
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '30 days'
);

-- Function to create notifications from activities
CREATE OR REPLACE FUNCTION create_notification_from_activity()
RETURNS TRIGGER AS $$
DECLARE
    notification_config RECORD;
    recipient_ids UUID[];
    recipient_id UUID;
    notification_title TEXT;
    notification_message TEXT;
BEGIN
    -- Skip if activity is private
    IF NEW.visibility = 'private' THEN
        RETURN NEW;
    END IF;
    
    -- Get notification configuration
    SELECT nt.* INTO notification_config
    FROM notification_types nt
    WHERE nt.type_code = NEW.activity_type;
    
    IF notification_config IS NULL THEN
        RETURN NEW;
    END IF;
    
    -- Determine recipients based on activity type
    recipient_ids := CASE NEW.activity_type
        WHEN 'comment_added' THEN 
            -- Notify post author and parent comment author
            ARRAY(
                SELECT DISTINCT user_id 
                FROM posts p
                WHERE p.id = NEW.target_id
                AND p.user_id != NEW.actor_id
                UNION
                SELECT c.user_id
                FROM comments c
                WHERE c.id = NEW.secondary_target_id
                AND c.user_id != NEW.actor_id
            )
        WHEN 'post_liked' THEN
            -- Notify post author
            ARRAY(
                SELECT user_id 
                FROM posts 
                WHERE id = NEW.target_id 
                AND user_id != NEW.actor_id
            )
        WHEN 'user_followed' THEN
            -- Notify followed user
            ARRAY[NEW.target_id]
        ELSE
            ARRAY[]::UUID[]
    END;
    
    -- Create notifications for each recipient
    FOREACH recipient_id IN ARRAY recipient_ids LOOP
        -- Generate personalized message
        -- This would typically involve a more sophisticated templating system
        notification_title := notification_config.display_name;
        notification_message := notification_config.template;
        
        INSERT INTO notifications (
            recipient_id, notification_type, actor_id,
            target_type, target_id,
            title, message, data
        ) VALUES (
            recipient_id, NEW.activity_type, NEW.actor_id,
            NEW.target_type, NEW.target_id,
            notification_title, notification_message,
            NEW.activity_data
        );
    END LOOP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER activity_notification_trigger
    AFTER INSERT ON activities
    FOR EACH ROW EXECUTE FUNCTION create_notification_from_activity();

-- Indexes for notifications
CREATE INDEX idx_notifications_recipient ON notifications (recipient_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications (recipient_id, is_read, created_at) 
WHERE is_read = FALSE;
CREATE INDEX idx_notifications_delivery ON notifications (created_at) 
WHERE email_sent_at IS NULL OR push_sent_at IS NULL;
```

## 📊 Querying Polymorphic Data

### Basic Queries
```sql
-- Get all comments for a specific post
SELECT c.*, u.username, u.avatar_url
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.content_type = 'blog_post' 
AND c.content_id = 'specific-post-uuid'
AND c.status = 'approved'
ORDER BY c.created_at DESC;

-- Get comment counts by content type
SELECT 
    content_type,
    COUNT(*) as total_comments,
    COUNT(*) FILTER (WHERE status = 'approved') as approved_comments,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as recent_comments
FROM comments
GROUP BY content_type
ORDER BY total_comments DESC;

-- Activity feed for a user's followers
SELECT 
    a.id,
    a.activity_type,
    a.target_type,
    a.target_id,
    a.created_at,
    
    -- Actor information
    u.username as actor_username,
    u.avatar_url as actor_avatar,
    
    -- Target information (polymorphic)
    CASE a.target_type
        WHEN 'blog_post' THEN (SELECT title FROM blog_posts WHERE id = a.target_id)
        WHEN 'product' THEN (SELECT name FROM products WHERE id = a.target_id)
        WHEN 'video' THEN (SELECT title FROM videos WHERE id = a.target_id)
    END as target_title

FROM activities a
JOIN users u ON u.id = a.actor_id
WHERE a.actor_id IN (
    SELECT following_id 
    FROM user_follows 
    WHERE follower_id = 'current-user-uuid'
)
AND a.visibility = 'public'
AND a.created_at > NOW() - INTERVAL '7 days'
ORDER BY a.created_at DESC
LIMIT 50;
```

### Advanced Aggregations
```sql
-- Content engagement metrics across all types
WITH engagement_stats AS (
    SELECT 
        content_type,
        content_id,
        COUNT(*) as comment_count,
        COUNT(DISTINCT user_id) as unique_commenters,
        AVG(LENGTH(body)) as avg_comment_length,
        MAX(created_at) as last_comment_at
    FROM comments
    WHERE status = 'approved'
    AND created_at > NOW() - INTERVAL '30 days'
    GROUP BY content_type, content_id
),
like_stats AS (
    SELECT 
        likeable_type as content_type,
        likeable_id as content_id,
        COUNT(*) as like_count,
        COUNT(DISTINCT user_id) as unique_likers
    FROM likes
    WHERE created_at > NOW() - INTERVAL '30 days'
    GROUP BY likeable_type, likeable_id
)
SELECT 
    COALESCE(e.content_type, l.content_type) as content_type,
    COALESCE(e.content_id, l.content_id) as content_id,
    
    COALESCE(e.comment_count, 0) as comments,
    COALESCE(l.like_count, 0) as likes,
    
    -- Engagement score
    (COALESCE(e.comment_count, 0) * 3 + COALESCE(l.like_count, 0)) as engagement_score,
    
    COALESCE(e.unique_commenters, 0) as unique_commenters,
    COALESCE(l.unique_likers, 0) as unique_likers,
    
    e.last_comment_at

FROM engagement_stats e
FULL OUTER JOIN like_stats l ON l.content_type = e.content_type AND l.content_id = e.content_id
ORDER BY engagement_score DESC
LIMIT 100;
```

## ⚡ Performance Optimization

### Indexing Strategies
```sql
-- Composite indexes for common query patterns
CREATE INDEX idx_comments_content_status_time 
ON comments (content_type, content_id, status, created_at DESC);

-- Partial indexes for active data
CREATE INDEX idx_comments_approved 
ON comments (content_type, content_id, created_at DESC) 
WHERE status = 'approved';

-- GIN indexes for JSONB data
CREATE INDEX idx_activities_data_gin 
ON activities USING gin(activity_data);

-- Expression indexes for computed values
CREATE INDEX idx_notifications_urgency 
ON notifications ((
    CASE notification_type
        WHEN 'order_shipped' THEN 1
        WHEN 'comment_reply' THEN 2  
        WHEN 'post_liked' THEN 3
        ELSE 4
    END
)) WHERE is_read = FALSE;
```

### Materialized Views for Aggregations
```sql
-- Pre-computed content statistics
CREATE MATERIALIZED VIEW content_engagement_summary AS
SELECT 
    content_type,
    content_id,
    COUNT(DISTINCT CASE WHEN activity_type = 'comment_added' THEN actor_id END) as commenters,
    COUNT(DISTINCT CASE WHEN activity_type = 'post_liked' THEN actor_id END) as likers,
    COUNT(DISTINCT CASE WHEN activity_type = 'content_shared' THEN actor_id END) as sharers,
    MAX(created_at) as last_activity_at,
    
    -- Engagement score
    COUNT(DISTINCT CASE WHEN activity_type = 'comment_added' THEN actor_id END) * 3 +
    COUNT(DISTINCT CASE WHEN activity_type = 'post_liked' THEN actor_id END) * 1 +
    COUNT(DISTINCT CASE WHEN activity_type = 'content_shared' THEN actor_id END) * 5 as engagement_score

FROM activities
WHERE created_at > NOW() - INTERVAL '30 days'
AND target_type IS NOT NULL
GROUP BY content_type, content_id;

-- Refresh strategy
CREATE OR REPLACE FUNCTION refresh_engagement_summary()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY content_engagement_summary;
END;
$$ LANGUAGE plpgsql;

-- Schedule refresh every hour
-- SELECT cron.schedule('refresh-engagement', '0 * * * *', 'SELECT refresh_engagement_summary();');
```

## 🚨 Common Pitfalls & Solutions

### Pitfall 1: No Referential Integrity
```sql
-- ❌ Problem: Orphaned polymorphic references
CREATE TABLE comments (
    commentable_type TEXT,
    commentable_id UUID
    -- No foreign keys = no integrity
);

-- ✅ Solution: Use constrained polymorphism or validation
CREATE OR REPLACE FUNCTION validate_polymorphic_reference()
RETURNS TRIGGER AS $$
DECLARE
    table_name TEXT;
    exists_check BOOLEAN;
BEGIN
    -- Map types to table names
    table_name := CASE NEW.commentable_type
        WHEN 'post' THEN 'posts'
        WHEN 'video' THEN 'videos'
        ELSE NULL
    END;
    
    IF table_name IS NULL THEN
        RAISE EXCEPTION 'Invalid commentable_type: %', NEW.commentable_type;
    END IF;
    
    -- Check if reference exists
    EXECUTE format('SELECT EXISTS(SELECT 1 FROM %I WHERE id = $1)', table_name)
    INTO exists_check USING NEW.commentable_id;
    
    IF NOT exists_check THEN
        RAISE EXCEPTION 'Referenced % with id % does not exist', NEW.commentable_type, NEW.commentable_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### Pitfall 2: Poor Query Performance
```sql
-- ❌ Problem: Full table scans on polymorphic queries
SELECT * FROM activities 
WHERE target_id = 'some-uuid'; -- Scans all activities

-- ✅ Solution: Include type in queries and indexes
SELECT * FROM activities 
WHERE target_type = 'post' AND target_id = 'some-uuid';

-- With proper index
CREATE INDEX idx_activities_target_typed 
ON activities (target_type, target_id, created_at);
```

### Pitfall 3: Complex Application Logic
```sql
-- ❌ Problem: Complex application joins
-- Application code:
// for each comment
//   if comment.commentable_type == 'post'
//     post = fetch_post(comment.commentable_id)
//   elsif comment.commentable_type == 'video'
//     video = fetch_video(comment.commentable_id)

-- ✅ Solution: Database views with UNION ALL
CREATE VIEW comments_with_content AS
SELECT 
    c.*,
    p.title as content_title,
    p.created_at as content_created_at,
    'post' as content_type
FROM comments c
JOIN posts p ON p.id = c.post_id
WHERE c.commentable_type = 'post'

UNION ALL

SELECT 
    c.*,
    v.title as content_title,
    v.created_at as content_created_at,
    'video' as content_type
FROM comments c  
JOIN videos v ON v.id = c.video_id
WHERE c.commentable_type = 'video';
```

## 💡 Best Practices

1. **Start Simple** - Use separate tables before polymorphic associations
2. **Validate References** - Ensure polymorphic references point to existing records
3. **Index Strategically** - Include type in composite indexes
4. **Limit Types** - Keep the number of target types manageable
5. **Use Views** - Simplify complex polymorphic queries with views
6. **Consider JSONB** - For highly flexible schemas, JSONB might be better
7. **Monitor Performance** - Polymorphic queries can be slow; profile regularly
8. **Document Patterns** - Make the polymorphic relationships clear to other developers

## 🔄 Migration Strategies

### From Separate Tables to Polymorphic
```sql
-- Before: Separate comment tables
CREATE TABLE post_comments (id UUID, post_id UUID, content TEXT);
CREATE TABLE video_comments (id UUID, video_id UUID, content TEXT);

-- Migration to polymorphic structure
BEGIN;

-- Create new polymorphic table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commentable_type TEXT NOT NULL,
    commentable_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Migrate data
INSERT INTO comments (id, commentable_type, commentable_id, content, created_at)
SELECT id, 'post', post_id, content, created_at FROM post_comments
UNION ALL
SELECT id, 'video', video_id, content, created_at FROM video_comments;

-- Verify migration
SELECT 
    'post_comments' as source_table,
    COUNT(*) as count
FROM post_comments
UNION ALL
SELECT 'video_comments', COUNT(*) FROM video_comments
UNION ALL  
SELECT 'comments (posts)', COUNT(*) FROM comments WHERE commentable_type = 'post'
UNION ALL
SELECT 'comments (videos)', COUNT(*) FROM comments WHERE commentable_type = 'video';

-- Drop old tables (after verification)
-- DROP TABLE post_comments;
-- DROP TABLE video_comments;

COMMIT;
```

Polymorphic associations are powerful but complex. Use them judiciously and always prioritize data integrity and query performance over flexibility.
- the rails way (creating a `polymorphic_type` and `polymorphic_id` column. Don't do this. Even though it is simple, you really lose the referential integrity. Deleting the associations will also leave the association "hanging")
- multiple database
- table inheritance
- using union to provide interface for polymorphic types!
- multiple foreign keys, with a constraint to check the type

https://hashrocket.com/blog/posts/modeling-polymorphic-associations-in-a-relational-database
http://duhallowgreygeek.com/polymorphic-association-bad-sql-smell/
https://www.vertabelo.com/blog/inheritance-in-a-relational-database/
https://stackoverflow.com/questions/5466163/same-data-from-different-entities-in-database-best-practice-phone-numbers-ex/5471265#5471265


## Achieving true polymorphic?

This is inspired from graphql [global object identification](https://graphql.org/learn/global-object-identification/) implementation, where all entities inherit a single node and each entity has a unique id.


```sql
create table if not exists pg_temp.node (
	id uuid not null default gen_random_uuid(),
	type text not null check (type in ('feed', 'post', 'comment', 'human')),
	primary key (id, type),
	unique (id)
);

create or replace function pg_temp.gen_node_id(_type text) returns uuid as $$
	declare
		_id uuid;
	begin
		insert into pg_temp.node (type) values (_type)
		returning id into _id;
		return _id;
	end;
$$ language plpgsql;


create table if not exists pg_temp.human (
	id uuid not null default pg_temp.gen_node_id('human'),
	type text not null default 'human' check (type = 'human'),
	name text not null,
	email text not null,
	unique (email),
	primary key(id), -- We don't need to define the 'type' here as primary key, because the constraint will have already been applied in the node table.
	foreign key(id, type) references node(id, type)
);

insert into pg_temp.human(name, email) values ('john', 'john.doe@mail.com');


create table if not exists pg_temp.feed (
	id uuid not null default pg_temp.gen_node_id('feed'),
	type text not null default 'feed' check (type = 'feed'),
	user_id uuid not null,
	body text not null,
	primary key(id),
	foreign key(id, type) references pg_temp.node(id, type),
	foreign key(user_id) references pg_temp.human(id)
);

create table if not exists pg_temp.post (
	id uuid not null default pg_temp.gen_node_id('post'),
	type text not null default 'post' check (type = 'post'),
	user_id uuid not null,
	body text not null,
	primary key(id),
	foreign key(id, type) references pg_temp.node(id, type),
	foreign key(user_id) references pg_temp.human(id)
);

create table if not exists pg_temp.comment (
	id uuid not null default pg_temp.gen_node_id('comment'),
	type text not null default 'comment' check (type = 'comment'),
	user_id uuid not null,
	body text not null,
	commentable_id uuid not null,
	commentable_type text not null check (commentable_type in ('post', 'feed')),
	primary key(id),
	foreign key(id, type) references node(id, type),
	foreign key(commentable_id, commentable_type) references node(id, type),
	foreign key(user_id) references pg_temp.human(id)
);


insert into pg_temp.feed(user_id, body) values 
((select id from pg_temp.human), 'this is a new feed');

insert into pg_temp.post(user_id, body) values 
((select id from pg_temp.human), 'this is a new post');

insert into pg_temp.comment (commentable_id, commentable_type, user_id, body) values
((select id from pg_temp.post), 'post', (select id from pg_temp.human), 'this is a comment on post');

insert into pg_temp.comment (commentable_id, commentable_type, user_id, body) values
((select id from pg_temp.feed), 'feed', (select id from pg_temp.human), 'this is a comment on feed');
```

Polymorphic table comment:
```sql
select * from pg_temp.comment;
```


| id | type | user_id | body | commentable_id | commentable_type |
| -- | ---- | --------| ---- | -------------- | ---------------- |
| 31e12f91-ea42-492f-988e-3eb17bb1b9dc	| comment	| 396b8bae-fff9-44cc-b28e-49a22625a671	| this is a comment on post	| 09b8ccb0-8dd9-4a8e-b0d6-a28b296ac7a4	| post| 
| 6704d0cb-4d83-41c0-b3eb-1d32bc05452f| 	comment| 	396b8bae-fff9-44cc-b28e-49a22625a671| 	this is a comment on feed| 	b64b571b-02ae-4c55-b786-cc6764b40eda| 	feed| 
