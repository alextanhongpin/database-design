# Multilingual Database Design: Complete Guide

Designing databases to support multiple languages requires careful consideration of storage patterns, query efficiency, and maintenance complexity. This guide covers various approaches to internationalization (i18n) and localization (l10n) in database design.

## Table of Contents
- [Design Approaches](#design-approaches)
- [Translation Table Pattern](#translation-table-pattern)
- [Column-Based Approach](#column-based-approach)
- [JSON-Based Storage](#json-based-storage)
- [Real-World Examples](#real-world-examples)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Design Approaches

### 1. Separate Translation Tables (Recommended)

Store base entity data separately from translations:

```sql
-- Base entity table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category_id INTEGER,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Supported languages
CREATE TABLE languages (
    code CHAR(2) PRIMARY KEY, -- ISO 639-1
    name TEXT NOT NULL,
    native_name TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE
);

INSERT INTO languages (code, name, native_name, is_default) VALUES
('en', 'English', 'English', TRUE),
('es', 'Spanish', 'Español', FALSE),
('fr', 'French', 'Français', FALSE),
('de', 'German', 'Deutsch', FALSE),
('zh', 'Chinese', '中文', FALSE);

-- Translation table
CREATE TABLE product_translations (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    name TEXT NOT NULL,
    description TEXT,
    short_description TEXT,
    meta_title TEXT,
    meta_description TEXT,
    
    UNIQUE(product_id, language_code)
);

-- Indexes for efficient queries
CREATE INDEX idx_product_translations_lookup ON product_translations(product_id, language_code);
CREATE INDEX idx_product_translations_language ON product_translations(language_code);
```

### 2. Column-Based Approach

Store translations in separate columns (suitable for few languages):

```sql
-- Multi-column approach (not recommended for many languages)
CREATE TABLE products_multicol (
    id SERIAL PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    
    -- English (default)
    name_en TEXT NOT NULL,
    description_en TEXT,
    
    -- Spanish
    name_es TEXT,
    description_es TEXT,
    
    -- French  
    name_fr TEXT,
    description_fr TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. JSON-Based Storage

Store all translations in a JSON column:

```sql
-- JSON-based translations
CREATE TABLE products_json (
    id SERIAL PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    
    -- Store all translations as JSON
    translations JSONB NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure default language exists
    CONSTRAINT has_default_translation CHECK (
        translations ? 'en' AND 
        translations->'en' ? 'name'
    )
);

-- GIN index for efficient JSON queries
CREATE INDEX idx_products_translations_gin ON products_json USING GIN (translations);

-- Example data structure
INSERT INTO products_json (sku, price, translations) VALUES
('LAPTOP-001', 999.99, '{
    "en": {
        "name": "Gaming Laptop",
        "description": "High-performance gaming laptop",
        "short_description": "Gaming laptop with RTX graphics"
    },
    "es": {
        "name": "Portátil Gaming",
        "description": "Portátil gaming de alto rendimiento",
        "short_description": "Portátil gaming con gráficos RTX"
    },
    "fr": {
        "name": "Ordinateur Portable Gaming",
        "description": "Ordinateur portable gaming haute performance",
        "short_description": "Portable gaming avec graphiques RTX"
    }
}');
```

## Translation Table Pattern

### Complete Implementation

```sql
-- Enhanced translation system with fallbacks
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    parent_id INTEGER REFERENCES categories(id),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE category_translations (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    name TEXT NOT NULL,
    description TEXT,
    slug TEXT NOT NULL,
    
    -- SEO fields
    meta_title TEXT,
    meta_description TEXT,
    meta_keywords TEXT,
    
    -- Content status
    is_published BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(category_id, language_code),
    UNIQUE(language_code, slug) -- Unique slugs per language
);

-- Function to get category with translation fallback
CREATE OR REPLACE FUNCTION get_category_translated(
    category_id INTEGER,
    preferred_language CHAR(2) DEFAULT 'en',
    fallback_language CHAR(2) DEFAULT 'en'
)
RETURNS TABLE(
    id INTEGER,
    name TEXT,
    description TEXT,
    slug TEXT,
    language_used CHAR(2)
) AS $$
BEGIN
    -- Try to get translation in preferred language
    RETURN QUERY
    SELECT 
        c.id,
        COALESCE(ct_preferred.name, ct_fallback.name) as name,
        COALESCE(ct_preferred.description, ct_fallback.description) as description,
        COALESCE(ct_preferred.slug, ct_fallback.slug) as slug,
        CASE 
            WHEN ct_preferred.name IS NOT NULL THEN preferred_language
            ELSE fallback_language
        END as language_used
    FROM categories c
    LEFT JOIN category_translations ct_preferred ON 
        ct_preferred.category_id = c.id AND 
        ct_preferred.language_code = preferred_language AND
        ct_preferred.is_published = TRUE
    LEFT JOIN category_translations ct_fallback ON 
        ct_fallback.category_id = c.id AND 
        ct_fallback.language_code = fallback_language AND
        ct_fallback.is_published = TRUE
    WHERE c.id = category_id
    AND c.is_active = TRUE;
END;
$$ LANGUAGE plpgsql;
```

### Translation Management

```sql
-- Translation status tracking
CREATE TABLE translation_status (
    id SERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    record_id INTEGER NOT NULL,
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    
    -- Translation completeness
    fields_total INTEGER NOT NULL DEFAULT 0,
    fields_translated INTEGER NOT NULL DEFAULT 0,
    completion_percentage DECIMAL(5,2) GENERATED ALWAYS AS 
        (CASE WHEN fields_total > 0 THEN (fields_translated::DECIMAL / fields_total * 100) ELSE 0 END) STORED,
    
    -- Translation quality
    is_machine_translated BOOLEAN DEFAULT FALSE,
    is_reviewed BOOLEAN DEFAULT FALSE,
    translated_by INTEGER, -- Reference to users table
    reviewed_by INTEGER,   -- Reference to users table
    
    last_source_update TIMESTAMPTZ,
    last_translation_update TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(table_name, record_id, language_code)
);

-- Function to update translation status
CREATE OR REPLACE FUNCTION update_translation_status()
RETURNS TRIGGER AS $$
DECLARE
    total_fields INTEGER;
    translated_fields INTEGER;
BEGIN
    -- Count total translatable fields and non-null fields
    CASE TG_TABLE_NAME
        WHEN 'product_translations' THEN
            total_fields := 6; -- name, description, short_description, meta_title, meta_description, meta_keywords
            translated_fields := (
                CASE WHEN NEW.name IS NOT NULL AND LENGTH(NEW.name) > 0 THEN 1 ELSE 0 END +
                CASE WHEN NEW.description IS NOT NULL AND LENGTH(NEW.description) > 0 THEN 1 ELSE 0 END +
                CASE WHEN NEW.short_description IS NOT NULL AND LENGTH(NEW.short_description) > 0 THEN 1 ELSE 0 END +
                CASE WHEN NEW.meta_title IS NOT NULL AND LENGTH(NEW.meta_title) > 0 THEN 1 ELSE 0 END +
                CASE WHEN NEW.meta_description IS NOT NULL AND LENGTH(NEW.meta_description) > 0 THEN 1 ELSE 0 END +
                CASE WHEN NEW.meta_keywords IS NOT NULL AND LENGTH(NEW.meta_keywords) > 0 THEN 1 ELSE 0 END
            );
    END CASE;
    
    -- Update translation status
    INSERT INTO translation_status (
        table_name, record_id, language_code, 
        fields_total, fields_translated
    ) VALUES (
        TG_TABLE_NAME, NEW.product_id, NEW.language_code,
        total_fields, translated_fields
    )
    ON CONFLICT (table_name, record_id, language_code)
    DO UPDATE SET
        fields_total = EXCLUDED.fields_total,
        fields_translated = EXCLUDED.fields_translated,
        last_translation_update = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to translation tables
CREATE TRIGGER update_product_translation_status
    AFTER INSERT OR UPDATE ON product_translations
    FOR EACH ROW
    EXECUTE FUNCTION update_translation_status();
```

## JSON-Based Storage

### Advanced JSON Implementation

```sql
-- Enhanced JSON-based multilingual support
CREATE TABLE content_pages (
    id SERIAL PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('page', 'blog_post', 'news')),
    status TEXT DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    
    -- Store all translations in structured JSON
    content JSONB NOT NULL DEFAULT '{}',
    
    -- Metadata
    author_id INTEGER NOT NULL,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure at least default language content exists
    CONSTRAINT has_default_content CHECK (
        content ? 'en' AND 
        content->'en' ? 'title' AND
        content->'en' ? 'body'
    )
);

-- Comprehensive GIN indexes for JSON queries
CREATE INDEX idx_content_pages_translations ON content_pages USING GIN (content);
CREATE INDEX idx_content_pages_published ON content_pages (published_at, status) WHERE status = 'published';

-- Function to get content with fallback
CREATE OR REPLACE FUNCTION get_content_page(
    page_slug TEXT,
    preferred_lang CHAR(2) DEFAULT 'en',
    fallback_lang CHAR(2) DEFAULT 'en'
)
RETURNS TABLE(
    id INTEGER,
    slug TEXT,
    title TEXT,
    body TEXT,
    excerpt TEXT,
    meta_title TEXT,
    meta_description TEXT,
    language_used CHAR(2),
    published_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        cp.id,
        cp.slug,
        COALESCE(
            cp.content->>preferred_lang->>'title',
            cp.content->>fallback_lang->>'title'
        ) as title,
        COALESCE(
            cp.content->>preferred_lang->>'body',
            cp.content->>fallback_lang->>'body'
        ) as body,
        COALESCE(
            cp.content->>preferred_lang->>'excerpt',
            cp.content->>fallback_lang->>'excerpt'
        ) as excerpt,
        COALESCE(
            cp.content->>preferred_lang->>'meta_title',
            cp.content->>fallback_lang->>'meta_title'
        ) as meta_title,
        COALESCE(
            cp.content->>preferred_lang->>'meta_description',
            cp.content->>fallback_lang->>'meta_description'
        ) as meta_description,
        CASE 
            WHEN cp.content ? preferred_lang THEN preferred_lang
            ELSE fallback_lang
        END as language_used,
        cp.published_at
    FROM content_pages cp
    WHERE cp.slug = page_slug
    AND cp.status = 'published';
END;
$$ LANGUAGE plpgsql;

-- Function to update specific language content
CREATE OR REPLACE FUNCTION update_content_translation(
    page_id INTEGER,
    lang_code CHAR(2),
    new_content JSONB
)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE content_pages 
    SET 
        content = jsonb_set(content, ARRAY[lang_code], new_content),
        updated_at = NOW()
    WHERE id = page_id;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT update_content_translation(1, 'es', '{
    "title": "Página de Inicio",
    "body": "Contenido de la página principal...",
    "excerpt": "Resumen de la página",
    "meta_title": "Inicio - Mi Sitio Web",
    "meta_description": "Página principal de mi sitio web"
}');
```

## Real-World Examples

### E-commerce Product Catalog

```sql
-- Complete multilingual e-commerce setup
CREATE TABLE brands (
    id SERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    logo_url TEXT,
    website_url TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE brand_translations (
    id SERIAL PRIMARY KEY,
    brand_id INTEGER NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    name TEXT NOT NULL,
    description TEXT,
    
    UNIQUE(brand_id, language_code)
);

-- Enhanced product system with attributes
CREATE TABLE product_attributes (
    id SERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL, -- color, size, material, etc.
    data_type TEXT NOT NULL CHECK (data_type IN ('text', 'number', 'boolean', 'select'))
);

CREATE TABLE product_attribute_translations (
    id SERIAL PRIMARY KEY,
    attribute_id INTEGER NOT NULL REFERENCES product_attributes(id) ON DELETE CASCADE,
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    name TEXT NOT NULL,
    description TEXT,
    
    UNIQUE(attribute_id, language_code)
);

CREATE TABLE product_attribute_values (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute_id INTEGER NOT NULL REFERENCES product_attributes(id),
    value_text TEXT,
    value_number DECIMAL,
    value_boolean BOOLEAN,
    
    UNIQUE(product_id, attribute_id)
);

-- Comprehensive product query with all translations
CREATE OR REPLACE FUNCTION get_product_complete(
    product_id INTEGER,
    lang_code CHAR(2) DEFAULT 'en'
)
RETURNS TABLE(
    id INTEGER,
    sku TEXT,
    price DECIMAL,
    name TEXT,
    description TEXT,
    brand_name TEXT,
    attributes JSONB
) AS $$
BEGIN
    RETURN QUERY
    WITH product_attrs AS (
        SELECT 
            pav.product_id,
            jsonb_object_agg(
                COALESCE(pat.name, pa.code),
                CASE pa.data_type
                    WHEN 'text' THEN to_jsonb(pav.value_text)
                    WHEN 'number' THEN to_jsonb(pav.value_number)
                    WHEN 'boolean' THEN to_jsonb(pav.value_boolean)
                END
            ) as attributes
        FROM product_attribute_values pav
        JOIN product_attributes pa ON pa.id = pav.attribute_id
        LEFT JOIN product_attribute_translations pat ON 
            pat.attribute_id = pa.id AND pat.language_code = lang_code
        WHERE pav.product_id = product_id
        GROUP BY pav.product_id
    )
    SELECT 
        p.id,
        p.sku,
        p.price,
        COALESCE(pt.name, pt_en.name) as name,
        COALESCE(pt.description, pt_en.description) as description,
        COALESCE(bt.name, bt_en.name) as brand_name,
        COALESCE(pa.attributes, '{}'::jsonb) as attributes
    FROM products p
    LEFT JOIN product_translations pt ON pt.product_id = p.id AND pt.language_code = lang_code
    LEFT JOIN product_translations pt_en ON pt_en.product_id = p.id AND pt_en.language_code = 'en'
    LEFT JOIN brands b ON b.id = p.brand_id
    LEFT JOIN brand_translations bt ON bt.brand_id = b.id AND bt.language_code = lang_code
    LEFT JOIN brand_translations bt_en ON bt_en.brand_id = b.id AND bt_en.language_code = 'en'
    LEFT JOIN product_attrs pa ON pa.product_id = p.id
    WHERE p.id = product_id;
END;
$$ LANGUAGE plpgsql;
```

### Content Management System

```sql
-- CMS with multilingual navigation and menus
CREATE TABLE navigation_menus (
    id SERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE navigation_items (
    id SERIAL PRIMARY KEY,
    menu_id INTEGER NOT NULL REFERENCES navigation_menus(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES navigation_items(id),
    sort_order INTEGER DEFAULT 0,
    url TEXT,
    target TEXT DEFAULT '_self',
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE navigation_item_translations (
    id SERIAL PRIMARY KEY,
    item_id INTEGER NOT NULL REFERENCES navigation_items(id) ON DELETE CASCADE,
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    title TEXT NOT NULL,
    
    UNIQUE(item_id, language_code)
);

-- Function to get complete navigation structure
CREATE OR REPLACE FUNCTION get_navigation_menu(
    menu_code TEXT,
    lang_code CHAR(2) DEFAULT 'en'
)
RETURNS TABLE(
    id INTEGER,
    parent_id INTEGER,
    title TEXT,
    url TEXT,
    sort_order INTEGER,
    level INTEGER
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE nav_tree AS (
        -- Root level items
        SELECT 
            ni.id,
            ni.parent_id,
            COALESCE(nit.title, nit_en.title) as title,
            ni.url,
            ni.sort_order,
            0 as level
        FROM navigation_items ni
        JOIN navigation_menus nm ON nm.id = ni.menu_id
        LEFT JOIN navigation_item_translations nit ON 
            nit.item_id = ni.id AND nit.language_code = lang_code
        LEFT JOIN navigation_item_translations nit_en ON 
            nit_en.item_id = ni.id AND nit_en.language_code = 'en'
        WHERE nm.code = menu_code
        AND ni.parent_id IS NULL
        AND ni.is_active = TRUE
        AND nm.is_active = TRUE
        
        UNION ALL
        
        -- Child items
        SELECT 
            ni.id,
            ni.parent_id,
            COALESCE(nit.title, nit_en.title) as title,
            ni.url,
            ni.sort_order,
            nt.level + 1
        FROM navigation_items ni
        JOIN nav_tree nt ON nt.id = ni.parent_id
        LEFT JOIN navigation_item_translations nit ON 
            nit.item_id = ni.id AND nit.language_code = lang_code
        LEFT JOIN navigation_item_translations nit_en ON 
            nit_en.item_id = ni.id AND nit_en.language_code = 'en'
        WHERE ni.is_active = TRUE
    )
    SELECT * FROM nav_tree ORDER BY level, sort_order;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Indexing Strategy

```sql
-- Comprehensive indexing for multilingual queries
-- 1. Translation lookup indexes
CREATE INDEX idx_translations_lookup ON product_translations(product_id, language_code);
CREATE INDEX idx_translations_language ON product_translations(language_code) WHERE is_published = TRUE;

-- 2. Search indexes for each language
CREATE INDEX idx_product_search_en ON product_translations 
    USING GIN (to_tsvector('english', name || ' ' || COALESCE(description, '')))
    WHERE language_code = 'en';

CREATE INDEX idx_product_search_es ON product_translations 
    USING GIN (to_tsvector('spanish', name || ' ' || COALESCE(description, '')))
    WHERE language_code = 'es';

-- 3. Covering indexes to avoid table lookups
CREATE INDEX idx_product_translations_covering ON product_translations (product_id, language_code)
    INCLUDE (name, short_description, meta_title);

-- 4. JSON path indexes for JSON-based storage
CREATE INDEX idx_content_json_en ON content_pages 
    USING GIN ((content->'en'));
CREATE INDEX idx_content_json_es ON content_pages 
    USING GIN ((content->'es'));
```

### Query Optimization

```sql
-- Optimized bulk translation queries
CREATE OR REPLACE FUNCTION get_products_bulk_translated(
    product_ids INTEGER[],
    lang_code CHAR(2) DEFAULT 'en',
    fallback_lang CHAR(2) DEFAULT 'en'
)
RETURNS TABLE(
    id INTEGER,
    sku TEXT,
    name TEXT,
    price DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.sku,
        COALESCE(pt_preferred.name, pt_fallback.name) as name,
        p.price
    FROM products p
    LEFT JOIN product_translations pt_preferred ON 
        pt_preferred.product_id = p.id AND 
        pt_preferred.language_code = lang_code
    LEFT JOIN product_translations pt_fallback ON 
        pt_fallback.product_id = p.id AND 
        pt_fallback.language_code = fallback_lang
    WHERE p.id = ANY(product_ids)
    AND p.status = 'active';
END;
$$ LANGUAGE plpgsql;

-- Use materialized views for frequently accessed translations
CREATE MATERIALIZED VIEW product_translations_search AS
SELECT 
    p.id,
    p.sku,
    p.price,
    p.status,
    pt.language_code,
    pt.name,
    pt.description,
    to_tsvector(
        CASE pt.language_code
            WHEN 'en' THEN 'english'
            WHEN 'es' THEN 'spanish'
            WHEN 'fr' THEN 'french'
            WHEN 'de' THEN 'german'
            ELSE 'simple'
        END,
        pt.name || ' ' || COALESCE(pt.description, '')
    ) as search_vector
FROM products p
JOIN product_translations pt ON pt.product_id = p.id
WHERE p.status = 'active' AND pt.is_published = TRUE;

CREATE INDEX idx_product_search_mv ON product_translations_search 
    USING GIN (search_vector);

-- Refresh materialized view periodically
-- SELECT cron.schedule('refresh-translations', '0 * * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY product_translations_search;');
```

## Best Practices

### 1. Language Management

```sql
-- Comprehensive language configuration
CREATE TABLE language_configs (
    language_code CHAR(2) PRIMARY KEY REFERENCES languages(code),
    locale TEXT NOT NULL, -- en_US, es_ES, fr_FR
    text_direction TEXT DEFAULT 'ltr' CHECK (text_direction IN ('ltr', 'rtl')),
    date_format TEXT DEFAULT 'YYYY-MM-DD',
    number_format JSONB DEFAULT '{"decimal_separator": ".", "thousands_separator": ","}',
    currency_code CHAR(3),
    timezone TEXT DEFAULT 'UTC',
    is_enabled BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0
);

INSERT INTO language_configs VALUES
('en', 'en_US', 'ltr', 'MM/DD/YYYY', '{"decimal_separator": ".", "thousands_separator": ","}', 'USD', 'America/New_York', TRUE, 1),
('es', 'es_ES', 'ltr', 'DD/MM/YYYY', '{"decimal_separator": ",", "thousands_separator": "."}', 'EUR', 'Europe/Madrid', TRUE, 2),
('ar', 'ar_SA', 'rtl', 'DD/MM/YYYY', '{"decimal_separator": ".", "thousands_separator": ","}', 'SAR', 'Asia/Riyadh', TRUE, 3);
```

### 2. Translation Validation

```sql
-- Validation functions for translation quality
CREATE OR REPLACE FUNCTION validate_translation_completeness(
    table_name TEXT,
    record_id INTEGER,
    required_languages CHAR(2)[] DEFAULT ARRAY['en']
)
RETURNS TABLE(
    language_code CHAR(2),
    is_complete BOOLEAN,
    missing_fields TEXT[]
) AS $$
DECLARE
    lang CHAR(2);
    sql_query TEXT;
    translation_record RECORD;
BEGIN
    FOREACH lang IN ARRAY required_languages LOOP
        sql_query := format('
            SELECT * FROM %I 
            WHERE %s_id = $1 AND language_code = $2
        ', table_name || '_translations', 
          regexp_replace(table_name, 's$', '')); -- Remove plural 's'
        
        EXECUTE sql_query INTO translation_record USING record_id, lang;
        
        -- Check completeness based on table
        RETURN QUERY SELECT 
            lang,
            translation_record IS NOT NULL,
            CASE 
                WHEN translation_record IS NULL THEN ARRAY['all_fields']
                ELSE ARRAY[]::TEXT[] -- Implement field-specific validation
            END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

### 3. Fallback Strategies

```sql
-- Configurable fallback chain
CREATE TABLE language_fallbacks (
    language_code CHAR(2) NOT NULL REFERENCES languages(code),
    fallback_code CHAR(2) NOT NULL REFERENCES languages(code),
    priority INTEGER NOT NULL DEFAULT 1,
    
    PRIMARY KEY (language_code, fallback_code)
);

INSERT INTO language_fallbacks VALUES
('es', 'en', 1),  -- Spanish falls back to English
('fr', 'en', 1),  -- French falls back to English  
('de', 'en', 1),  -- German falls back to English
('pt', 'es', 1),  -- Portuguese falls back to Spanish
('pt', 'en', 2),  -- Then to English
('it', 'es', 1),  -- Italian falls back to Spanish
('it', 'en', 2);  -- Then to English

-- Enhanced fallback function
CREATE OR REPLACE FUNCTION get_translation_with_fallback(
    entity_table TEXT,
    entity_id INTEGER,  
    preferred_lang CHAR(2),
    field_name TEXT DEFAULT 'name'
)
RETURNS TEXT AS $$
DECLARE
    result TEXT;
    fallback_lang CHAR(2);
    sql_query TEXT;
BEGIN
    -- Try preferred language first
    sql_query := format('
        SELECT %I FROM %I 
        WHERE %s_id = $1 AND language_code = $2
    ', field_name, entity_table || '_translations',
       regexp_replace(entity_table, 's$', ''));
    
    EXECUTE sql_query INTO result USING entity_id, preferred_lang;
    
    IF result IS NOT NULL THEN
        RETURN result;
    END IF;
    
    -- Try fallback languages
    FOR fallback_lang IN 
        SELECT lf.fallback_code 
        FROM language_fallbacks lf 
        WHERE lf.language_code = preferred_lang 
        ORDER BY lf.priority
    LOOP
        EXECUTE sql_query INTO result USING entity_id, fallback_lang;
        
        IF result IS NOT NULL THEN
            RETURN result;
        END IF;
    END LOOP;
    
    -- Final fallback to any available translation
    EXECUTE format('
        SELECT %I FROM %I 
        WHERE %s_id = $1 
        ORDER BY CASE language_code WHEN ''en'' THEN 1 ELSE 2 END
        LIMIT 1
    ', field_name, entity_table || '_translations',
       regexp_replace(entity_table, 's$', '')) 
    INTO result USING entity_id;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

### 4. Migration and Maintenance

```sql
-- Migration helper for adding new languages
CREATE OR REPLACE FUNCTION add_language_to_existing_content(
    new_lang_code CHAR(2),
    source_lang_code CHAR(2) DEFAULT 'en',
    tables_to_migrate TEXT[] DEFAULT ARRAY['products', 'categories', 'brands']
)
RETURNS INTEGER AS $$
DECLARE
    table_name TEXT;
    sql_query TEXT;
    rows_inserted INTEGER := 0;
    total_rows INTEGER := 0;
BEGIN
    FOREACH table_name IN ARRAY tables_to_migrate LOOP
        sql_query := format('
            INSERT INTO %I (
                %s_id, language_code, name, description
            )
            SELECT 
                %s_id, $1, 
                ''[TO TRANSLATE] '' || name,
                ''[TO TRANSLATE] '' || description
            FROM %I 
            WHERE language_code = $2
            ON CONFLICT (%s_id, language_code) DO NOTHING
        ', table_name || '_translations',
           regexp_replace(table_name, 's$', ''),
           regexp_replace(table_name, 's$', ''),
           table_name || '_translations',
           regexp_replace(table_name, 's$', ''));
        
        EXECUTE sql_query USING new_lang_code, source_lang_code;
        GET DIAGNOSTICS rows_inserted = ROW_COUNT;
        total_rows := total_rows + rows_inserted;
        
        RAISE NOTICE 'Added % translation templates for %', rows_inserted, table_name;
    END LOOP;
    
    RETURN total_rows;
END;
$$ LANGUAGE plpgsql;

-- Cleanup orphaned translations
CREATE OR REPLACE FUNCTION cleanup_orphaned_translations()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
    table_record RECORD;
BEGIN
    FOR table_record IN 
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_name LIKE '%_translations' 
        AND table_schema = 'public'
    LOOP
        EXECUTE format('
            DELETE FROM %I t
            WHERE NOT EXISTS (
                SELECT 1 FROM %I p 
                WHERE p.id = t.%s_id
            )
        ', table_record.table_name,
           regexp_replace(table_record.table_name, '_translations$', ''),
           regexp_replace(regexp_replace(table_record.table_name, '_translations$', ''), 's$', ''));
        
        GET DIAGNOSTICS deleted_count = ROW_COUNT;
        
        IF deleted_count > 0 THEN
            RAISE NOTICE 'Cleaned up % orphaned translations from %', deleted_count, table_record.table_name;
        END IF;
    END LOOP;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

## Conclusion

When designing multilingual databases, consider:

**Translation Table Pattern (Recommended for most cases):**
- ✅ Clean separation of base data and translations
- ✅ Easy to add new languages
- ✅ Flexible querying with proper indexes
- ❌ Requires JOINs for most queries

**JSON-Based Storage:**
- ✅ Simple schema, all translations in one place
- ✅ Good for content-heavy applications
- ❌ Less efficient for complex queries
- ❌ Harder to enforce validation

**Column-Based Approach:**
- ✅ Simple queries, no JOINs needed
- ❌ Schema changes required for new languages
- ❌ Only suitable for few languages

**Key implementation principles:**
- Always implement fallback strategies
- Use proper indexing for performance
- Consider translation completeness tracking
- Plan for content migration and maintenance
- Implement proper validation and constraints

Choose the approach that best fits your application's specific requirements for language support, query patterns, and maintenance complexity.

