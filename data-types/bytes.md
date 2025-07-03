# Binary Data and Bytes: Complete Guide

Binary data storage is essential for handling files, images, cryptographic data, and optimized storage formats. This guide covers binary data types, storage strategies, and optimization techniques.

## Table of Contents
- [Binary Data Types](#binary-data-types)
- [Storage Strategies](#storage-strategies)
- [UUID Storage Optimization](#uuid-storage-optimization)
- [File and Media Storage](#file-and-media-storage)
- [Cryptographic Data](#cryptographic-data)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Binary Data Types

### PostgreSQL Binary Types

```sql
-- PostgreSQL binary data types
CREATE TABLE binary_data_examples (
    id SERIAL PRIMARY KEY,
    
    -- Variable-length binary string
    file_data BYTEA,                    -- Stores any binary data
    
    -- Fixed-length binary (rare usage)
    checksum_md5 BYTEA CHECK (LENGTH(checksum_md5) = 16),
    checksum_sha256 BYTEA CHECK (LENGTH(checksum_sha256) = 32),
    
    -- UUID stored as binary (16 bytes vs 36 characters)
    uuid_binary BYTEA CHECK (LENGTH(uuid_binary) = 16),
    uuid_text UUID DEFAULT gen_random_uuid(), -- Native UUID type
    
    -- Binary representation of images/files
    thumbnail BYTEA,
    document BYTEA,
    
    -- Metadata
    content_type TEXT,
    file_size INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### MySQL Binary Types

```sql
-- MySQL binary data types
CREATE TABLE mysql_binary_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    
    -- Fixed-length binary strings
    hash_md5 BINARY(16),                -- Exactly 16 bytes
    hash_sha1 BINARY(20),               -- Exactly 20 bytes
    uuid_binary BINARY(16),             -- UUID as 16 bytes
    
    -- Variable-length binary strings  
    small_file VARBINARY(255),          -- Up to 255 bytes
    
    -- BLOB types for larger binary data
    thumbnail BLOB,                     -- Up to 65,535 bytes
    image_data MEDIUMBLOB,              -- Up to 16,777,215 bytes  
    video_data LONGBLOB,                -- Up to 4,294,967,295 bytes
    
    -- Metadata
    filename VARCHAR(255),
    mime_type VARCHAR(100),
    file_size BIGINT UNSIGNED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Storage Size Comparison

```sql
-- Storage efficiency comparison
CREATE TABLE storage_comparison (
    id SERIAL PRIMARY KEY,
    
    -- UUID storage comparison
    uuid_text TEXT,           -- 36 characters = 36 bytes + length overhead
    uuid_native UUID,         -- 16 bytes (PostgreSQL native)
    uuid_binary BYTEA,        -- 16 bytes + length overhead
    
    -- Hash storage comparison  
    sha256_hex TEXT,          -- 64 characters = 64 bytes
    sha256_binary BYTEA,      -- 32 bytes + length overhead
    
    -- Integer vs binary comparison
    user_id INTEGER,          -- 4 bytes
    user_id_binary BYTEA,     -- 4 bytes + length overhead
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Demonstrate storage efficiency
INSERT INTO storage_comparison (
    uuid_text, 
    uuid_native, 
    uuid_binary,
    sha256_hex,
    sha256_binary,
    user_id,
    user_id_binary
) VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    '550e8400-e29b-41d4-a716-446655440000'::UUID,
    decode('550e8400e29b41d4a716446655440000', 'hex'),
    'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3',
    decode('a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', 'hex'),
    12345,
    decode(to_hex(12345), 'hex')
);
```

## Storage Strategies

### 1. Database vs External Storage

```sql
-- Decision matrix for binary data storage
CREATE TABLE file_storage_strategy (
    id SERIAL PRIMARY KEY,
    file_name TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    
    -- Store small files directly in database
    file_data BYTEA, -- For files < 1MB
    
    -- Store large files externally with metadata
    external_url TEXT, -- S3, CloudFlare, etc.
    external_key TEXT, -- Storage service key
    storage_provider TEXT CHECK (storage_provider IN ('local', 's3', 'gcs', 'azure')),
    
    -- Hybrid approach: store thumbnail in DB, full image externally
    thumbnail_data BYTEA,
    
    -- File integrity
    checksum_sha256 BYTEA,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure either internal or external storage
    CONSTRAINT storage_location_check 
    CHECK (
        (file_data IS NOT NULL AND external_url IS NULL) OR
        (file_data IS NULL AND external_url IS NOT NULL)
    )
);
```

### 2. File Metadata Pattern

```sql
-- Comprehensive file metadata management
CREATE TABLE file_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- File identification
    original_filename TEXT NOT NULL,
    internal_filename TEXT UNIQUE NOT NULL,
    file_extension TEXT NOT NULL,
    
    -- Content information
    mime_type TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    
    -- Storage information
    storage_path TEXT,
    storage_bucket TEXT,
    storage_region TEXT,
    
    -- Image-specific metadata (JSON for flexibility)
    image_metadata JSONB,
    
    -- File integrity and security
    checksum_md5 BYTEA,
    checksum_sha256 BYTEA,
    virus_scan_status TEXT CHECK (virus_scan_status IN ('pending', 'clean', 'infected', 'error')),
    virus_scan_date TIMESTAMPTZ,
    
    -- Access control
    uploaded_by INTEGER REFERENCES users(id),
    is_public BOOLEAN DEFAULT FALSE,
    access_permissions JSONB,
    
    -- Audit trail
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT positive_file_size CHECK (file_size > 0),
    CONSTRAINT valid_extension CHECK (file_extension ~ '^[a-z0-9]+$')
);

-- Image-specific metadata example
INSERT INTO file_metadata (
    original_filename, internal_filename, file_extension,
    mime_type, file_size, image_metadata,
    checksum_sha256, uploaded_by
) VALUES (
    'vacation-photo.jpg',
    'img_2024_07_04_12345.jpg', 
    'jpg',
    'image/jpeg',
    2048576,
    '{
        "width": 1920,
        "height": 1080,
        "color_space": "sRGB",
        "compression": "JPEG",
        "exif": {
            "camera": "Canon EOS R5",
            "lens": "RF 24-70mm f/2.8L",
            "iso": 400,
            "aperture": "f/5.6",
            "shutter_speed": "1/125"
        }
    }'::JSONB,
    decode('a1b2c3d4e5f6...', 'hex'),
    123
);
```

## UUID Storage Optimization

### Binary UUID Functions

```sql
-- UUID conversion functions for optimal storage
CREATE OR REPLACE FUNCTION uuid_to_binary(uuid_val UUID)
RETURNS BYTEA AS $$
BEGIN
    RETURN decode(replace(uuid_val::TEXT, '-', ''), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION binary_to_uuid(binary_val BYTEA)
RETURNS UUID AS $$
DECLARE
    hex_str TEXT;
BEGIN
    hex_str := encode(binary_val, 'hex');
    RETURN (
        substr(hex_str, 1, 8) || '-' ||
        substr(hex_str, 9, 4) || '-' ||
        substr(hex_str, 13, 4) || '-' ||
        substr(hex_str, 17, 4) || '-' ||
        substr(hex_str, 21, 12)
    )::UUID;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Optimized UUID storage table
CREATE TABLE uuid_optimized (
    id BYTEA PRIMARY KEY DEFAULT uuid_to_binary(gen_random_uuid()),
    data TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Helper view for easy UUID access
CREATE VIEW uuid_optimized_view AS
SELECT 
    binary_to_uuid(id) as id,
    data,
    created_at
FROM uuid_optimized;

-- Insert using the view
INSERT INTO uuid_optimized (data) VALUES ('Sample data');

-- Query using the view
SELECT * FROM uuid_optimized_view WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

### MySQL UUID Binary Optimization

```sql
-- MySQL UUID binary functions (MySQL 8.0+)
DELIMITER //

CREATE FUNCTION uuid_to_bin_ordered(uuid_string CHAR(36))
RETURNS BINARY(16)
DETERMINISTIC
SQL SECURITY INVOKER
RETURN UNHEX(CONCAT(
    SUBSTR(uuid_string, 15, 4),   -- time_mid
    SUBSTR(uuid_string, 10, 4),   -- time_low high
    SUBSTR(uuid_string, 1, 8),    -- time_low low
    SUBSTR(uuid_string, 20, 4),   -- time_hi_and_version
    SUBSTR(uuid_string, 25)       -- clock_seq_and_node
))//

CREATE FUNCTION bin_to_uuid_ordered(binary_uuid BINARY(16))
RETURNS CHAR(36)
DETERMINISTIC
SQL SECURITY INVOKER
RETURN LOWER(CONCAT(
    SUBSTR(HEX(binary_uuid), 9, 8), '-',
    SUBSTR(HEX(binary_uuid), 5, 4), '-',
    SUBSTR(HEX(binary_uuid), 1, 4), '-',
    SUBSTR(HEX(binary_uuid), 17, 4), '-',
    SUBSTR(HEX(binary_uuid), 21)
))//

DELIMITER ;

-- Optimized table using ordered binary UUIDs
CREATE TABLE mysql_uuid_optimized (
    id BINARY(16) PRIMARY KEY DEFAULT (uuid_to_bin_ordered(UUID())),
    data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_created_at (created_at)
);
```

## File and Media Storage

### Image Storage with Variants

```sql
-- Multi-variant image storage
CREATE TABLE image_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_image_id UUID NOT NULL,
    variant_type TEXT NOT NULL CHECK (variant_type IN ('original', 'thumbnail', 'medium', 'large', 'webp', 'avif')),
    
    -- Image data (for small variants) or external reference
    image_data BYTEA,
    external_url TEXT,
    
    -- Variant specifications
    width INTEGER,
    height INTEGER,
    quality INTEGER CHECK (quality BETWEEN 1 AND 100),
    format TEXT NOT NULL,
    
    -- File information
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(base_image_id, variant_type),
    
    CONSTRAINT storage_location_check 
    CHECK (
        (image_data IS NOT NULL AND external_url IS NULL) OR
        (image_data IS NULL AND external_url IS NOT NULL)
    )
);

-- Function to create image variants
CREATE OR REPLACE FUNCTION create_image_variants(
    p_base_image_id UUID,
    p_original_data BYTEA,
    p_mime_type TEXT
) RETURNS VOID AS $$
BEGIN
    -- Store original
    INSERT INTO image_variants (
        base_image_id, variant_type, image_data, 
        mime_type, file_size
    ) VALUES (
        p_base_image_id, 'original', p_original_data,
        p_mime_type, LENGTH(p_original_data)
    );
    
    -- In real implementation, you would:
    -- 1. Use external service to generate variants
    -- 2. Store smaller variants in DB, larger ones externally
    -- 3. This is a placeholder for the pattern
    
    -- Example: Create thumbnail entry (would be generated externally)
    INSERT INTO image_variants (
        base_image_id, variant_type, external_url,
        width, height, format, mime_type, file_size
    ) VALUES (
        p_base_image_id, 'thumbnail', 
        'https://cdn.example.com/thumbnails/' || p_base_image_id || '.webp',
        150, 150, 'webp', 'image/webp', 8192
    );
END;
$$ LANGUAGE plpgsql;
```

### Document Storage with Versions

```sql
-- Document versioning with binary storage
CREATE TABLE document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    
    -- Document content
    document_data BYTEA,
    external_storage_key TEXT,
    
    -- Document metadata
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    
    -- Content analysis
    text_content TEXT, -- Extracted text for search
    page_count INTEGER,
    
    -- Checksums for integrity
    checksum_md5 BYTEA,
    checksum_sha256 BYTEA,
    
    -- Version information
    version_notes TEXT,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(document_id, version_number),
    
    CONSTRAINT positive_version CHECK (version_number > 0),
    CONSTRAINT positive_file_size CHECK (file_size > 0)
);

-- Full-text search on documents
CREATE INDEX idx_document_text_search ON document_versions 
USING GIN(to_tsvector('english', text_content));
```

## Cryptographic Data

### Hash Storage and Verification

```sql
-- Secure hash storage patterns
CREATE TABLE password_hashes (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    
    -- Password hash (bcrypt, scrypt, argon2)
    password_hash BYTEA NOT NULL,
    hash_algorithm TEXT NOT NULL DEFAULT 'bcrypt',
    hash_cost INTEGER NOT NULL DEFAULT 12,
    
    -- Salt (if not included in hash)
    salt BYTEA,
    
    -- Security metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_changed TIMESTAMPTZ DEFAULT NOW(),
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,
    
    CONSTRAINT valid_hash_algorithm 
    CHECK (hash_algorithm IN ('bcrypt', 'scrypt', 'argon2', 'pbkdf2'))
);

-- API key storage with hashing
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id),
    
    -- Store hash of key, not the key itself
    key_hash BYTEA NOT NULL,
    key_prefix TEXT NOT NULL, -- First few characters for identification
    
    -- Key metadata
    name TEXT NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    
    -- Access control
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    usage_count BIGINT DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(key_hash),
    UNIQUE(user_id, name)
);

-- Secure key generation function
CREATE OR REPLACE FUNCTION generate_api_key()
RETURNS TEXT AS $$
DECLARE
    key_bytes BYTEA;
    key_string TEXT;
BEGIN
    -- Generate 32 random bytes
    key_bytes := gen_random_bytes(32);
    
    -- Encode as base64url (URL-safe)
    key_string := translate(encode(key_bytes, 'base64'), '+/', '-_');
    key_string := rtrim(key_string, '='); -- Remove padding
    
    RETURN key_string;
END;
$$ LANGUAGE plpgsql;
```

### Encryption Key Management

```sql
-- Key derivation and storage
CREATE TABLE encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Key identification
    key_name TEXT UNIQUE NOT NULL,
    key_purpose TEXT NOT NULL CHECK (key_purpose IN ('data', 'backup', 'transport')),
    
    -- Encrypted key material (encrypted with master key)
    encrypted_key BYTEA NOT NULL,
    key_derivation_info JSONB NOT NULL,
    
    -- Key metadata
    algorithm TEXT NOT NULL DEFAULT 'AES-256-GCM',
    key_length INTEGER NOT NULL DEFAULT 256,
    
    -- Key lifecycle
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    rotated_from UUID REFERENCES encryption_keys(id),
    is_active BOOLEAN DEFAULT TRUE,
    
    CONSTRAINT valid_key_length CHECK (key_length IN (128, 192, 256)),
    CONSTRAINT future_expiry CHECK (expires_at IS NULL OR expires_at > created_at)
);
```

## Performance Considerations

### Indexing Binary Data

```sql
-- Indexing strategies for binary data
CREATE TABLE performance_binary (
    id SERIAL PRIMARY KEY,
    
    -- Hash indexes for exact matches
    file_hash BYTEA,
    
    -- Binary UUID with B-tree index
    uuid_binary BYTEA,
    
    -- Partial content for prefix matching
    content_prefix BYTEA,
    
    -- Full binary content
    content_data BYTEA
);

-- Indexes for different access patterns
CREATE INDEX idx_file_hash ON performance_binary USING HASH (file_hash);
CREATE UNIQUE INDEX idx_uuid_binary ON performance_binary (uuid_binary);
CREATE INDEX idx_content_prefix ON performance_binary (content_prefix);

-- For prefix matching on binary data
CREATE INDEX idx_content_prefix_btree ON performance_binary (substring(content_data, 1, 16));
```

### Large Object Handling

```sql
-- PostgreSQL: Large object handling for very large files
CREATE TABLE large_files (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    
    -- PostgreSQL large object OID
    file_oid OID,
    
    -- Alternative: chunked storage
    chunk_count INTEGER,
    total_size BIGINT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chunked binary storage for streaming
CREATE TABLE file_chunks (
    file_id INTEGER REFERENCES large_files(id),
    chunk_number INTEGER NOT NULL,
    chunk_data BYTEA NOT NULL,
    chunk_size INTEGER NOT NULL,
    
    PRIMARY KEY (file_id, chunk_number),
    
    CONSTRAINT valid_chunk_number CHECK (chunk_number >= 0),
    CONSTRAINT valid_chunk_size CHECK (chunk_size > 0 AND chunk_size <= 1048576) -- 1MB max
);
```

## Best Practices

### 1. Choose Appropriate Storage

```sql
-- ✅ Good: Size-based storage decisions
CREATE TABLE smart_file_storage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    
    -- Small files: store in database
    small_file_data BYTEA, -- < 1MB
    
    -- Large files: external storage
    external_url TEXT,     -- >= 1MB
    
    -- Always store metadata
    mime_type TEXT NOT NULL,
    checksum_sha256 BYTEA NOT NULL,
    
    CONSTRAINT size_based_storage CHECK (
        (file_size < 1048576 AND small_file_data IS NOT NULL AND external_url IS NULL) OR
        (file_size >= 1048576 AND small_file_data IS NULL AND external_url IS NOT NULL)
    )
);
```

### 2. Data Integrity

```sql
-- Always include integrity checks
CREATE TABLE integrity_example (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data BYTEA NOT NULL,
    
    -- Multiple hash algorithms for security
    md5_hash BYTEA CHECK (LENGTH(md5_hash) = 16),
    sha256_hash BYTEA CHECK (LENGTH(sha256_hash) = 32),
    
    -- Content validation
    data_size INTEGER GENERATED ALWAYS AS (LENGTH(data)) STORED,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Trigger to automatically generate hashes
CREATE OR REPLACE FUNCTION generate_hashes()
RETURNS TRIGGER AS $$
BEGIN
    NEW.md5_hash := decode(md5(NEW.data), 'hex');
    NEW.sha256_hash := digest(NEW.data, 'sha256');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER auto_generate_hashes
    BEFORE INSERT OR UPDATE ON integrity_example
    FOR EACH ROW EXECUTE FUNCTION generate_hashes();
```

### 3. Security Considerations

```sql
-- Secure binary data handling
CREATE TABLE secure_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Encrypted content (never store sensitive data unencrypted)
    encrypted_content BYTEA NOT NULL,
    encryption_algorithm TEXT NOT NULL DEFAULT 'AES-256-GCM',
    
    -- Key reference (not the actual key)
    encryption_key_id UUID REFERENCES encryption_keys(id),
    
    -- Content metadata (can be unencrypted)
    content_type TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    
    -- Access control
    owner_id INTEGER REFERENCES users(id),
    access_level TEXT DEFAULT 'private' CHECK (access_level IN ('private', 'shared', 'public')),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 4. Monitoring and Maintenance

```sql
-- Monitor binary data usage
CREATE VIEW binary_storage_stats AS
SELECT 
    schemaname,
    tablename,
    attname as column_name,
    avg_width as avg_bytes,
    n_distinct,
    correlation
FROM pg_stats 
WHERE atttypid = 'bytea'::regtype;

-- Cleanup old binary data
CREATE OR REPLACE FUNCTION cleanup_old_files(
    older_than INTERVAL DEFAULT '1 year'
) RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Archive or delete old files
    WITH deleted AS (
        DELETE FROM file_metadata
        WHERE created_at < NOW() - older_than
        AND is_archived = FALSE
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

## Conclusion

Effective binary data handling requires:

1. **Storage Strategy**: Database for small files, external storage for large files
2. **Data Integrity**: Always use checksums and validation
3. **Security**: Encrypt sensitive data, hash credentials properly
4. **Performance**: Choose appropriate indexing and chunking strategies
5. **Optimization**: Use binary formats for UUIDs and hashes when possible
6. **Monitoring**: Track storage usage and implement cleanup procedures

The key is balancing performance, security, and maintainability while choosing the right storage approach for your specific use case.
