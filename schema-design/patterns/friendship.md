# Friendship Database Design Patterns

## Table of Contents
- [Overview](#overview)
- [Core Patterns](#core-patterns)
- [Friendship Status Management](#friendship-status-management)
- [Query Optimization](#query-optimization)
- [Real-World Examples](#real-world-examples)
- [Advanced Patterns](#advanced-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Overview

Friendship systems are fundamental to social networks and collaborative platforms. The main challenge is efficiently modeling bidirectional relationships while maintaining data integrity and query performance.

### Key Challenges
- **Bidirectional relationships**: Friends can see each other's content
- **Status management**: Pending, accepted, blocked states
- **Query complexity**: Finding mutual friends, friend suggestions
- **Performance**: Scaling to millions of relationships

## Core Patterns

### Pattern 1: Single Row with Hash (Recommended)

This pattern uses a computed hash to ensure unique bidirectional relationships.

```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Friendship table with computed hash
CREATE TABLE friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Status management
    status friendship_status NOT NULL DEFAULT 'pending',
    
    -- Computed hash for unique bidirectional relationships
    relationship_hash TEXT GENERATED ALWAYS AS (
        MD5(
            CASE 
                WHEN user_id < friend_id 
                THEN user_id::text || friend_id::text
                ELSE friend_id::text || user_id::text
            END
        )
    ) STORED,
    
    -- Metadata
    requested_by UUID NOT NULL REFERENCES users(id),
    requested_at TIMESTAMP DEFAULT NOW(),
    responded_at TIMESTAMP,
    
    -- Constraints
    CHECK (user_id != friend_id),
    UNIQUE (relationship_hash)
);

-- Custom enum for friendship status
CREATE TYPE friendship_status AS ENUM (
    'pending',
    'accepted', 
    'rejected',
    'blocked'
);
```

### Pattern 2: Dual Row Approach

Creates two rows for each friendship for easier querying.

```sql
CREATE TABLE friendships_dual (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status friendship_status NOT NULL DEFAULT 'pending',
    
    -- Relationship metadata
    is_initiator BOOLEAN NOT NULL DEFAULT FALSE,
    requested_at TIMESTAMP DEFAULT NOW(),
    responded_at TIMESTAMP,
    
    -- Constraints
    CHECK (user_id != friend_id),
    UNIQUE (user_id, friend_id)
);

-- Trigger to maintain dual entries
CREATE OR REPLACE FUNCTION maintain_dual_friendship()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Insert reciprocal relationship
        INSERT INTO friendships_dual (
            user_id, friend_id, status, is_initiator, 
            requested_at, responded_at
        ) VALUES (
            NEW.friend_id, NEW.user_id, NEW.status, FALSE,
            NEW.requested_at, NEW.responded_at
        ) ON CONFLICT (user_id, friend_id) DO UPDATE SET
            status = NEW.status,
            responded_at = NEW.responded_at;
    
    ELSIF TG_OP = 'UPDATE' THEN
        -- Update reciprocal relationship
        UPDATE friendships_dual 
        SET status = NEW.status,
            responded_at = NEW.responded_at
        WHERE user_id = NEW.friend_id AND friend_id = NEW.user_id;
    
    ELSIF TG_OP = 'DELETE' THEN
        -- Delete reciprocal relationship
        DELETE FROM friendships_dual 
        WHERE user_id = OLD.friend_id AND friend_id = OLD.user_id;
        RETURN OLD;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER friendship_dual_trigger
    AFTER INSERT OR UPDATE OR DELETE ON friendships_dual
    FOR EACH ROW EXECUTE FUNCTION maintain_dual_friendship();
```

## Friendship Status Management

### Status Transitions

```sql
-- Function to handle friendship requests
CREATE OR REPLACE FUNCTION request_friendship(
    requester_id UUID,
    requested_id UUID
) RETURNS UUID AS $$
DECLARE
    friendship_id UUID;
    existing_hash TEXT;
BEGIN
    -- Check if friendship already exists
    SELECT relationship_hash INTO existing_hash
    FROM friendships
    WHERE relationship_hash = MD5(
        CASE 
            WHEN requester_id < requested_id 
            THEN requester_id::text || requested_id::text
            ELSE requested_id::text || requester_id::text
        END
    );
    
    IF existing_hash IS NOT NULL THEN
        RAISE EXCEPTION 'Friendship already exists or was previously rejected';
    END IF;
    
    -- Create new friendship request
    INSERT INTO friendships (user_id, friend_id, requested_by, status)
    VALUES (requester_id, requested_id, requester_id, 'pending')
    RETURNING id INTO friendship_id;
    
    RETURN friendship_id;
END;
$$ LANGUAGE plpgsql;

-- Function to respond to friendship request
CREATE OR REPLACE FUNCTION respond_to_friendship(
    responder_id UUID,
    requester_id UUID,
    response friendship_status
) RETURNS BOOLEAN AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    -- Update friendship status
    UPDATE friendships 
    SET status = response,
        responded_at = NOW()
    WHERE (
        (user_id = requester_id AND friend_id = responder_id) OR
        (user_id = responder_id AND friend_id = requester_id)
    ) 
    AND status = 'pending';
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    
    IF updated_count = 0 THEN
        RAISE EXCEPTION 'No pending friendship request found';
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## Query Optimization

### Essential Indexes

```sql
-- Primary indexes for friendship queries
CREATE INDEX idx_friendships_hash ON friendships (relationship_hash);
CREATE INDEX idx_friendships_user_status ON friendships (user_id, status);
CREATE INDEX idx_friendships_friend_status ON friendships (friend_id, status);
CREATE INDEX idx_friendships_requested_by ON friendships (requested_by, requested_at);

-- Partial indexes for common queries
CREATE INDEX idx_friendships_pending ON friendships (user_id, requested_at) 
WHERE status = 'pending';

CREATE INDEX idx_friendships_accepted ON friendships (user_id, friend_id) 
WHERE status = 'accepted';
```

### Common Queries

```sql
-- Get all friends for a user
SELECT u.id, u.username, u.display_name, f.status
FROM friendships f
JOIN users u ON (
    CASE 
        WHEN f.user_id = $1 THEN u.id = f.friend_id
        ELSE u.id = f.user_id
    END
)
WHERE (f.user_id = $1 OR f.friend_id = $1)
  AND f.status = 'accepted';

-- Get pending friend requests sent by user
SELECT u.id, u.username, u.display_name, f.requested_at
FROM friendships f
JOIN users u ON u.id = CASE 
    WHEN f.user_id = $1 THEN f.friend_id
    ELSE f.user_id
END
WHERE f.requested_by = $1 
  AND f.status = 'pending';

-- Get pending friend requests received by user
SELECT u.id, u.username, u.display_name, f.requested_at
FROM friendships f
JOIN users u ON u.id = f.requested_by
WHERE (f.user_id = $1 OR f.friend_id = $1)
  AND f.requested_by != $1
  AND f.status = 'pending';

-- Check if two users are friends
SELECT EXISTS(
    SELECT 1 FROM friendships 
    WHERE relationship_hash = MD5(
        CASE 
            WHEN $1 < $2 THEN $1::text || $2::text
            ELSE $2::text || $1::text
        END
    )
    AND status = 'accepted'
) AS are_friends;

-- Find mutual friends
WITH user_friends AS (
    SELECT CASE 
        WHEN user_id = $1 THEN friend_id
        ELSE user_id
    END AS friend_id
    FROM friendships
    WHERE (user_id = $1 OR friend_id = $1)
      AND status = 'accepted'
),
other_friends AS (
    SELECT 
        CASE 
            WHEN user_id = tf.friend_id THEN f.friend_id
            ELSE f.user_id
        END AS candidate_id,
        COUNT(*) AS mutual_count
    FROM target_friends tf
    JOIN friendships f ON (f.user_id = tf.friend_id OR f.friend_id = tf.friend_id)
    WHERE f.status = 'accepted'
      AND CASE 
            WHEN f.user_id = tf.friend_id THEN f.friend_id
            ELSE f.user_id
          END != target_user_id
    GROUP BY candidate_id
)
SELECT u.id, u.username, u.display_name
FROM user_friends uf
JOIN other_friends of ON uf.friend_id = of.friend_id
JOIN users u ON u.id = uf.friend_id;
```

## Real-World Examples

### Social Media Platform

```sql
-- Extended friendship table for social media
CREATE TABLE social_friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Enhanced status management
    status friendship_status NOT NULL DEFAULT 'pending',
    visibility_level visibility_type DEFAULT 'normal',
    
    -- Social features
    is_close_friend BOOLEAN DEFAULT FALSE,
    is_favorite BOOLEAN DEFAULT FALSE,
    custom_label VARCHAR(50),
    
    -- Computed hash
    relationship_hash TEXT GENERATED ALWAYS AS (
        MD5(
            CASE 
                WHEN user_id < friend_id 
                THEN user_id::text || friend_id::text
                ELSE friend_id::text || user_id::text
            END
        )
    ) STORED,
    
    -- Metadata
    requested_by UUID NOT NULL REFERENCES users(id),
    requested_at TIMESTAMP DEFAULT NOW(),
    responded_at TIMESTAMP,
    last_interaction_at TIMESTAMP,
    
    -- Constraints
    CHECK (user_id != friend_id),
    UNIQUE (relationship_hash)
);

-- Visibility levels
CREATE TYPE visibility_type AS ENUM ('hidden', 'limited', 'normal', 'close');

-- Friend list management
CREATE TABLE friend_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (user_id, name)
);

-- Many-to-many relationship for friend lists
CREATE TABLE friend_list_members (
    friend_list_id UUID REFERENCES friend_lists(id) ON DELETE CASCADE,
    friendship_id UUID REFERENCES social_friendships(id) ON DELETE CASCADE,
    added_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (friend_list_id, friendship_id)
);
```

### Professional Network

```sql
-- Professional networking friendship model
CREATE TABLE professional_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Connection type
    connection_type connection_type NOT NULL DEFAULT 'colleague',
    
    -- Professional context
    company_name VARCHAR(255),
    shared_company_id UUID REFERENCES companies(id),
    introduction_message TEXT,
    
    -- Status and hash
    status friendship_status NOT NULL DEFAULT 'pending',
    relationship_hash TEXT GENERATED ALWAYS AS (
        MD5(
            CASE 
                WHEN user_id < connection_id 
                THEN user_id::text || connection_id::text
                ELSE connection_id::text || user_id::text
            END
        )
    ) STORED,
    
    -- Metadata
    requested_by UUID NOT NULL REFERENCES users(id),
    requested_at TIMESTAMP DEFAULT NOW(),
    connected_at TIMESTAMP,
    
    -- Constraints
    CHECK (user_id != connection_id),
    UNIQUE (relationship_hash)
);

-- Professional connection types
CREATE TYPE connection_type AS ENUM (
    'colleague', 'manager', 'report', 'client', 
    'vendor', 'mentor', 'mentee', 'classmate'
);
```

## Advanced Patterns

### Friendship Analytics

```sql
-- Friendship statistics view
CREATE VIEW friendship_stats AS
SELECT 
    u.id AS user_id,
    u.username,
    COUNT(CASE WHEN f.status = 'accepted' THEN 1 END) AS friend_count,
    COUNT(CASE WHEN f.status = 'pending' AND f.requested_by = u.id THEN 1 END) AS sent_requests,
    COUNT(CASE WHEN f.status = 'pending' AND f.requested_by != u.id THEN 1 END) AS received_requests,
    COUNT(CASE WHEN f.status = 'blocked' THEN 1 END) AS blocked_count,
    MAX(f.responded_at) AS last_activity_at
FROM users u
LEFT JOIN friendships f ON (u.id = f.user_id OR u.id = f.friend_id)
GROUP BY u.id, u.username;

-- Friend suggestion algorithm
CREATE OR REPLACE FUNCTION suggest_friends(
    target_user_id UUID,
    limit_count INTEGER DEFAULT 10
) RETURNS TABLE(
    user_id UUID,
    username VARCHAR(255),
    mutual_friends_count INTEGER,
    suggestion_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH target_friends AS (
        SELECT CASE 
            WHEN f.user_id = target_user_id THEN f.friend_id
            ELSE f.user_id
        END AS friend_id
        FROM friendships f
        WHERE (f.user_id = target_user_id OR f.friend_id = target_user_id)
          AND f.status = 'accepted'
    ),
    friend_of_friends AS (
        SELECT 
            CASE 
                WHEN f.user_id = tf.friend_id THEN f.friend_id
                ELSE f.user_id
            END AS candidate_id,
            COUNT(*) AS mutual_count
        FROM target_friends tf
        JOIN friendships f ON (f.user_id = tf.friend_id OR f.friend_id = tf.friend_id)
        WHERE f.status = 'accepted'
          AND CASE 
                WHEN f.user_id = tf.friend_id THEN f.friend_id
                ELSE f.user_id
              END != target_user_id
        GROUP BY candidate_id
    )
    SELECT 
        u.id,
        u.username,
        fof.mutual_count::INTEGER,
        (fof.mutual_count * 1.0 + RANDOM() * 0.1)::NUMERIC AS score
    FROM friend_of_friends fof
    JOIN users u ON u.id = fof.candidate_id
    WHERE NOT EXISTS (
        SELECT 1 FROM friendships f
        WHERE (
            (f.user_id = target_user_id AND f.friend_id = fof.candidate_id) OR
            (f.user_id = fof.candidate_id AND f.friend_id = target_user_id)
        )
    )
    ORDER BY score DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Scaling Strategies

1. **Partitioning**: Partition friendships table by user_id hash
```sql
-- Partition by user_id hash
CREATE TABLE friendships_partitioned (
    LIKE friendships INCLUDING ALL
) PARTITION BY HASH (user_id);

-- Create partitions
CREATE TABLE friendships_p0 PARTITION OF friendships_partitioned
FOR VALUES WITH (modulus 4, remainder 0);

CREATE TABLE friendships_p1 PARTITION OF friendships_partitioned
FOR VALUES WITH (modulus 4, remainder 1);

CREATE TABLE friendships_p2 PARTITION OF friendships_partitioned
FOR VALUES WITH (modulus 4, remainder 2);

CREATE TABLE friendships_p3 PARTITION OF friendships_partitioned
FOR VALUES WITH (modulus 4, remainder 3);
```

2. **Caching Strategy**: Cache friend lists and mutual friend counts
```sql
-- Materialized view for friend counts
CREATE MATERIALIZED VIEW friendship_counts AS
SELECT 
    user_id,
    COUNT(*) as friend_count
FROM (
    SELECT user_id FROM friendships WHERE status = 'accepted'
    UNION ALL
    SELECT friend_id FROM friendships WHERE status = 'accepted'
) friends
GROUP BY user_id;

-- Refresh periodically
CREATE INDEX ON friendship_counts (user_id);
```

## Best Practices

### 1. Data Integrity
- Always use check constraints to prevent self-friendships
- Implement proper foreign key constraints
- Use computed hash columns to prevent duplicate relationships
- Consider using triggers for complex business logic

### 2. Performance
- Index on status and user combinations for common queries
- Use partial indexes for status-specific queries
- Consider partitioning for very large datasets
- Cache friend counts and mutual friend data

### 3. Privacy and Security
- Implement proper access controls
- Consider visibility levels (public, friends, private)
- Log friendship events for audit trails
- Handle blocked users appropriately

### 4. Scalability
- Design for eventual sharding
- Use UUIDs for globally unique identifiers
- Consider async processing for friend suggestions
- Monitor query performance and optimize indexes

### 5. User Experience
- Provide clear status indicators
- Implement friend suggestion algorithms
- Support friend lists and groups
- Allow users to control privacy settings

### Migration Strategy

```sql
-- Example migration from simple to hash-based approach
BEGIN;

-- Add new columns
ALTER TABLE friendships 
ADD COLUMN relationship_hash TEXT,
ADD COLUMN requested_by UUID,
ADD COLUMN requested_at TIMESTAMP DEFAULT NOW();

-- Populate hash for existing data
UPDATE friendships SET relationship_hash = MD5(
    CASE 
        WHEN user_id < friend_id 
        THEN user_id::text || friend_id::text
        ELSE friend_id::text || user_id::text
    END
);

-- Add constraints
ALTER TABLE friendships 
ADD CONSTRAINT unique_relationship_hash UNIQUE (relationship_hash),
ALTER COLUMN relationship_hash SET NOT NULL;

-- Remove duplicate relationships (keep newest)
DELETE FROM friendships f1 
WHERE f1.id < (
    SELECT MAX(f2.id) 
    FROM friendships f2 
    WHERE f2.relationship_hash = f1.relationship_hash
);

COMMIT;
```

This comprehensive friendship system provides a solid foundation for social features while maintaining performance and data integrity at scale.
