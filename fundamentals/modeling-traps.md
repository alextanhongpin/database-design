# Database Modeling Traps and Anti-Patterns

Common pitfalls in database design that can lead to poor performance, data integrity issues, and maintenance problems.

## Chasm and Fan Traps

### What are Chasm and Fan Traps?

**Chasm Trap**: Occurs when a model suggests a relationship between entity sets, but the pathway between certain entity occurrences is ambiguous.

**Fan Trap**: Occurs when a model represents a relationship between entity sets, but the pathway between certain entity occurrences is not unique.

### Chasm Trap Example
```sql
-- Problematic design
Customer -> Order -> OrderItem
Customer -> CustomerService

-- Issue: Can't directly relate CustomerService to specific Orders
-- Some customers may have services but no orders, or orders but no services
```

### Fan Trap Example  
```sql
-- Problematic design
Department -> Employee
Department -> Project

-- Issue: Implies all employees work on all department projects
-- Creates a many-to-many relationship through Department
```

### Solutions

#### For Chasm Traps
```sql
-- Add direct relationships where needed
CREATE TABLE customer_service_orders (
    service_id INT REFERENCES customer_services(id),
    order_id INT REFERENCES orders(id),
    PRIMARY KEY (service_id, order_id)
);
```

#### For Fan Traps
```sql
-- Create explicit many-to-many relationship
CREATE TABLE employee_projects (
    employee_id INT REFERENCES employees(id),
    project_id INT REFERENCES projects(id),
    role VARCHAR(50),
    start_date DATE,
    end_date DATE,
    PRIMARY KEY (employee_id, project_id)
);
```

## Other Common Modeling Traps

### 1. Over-Normalization Trap
```sql
-- Over-normalized (problematic for queries)
CREATE TABLE addresses (
    id SERIAL PRIMARY KEY,
    street_number VARCHAR(10),
    street_name VARCHAR(100),
    city_id INT REFERENCES cities(id)
);

CREATE TABLE cities (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    state_id INT REFERENCES states(id)
);

-- Better: Balance normalization with query needs
CREATE TABLE addresses (
    id SERIAL PRIMARY KEY,
    street_address VARCHAR(200),
    city VARCHAR(100),
    state VARCHAR(50),
    postal_code VARCHAR(20),
    country VARCHAR(50)
);
```

### 2. Generic Trap (EAV Anti-Pattern)
```sql
-- Problematic: Entity-Attribute-Value
CREATE TABLE entities (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50)
);

CREATE TABLE attributes (
    entity_id INT,
    attribute_name VARCHAR(100),
    attribute_value TEXT
);

-- Better: Specific tables for each entity type
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200),
    price DECIMAL(10,2),
    category VARCHAR(100)
);
```

### 3. Inheritance Trap
```sql
-- Problematic: Complex inheritance hierarchy
CREATE TABLE vehicles (
    id SERIAL PRIMARY KEY,
    type VARCHAR(20), -- 'car', 'truck', 'motorcycle'
    make VARCHAR(50),
    model VARCHAR(50),
    -- Car-specific fields
    num_doors INT,
    -- Truck-specific fields  
    payload_capacity INT,
    -- Motorcycle-specific fields
    engine_cc INT
);

-- Better: Table per type or proper inheritance
CREATE TABLE vehicles (
    id SERIAL PRIMARY KEY,
    make VARCHAR(50),
    model VARCHAR(50),
    year INT
);

CREATE TABLE cars (
    vehicle_id INT PRIMARY KEY REFERENCES vehicles(id),
    num_doors INT,
    fuel_type VARCHAR(20)
);

CREATE TABLE trucks (
    vehicle_id INT PRIMARY KEY REFERENCES vehicles(id),
    payload_capacity INT,
    cargo_volume DECIMAL(8,2)
);
```

## Prevention Strategies

### 1. Clear Requirements Analysis
- Define all relationships explicitly
- Identify all business rules
- Map out data access patterns

### 2. Entity-Relationship Modeling
- Use proper ER diagrams
- Validate relationships with stakeholders
- Test with sample queries

### 3. Iterative Design
- Start simple, refine based on usage
- Monitor query patterns in production
- Be prepared to refactor

## Related Resources

- [Entity-Relationship Modeling](entity-relationship-diagrams.md)
- [Normalization Best Practices](../schema-design/normalization.md)
- [Performance Considerations](../performance/README.md)

## External References

- [Sisense: Chasm and Fan Traps](https://documentation.sisense.com/docs/chasm-and-fan-traps)
- [Qlik: Fan traps and Chasm traps](https://community.qlik.com/t5/Design/Fan-traps-and-Chasm-traps/ba-p/1463093)