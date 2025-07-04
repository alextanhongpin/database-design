# Image Storage and Management: Complete Guide

Proper image handling in database applications involves choosing the right storage strategy, managing metadata, optimizing performance, and implementing proper image processing workflows.

## Table of Contents
- [Storage Strategies](#storage-strategies)
- [Image Metadata Management](#image-metadata-management)
- [Multi-Variant Image Handling](#multi-variant-image-handling)
- [Database Schema Patterns](#database-schema-patterns)
- [Performance Optimization](#performance-optimization)
- [Security Considerations](#security-considerations)
- [Best Practices](#best-practices)

## Storage Strategies

### 1. External Storage (Recommended)

```sql
-- Store only metadata and URLs in database
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- File identification
    original_filename TEXT NOT NULL,
    internal_filename TEXT UNIQUE NOT NULL,
    file_extension TEXT NOT NULL CHECK (file_extension IN ('jpg', 'jpeg', 'png', 'webp', 'avif', 'gif')),
    
    -- Storage location
    storage_provider TEXT NOT NULL CHECK (storage_provider IN ('s3', 'gcs', 'azure', 'cloudinary', 'local')),
    storage_bucket TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    storage_region TEXT,
    
    -- CDN information
    cdn_url TEXT,
    cdn_cache_ttl INTEGER DEFAULT 86400, -- 24 hours
    
    -- Image properties
    width INTEGER NOT NULL CHECK (width > 0),
    height INTEGER NOT NULL CHECK (height > 0),
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    mime_type TEXT NOT NULL,
    
    -- Quality and processing
    quality INTEGER CHECK (quality BETWEEN 1 AND 100),
    is_optimized BOOLEAN DEFAULT FALSE,
    
    -- Metadata
    uploaded_by INTEGER REFERENCES users(id),
    upload_session_id UUID,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_mime_type CHECK (
        mime_type IN ('image/jpeg', 'image/png', 'image/webp', 'image/avif', 'image/gif')
    )
);

-- Indexes for common queries
CREATE INDEX idx_images_uploaded_by ON images(uploaded_by);
CREATE INDEX idx_images_storage_provider ON images(storage_provider, storage_bucket);
CREATE INDEX idx_images_mime_type ON images(mime_type);
CREATE INDEX idx_images_created_at ON images(created_at);
```

### 2. Database Storage (Limited Use Cases)

```sql
-- Store small images directly in database (< 1MB)
CREATE TABLE image_blobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Image data
    image_data BYTEA NOT NULL,
    
    -- Metadata
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    
    -- Checksums for integrity
    md5_hash BYTEA NOT NULL,
    sha256_hash BYTEA NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Size constraint for database storage
    CONSTRAINT reasonable_size CHECK (file_size <= 1048576), -- 1MB max
    
    -- Generate hashes automatically
    CONSTRAINT valid_md5 CHECK (length(md5_hash) = 16),
    CONSTRAINT valid_sha256 CHECK (length(sha256_hash) = 32)
);

-- Trigger to generate hashes
CREATE OR REPLACE FUNCTION generate_image_hashes()
RETURNS TRIGGER AS $$
BEGIN
    NEW.md5_hash := decode(md5(NEW.image_data), 'hex');
    NEW.sha256_hash := digest(NEW.image_data, 'sha256');
    NEW.file_size := length(NEW.image_data);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER auto_generate_image_hashes
    BEFORE INSERT OR UPDATE ON image_blobs
    FOR EACH ROW EXECUTE FUNCTION generate_image_hashes();
```

### 3. Hybrid Storage Strategy

```sql
-- Decision-based storage: small images in DB, large images external
CREATE TABLE smart_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    
    -- For small images (< 100KB)
    thumbnail_data BYTEA,
    
    -- For large images
    external_url TEXT,
    storage_provider TEXT,
    storage_key TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure either internal or external storage
    CONSTRAINT storage_choice CHECK (
        (file_size < 102400 AND thumbnail_data IS NOT NULL AND external_url IS NULL) OR
        (file_size >= 102400 AND thumbnail_data IS NULL AND external_url IS NOT NULL)
    )
);
```

## Image Metadata Management

### Comprehensive Metadata Schema

```sql
-- Detailed image metadata with EXIF support
CREATE TABLE image_metadata (
    image_id UUID PRIMARY KEY REFERENCES images(id) ON DELETE CASCADE,
    
    -- Basic image properties
    color_space TEXT, -- 'sRGB', 'Adobe RGB', 'P3', etc.
    bit_depth INTEGER CHECK (bit_depth IN (8, 16, 24, 32)),
    compression_type TEXT,
    
    -- Camera EXIF data (stored as JSONB for flexibility)
    exif_data JSONB,
    
    -- Geographic information
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    altitude DECIMAL(8, 2),
    
    -- Image processing history
    processing_operations JSONB DEFAULT '[]',
    
    -- Content analysis (can be populated by AI services)
    detected_objects JSONB,
    detected_faces INTEGER DEFAULT 0,
    content_tags TEXT[],
    
    -- Accessibility
    alt_text TEXT,
    caption TEXT,
    
    -- Copyright and licensing
    copyright_notice TEXT,
    license_type TEXT,
    attribution_required BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for metadata searches
CREATE INDEX idx_image_metadata_tags ON image_metadata USING GIN(content_tags);
CREATE INDEX idx_image_metadata_location ON image_metadata (latitude, longitude) 
WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX idx_image_metadata_faces ON image_metadata (detected_faces) 
WHERE detected_faces > 0;
```

### EXIF Data Handling

```sql
-- Example EXIF data structure
INSERT INTO image_metadata (image_id, exif_data) VALUES 
('550e8400-e29b-41d4-a716-446655440000', '{
    "camera": {
        "make": "Canon",
        "model": "EOS R5",
        "serial_number": "123456789"
    },
    "lens": {
        "make": "Canon",
        "model": "RF 24-70mm f/2.8L IS USM",
        "focal_length": "50mm",
        "aperture": "f/2.8"
    },
    "settings": {
        "iso": 400,
        "shutter_speed": "1/125",
        "exposure_mode": "Manual",
        "white_balance": "Auto",
        "flash": "Off"
    },
    "datetime": {
        "captured": "2024-07-04T14:30:00Z",
        "digitized": "2024-07-04T14:30:00Z",
        "modified": "2024-07-04T14:31:15Z"
    },
    "technical": {
        "color_space": "sRGB",
        "resolution_x": 72,
        "resolution_y": 72,
        "resolution_unit": "inches"
    }
}'::JSONB);

-- Query EXIF data
SELECT 
    i.filename,
    m.exif_data->>'camera'->>'make' as camera_make,
    m.exif_data->>'camera'->>'model' as camera_model,
    m.exif_data->>'settings'->>'iso' as iso,
    m.exif_data->>'lens'->>'focal_length' as focal_length
FROM images i
JOIN image_metadata m ON i.id = m.image_id
WHERE m.exif_data->>'camera'->>'make' = 'Canon';
```

## Multi-Variant Image Handling

### Image Variants System

```sql
-- Manage multiple sizes/formats of the same image
CREATE TABLE image_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    
    -- Variant classification
    variant_type TEXT NOT NULL CHECK (variant_type IN (
        'original', 'thumbnail', 'small', 'medium', 'large', 'xl',
        'webp', 'avif', 'progressive', 'placeholder'
    )),
    
    -- Format and quality
    format TEXT NOT NULL CHECK (format IN ('jpeg', 'png', 'webp', 'avif')),
    quality INTEGER CHECK (quality BETWEEN 1 AND 100),
    
    -- Dimensions
    width INTEGER NOT NULL CHECK (width > 0),
    height INTEGER NOT NULL CHECK (height > 0),
    
    -- Storage information
    file_size BIGINT NOT NULL,
    storage_url TEXT NOT NULL,
    
    -- Processing information
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    generation_method TEXT, -- 'upload', 'resize', 'convert', 'ai_upscale'
    
    -- Performance hints
    lazy_load_priority INTEGER DEFAULT 0, -- 0=low, 1=normal, 2=high
    preload_hint BOOLEAN DEFAULT FALSE,
    
    UNIQUE(base_image_id, variant_type, format)
);

-- Function to get responsive image sources
CREATE OR REPLACE FUNCTION get_responsive_image_sources(p_image_id UUID)
RETURNS TABLE(
    variant_type TEXT,
    format TEXT,
    width INTEGER,
    height INTEGER,
    url TEXT,
    file_size BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        iv.variant_type,
        iv.format,
        iv.width,
        iv.height,
        iv.storage_url,
        iv.file_size
    FROM image_variants iv
    WHERE iv.base_image_id = p_image_id
    ORDER BY 
        CASE iv.variant_type
            WHEN 'thumbnail' THEN 1
            WHEN 'small' THEN 2 
            WHEN 'medium' THEN 3
            WHEN 'large' THEN 4
            WHEN 'xl' THEN 5
            ELSE 6
        END,
        iv.format;
END;
$$ LANGUAGE plpgsql;
```

### Responsive Image Generation

```sql
-- Configuration for automatic variant generation
CREATE TABLE image_variant_configs (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    
    -- Size specifications
    max_width INTEGER NOT NULL,
    max_height INTEGER NOT NULL,
    resize_mode TEXT DEFAULT 'fit' CHECK (resize_mode IN ('fit', 'fill', 'crop', 'scale')),
    
    -- Quality settings per format
    jpeg_quality INTEGER DEFAULT 85 CHECK (jpeg_quality BETWEEN 1 AND 100),
    webp_quality INTEGER DEFAULT 80 CHECK (webp_quality BETWEEN 1 AND 100),
    avif_quality INTEGER DEFAULT 75 CHECK (avif_quality BETWEEN 1 AND 100),
    
    -- Generation rules
    auto_generate BOOLEAN DEFAULT TRUE,
    generate_webp BOOLEAN DEFAULT TRUE,
    generate_avif BOOLEAN DEFAULT FALSE,
    
    -- Use case
    description TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Standard responsive image sizes
INSERT INTO image_variant_configs (name, max_width, max_height, description) VALUES
('thumbnail', 150, 150, 'Small thumbnail for listings'),
('small', 320, 240, 'Mobile portrait view'),
('medium', 768, 576, 'Tablet and small desktop'),
('large', 1200, 900, 'Desktop and large screens'),
('xl', 1920, 1440, 'High-resolution displays'),
('hero', 2560, 1440, 'Hero banners and full-screen');
```

## Database Schema Patterns

### Single vs Multiple Images

```sql
-- Pattern 1: Single image per entity
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE REFERENCES users(id),
    avatar_image_id UUID REFERENCES images(id),
    cover_image_id UUID REFERENCES images(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pattern 2: Multiple images per entity
CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id INTEGER NOT NULL REFERENCES products(id),
    image_id UUID NOT NULL REFERENCES images(id),
    
    -- Ordering and categorization
    display_order INTEGER NOT NULL DEFAULT 0,
    image_type TEXT NOT NULL DEFAULT 'gallery' CHECK (
        image_type IN ('primary', 'gallery', 'thumbnail', 'detail', 'lifestyle')
    ),
    
    -- Visibility
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(product_id, image_id),
    
    -- Only one primary image per product
    CONSTRAINT unique_primary_image 
    EXCLUDE (product_id WITH =) WHERE (image_type = 'primary' AND is_active = TRUE)
);

-- Pattern 3: Flexible image associations
CREATE TABLE entity_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id),
    
    -- Polymorphic association
    entity_type TEXT NOT NULL, -- 'user', 'product', 'article', etc.
    entity_id TEXT NOT NULL,   -- Can reference any entity
    
    -- Association details
    relationship_type TEXT NOT NULL, -- 'avatar', 'gallery', 'attachment', etc.
    display_order INTEGER DEFAULT 0,
    
    -- Metadata specific to the association
    caption TEXT,
    alt_text TEXT,
    is_primary BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Composite indexes
    INDEX idx_entity_images_entity (entity_type, entity_id),
    INDEX idx_entity_images_relationship (entity_type, relationship_type),
    
    -- Unique primary image per entity/relationship
    CONSTRAINT unique_primary_per_relationship
    EXCLUDE (entity_type WITH =, entity_id WITH =, relationship_type WITH =)
    WHERE (is_primary = TRUE)
);
```

### Image Collections and Albums

```sql
-- Image collections/albums
CREATE TABLE image_collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    
    -- Collection properties
    is_public BOOLEAN DEFAULT FALSE,
    sort_order TEXT DEFAULT 'date_desc' CHECK (
        sort_order IN ('date_asc', 'date_desc', 'name_asc', 'name_desc', 'manual')
    ),
    
    -- Access control
    owner_id INTEGER REFERENCES users(id),
    
    -- Cover image
    cover_image_id UUID REFERENCES images(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE collection_images (
    collection_id UUID REFERENCES image_collections(id) ON DELETE CASCADE,
    image_id UUID REFERENCES images(id) ON DELETE CASCADE,
    
    -- Position for manual sorting
    position INTEGER DEFAULT 0,
    
    -- Individual image settings within collection
    caption TEXT,
    is_featured BOOLEAN DEFAULT FALSE,
    
    added_at TIMESTAMPTZ DEFAULT NOW(),
    added_by INTEGER REFERENCES users(id),
    
    PRIMARY KEY (collection_id, image_id)
);
```

## Performance Optimization

### Efficient Image Serving

```sql
-- Image serving optimization table
CREATE TABLE image_serving_stats (
    image_id UUID REFERENCES images(id),
    variant_type TEXT NOT NULL,
    
    -- Performance metrics
    total_requests BIGINT DEFAULT 0,
    total_bytes_served BIGINT DEFAULT 0,
    cache_hit_ratio DECIMAL(5,4) DEFAULT 0.0000,
    avg_response_time_ms INTEGER DEFAULT 0,
    
    -- Time-based metrics
    last_accessed TIMESTAMPTZ,
    last_cache_refresh TIMESTAMPTZ,
    
    -- Optimization hints
    should_preload BOOLEAN DEFAULT FALSE,
    cdn_cache_ttl INTEGER DEFAULT 86400,
    
    PRIMARY KEY (image_id, variant_type)
);

-- Function to get optimized image URL
CREATE OR REPLACE FUNCTION get_optimized_image_url(
    p_image_id UUID,
    p_max_width INTEGER DEFAULT NULL,
    p_format TEXT DEFAULT 'auto'
) RETURNS TEXT AS $$
DECLARE
    best_variant RECORD;
    result_url TEXT;
BEGIN
    -- Find the best matching variant
    SELECT iv.storage_url, iv.width, iv.format
    INTO best_variant
    FROM image_variants iv
    WHERE iv.base_image_id = p_image_id
      AND (p_max_width IS NULL OR iv.width <= p_max_width)
      AND (p_format = 'auto' OR iv.format = p_format)
    ORDER BY 
        CASE WHEN p_format = 'auto' THEN
            CASE iv.format
                WHEN 'avif' THEN 1  -- Prefer AVIF
                WHEN 'webp' THEN 2  -- Then WebP
                WHEN 'jpeg' THEN 3  -- Then JPEG
                ELSE 4
            END
        ELSE 1
        END,
        ABS(iv.width - COALESCE(p_max_width, iv.width))
    LIMIT 1;
    
    result_url := COALESCE(best_variant.storage_url, '');
    
    -- Update serving stats
    INSERT INTO image_serving_stats (image_id, variant_type, total_requests, last_accessed)
    VALUES (p_image_id, 'optimized', 1, NOW())
    ON CONFLICT (image_id, variant_type)
    DO UPDATE SET 
        total_requests = image_serving_stats.total_requests + 1,
        last_accessed = NOW();
    
    RETURN result_url;
END;
$$ LANGUAGE plpgsql;
```

### Caching Strategy

```sql
-- Image cache management
CREATE TABLE image_cache_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id),
    cache_key TEXT UNIQUE NOT NULL,
    
    -- Cache parameters
    width INTEGER,
    height INTEGER,
    format TEXT,
    quality INTEGER,
    
    -- Cache metadata
    cached_url TEXT NOT NULL,
    cache_size BIGINT,
    cache_region TEXT,
    
    -- Cache lifecycle
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_accessed TIMESTAMPTZ,
    access_count BIGINT DEFAULT 0,
    
    -- Cache status
    is_valid BOOLEAN DEFAULT TRUE
);

-- Cleanup expired cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM image_cache_entries
    WHERE expires_at < NOW() OR is_valid = FALSE;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

## Security Considerations

### Upload Validation

```sql
-- File upload validation and security
CREATE TABLE upload_security_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- File information
    original_filename TEXT NOT NULL,
    detected_mime_type TEXT,
    file_size BIGINT,
    file_hash TEXT, -- SHA-256
    
    -- Security checks
    virus_scan_result TEXT CHECK (virus_scan_result IN ('clean', 'infected', 'suspicious', 'error')),
    content_validation_result TEXT CHECK (content_validation_result IN ('valid', 'invalid', 'suspicious')),
    
    -- Metadata validation
    exif_stripped BOOLEAN DEFAULT FALSE,
    contains_metadata BOOLEAN DEFAULT FALSE,
    
    -- Upload context
    uploaded_by INTEGER REFERENCES users(id),
    upload_ip INET,
    user_agent TEXT,
    
    -- Processing results
    upload_allowed BOOLEAN DEFAULT FALSE,
    rejection_reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to validate image upload
CREATE OR REPLACE FUNCTION validate_image_upload(
    p_filename TEXT,
    p_mime_type TEXT,
    p_file_size BIGINT,
    p_uploaded_by INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    is_valid BOOLEAN := TRUE;
    rejection_reasons TEXT[] := '{}';
BEGIN
    -- File size validation
    IF p_file_size > 10485760 THEN -- 10MB
        is_valid := FALSE;
        rejection_reasons := array_append(rejection_reasons, 'File too large');
    END IF;
    
    -- MIME type validation
    IF p_mime_type NOT IN ('image/jpeg', 'image/png', 'image/webp', 'image/gif') THEN
        is_valid := FALSE;
        rejection_reasons := array_append(rejection_reasons, 'Invalid file type');
    END IF;
    
    -- Filename validation
    IF p_filename ~ '\.(php|exe|bat|sh|cmd)$' THEN
        is_valid := FALSE;
        rejection_reasons := array_append(rejection_reasons, 'Dangerous file extension');
    END IF;
    
    -- Log the validation attempt
    INSERT INTO upload_security_logs (
        original_filename, detected_mime_type, file_size, uploaded_by,
        upload_allowed, rejection_reason
    ) VALUES (
        p_filename, p_mime_type, p_file_size, p_uploaded_by,
        is_valid, array_to_string(rejection_reasons, '; ')
    );
    
    RETURN is_valid;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Storage Strategy Selection

```sql
-- Decision matrix for storage strategy
-- ✅ External storage for:
-- - Images > 1MB
-- - Public-facing images
-- - Images requiring CDN
-- - High-traffic applications

-- ✅ Database storage for:
-- - Small images < 100KB (thumbnails, icons)
-- - Private/secure images
-- - Images requiring ACID transactions

-- ✅ Hybrid approach:
-- - Thumbnails in database, full images external
-- - Critical metadata in database, bulk data external
```

### 2. Performance Optimization

```sql
-- Index strategy for image queries
CREATE INDEX CONCURRENTLY idx_images_user_created ON images(uploaded_by, created_at DESC);
CREATE INDEX CONCURRENTLY idx_images_type_size ON images(mime_type, file_size);
CREATE INDEX CONCURRENTLY idx_variants_base_type ON image_variants(base_image_id, variant_type);

-- Partial indexes for common filters
CREATE INDEX CONCURRENTLY idx_images_recent ON images(created_at) 
WHERE created_at >= NOW() - INTERVAL '30 days';
```

### 3. Monitoring and Maintenance

```sql
-- Storage usage monitoring
CREATE VIEW image_storage_summary AS
SELECT 
    storage_provider,
    COUNT(*) as image_count,
    SUM(file_size) as total_size_bytes,
    ROUND(SUM(file_size) / 1024.0 / 1024.0, 2) as total_size_mb,
    AVG(file_size) as avg_size_bytes,
    MIN(created_at) as oldest_image,
    MAX(created_at) as newest_image
FROM images
GROUP BY storage_provider;

-- Cleanup orphaned images
CREATE OR REPLACE FUNCTION cleanup_orphaned_images()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete image variants with no base image
    DELETE FROM image_variants 
    WHERE base_image_id NOT IN (SELECT id FROM images);
    
    -- Delete metadata with no image
    DELETE FROM image_metadata 
    WHERE image_id NOT IN (SELECT id FROM images);
    
    -- Count and return deleted images
    WITH deleted AS (
        DELETE FROM images 
        WHERE id NOT IN (
            SELECT DISTINCT image_id FROM entity_images WHERE image_id IS NOT NULL
            UNION
            SELECT DISTINCT avatar_image_id FROM user_profiles WHERE avatar_image_id IS NOT NULL
            UNION
            SELECT DISTINCT cover_image_id FROM user_profiles WHERE cover_image_id IS NOT NULL
        )
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

## Conclusion

Effective image management requires:

1. **Storage Strategy**: Choose between database, external storage, or hybrid based on use case
2. **Metadata Management**: Store comprehensive metadata for search and organization
3. **Multi-Format Support**: Generate multiple variants for responsive design and performance
4. **Security**: Validate uploads and protect against malicious files
5. **Performance**: Implement proper indexing, caching, and CDN strategies
6. **Maintenance**: Regular cleanup of orphaned images and cache management

The key is balancing storage costs, performance requirements, and feature needs while maintaining data integrity and security.
