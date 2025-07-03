# JSON Data Types

Comprehensive guide to storing, querying, and managing JSON data in databases, covering PostgreSQL, MySQL, and best practices for document storage.

## 📚 Table of Contents

- [JSON vs JSONB](#json-vs-jsonb)
- [Schema Design Patterns](#schema-design-patterns)
- [Querying JSON Data](#querying-json-data)
- [Indexing Strategies](#indexing-strategies)
- [Validation and Constraints](#validation-and-constraints)
- [Performance Optimization](#performance-optimization)
- [Migration Strategies](#migration-strategies)
- [Best Practices](#best-practices)

## JSON vs JSONB

### PostgreSQL: JSON vs JSONB

```sql
-- JSON: Stores exact text representation
CREATE TABLE documents_json (
    id SERIAL PRIMARY KEY,
    data JSON NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- JSONB: Binary format, processed and optimized (recommended)
CREATE TABLE documents_jsonb (
    id SERIAL PRIMARY KEY,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Key differences demonstration
INSERT INTO documents_json (data) VALUES ('{"name": "John", "age": 30, "name": "Jane"}');
INSERT INTO documents_jsonb (data) VALUES ('{"name": "John", "age": 30, "name": "Jane"}');

-- JSON preserves duplicates and whitespace
-- JSONB removes duplicates (keeps last) and whitespace
SELECT data FROM documents_json;   -- {"name": "John", "age": 30, "name": "Jane"}
SELECT data FROM documents_jsonb;  -- {"age": 30, "name": "Jane"}
```

### MySQL JSON Support

```sql
-- MySQL 5.7+ JSON type
CREATE TABLE documents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    data JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- JSON validation is automatic
INSERT INTO documents (data) VALUES ('{"name": "John", "age": 30}');
-- INSERT INTO documents (data) VALUES ('invalid json'); -- Error
```

## Schema Design Patterns

### Structured Documents

```sql
-- User profiles with flexible attributes
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE REFERENCES users(id),
    profile_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sample data structure
INSERT INTO user_profiles (user_id, profile_data) VALUES 
(1, '{
    "personal": {
        "firstName": "John",
        "lastName": "Doe",
        "dateOfBirth": "1990-01-15"
    },
    "contact": {
        "email": "john@example.com",
        "phone": "+1234567890",
        "address": {
            "street": "123 Main St",
            "city": "Boston",
            "state": "MA",
            "zipCode": "02101"
        }
    },
    "preferences": {
        "theme": "dark",
        "language": "en",
        "notifications": {
            "email": true,
            "push": false,
            "sms": false
        }
    },
    "metadata": {
        "lastLogin": "2023-12-01T10:30:00Z",
        "loginCount": 42,
        "source": "web"
    }
}');
```

### Product Catalog with Variants

```sql
-- E-commerce products with flexible attributes
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category_id INT REFERENCES categories(id),
    
    -- Core product data as JSON
    product_data JSONB NOT NULL,
    
    -- Extracted fields for common queries
    price DECIMAL(10,2) GENERATED ALWAYS AS ((product_data->>'price')::decimal) STORED,
    brand VARCHAR(100) GENERATED ALWAYS AS (product_data->>'brand') STORED,
    in_stock BOOLEAN GENERATED ALWAYS AS ((product_data->>'inStock')::boolean) STORED,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sample product data
INSERT INTO products (sku, name, category_id, product_data) VALUES 
('SKU-001', 'Wireless Headphones', 1, '{
    "price": 199.99,
    "brand": "TechBrand",
    "inStock": true,
    "specifications": {
        "wireless": true,
        "batteryLife": "30 hours",
        "weight": "250g",
        "colors": ["black", "white", "blue"]
    },
    "features": [
        "Noise Cancellation",
        "Bluetooth 5.0",
        "Quick Charge"
    ],
    "dimensions": {
        "length": 20,
        "width": 15,
        "height": 8,
        "unit": "cm"
    },
    "warranty": {
        "duration": 24,
        "unit": "months",
        "type": "manufacturer"
    }
}');
```

### Event Logging

```sql
-- System events with flexible payloads
CREATE TABLE event_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    user_id INT REFERENCES users(id),
    event_data JSONB NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for event querying
CREATE INDEX idx_events_type_time ON event_logs(event_type, created_at);
CREATE INDEX idx_events_user_time ON event_logs(user_id, created_at);

-- Sample events
INSERT INTO event_logs (event_type, user_id, event_data) VALUES 
('user_login', 1, '{
    "success": true,
    "method": "password",
    "sessionId": "sess_abc123",
    "deviceInfo": {
        "platform": "web",
        "browser": "Chrome",
        "version": "91.0"
    }
}'),
('purchase_completed', 1, '{
    "orderId": "order_xyz789",
    "amount": 299.99,
    "currency": "USD",
    "items": [
        {"sku": "SKU-001", "quantity": 1, "price": 199.99},
        {"sku": "SKU-002", "quantity": 2, "price": 50.00}
    ],
    "paymentMethod": "credit_card",
    "shippingAddress": {
        "street": "123 Main St",
        "city": "Boston",
        "state": "MA"
    }
}');
```

## Querying JSON Data

### PostgreSQL JSONB Operators

```sql
-- Extract values
SELECT 
    profile_data->>'personal'->>'firstName' as first_name,
    profile_data->'contact'->>'email' as email,
    profile_data->'preferences'->>'theme' as theme
FROM user_profiles;

-- Check if key exists
SELECT * FROM user_profiles 
WHERE profile_data ? 'preferences';

-- Check if nested key exists
SELECT * FROM user_profiles 
WHERE profile_data->'contact' ? 'phone';

-- Array operations
SELECT * FROM products 
WHERE product_data->'specifications'->'colors' ? 'black';

-- Contains operation
SELECT * FROM products 
WHERE product_data @> '{"brand": "TechBrand"}';

-- Path-based queries
SELECT * FROM user_profiles 
WHERE profile_data #> '{preferences, notifications, email}' = 'true';

-- Array length
SELECT 
    name,
    jsonb_array_length(product_data->'features') as feature_count
FROM products;
```

### MySQL JSON Functions

```sql
-- Extract values
SELECT 
    JSON_EXTRACT(data, '$.name') as name,
    JSON_UNQUOTE(JSON_EXTRACT(data, '$.email')) as email,
    data->>'$.age' as age  -- MySQL 5.7.13+
FROM documents;

-- Check if key exists
SELECT * FROM documents 
WHERE JSON_EXTRACT(data, '$.email') IS NOT NULL;

-- Array operations
SELECT * FROM documents 
WHERE JSON_CONTAINS(JSON_EXTRACT(data, '$.hobbies'), '"reading"');

-- Update JSON fields
UPDATE documents 
SET data = JSON_SET(data, '$.age', 31) 
WHERE JSON_EXTRACT(data, '$.name') = 'John';

-- Array aggregation
SELECT JSON_ARRAYAGG(JSON_EXTRACT(data, '$.name')) as names
FROM documents 
WHERE JSON_EXTRACT(data, '$.active') = true;
```

### Complex Queries

```sql
-- PostgreSQL: Find users with specific preferences
SELECT 
    u.id,
    u.email,
    up.profile_data->'personal'->>'firstName' as name
FROM users u
JOIN user_profiles up ON u.id = up.user_id
WHERE up.profile_data->'preferences'->>'theme' = 'dark'
  AND up.profile_data->'preferences'->'notifications'->>'email' = 'true';

-- Find products by price range and features
SELECT 
    name,
    (product_data->>'price')::decimal as price,
    product_data->'features' as features
FROM products 
WHERE (product_data->>'price')::decimal BETWEEN 100 AND 500
  AND product_data->'features' ? 'Bluetooth 5.0';

-- Aggregate queries
SELECT 
    product_data->>'brand' as brand,
    COUNT(*) as product_count,
    AVG((product_data->>'price')::decimal) as avg_price,
    jsonb_agg(DISTINCT product_data->>'category') as categories
FROM products 
GROUP BY product_data->>'brand'
ORDER BY avg_price DESC;
```

## Indexing Strategies

### PostgreSQL JSONB Indexes

```sql
-- GIN index for general JSON queries (recommended for most cases)
CREATE INDEX idx_products_data_gin ON products USING GIN (product_data);

-- Specific path indexes for frequent queries
CREATE INDEX idx_products_brand ON products ((product_data->>'brand'));
CREATE INDEX idx_products_price ON products (((product_data->>'price')::decimal));
CREATE INDEX idx_products_in_stock ON products (((product_data->>'inStock')::boolean));

-- Partial indexes for filtered queries
CREATE INDEX idx_products_expensive 
ON products ((product_data->>'price')::decimal) 
WHERE (product_data->>'price')::decimal > 1000;

-- Composite indexes for complex queries
CREATE INDEX idx_products_brand_price 
ON products ((product_data->>'brand'), ((product_data->>'price')::decimal));

-- Expression indexes for JSON operations
CREATE INDEX idx_products_features_count 
ON products (jsonb_array_length(product_data->'features'));
```

### MySQL JSON Indexes

```sql
-- Functional indexes on JSON expressions (MySQL 8.0+)
CREATE INDEX idx_documents_name ON documents ((JSON_UNQUOTE(JSON_EXTRACT(data, '$.name'))));
CREATE INDEX idx_documents_age ON documents ((CAST(JSON_EXTRACT(data, '$.age') AS UNSIGNED)));

-- Multi-value indexes for JSON arrays (MySQL 8.0.17+)
CREATE INDEX idx_documents_hobbies ON documents ((CAST(data->'$.hobbies[*]' AS CHAR(50) ARRAY)));

-- Virtual columns with indexes
ALTER TABLE documents 
ADD COLUMN name_virtual VARCHAR(100) 
GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(data, '$.name'))) VIRTUAL;

CREATE INDEX idx_documents_name_virtual ON documents (name_virtual);
```

## Validation and Constraints

### JSON Schema Validation

```sql
-- PostgreSQL: Custom validation function
CREATE OR REPLACE FUNCTION validate_user_profile(profile JSONB)
RETURNS BOOLEAN AS $$
BEGIN
    -- Check required fields
    IF NOT (profile ? 'personal' AND profile ? 'contact') THEN
        RETURN FALSE;
    END IF;
    
    -- Validate email format
    IF profile->'contact'->>'email' !~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$' THEN
        RETURN FALSE;
    END IF;
    
    -- Validate age is a number
    IF profile->'personal'->>'age' IS NOT NULL 
       AND NOT (profile->'personal'->>'age' ~ '^\d+$') THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Add constraint
ALTER TABLE user_profiles 
ADD CONSTRAINT valid_profile_data 
CHECK (validate_user_profile(profile_data));
```

### MySQL JSON Schema Validation

```sql
-- MySQL 8.0.17+: JSON Schema validation
ALTER TABLE documents 
ADD CONSTRAINT valid_document_schema 
CHECK (JSON_SCHEMA_VALID('{
    "type": "object",
    "required": ["name", "email"],
    "properties": {
        "name": {"type": "string", "minLength": 1},
        "email": {"type": "string", "format": "email"},
        "age": {"type": "integer", "minimum": 0, "maximum": 150}
    }
}', data));
```

### Application-Level Validation

```javascript
// JavaScript JSON schema validation with Ajv
const Ajv = require('ajv');
const addFormats = require('ajv-formats');

const ajv = new Ajv();
addFormats(ajv);

const userProfileSchema = {
    type: 'object',
    required: ['personal', 'contact'],
    properties: {
        personal: {
            type: 'object',
            required: ['firstName', 'lastName'],
            properties: {
                firstName: { type: 'string', minLength: 1 },
                lastName: { type: 'string', minLength: 1 },
                age: { type: 'integer', minimum: 0, maximum: 150 }
            }
        },
        contact: {
            type: 'object',
            required: ['email'],
            properties: {
                email: { type: 'string', format: 'email' },
                phone: { type: 'string', pattern: '^\\+?[1-9]\\d{1,14}$' }
            }
        },
        preferences: {
            type: 'object',
            properties: {
                theme: { type: 'string', enum: ['light', 'dark'] },
                language: { type: 'string', pattern: '^[a-z]{2}$' }
            }
        }
    }
};

const validate = ajv.compile(userProfileSchema);

function validateUserProfile(profileData) {
    const valid = validate(profileData);
    if (!valid) {
        return { isValid: false, errors: validate.errors };
    }
    return { isValid: true };
}

// Usage
const profile = {
    personal: { firstName: 'John', lastName: 'Doe', age: 30 },
    contact: { email: 'john@example.com' },
    preferences: { theme: 'dark', language: 'en' }
};

const result = validateUserProfile(profile);
console.log(result.isValid); // true
```

## Performance Optimization

### Query Optimization

```sql
-- Use extracted columns for frequent queries
ALTER TABLE products 
ADD COLUMN brand VARCHAR(100) GENERATED ALWAYS AS (product_data->>'brand') STORED,
ADD COLUMN price DECIMAL(10,2) GENERATED ALWAYS AS ((product_data->>'price')::decimal) STORED;

CREATE INDEX idx_products_brand_stored ON products(brand);
CREATE INDEX idx_products_price_stored ON products(price);

-- Query using extracted columns (faster)
SELECT * FROM products 
WHERE brand = 'TechBrand' 
  AND price BETWEEN 100 AND 500;

-- vs JSON path query (slower)
SELECT * FROM products 
WHERE product_data->>'brand' = 'TechBrand' 
  AND (product_data->>'price')::decimal BETWEEN 100 AND 500;
```

### Batch Operations

```sql
-- Efficient bulk updates
UPDATE products 
SET product_data = jsonb_set(
    product_data, 
    '{lastUpdated}', 
    to_jsonb(NOW()::text)
)
WHERE created_at < NOW() - INTERVAL '1 day';

-- Bulk upsert with JSON data
INSERT INTO user_profiles (user_id, profile_data)
VALUES 
    (1, '{"theme": "dark"}'),
    (2, '{"theme": "light"}'),
    (3, '{"theme": "dark"}')
ON CONFLICT (user_id) 
DO UPDATE SET 
    profile_data = user_profiles.profile_data || EXCLUDED.profile_data,
    updated_at = NOW();
```

### Memory and Storage Optimization

```sql
-- Use specific data types for known structures
-- Instead of storing everything in JSON:
CREATE TABLE user_settings (
    user_id INT PRIMARY KEY REFERENCES users(id),
    theme VARCHAR(10) DEFAULT 'light',
    language CHAR(2) DEFAULT 'en',
    email_notifications BOOLEAN DEFAULT true,
    push_notifications BOOLEAN DEFAULT false,
    
    -- Use JSON only for truly flexible data
    custom_settings JSONB DEFAULT '{}'
);

-- Compress large JSON documents
-- PostgreSQL automatically compresses, but consider:
-- 1. Normalizing frequently accessed fields
-- 2. Archiving old JSON data
-- 3. Using separate tables for large documents
```

## Migration Strategies

### Schema Evolution

```sql
-- Add new fields to existing JSON
UPDATE user_profiles 
SET profile_data = jsonb_set(
    profile_data,
    '{preferences, darkMode}',
    'true'::jsonb
)
WHERE profile_data->'preferences'->>'theme' = 'dark';

-- Migrate from separate columns to JSON
UPDATE products 
SET product_data = jsonb_build_object(
    'name', name,
    'description', description,
    'price', price,
    'category', category
);

-- Then drop old columns
ALTER TABLE products 
DROP COLUMN name,
DROP COLUMN description,
DROP COLUMN price,
DROP COLUMN category;
```

### Data Restructuring

```sql
-- Restructure nested JSON
UPDATE user_profiles 
SET profile_data = jsonb_set(
    profile_data,
    '{contact}',
    jsonb_build_object(
        'email', profile_data->>'email',
        'phone', profile_data->>'phone',
        'address', profile_data->'address'
    )
)
WHERE profile_data ? 'email';

-- Remove old structure
UPDATE user_profiles 
SET profile_data = profile_data - 'email' - 'phone';
```

## Best Practices

### When to Use JSON

✅ **Good Use Cases:**
- Configuration data with flexible schemas
- Product catalogs with varying attributes
- Event logging and analytics data
- User preferences and settings
- API response caching
- Audit trails and change logs

❌ **Avoid JSON For:**
- Data with fixed, well-known structure
- Frequently joined relationships
- Data requiring complex transactions
- High-frequency write operations
- Data needing strong consistency guarantees

### Design Guidelines

1. **Hybrid Approach**: Extract frequently queried fields as regular columns
2. **Index Strategically**: Create indexes on commonly queried JSON paths
3. **Validate Early**: Implement schema validation at application and database levels
4. **Version Schemas**: Plan for JSON schema evolution
5. **Monitor Performance**: Track query performance on JSON operations
6. **Limit Nesting**: Avoid deeply nested structures (3-4 levels max)
7. **Document Structure**: Maintain clear documentation of JSON schemas

### Performance Tips

```sql
-- ✅ Good: Use extracted columns for filtering
SELECT * FROM products 
WHERE brand = 'TechBrand'  -- Regular column
  AND product_data->'features' ? 'Bluetooth';  -- JSON for flexible data

-- ❌ Avoid: Complex JSON operations in WHERE clauses
SELECT * FROM products 
WHERE jsonb_array_length(product_data->'features') > 5
  AND (product_data->'specifications'->>'weight')::int < 500;

-- ✅ Good: Use appropriate indexes
CREATE INDEX idx_products_features ON products USING GIN ((product_data->'features'));

-- ✅ Good: Batch JSON updates
UPDATE products 
SET product_data = product_data || '{"lastUpdated": "2023-12-01"}'::jsonb
WHERE updated_at < NOW() - INTERVAL '1 day';
```

### Security Considerations

```sql
-- Validate JSON size to prevent DoS
ALTER TABLE user_profiles 
ADD CONSTRAINT profile_size_limit 
CHECK (pg_column_size(profile_data) < 1048576); -- 1MB limit

-- Sanitize JSON input at application level
-- Never trust user input for JSON structure
```

JSON data types provide powerful flexibility for modern applications, but require careful consideration of performance, indexing, and schema design to use effectively.

## Privilege for functions

```sql
mysql -u USERNAME -p
set global log_bin_trust_function_creators=1;
```

## Thoughts on storing json data as object vs array

with object:

- we probably need to create a static struct to manage the growing keys
- no identity on the kind of data (unless determined through column name)
- once unmarshalled, the values can be used straight away

with array:

- more generic approach
- need to loop through each key value pairs to get the data 
- easier to extend in the future

```js
{
  "a": "1",
  "b": "2"
}

// vs
{
  "data": [{"key": "a", "value": "1"}]
}
```

## Json or not?

Why json is not a good candidate

- no protection against referential integrity (if something gets deleted etc)
- no sorting
- no joining
- no constraints (uniqueness)
- no aggregation (actually it is possible, but not performant)

When to use json column

- if the payload is highly dynamic unstructured, then json is a good candidate. For example, when you are storing `json schema` or `api response payload` in the column. For the opposite, if the shape is almost fixed, store those fields in separate columns instead, or store a custom type as the column

- when you don't need to apply other contraints etc, this is best done at column level



Using `custom type` is akin to strongly typed language vs using `json`. In most cases, having a type to represent data makes a difference. Prefer this over dynamic json.

## Converting JSON to a database row (Postgres)

For single row:

```sql
SELECT * 
FROM json_populate_record(null::account, '{"email": "john.doe@mail.com"}');
```

For multiple rows:

```sql
SELECT * 
FROM json_populate_recordset(null::account, '[{"email": "john.doe@mail.com"}, {"email": "janedoe@mail.com"}]');
```

To build it from a dynamic list:

```sql
-- json_populate_record(record, json) <- convert the jsonb format to json. Merge with || only works with jsonb.
SELECT * FROM json_populate_record(null::account, ('{"email": "john.doe@mail.com"}'::jsonb || '{"token": "hello"}')::json);
SELECT * FROM json_populate_record(null::account, ('{"email": "john.doe@mail.com"}'::jsonb || json_build_object('token', 'hello')::jsonb)::json);
SELECT * FROM json_populate_recordset(null::account, '[{"email": "john.doe@mail.com"}, {"email": "janedoe@mail.com"}]');
```

## Building json object (Postgres)

The merge only works for `jsonb`, not `json`:

```sql
SELECT '{"email": "john.doe@mail.com"}'::jsonb || '{"token": "hello"}'; -- {"email": "john.doe@mail.com", "token": "hello"}
SELECT '{"email": "john.doe@mail.com"}'::json || '{"token": "hello"}'; -- {"email": "john.doe@mail.com"}{"token": "hello"}
```

## Insert json into table (Postgres)

Some limitations - if the field value is not provided in json, it will be treated as null. So for strings, it will throw an error if there is a text column with `not null` constraint.

```sql
  INSERT INTO pg_temp.person (name, picture, display_name)
  -- Don't include fields like ids.
  SELECT name, picture, display_name
    FROM json_populate_record(
      null::pg_temp.person, 
      (_extra::jsonb || json_build_object('name', _name, 'display_name', _display_name)::jsonb)::json
    )
  RETURNING *
```

## Check if a json field exists (Postgres)

```sql
SELECT '{"name": "a", "age": 10}' ? 'age';
```

## Aggregating rows as json in (Postgres)

There are times we want to aggregate a row as json, so that we can deserialize it back to full objects at the application layer:

```sql
-- NOTE: The table name (notification) must be specified before the .*
SELECT array_agg(to_json(notification.*)), subscriber_id, count(*)
FROM notification
GROUP BY subscriber_id;
```

## Update json data with jsonb set (Postgres)

Idempotent update of a json object counter.

```sql
select jsonb_set(
    '{"video": 1}'::jsonb, 
    '{video}', 
    (SELECT (SELECT '{"video": 1}'::jsonb-> 'video')::int + 1)::text::jsonb
);
```

## Convert row to json, and add additional fields

```sql
SELECT row_to_json(reservation_created.*)::jsonb || json_build_object('start_date', lower(validity), 'end_date', upper(validity))::jsonb
FROM reservation_created
```

## Prettify JSONB array column

```sql
SELECT id, jsonb_pretty(log::jsonb) FROM saga, UNNEST(logs) AS log;
```



## Using custom type vs JSON

If the shape is known, store a custom type instead of json.

```sql
create type translations as (
	en text,
	ms text
);


create table products (
	id int generated always as identity,
	
	name text not null,
	translations translations not null,
	primary key (id)
);

insert into products (name, translations) values
('test', '(en-test,ms-test)'::translations);
table products;
alter type translations drop attribute ms; -- This will drop the data.
alter type translations add attribute ms text;

select *, (translations).en, (translations).ms from products;

-- Updating doesn't require the column.
update products set translations.ms = 'ms-test';
update products set translations.ms = null;

select *, (translations).en, (translations).ms from products;

-- However, we are unable to enforce the constraint. Let's add a domain type that is derived from the base type with some constraint - both values must be either set or null.
create domain app_translations as translations check (
	((value).en, (value).ms) is not null
);

drop table products;
create table products (
	id int generated always as identity,
	
	name text not null,
	translations app_translations not null,
	primary key (id)
);

insert into products (name, translations) values
('test', '(en-test,ms-test)'::translations);
update products set translations.ms = null;
select *, (translations).en, (translations).ms from products;



drop table currencies;
create table currencies (
	name text not null,
	primary key (name)
);

insert into currencies (name) values ('sgd');

create type money2 as (
	currency text,
	amount int
);
drop type money;
select '(idr,100)'::money2;

create table products (
	id int generated always as identity,
	price money2,
	
	primary key (id),
	foreign key (price.currency) references currencies (name) -- NOT POSSIBLE
);

drop table products;

-- We can however, make use of generated columns to enforce foreign key.
create table products (
	id int generated always as identity,
	price money2 not null,
	currency text generated always as ((price).currency) stored,
	
	
	primary key (id),
	foreign key (currency) references currencies (name)
);
insert into products (price) values ('(sgd,1000)'::money2); -- Works
insert into products (price) values ('(idr,1000)'::money2); -- Fails

table products;
```

Updating constraints for custom types is easy. For the `app_translations` example, say if we want to add a new translation `id`.



```sql
-- Add a new translation `id`
alter type translations add attribute id text;

-- Update existing data first.
update products set translations.id = 'some value';

SELECT * FROM pg_catalog.pg_constraint where conname = 'app_translations_check';
-- Drop the existing constraint.
alter domain app_translations drop constraint app_translations_check;

-- Now make the the field `id` mandatory.
alter domain app_translations add constraint app_translations_check check (
	((value).en, (value).ms, (value).id) is not null
);
```
