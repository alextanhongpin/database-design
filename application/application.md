# Database Integration in Applications

A comprehensive guide to integrating databases into applications, covering connection management, testing strategies, and architectural patterns.

## Overview

Most applications require database integration for data persistence, retrieval, and management. This guide covers essential patterns and practices for robust database integration.

## Database Connection Patterns

### 1. Basic Database Connection

```go
// Go example with database/sql
import (
    "database/sql"
    "time"
    _ "github.com/lib/pq" // PostgreSQL driver
)

func NewDBConnection(connStr string) (*sql.DB, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    
    // Test the connection
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

```python
# Python example with psycopg2
import psycopg2
from psycopg2 import pool

def create_connection_pool(conn_str, min_conn=1, max_conn=20):
    return psycopg2.pool.ThreadedConnectionPool(
        min_conn, max_conn, conn_str
    )
```

### 2. Connection Pool Configuration

Connection pools reuse existing connections to avoid the overhead of creating new connections for each request.

```go
// Go connection pool setup
func ConfigureConnectionPool(db *sql.DB) {
    // Maximum number of open connections
    db.SetMaxOpenConns(25)
    
    // Maximum number of idle connections
    db.SetMaxIdleConns(5)
    
    // Maximum lifetime of connections
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Maximum idle time
    db.SetConnMaxIdleTime(2 * time.Minute)
}
```

```javascript
// Node.js with pg pool
const { Pool } = require('pg');

const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
    max: 20,                    // Maximum connections
    idleTimeoutMillis: 30000,   // Close idle connections after 30s
    connectionTimeoutMillis: 2000, // Connection timeout
});
```

### 3. Statement Timeouts

Prevent long-running queries from blocking the application.

```sql
-- PostgreSQL: Set statement timeout
SET statement_timeout = '30s';

-- MySQL: Set query timeout
SET SESSION max_execution_time = 30000; -- 30 seconds
```

```go
// Go: Context-based timeouts
func QueryWithTimeout(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    return db.QueryContext(ctx, query, args...)
}
```

## Architecture Patterns

### Repository Pattern

Separate data access logic from business logic.

```go
// User repository interface
type UserRepository interface {
    GetByID(ctx context.Context, id int) (*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id int) error
}

// PostgreSQL implementation
type PostgreSQLUserRepository struct {
    db *sql.DB
}

func (r *PostgreSQLUserRepository) GetByID(ctx context.Context, id int) (*User, error) {
    query := `SELECT id, email, username, created_at FROM users WHERE id = $1`
    
    var user User
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Email, &user.Username, &user.CreatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, ErrUserNotFound
    }
    
    return &user, err
}
```

### ORM vs SQL Query Builder vs Raw SQL

| Approach | Pros | Cons | Best For |
|----------|------|------|----------|
| **ORM** | Type safety, rapid development | Performance overhead, limited control | Rapid prototyping, simple CRUD |
| **Query Builder** | Flexible, type-safe, readable | Learning curve, still abstraction layer | Complex queries, dynamic filters |
| **Raw SQL** | Full control, optimal performance | Manual mapping, SQL injection risk | Performance-critical, complex operations |

```go
// Raw SQL example
func (r *UserRepository) GetActiveUsers(limit int) ([]User, error) {
    query := `
        SELECT u.id, u.email, u.username, u.created_at,
               COUNT(o.id) as order_count
        FROM users u
        LEFT JOIN orders o ON u.id = o.user_id AND o.status = 'completed'
        WHERE u.status = 'active'
        GROUP BY u.id, u.email, u.username, u.created_at
        HAVING COUNT(o.id) > 0
        ORDER BY order_count DESC
        LIMIT $1
    `
    // Implementation...
}
```

## Database Migrations

### Migration Strategy

```sql
-- Example migration: 001_create_users_table.up.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
```

```sql
-- 001_create_users_table.down.sql
DROP TABLE IF EXISTS users;
```

### Migration Tools and Approaches

#### External Migration Control (Recommended)
```bash
# Using migrate CLI tool
migrate -path ./migrations -database "postgres://user:pass@host/db?sslmode=disable" up

# Using Flyway
flyway -url=jdbc:postgresql://host/db -user=user -password=pass migrate
```

#### Application-Controlled Migrations
```go
// Go example with golang-migrate
import "github.com/golang-migrate/migrate/v4"

func RunMigrations(databaseURL string) error {
    m, err := migrate.New(
        "file://migrations",
        databaseURL,
    )
    if err != nil {
        return err
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    
    return nil
}
```

### Data Seeding

Separate reference data and application data seeding.

```sql
-- Reference data seeding
INSERT INTO countries (code, name, region) VALUES
('US', 'United States', 'North America'),
('GB', 'United Kingdom', 'Europe'),
('JP', 'Japan', 'Asia');

INSERT INTO currencies (code, name, symbol) VALUES
('USD', 'US Dollar', '$'),
('GBP', 'British Pound', '£'),
('JPY', 'Japanese Yen', '¥');
```

```go
// Development data seeding
func SeedDevelopmentData(db *sql.DB) error {
    users := []User{
        {Email: "admin@example.com", Username: "admin", Role: "admin"},
        {Email: "user@example.com", Username: "user", Role: "user"},
    }
    
    for _, user := range users {
        if err := createUser(db, user); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Query Patterns

### Static vs Dynamic Queries

#### Static Queries (Preferred)
```go
// Prepared statements for static queries
const getUserByEmailQuery = `
    SELECT id, email, username, created_at 
    FROM users 
    WHERE email = $1 AND status = 'active'
`

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := r.db.QueryRowContext(ctx, getUserByEmailQuery, email).Scan(
        &user.ID, &user.Email, &user.Username, &user.CreatedAt,
    )
    return &user, err
}
```

#### Dynamic Queries (When Necessary)
```go
// Dynamic query builder for complex filtering
type UserFilter struct {
    Status    *string
    Role      *string
    CreatedAfter *time.Time
    Limit     int
}

func (r *UserRepository) GetUsers(ctx context.Context, filter UserFilter) ([]User, error) {
    query := "SELECT id, email, username, created_at FROM users WHERE 1=1"
    args := []interface{}{}
    argCount := 0
    
    if filter.Status != nil {
        argCount++
        query += fmt.Sprintf(" AND status = $%d", argCount)
        args = append(args, *filter.Status)
    }
    
    if filter.Role != nil {
        argCount++
        query += fmt.Sprintf(" AND role = $%d", argCount)
        args = append(args, *filter.Role)
    }
    
    if filter.CreatedAfter != nil {
        argCount++
        query += fmt.Sprintf(" AND created_at > $%d", argCount)
        args = append(args, *filter.CreatedAfter)
    }
    
    query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", filter.Limit)
    
    rows, err := r.db.QueryContext(ctx, query, args...)
    // Process rows...
}
```

## Transaction Management

### Basic Transaction Pattern
```go
func (r *UserRepository) TransferCredits(ctx context.Context, fromUserID, toUserID int, amount decimal.Decimal) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback() // Rollback if not committed
    
    // Deduct from sender
    _, err = tx.ExecContext(ctx, 
        "UPDATE users SET credits = credits - $1 WHERE id = $2 AND credits >= $1",
        amount, fromUserID,
    )
    if err != nil {
        return err
    }
    
    // Add to receiver
    _, err = tx.ExecContext(ctx,
        "UPDATE users SET credits = credits + $1 WHERE id = $2",
        amount, toUserID,
    )
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

### Repository Transaction Pattern
```go
// Transaction-aware repository
type TxRepository struct {
    db *sql.DB
    tx *sql.Tx
}

func (r *TxRepository) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    if r.tx != nil {
        return r.tx.ExecContext(ctx, query, args...)
    }
    return r.db.ExecContext(ctx, query, args...)
}

func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
    return &UserRepository{
        db: r.db,
        tx: tx,
    }
}
```

## Testing Strategies

### Database Testing Approaches

1. **Application Layer Testing** - Test through your application code
2. **External Testing** - Use tools like pgTAP for PostgreSQL
3. **Type-Safe Approaches** - Generate application code from SQL (sqlc, pgtyped)

### Integration Testing Setup

```go
// Global test setup
func TestMain(m *testing.M) {
    // Setup test database
    testDB := setupTestDatabase()
    defer teardownTestDatabase(testDB)
    
    // Run migrations
    if err := runMigrations(testDB); err != nil {
        log.Fatal(err)
    }
    
    // Seed test data
    if err := seedTestData(testDB); err != nil {
        log.Fatal(err)
    }
    
    // Run tests
    code := m.Run()
    os.Exit(code)
}

func setupTestDatabase() *sql.DB {
    // Use Docker container for testing
    connStr := "postgres://test:test@localhost:5433/testdb?sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    return db
}
```

### Parallel Testing with Separate Databases

```go
// Worker-based parallel testing
func TestWithWorkerDB(t *testing.T) {
    workerID := getWorkerID() // From test environment
    dbName := fmt.Sprintf("test_worker_%d", workerID)
    
    // Create worker-specific database from template
    db := createWorkerDatabase(dbName)
    defer dropWorkerDatabase(dbName)
    
    // Run tests...
}

func createWorkerDatabase(dbName string) *sql.DB {
    // Connect to template database
    templateDB := connectToTemplate()
    defer templateDB.Close()
    
    // Create worker database from template
    _, err := templateDB.Exec(fmt.Sprintf(
        "CREATE DATABASE %s WITH TEMPLATE test_template", dbName))
    if err != nil {
        log.Fatal(err)
    }
    
    // Connect to worker database
    return connectToDatabase(dbName)
}
```

### Transaction-Based Testing (For Mutations)

```go
func TestUserRepository_Create(t *testing.T) {
    db := getTestDB()
    
    // Start transaction
    tx, err := db.Begin()
    require.NoError(t, err)
    defer tx.Rollback() // Always rollback
    
    repo := NewUserRepository(db).WithTx(tx)
    
    user := &User{
        Email:    "test@example.com",
        Username: "testuser",
    }
    
    err = repo.Create(context.Background(), user)
    require.NoError(t, err)
    assert.NotZero(t, user.ID)
    
    // Verify user was created
    retrieved, err := repo.GetByID(context.Background(), user.ID)
    require.NoError(t, err)
    assert.Equal(t, user.Email, retrieved.Email)
    
    // Transaction automatically rolled back
}
```

### Read-Only Testing (No Transaction Needed)

```go
func TestUserRepository_GetByEmail(t *testing.T) {
    db := getTestDB()
    repo := NewUserRepository(db)
    
    // Use seeded test data
    user, err := repo.GetByEmail(context.Background(), "seed@example.com")
    require.NoError(t, err)
    assert.Equal(t, "seed@example.com", user.Email)
}
```

## Advanced Patterns

### Database Functions vs Application Logic

#### Using Database Functions
```sql
-- PostgreSQL function for email validation
CREATE OR REPLACE FUNCTION validate_email(email_input TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    IF email_input IS NULL OR email_input = '' THEN
        RAISE EXCEPTION 'Email cannot be empty';
    END IF;
    
    IF email_input !~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$' THEN
        RAISE EXCEPTION 'Invalid email format: %', email_input;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Use in table constraint
ALTER TABLE users 
ADD CONSTRAINT check_email_valid 
CHECK (validate_email(email));
```

#### Benefits of Database Functions
- **Single Source of Truth**: Business logic centralized in database
- **Consistent Validation**: Same rules applied regardless of application
- **Performance**: Validation happens close to data
- **Custom Error Messages**: Better error handling than simple constraints

```sql
-- Multi-validation function
CREATE OR REPLACE FUNCTION validate_password(password_input TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    IF password_input IS NULL OR LENGTH(password_input) = 0 THEN
        RAISE EXCEPTION 'Password cannot be empty';
    END IF;
    
    IF LENGTH(password_input) < 8 THEN
        RAISE EXCEPTION 'Password must be at least 8 characters long';
    END IF;
    
    IF password_input !~ '[A-Z]' THEN
        RAISE EXCEPTION 'Password must contain at least one uppercase letter';
    END IF;
    
    IF password_input !~ '[0-9]' THEN
        RAISE EXCEPTION 'Password must contain at least one number';
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

### Schema Management Strategies

#### Schema in Application (Monolith)
```
Pros:
- Easier to maintain and develop
- Single source of truth
- Code generation tools work well
- Tight coupling between app and schema

Cons:
- Hard to break out into microservices
- Other applications can't easily share schema
- Schema becomes tightly coupled to one application
```

#### Schema as Separate Service (Microservices)
```
Pros:
- Multiple applications can share schema
- Easier to evolve independently
- Better separation of concerns
- Can use schema-first development

Cons:
- More complex deployment pipeline
- Potential version conflicts
- Additional coordination overhead
- Code generation becomes more complex
```

### Entity Timestamp Patterns

```sql
-- Mutable entity with full timestamp tracking
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL -- Soft delete
);

-- Event entity (immutable, only created_at needed)
CREATE TABLE audit_events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INT NOT NULL,
    event_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Entity with validity periods
CREATE TABLE price_history (
    id SERIAL PRIMARY KEY,
    product_id INT REFERENCES products(id),
    price DECIMAL(10,2) NOT NULL,
    valid_from TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP NULL, -- NULL means current price
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Best Practices Summary

### Connection Management
1. **Use connection pools** - Reuse connections efficiently
2. **Set appropriate timeouts** - Prevent hanging connections
3. **Monitor connection health** - Implement health checks
4. **Close connections properly** - Prevent connection leaks

### Query Design
1. **Prefer static queries** - Better performance and security
2. **Use prepared statements** - Protection against SQL injection
3. **Implement proper indexing** - Optimize query performance
4. **Avoid N+1 queries** - Use joins or batch operations

### Testing
1. **Test against real databases** - Don't mock what you can test
2. **Use transactions for cleanup** - Isolate test data changes
3. **Separate unit and integration tests** - Different purposes and speeds
4. **Validate both happy and error paths** - Test constraint violations

### Architecture
1. **Use repository pattern** - Separate data access from business logic
2. **Implement proper error handling** - Distinguish between different error types
3. **Design for observability** - Log queries and performance metrics
4. **Plan for schema evolution** - Design migrations carefully

## Related Topics

- [Query Patterns](../query-patterns/README.md) - SQL query design patterns
- [Performance Optimization](../performance/README.md) - Database performance tuning
- [Security Patterns](../security/README.md) - Database security implementation
- [Testing Strategies](../operations/testing.md) - Database testing approaches
- [Migration Patterns](../operations/migrations.md) - Schema evolution strategies
