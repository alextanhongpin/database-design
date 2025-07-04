# Vector Embeddings in Database Design

Vector embeddings are numerical representations of data (text, images, audio) that capture semantic meaning. This guide covers storage strategies, versioning, and best practices for embedding-based applications.

## Introduction to Vector Embeddings

Embeddings transform high-dimensional data into dense vector representations that can be used for:
- Semantic search
- Recommendation systems
- Similarity detection
- Content classification
- Clustering and analysis

## Storage Strategies

### Using pgvector Extension

PostgreSQL's `pgvector` extension provides efficient vector storage and operations:

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Basic embedding storage
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    embedding VECTOR(1536), -- OpenAI embedding dimension
    
    -- Metadata
    document_type VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for efficient similarity search
CREATE INDEX ON documents USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Alternative: HNSW index for better recall
CREATE INDEX ON documents USING hnsw (embedding vector_cosine_ops);
```

### Multi-Model Embedding Storage

Handle different embedding models and dimensions:

```sql
-- Embedding models registry
CREATE TABLE embedding_models (
    id SERIAL PRIMARY KEY,
    model_name VARCHAR(100) NOT NULL UNIQUE,
    model_version VARCHAR(50) NOT NULL,
    dimensions INTEGER NOT NULL,
    provider VARCHAR(50) NOT NULL, -- 'openai', 'huggingface', 'sentence-transformers'
    
    -- Model metadata
    max_input_length INTEGER,
    cost_per_token DECIMAL(10,8),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    deprecated_at TIMESTAMPTZ,
    
    UNIQUE(model_name, model_version)
);

-- Flexible embedding storage
CREATE TABLE document_embeddings (
    id SERIAL PRIMARY KEY,
    document_id INTEGER NOT NULL,
    model_id INTEGER NOT NULL,
    embedding VECTOR NOT NULL, -- Dynamic dimensions
    
    -- Chunk information for large documents
    chunk_index INTEGER DEFAULT 0,
    chunk_content TEXT,
    chunk_start_pos INTEGER,
    chunk_end_pos INTEGER,
    
    -- Generation metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generation_cost DECIMAL(10,6),
    processing_time_ms INTEGER,
    
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE,
    FOREIGN KEY (model_id) REFERENCES embedding_models(id),
    
    UNIQUE(document_id, model_id, chunk_index)
);
```

## Handling Embedding Changes

### Document Updates

When documents change, embeddings need to be regenerated:

```sql
-- Track document changes
CREATE TABLE document_change_log (
    id SERIAL PRIMARY KEY,
    document_id INTEGER NOT NULL,
    change_type VARCHAR(20) NOT NULL, -- 'create', 'update', 'delete'
    old_content_hash VARCHAR(64),
    new_content_hash VARCHAR(64),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Embedding regeneration status
    embeddings_updated BOOLEAN DEFAULT FALSE,
    embeddings_updated_at TIMESTAMPTZ,
    
    FOREIGN KEY (document_id) REFERENCES documents(id)
);

-- Function to handle document updates
CREATE OR REPLACE FUNCTION handle_document_update()
RETURNS TRIGGER AS $$
DECLARE
    content_hash VARCHAR(64);
BEGIN
    -- Calculate content hash
    content_hash := encode(sha256(NEW.content::bytea), 'hex');
    
    -- Log the change
    INSERT INTO document_change_log (
        document_id, change_type, 
        old_content_hash, new_content_hash
    ) VALUES (
        NEW.id, 'update',
        encode(sha256(OLD.content::bytea), 'hex'),
        content_hash
    );
    
    -- Mark embeddings as outdated
    UPDATE document_embeddings 
    SET embedding = NULL  -- Or mark with a flag
    WHERE document_id = NEW.id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for document updates
CREATE TRIGGER document_update_trigger
    AFTER UPDATE ON documents
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content)
    EXECUTE FUNCTION handle_document_update();
```

### Model Version Updates

Handle embedding model changes gracefully:

```sql
-- Migration strategy for model updates
CREATE OR REPLACE FUNCTION migrate_embeddings_to_new_model(
    p_old_model_id INTEGER,
    p_new_model_id INTEGER
)
RETURNS INTEGER AS $$
DECLARE
    migrated_count INTEGER := 0;
    doc_record RECORD;
BEGIN
    -- Process documents in batches
    FOR doc_record IN 
        SELECT DISTINCT document_id 
        FROM document_embeddings 
        WHERE model_id = p_old_model_id
    LOOP
        -- Create new embedding entries (to be populated by external process)
        INSERT INTO document_embeddings (
            document_id, model_id, chunk_index, chunk_content,
            chunk_start_pos, chunk_end_pos
        )
        SELECT 
            document_id, p_new_model_id, chunk_index, chunk_content,
            chunk_start_pos, chunk_end_pos
        FROM document_embeddings
        WHERE document_id = doc_record.document_id 
          AND model_id = p_old_model_id
        ON CONFLICT (document_id, model_id, chunk_index) DO NOTHING;
        
        migrated_count := migrated_count + 1;
    END LOOP;
    
    RETURN migrated_count;
END;
$$ LANGUAGE plpgsql;
```

## Chunking Strategies

### Fixed-Size Chunking

```sql
-- Function to split documents into chunks
CREATE OR REPLACE FUNCTION create_document_chunks(
    p_document_id INTEGER,
    p_chunk_size INTEGER DEFAULT 1000,
    p_overlap INTEGER DEFAULT 100
)
RETURNS INTEGER AS $$
DECLARE
    doc_content TEXT;
    chunk_start INTEGER := 1;
    chunk_end INTEGER;
    chunk_index INTEGER := 0;
    chunks_created INTEGER := 0;
BEGIN
    -- Get document content
    SELECT content INTO doc_content FROM documents WHERE id = p_document_id;
    
    -- Clear existing chunks
    DELETE FROM document_embeddings WHERE document_id = p_document_id;
    
    -- Create chunks
    WHILE chunk_start <= LENGTH(doc_content) LOOP
        chunk_end := LEAST(chunk_start + p_chunk_size - 1, LENGTH(doc_content));
        
        INSERT INTO document_embeddings (
            document_id, model_id, chunk_index, chunk_content,
            chunk_start_pos, chunk_end_pos
        ) VALUES (
            p_document_id, 1, chunk_index, -- Default to first model
            SUBSTRING(doc_content FROM chunk_start FOR chunk_end - chunk_start + 1),
            chunk_start, chunk_end
        );
        
        chunks_created := chunks_created + 1;
        chunk_index := chunk_index + 1;
        chunk_start := chunk_end - p_overlap + 1;
    END LOOP;
    
    RETURN chunks_created;
END;
$$ LANGUAGE plpgsql;
```

### Semantic Chunking

```sql
-- Store semantic boundaries
CREATE TABLE document_sections (
    id SERIAL PRIMARY KEY,
    document_id INTEGER NOT NULL,
    section_type VARCHAR(50), -- 'paragraph', 'sentence', 'section'
    start_pos INTEGER NOT NULL,
    end_pos INTEGER NOT NULL,
    content TEXT NOT NULL,
    
    -- Hierarchy
    parent_section_id INTEGER,
    section_order INTEGER,
    
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_section_id) REFERENCES document_sections(id)
);

-- Link embeddings to sections
ALTER TABLE document_embeddings 
ADD COLUMN section_id INTEGER,
ADD FOREIGN KEY (section_id) REFERENCES document_sections(id);
```

## Similarity Search and Querying

### Basic Similarity Search

```sql
-- Find similar documents
CREATE OR REPLACE FUNCTION find_similar_documents(
    p_query_embedding VECTOR,
    p_limit INTEGER DEFAULT 10,
    p_threshold FLOAT DEFAULT 0.7
)
RETURNS TABLE(
    document_id INTEGER,
    title VARCHAR(500),
    similarity_score FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        d.id,
        d.title,
        1 - (de.embedding <=> p_query_embedding) as similarity
    FROM documents d
    JOIN document_embeddings de ON d.id = de.document_id
    WHERE de.embedding IS NOT NULL
      AND (1 - (de.embedding <=> p_query_embedding)) > p_threshold
    ORDER BY de.embedding <=> p_query_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

### Advanced Filtering

```sql
-- Semantic search with metadata filtering
CREATE OR REPLACE FUNCTION semantic_search(
    p_query_embedding VECTOR,
    p_document_types VARCHAR[] DEFAULT NULL,
    p_date_from TIMESTAMPTZ DEFAULT NULL,
    p_date_to TIMESTAMPTZ DEFAULT NULL,
    p_limit INTEGER DEFAULT 10
)
RETURNS TABLE(
    document_id INTEGER,
    title VARCHAR(500),
    document_type VARCHAR(50),
    similarity_score FLOAT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        d.id,
        d.title,
        d.document_type,
        1 - (de.embedding <=> p_query_embedding) as similarity,
        d.created_at
    FROM documents d
    JOIN document_embeddings de ON d.id = de.document_id
    WHERE de.embedding IS NOT NULL
      AND (p_document_types IS NULL OR d.document_type = ANY(p_document_types))
      AND (p_date_from IS NULL OR d.created_at >= p_date_from)
      AND (p_date_to IS NULL OR d.created_at <= p_date_to)
    ORDER BY de.embedding <=> p_query_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

## Hybrid Search

Combine semantic and keyword search:

```sql
-- Add full-text search capabilities
ALTER TABLE documents ADD COLUMN search_vector TSVECTOR;

-- Update search vector
UPDATE documents SET search_vector = to_tsvector('english', title || ' ' || content);

-- Create index for full-text search
CREATE INDEX ON documents USING gin(search_vector);

-- Hybrid search function
CREATE OR REPLACE FUNCTION hybrid_search(
    p_query_text TEXT,
    p_query_embedding VECTOR,
    p_semantic_weight FLOAT DEFAULT 0.7,
    p_keyword_weight FLOAT DEFAULT 0.3,
    p_limit INTEGER DEFAULT 10
)
RETURNS TABLE(
    document_id INTEGER,
    title VARCHAR(500),
    semantic_score FLOAT,
    keyword_score FLOAT,
    combined_score FLOAT
) AS $$
BEGIN
    RETURN QUERY
    WITH semantic_results AS (
        SELECT 
            d.id,
            d.title,
            1 - (de.embedding <=> p_query_embedding) as semantic_score
        FROM documents d
        JOIN document_embeddings de ON d.id = de.document_id
        WHERE de.embedding IS NOT NULL
    ),
    keyword_results AS (
        SELECT 
            d.id,
            d.title,
            ts_rank(d.search_vector, plainto_tsquery('english', p_query_text)) as keyword_score
        FROM documents d
        WHERE d.search_vector @@ plainto_tsquery('english', p_query_text)
    )
    SELECT 
        COALESCE(s.id, k.id) as document_id,
        COALESCE(s.title, k.title) as title,
        COALESCE(s.semantic_score, 0.0) as semantic_score,
        COALESCE(k.keyword_score, 0.0) as keyword_score,
        (COALESCE(s.semantic_score, 0.0) * p_semantic_weight + 
         COALESCE(k.keyword_score, 0.0) * p_keyword_weight) as combined_score
    FROM semantic_results s
    FULL OUTER JOIN keyword_results k ON s.id = k.id
    ORDER BY combined_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

## Performance Optimization

### Indexing Strategies

```sql
-- Different index types for different use cases
-- IVFFlat: Good for recall/speed balance
CREATE INDEX documents_embedding_ivfflat_idx 
ON document_embeddings USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- HNSW: Better recall, more memory usage
CREATE INDEX documents_embedding_hnsw_idx 
ON document_embeddings USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Partial indexes for active embeddings
CREATE INDEX documents_embedding_active_idx 
ON document_embeddings USING ivfflat (embedding vector_cosine_ops)
WHERE embedding IS NOT NULL;
```

### Query Optimization

```sql
-- Materialized view for frequent searches
CREATE MATERIALIZED VIEW popular_document_embeddings AS
SELECT 
    de.document_id,
    de.embedding,
    d.title,
    d.document_type,
    d.created_at,
    s.search_count
FROM document_embeddings de
JOIN documents d ON de.document_id = d.id
JOIN (
    SELECT document_id, COUNT(*) as search_count
    FROM search_logs
    WHERE created_at >= NOW() - INTERVAL '30 days'
    GROUP BY document_id
    HAVING COUNT(*) >= 10
) s ON de.document_id = s.document_id
WHERE de.embedding IS NOT NULL;

-- Index on materialized view
CREATE INDEX ON popular_document_embeddings USING ivfflat (embedding vector_cosine_ops);

-- Refresh schedule
CREATE OR REPLACE FUNCTION refresh_popular_embeddings()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW popular_document_embeddings;
END;
$$ LANGUAGE plpgsql;
```

## Monitoring and Analytics

### Embedding Quality Metrics

```sql
-- Track embedding generation metrics
CREATE TABLE embedding_metrics (
    id SERIAL PRIMARY KEY,
    model_id INTEGER NOT NULL,
    
    -- Generation metrics
    documents_processed INTEGER NOT NULL,
    total_tokens INTEGER,
    total_cost DECIMAL(10,6),
    avg_processing_time_ms INTEGER,
    
    -- Quality metrics
    avg_embedding_norm DECIMAL(10,6),
    embedding_variance DECIMAL(10,6),
    
    -- Time period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    
    FOREIGN KEY (model_id) REFERENCES embedding_models(id)
);

-- Search performance tracking
CREATE TABLE search_performance (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(64) NOT NULL,
    query_type VARCHAR(20) NOT NULL, -- 'semantic', 'keyword', 'hybrid'
    
    -- Performance metrics
    execution_time_ms INTEGER NOT NULL,
    results_count INTEGER NOT NULL,
    
    -- Quality metrics
    avg_similarity_score DECIMAL(10,6),
    click_through_rate DECIMAL(5,4),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Migration and Cleanup

### Embedding Cleanup

```sql
-- Remove outdated embeddings
CREATE OR REPLACE FUNCTION cleanup_old_embeddings()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete embeddings for deprecated models
    DELETE FROM document_embeddings de
    WHERE de.model_id IN (
        SELECT id FROM embedding_models 
        WHERE deprecated_at < NOW() - INTERVAL '30 days'
    );
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Delete embeddings for deleted documents
    DELETE FROM document_embeddings de
    WHERE NOT EXISTS (
        SELECT 1 FROM documents d WHERE d.id = de.document_id
    );
    
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

### Batch Processing

```sql
-- Queue for embedding generation
CREATE TABLE embedding_queue (
    id SERIAL PRIMARY KEY,
    document_id INTEGER NOT NULL,
    model_id INTEGER NOT NULL,
    priority INTEGER DEFAULT 0,
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Error handling
    error_message TEXT,
    
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE,
    FOREIGN KEY (model_id) REFERENCES embedding_models(id)
);

-- Process embedding queue
CREATE OR REPLACE FUNCTION process_embedding_queue(p_batch_size INTEGER DEFAULT 100)
RETURNS INTEGER AS $$
DECLARE
    processed_count INTEGER := 0;
    queue_record RECORD;
BEGIN
    FOR queue_record IN
        SELECT * FROM embedding_queue
        WHERE status = 'pending'
        ORDER BY priority DESC, created_at ASC
        LIMIT p_batch_size
    LOOP
        -- Mark as processing
        UPDATE embedding_queue 
        SET status = 'processing', started_at = NOW()
        WHERE id = queue_record.id;
        
        -- Process would happen in external application
        -- For now, just mark as completed
        UPDATE embedding_queue 
        SET status = 'completed', completed_at = NOW()
        WHERE id = queue_record.id;
        
        processed_count := processed_count + 1;
    END LOOP;
    
    RETURN processed_count;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

1. **Use appropriate dimensions** - Balance between accuracy and storage/performance
2. **Implement chunking** - Handle large documents efficiently
3. **Version your models** - Support multiple embedding models simultaneously
4. **Monitor quality** - Track embedding generation metrics and search performance
5. **Optimize indexes** - Choose the right index type for your use case
6. **Handle updates gracefully** - Implement strategies for content and model changes
7. **Use hybrid search** - Combine semantic and keyword search for better results
8. **Clean up regularly** - Remove outdated embeddings and optimize storage

## Common Pitfalls

- Not handling document updates properly
- Using wrong similarity metrics
- Poor chunking strategies
- Inadequate indexing
- Not versioning embedding models
- Ignoring performance monitoring
- Insufficient error handling in batch processing
- Not considering storage costs

This comprehensive approach ensures robust, scalable, and maintainable vector embedding systems in your database design.

