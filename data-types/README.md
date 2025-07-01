# Data Types

Comprehensive guide to choosing and implementing database data types for different use cases.

## 📚 Contents

### Primitive Types
- **[Arrays](primitives/array.md)** - Working with array data types
- **[Binary & Bytes](primitives/bytes.md)** - Binary data storage
- **[Character Sets](primitives/charset.md)** - Text encoding and character sets
- **[Currency](primitives/currency.md)** - Monetary value storage
- **[Dates & Times](primitives/date.md)** - Temporal data types
- **[Email](primitives/email.md)** - Email address validation and storage
- **[Enums](primitives/enum.md)** - Enumerated types and values
- **[Geographic](primitives/geo.md)** - Spatial and geographic data
- **[IDs & Identity](primitives/id.md)** - Primary key strategies
- **[Images](primitives/image.md)** - Image metadata and storage
- **[JSON](primitives/json.md)** - Structured document storage
- **[Null Values](primitives/null.md)** - Handling null and optional data
- **[Strings](primitives/string.md)** - Text data optimization
- **[UUIDs](primitives/uuid.md)** - Unique identifier strategies

### Custom Types
- **[Friendly IDs](custom/friendly-ids.md)** - Human-readable identifiers
- **[URL Slugs](custom/url-slugs.md)** - SEO-friendly URL components
- **[Custom Types](primitives/custom-type.md)** - Creating domain-specific types

### Complex Types
- **[Composite Types](complex/)** - Multi-field custom types
- **[Inheritance](complex/)** - Type inheritance patterns

### Validation & Constraints
- **[Data Sanitization](validation/data-sanitization.md)** - Input cleaning and validation
- **[Hashing](validation/hashing.md)** - Password and data hashing strategies

## 🎯 Database-Specific Guides

### MySQL
- **[MySQL 8 UUID v4](primitives/mysql_8_uuid_v4.md)** - UUID generation in MySQL
- **[MySQL Time Types](primitives/time.mysql.md)** - Time handling in MySQL
- **[MySQL GIPK](primitives/mysql_gipk.md)** - Generated Invisible Primary Keys

### PostgreSQL
- **[PostgreSQL UUIDs](primitives/uuid-postgres.md)** - UUID support in PostgreSQL
- **[BRIN Indexes](primitives/brin.md)** - Block Range Index types

## 🛠️ Type Selection Guidelines

### Performance Considerations
1. **Size Optimization** - Choose the smallest type that fits your data
2. **Index Efficiency** - Consider how types affect index performance
3. **Query Performance** - Impact on WHERE clauses and JOINs
4. **Storage Costs** - Balance between precision and storage requirements

### Data Integrity
1. **Validation Rules** - Built-in vs application-level validation
2. **Range Constraints** - Ensuring data stays within valid ranges
3. **Format Consistency** - Standardizing data formats
4. **Null Handling** - When to allow and disallow null values

### Scalability
1. **Growth Planning** - Types that can accommodate future growth
2. **Migration Considerations** - Easy type changes vs difficult ones
3. **Cross-Platform** - Types that work across different databases
4. **Application Integration** - Compatibility with programming languages

## 🎓 Learning Path

### Beginner
1. Primitive Types → Strings → Numbers
2. Dates & Times → Null Handling
3. Basic Validation → Simple Constraints

### Intermediate
1. JSON → Arrays → Geographic Types
2. Custom Types → Enums
3. Advanced Validation → Complex Constraints

### Advanced
1. Composite Types → Inheritance
2. Database-Specific Features
3. Performance Optimization → Migration Strategies

## 🔗 Related Topics

- **[Schema Design](../schema-design/README.md)** - Using types in schema design
- **[Performance](../performance/README.md)** - Type performance implications
- **[Security](../security/README.md)** - Data type security considerations
- **[Examples](../examples/README.md)** - Real-world type usage examples

## 📋 Type Selection Checklist

### Before Choosing a Type
- [ ] Understand the data domain and requirements
- [ ] Consider current and future data volume
- [ ] Evaluate query patterns and performance needs
- [ ] Check application language compatibility
- [ ] Plan for data validation and constraints

### During Implementation
- [ ] Add appropriate constraints and validation
- [ ] Document type choices and reasoning
- [ ] Test with realistic data volumes
- [ ] Validate cross-platform compatibility
- [ ] Consider indexing strategy

### After Implementation
- [ ] Monitor query performance
- [ ] Validate data integrity
- [ ] Document any type-specific quirks
- [ ] Plan for future migrations
- [ ] Review and optimize as needed

## 🎯 Learning Objectives

After completing this section, you will be able to:
- Choose appropriate data types for different use cases
- Implement custom types and validation rules
- Optimize storage and query performance through type selection
- Handle complex data structures effectively
- Plan for data type evolution and migration
- Apply database-specific type features appropriately
