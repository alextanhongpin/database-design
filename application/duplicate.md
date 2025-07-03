# Handling Duplicate Keys in Database Applications

A comprehensive guide to detecting, preventing, and handling duplicate key errors across different programming languages and database systems.

## Overview

Duplicate key errors occur when attempting to insert or update data that violates unique constraints. Proper handling of these errors is crucial for application reliability and user experience.

## Database-Level Duplicate Prevention

### 1. Unique Constraints

```sql
-- Single column unique constraint
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL
);

-- Composite unique constraint
CREATE TABLE user_roles (
    user_id INT REFERENCES users(id),
    role_id INT REFERENCES roles(id),
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id, role_id)
);

-- Named unique constraint for better error handling
ALTER TABLE products 
ADD CONSTRAINT uk_products_sku UNIQUE (sku);
```

### 2. Partial Unique Constraints

```sql
-- PostgreSQL: Unique constraint with condition
CREATE UNIQUE INDEX uk_users_active_email 
ON users (email) 
WHERE status = 'active';

-- Allows multiple inactive users with same email
INSERT INTO users (email, status) VALUES 
('user@example.com', 'active'),   -- OK
('user@example.com', 'inactive'), -- OK
('user@example.com', 'inactive'); -- OK
-- ('user@example.com', 'active'); -- Would fail
```

## Application-Level Duplicate Handling

### Node.js with MySQL

```javascript
const mysql = require('mysql2/promise');

class UserService {
    constructor(connection) {
        this.db = connection;
    }
    
    async createUser(userData) {
        try {
            const [result] = await this.db.execute(
                'INSERT INTO users (email, username) VALUES (?, ?)',
                [userData.email, userData.username]
            );
            
            return { 
                id: result.insertId, 
                ...userData 
            };
            
        } catch (error) {
            if (error.code === 'ER_DUP_ENTRY') {
                // Parse which field caused the duplicate
                const field = this.parseDuplicateField(error.message);
                throw new DuplicateError(`${field} already exists`, field);
            }
            throw error;
        }
    }
    
    parseDuplicateField(message) {
        // MySQL error message: "Duplicate entry 'user@example.com' for key 'users.email'"
        if (message.includes("for key 'users.email'")) return 'email';
        if (message.includes("for key 'users.username'")) return 'username';
        return 'unknown';
    }
}

class DuplicateError extends Error {
    constructor(message, field) {
        super(message);
        this.name = 'DuplicateError';
        this.field = field;
    }
}

// Usage example
async function registerUser(userData) {
    try {
        const user = await userService.createUser(userData);
        return { success: true, user };
    } catch (error) {
        if (error instanceof DuplicateError) {
            return { 
                success: false, 
                error: `${error.field} is already taken` 
            };
        }
        throw error; // Re-throw unexpected errors
    }
}
```

### Go with MySQL

```go
package main

import (
    "database/sql"
    "errors"
    "fmt"
    
    "github.com/go-sql-driver/mysql"
    "github.com/VividCortex/mysqlerr"
)

type User struct {
    ID       int    `json:"id"`
    Email    string `json:"email"`
    Username string `json:"username"`
}

type DuplicateError struct {
    Field   string
    Value   string
    Message string
}

func (e DuplicateError) Error() string {
    return e.Message
}

type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) Create(user *User) error {
    query := "INSERT INTO users (email, username) VALUES (?, ?)"
    
    result, err := r.db.Exec(query, user.Email, user.Username)
    if err != nil {
        // Check for MySQL duplicate key error
        if mysqlError, ok := err.(*mysql.MySQLError); ok {
            if mysqlError.Number == mysqlerr.ER_DUP_ENTRY {
                field, value := parseMySQLDuplicateError(mysqlError.Message)
                return DuplicateError{
                    Field:   field,
                    Value:   value,
                    Message: fmt.Sprintf("%s '%s' already exists", field, value),
                }
            }
        }
        return err
    }
    
    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    
    user.ID = int(id)
    return nil
}

func parseMySQLDuplicateError(message string) (field, value string) {
    // Parse MySQL error message to extract field and value
    // "Duplicate entry 'user@example.com' for key 'users.email'"
    
    // This is a simplified parser - in production, use regex
    if strings.Contains(message, "for key 'users.email'") {
        return "email", extractValueFromMessage(message)
    }
    if strings.Contains(message, "for key 'users.username'") {
        return "username", extractValueFromMessage(message)
    }
    return "unknown", ""
}

// Service layer handling
func (s *UserService) RegisterUser(user *User) (*User, error) {
    err := s.repo.Create(user)
    if err != nil {
        var dupErr DuplicateError
        if errors.As(err, &dupErr) {
            return nil, fmt.Errorf("registration failed: %s", dupErr.Message)
        }
        return nil, fmt.Errorf("unexpected error: %w", err)
    }
    
    return user, nil
}
```

### Go with PostgreSQL

```go
package main

import (
    "database/sql"
    "errors"
    "fmt"
    "strings"
    
    "github.com/lib/pq"
)

const (
    UniqueViolationCode = "23505"
    CheckViolationCode  = "23514"
    ForeignKeyViolationCode = "23503"
)

type PostgresDuplicateError struct {
    Constraint string
    Detail     string
    Field      string
}

func (e PostgresDuplicateError) Error() string {
    return fmt.Sprintf("duplicate key violation: %s", e.Detail)
}

func (r *UserRepository) CreatePostgres(user *User) error {
    query := `
        INSERT INTO users (email, username) 
        VALUES ($1, $2) 
        RETURNING id
    `
    
    err := r.db.QueryRow(query, user.Email, user.Username).Scan(&user.ID)
    if err != nil {
        var pqErr *pq.Error
        if errors.As(err, &pqErr) {
            if pqErr.Code == UniqueViolationCode {
                field := parsePostgresDuplicateField(pqErr.Constraint)
                return PostgresDuplicateError{
                    Constraint: pqErr.Constraint,
                    Detail:     pqErr.Detail,
                    Field:      field,
                }
            }
        }
        return err
    }
    
    return nil
}

func parsePostgresDuplicateField(constraint string) string {
    // Parse constraint name to determine field
    // e.g., "users_email_key" -> "email"
    parts := strings.Split(constraint, "_")
    if len(parts) >= 2 {
        return parts[1] // Assuming naming convention: table_field_key
    }
    return "unknown"
}
```

### Python with SQLAlchemy

```python
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session
import re

class DuplicateError(Exception):
    def __init__(self, field, value, message):
        self.field = field
        self.value = value
        self.message = message
        super().__init__(message)

class UserRepository:
    def __init__(self, session: Session):
        self.session = session
    
    def create_user(self, user_data: dict) -> User:
        user = User(**user_data)
        
        try:
            self.session.add(user)
            self.session.commit()
            return user
            
        except IntegrityError as e:
            self.session.rollback()
            
            # Parse the error to determine the duplicate field
            error_msg = str(e.orig)
            
            if 'users_email_key' in error_msg:
                raise DuplicateError('email', user_data['email'], 
                                   f"Email {user_data['email']} already exists")
            elif 'users_username_key' in error_msg:
                raise DuplicateError('username', user_data['username'],
                                   f"Username {user_data['username']} already exists")
            else:
                # Re-raise if we can't parse the specific field
                raise e

# Usage in service layer
class UserService:
    def __init__(self, repository: UserRepository):
        self.repository = repository
    
    def register_user(self, user_data: dict) -> dict:
        try:
            user = self.repository.create_user(user_data)
            return {
                'success': True,
                'user': {
                    'id': user.id,
                    'email': user.email,
                    'username': user.username
                }
            }
        except DuplicateError as e:
            return {
                'success': False,
                'error': e.message,
                'field': e.field
            }
```

## UPSERT Operations

### PostgreSQL UPSERT

```sql
-- Insert or update on conflict
INSERT INTO user_preferences (user_id, preference_key, preference_value)
VALUES (1, 'theme', 'dark')
ON CONFLICT (user_id, preference_key) 
DO UPDATE SET 
    preference_value = EXCLUDED.preference_value,
    updated_at = CURRENT_TIMESTAMP;

-- Insert or do nothing
INSERT INTO user_visits (user_id, page_url, visited_at)
VALUES (1, '/dashboard', CURRENT_TIMESTAMP)
ON CONFLICT (user_id, page_url, DATE(visited_at)) 
DO NOTHING;
```

### MySQL UPSERT

```sql
-- Insert or update on duplicate key
INSERT INTO user_stats (user_id, login_count, last_login)
VALUES (1, 1, NOW())
ON DUPLICATE KEY UPDATE 
    login_count = login_count + 1,
    last_login = VALUES(last_login);

-- Using REPLACE (deletes and inserts)
REPLACE INTO user_sessions (user_id, session_token, expires_at)
VALUES (1, 'abc123', DATE_ADD(NOW(), INTERVAL 1 HOUR));
```

### Application-Level UPSERT

```go
// Go example: Safe upsert with retry
func (r *UserRepository) UpsertUserPreference(userID int, key, value string) error {
    const maxRetries = 3
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        // Try insert first
        err := r.insertPreference(userID, key, value)
        if err == nil {
            return nil // Success
        }
        
        // Check if it's a duplicate error
        var dupErr DuplicateError
        if errors.As(err, &dupErr) {
            // Try update instead
            updateErr := r.updatePreference(userID, key, value)
            if updateErr == nil {
                return nil // Success
            }
            
            // If update failed, retry (might be race condition)
            continue
        }
        
        // Non-duplicate error, return immediately
        return err
    }
    
    return fmt.Errorf("failed to upsert after %d attempts", maxRetries)
}
```

## Advanced Duplicate Handling Patterns

### 1. Optimistic Locking

```sql
-- Add version column for optimistic locking
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    version INT DEFAULT 1,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

```go
func (r *UserRepository) UpdateWithOptimisticLock(user *User) error {
    query := `
        UPDATE users 
        SET email = $1, username = $2, version = version + 1, updated_at = CURRENT_TIMESTAMP
        WHERE id = $3 AND version = $4
        RETURNING version
    `
    
    var newVersion int
    err := r.db.QueryRow(query, user.Email, user.Username, user.ID, user.Version).Scan(&newVersion)
    
    if err == sql.ErrNoRows {
        return errors.New("optimistic lock failure: record was modified by another process")
    }
    
    if err != nil {
        // Handle duplicate errors as before
        return err
    }
    
    user.Version = newVersion
    return nil
}
```

### 2. Distributed Duplicate Prevention

```go
// Using Redis for distributed duplicate prevention
type DistributedUserService struct {
    repo  UserRepository
    redis *redis.Client
}

func (s *DistributedUserService) CreateUserSafely(user *User) error {
    lockKey := fmt.Sprintf("user:create:email:%s", user.Email)
    
    // Acquire distributed lock
    lock, err := s.redis.SetNX(context.Background(), lockKey, "locked", 30*time.Second).Result()
    if err != nil {
        return err
    }
    
    if !lock {
        return errors.New("another process is creating user with this email")
    }
    
    defer s.redis.Del(context.Background(), lockKey)
    
    // Check if user already exists
    existing, err := s.repo.GetByEmail(user.Email)
    if err != nil && err != sql.ErrNoRows {
        return err
    }
    
    if existing != nil {
        return DuplicateError{Field: "email", Message: "email already exists"}
    }
    
    // Create user
    return s.repo.Create(user)
}
```

### 3. Bulk Insert with Duplicate Handling

```sql
-- PostgreSQL: Bulk insert with conflict resolution
INSERT INTO products (sku, name, price)
VALUES 
    ('SKU001', 'Product 1', 10.99),
    ('SKU002', 'Product 2', 15.99),
    ('SKU003', 'Product 3', 20.99)
ON CONFLICT (sku) 
DO UPDATE SET 
    name = EXCLUDED.name,
    price = EXCLUDED.price,
    updated_at = CURRENT_TIMESTAMP;
```

```go
// Go: Batch processing with error collection
func (r *ProductRepository) BulkCreate(products []Product) ([]Product, []error) {
    var created []Product
    var errors []error
    
    for i, product := range products {
        err := r.Create(&product)
        if err != nil {
            var dupErr DuplicateError
            if errors.As(err, &dupErr) {
                errors = append(errors, fmt.Errorf("row %d: %w", i, err))
                continue
            }
            // For non-duplicate errors, fail fast
            return created, []error{fmt.Errorf("row %d: %w", i, err)}
        }
        created = append(created, product)
    }
    
    return created, errors
}
```

## Error Response Patterns

### RESTful API Error Responses

```json
{
    "error": {
        "code": "DUPLICATE_EMAIL",
        "message": "Email address already exists",
        "field": "email",
        "value": "user@example.com"
    }
}
```

```go
// Go: Structured error response
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Field   string `json:"field,omitempty"`
    Value   string `json:"value,omitempty"`
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    json.NewDecoder(r.Body).Decode(&user)
    
    err := userService.Create(&user)
    if err != nil {
        var dupErr DuplicateError
        if errors.As(err, &dupErr) {
            apiErr := APIError{
                Code:    fmt.Sprintf("DUPLICATE_%s", strings.ToUpper(dupErr.Field)),
                Message: dupErr.Message,
                Field:   dupErr.Field,
            }
            
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusConflict)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": apiErr})
            return
        }
        
        // Handle other errors...
    }
    
    // Success response...
}
```

## Best Practices

### 1. Error Handling Strategy
- **Fail Fast**: Detect duplicates as early as possible
- **Specific Messages**: Provide clear, field-specific error messages
- **Consistent Responses**: Use consistent error formats across your API
- **Log Appropriately**: Log duplicate attempts for monitoring

### 2. Performance Considerations
- **Use Unique Indexes**: Ensure database-level uniqueness constraints
- **Batch Operations**: Handle duplicates efficiently in bulk operations
- **Caching**: Cache existence checks for frequently validated fields
- **Async Validation**: Perform expensive uniqueness checks asynchronously when possible

### 3. User Experience
- **Real-time Validation**: Check for duplicates before form submission
- **Helpful Suggestions**: Suggest alternatives when duplicates are found
- **Clear Recovery**: Provide clear paths to resolve duplicate conflicts
- **Progressive Enhancement**: Handle duplicates gracefully even with JavaScript disabled

## Related Topics

- [Database Constraints](../schema-design/constraints.md) - Comprehensive constraint design
- [Error Handling Patterns](../application/error-handling.md) - Application error strategies  
- [Concurrent Programming](../application/concurrency.md) - Handling race conditions
- [API Design](../application/api-design.md) - RESTful error responses
- [Performance Optimization](../performance/README.md) - Optimizing uniqueness checks
