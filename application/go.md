# Go Database Programming Guide

A comprehensive guide to database programming in Go, covering connection management, prepared statements, JSON handling, and advanced patterns.

## Database Connection Setup

### Basic Connection Configuration

```go
package main

import (
    "database/sql"
    "log"
    "time"
    
    _ "github.com/go-sql-driver/mysql"
    _ "github.com/lib/pq" // PostgreSQL
)

func setupDatabase(dsn string) (*sql.DB, error) {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)        // Maximum number of open connections
    db.SetMaxIdleConns(5)         // Maximum number of idle connections  
    db.SetConnMaxLifetime(time.Hour) // Maximum connection lifetime
    db.SetConnMaxIdleTime(10 * time.Minute) // Maximum idle time
    
    // Test the connection
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

### Connection Pool Best Practices

```go
type DatabaseConfig struct {
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    ConnMaxIdleTime time.Duration
}

func configureProdDatabase(db *sql.DB) {
    config := DatabaseConfig{
        MaxOpenConns:    100,                 // High traffic production
        MaxIdleConns:    10,                  // Keep some connections ready
        ConnMaxLifetime: 1 * time.Hour,       // Rotate connections hourly
        ConnMaxIdleTime: 10 * time.Minute,    // Close idle connections
    }
    
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

func configureDevDatabase(db *sql.DB) {
    config := DatabaseConfig{
        MaxOpenConns:    5,                   // Low resource usage
        MaxIdleConns:    2,                   // Minimal idle connections
        ConnMaxLifetime: 5 * time.Minute,     // Quick rotation for testing
        ConnMaxIdleTime: 1 * time.Minute,     // Aggressive cleanup
    }
    
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}
```

### Health Checks and Monitoring

```go
func (db *Database) HealthCheck() error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    return db.PingContext(ctx)
}

func (db *Database) GetStats() sql.DBStats {
    return db.Stats()
}

func logDatabaseStats(db *sql.DB) {
    stats := db.Stats()
    log.Printf("DB Stats - Open: %d, InUse: %d, Idle: %d, WaitCount: %d, WaitDuration: %v",
        stats.OpenConnections,
        stats.InUse,
        stats.Idle,
        stats.WaitCount,
        stats.WaitDuration,
    )
}
```

## Prepared Statements

### Statement Management Pattern

```go
package database

import (
    "database/sql"
    "fmt"
)

type StatementID int

const (
    CreateUserStmt StatementID = iota
    GetUserByIDStmt
    GetUserByEmailStmt
    UpdateUserStmt
    DeleteUserStmt
    ListUsersStmt
)

type StatementDefinition struct {
    ID   StatementID
    SQL  string
    Name string
}

var statementDefinitions = []StatementDefinition{
    {
        ID:   CreateUserStmt,
        Name: "CreateUser",
        SQL: `INSERT INTO users (email, username, password_hash, created_at) 
              VALUES (?, ?, ?, NOW())`,
    },
    {
        ID:   GetUserByIDStmt,
        Name: "GetUserByID", 
        SQL:  `SELECT id, email, username, created_at, updated_at FROM users WHERE id = ?`,
    },
    {
        ID:   GetUserByEmailStmt,
        Name: "GetUserByEmail",
        SQL:  `SELECT id, email, username, created_at, updated_at FROM users WHERE email = ?`,
    },
    {
        ID:   UpdateUserStmt,
        Name: "UpdateUser",
        SQL: `UPDATE users SET username = ?, updated_at = NOW() 
              WHERE id = ? AND version = ?`,
    },
    {
        ID:   DeleteUserStmt,
        Name: "DeleteUser",
        SQL:  `DELETE FROM users WHERE id = ?`,
    },
    {
        ID:   ListUsersStmt,
        Name: "ListUsers",
        SQL:  `SELECT id, email, username, created_at FROM users LIMIT ? OFFSET ?`,
    },
}

type PreparedStatements struct {
    statements map[StatementID]*sql.Stmt
}

func NewPreparedStatements(db *sql.DB) (*PreparedStatements, error) {
    statements := make(map[StatementID]*sql.Stmt)
    
    for _, def := range statementDefinitions {
        stmt, err := db.Prepare(def.SQL)
        if err != nil {
            // Clean up already prepared statements
            for _, preparedStmt := range statements {
                preparedStmt.Close()
            }
            return nil, fmt.Errorf("failed to prepare statement %s: %w", def.Name, err)
        }
        statements[def.ID] = stmt
    }
    
    return &PreparedStatements{statements: statements}, nil
}

func (ps *PreparedStatements) Get(id StatementID) *sql.Stmt {
    return ps.statements[id]
}

func (ps *PreparedStatements) Close() error {
    for _, stmt := range ps.statements {
        if err := stmt.Close(); err != nil {
            return err
        }
    }
    return nil
}
```

### Repository Pattern with Prepared Statements

```go
type UserRepository struct {
    db    *sql.DB
    stmts *PreparedStatements
}

func NewUserRepository(db *sql.DB) (*UserRepository, error) {
    stmts, err := NewPreparedStatements(db)
    if err != nil {
        return nil, err
    }
    
    return &UserRepository{
        db:    db,
        stmts: stmts,
    }, nil
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    stmt := r.stmts.Get(CreateUserStmt)
    
    result, err := stmt.ExecContext(ctx, user.Email, user.Username, user.PasswordHash)
    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }
    
    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get user ID: %w", err)
    }
    
    user.ID = id
    return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
    stmt := r.stmts.Get(GetUserByIDStmt)
    
    var user User
    err := stmt.QueryRowContext(ctx, id).Scan(
        &user.ID,
        &user.Email,
        &user.Username,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, ErrUserNotFound
    }
    
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return &user, nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
    stmt := r.stmts.Get(ListUsersStmt)
    
    rows, err := stmt.QueryContext(ctx, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to query users: %w", err)
    }
    defer rows.Close()
    
    var users []*User
    for rows.Next() {
        var user User
        err := rows.Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, &user)
    }
    
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows iteration error: %w", err)
    }
    
    return users, nil
}

func (r *UserRepository) Close() error {
    return r.stmts.Close()
}
```

## JSON Handling in Go

### Database JSON Operations

```go
// MySQL JSON handling
func (r *UserRepository) GetUserAsJSON(ctx context.Context, id int64) ([]byte, error) {
    query := `
        SELECT JSON_OBJECT(
            'id', id,
            'email', email,
            'username', username,
            'emailVerified', email_verified = 1,
            'createdAt', DATE_FORMAT(created_at, '%Y-%m-%dT%H:%i:%sZ'),
            'profile', JSON_OBJECT(
                'firstName', first_name,
                'lastName', last_name,
                'avatar', avatar_url
            )
        ) FROM users WHERE id = ?
    `
    
    var jsonData []byte
    err := r.db.QueryRowContext(ctx, query, id).Scan(&jsonData)
    if err != nil {
        return nil, fmt.Errorf("failed to get user JSON: %w", err)
    }
    
    return jsonData, nil
}

// PostgreSQL JSON handling  
func (r *UserRepository) GetUserAsJSONPostgres(ctx context.Context, id int64) ([]byte, error) {
    query := `
        SELECT json_build_object(
            'id', id,
            'email', email,
            'username', username,
            'emailVerified', email_verified,
            'createdAt', to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
            'profile', json_build_object(
                'firstName', first_name,
                'lastName', last_name,
                'avatar', avatar_url
            )
        ) FROM users WHERE id = $1
    `
    
    var jsonData []byte
    err := r.db.QueryRowContext(ctx, query, id).Scan(&jsonData)
    if err != nil {
        return nil, fmt.Errorf("failed to get user JSON: %w", err)
    }
    
    return jsonData, nil
}
```

### JSON Column Operations

```go
// Working with JSON columns
type UserPreferences struct {
    Theme      string            `json:"theme"`
    Language   string            `json:"language"`
    Settings   map[string]string `json:"settings"`
}

func (r *UserRepository) UpdatePreferences(ctx context.Context, userID int64, prefs UserPreferences) error {
    prefsJSON, err := json.Marshal(prefs)
    if err != nil {
        return fmt.Errorf("failed to marshal preferences: %w", err)
    }
    
    query := `UPDATE users SET preferences = ? WHERE id = ?`
    _, err = r.db.ExecContext(ctx, query, prefsJSON, userID)
    if err != nil {
        return fmt.Errorf("failed to update preferences: %w", err)
    }
    
    return nil
}

func (r *UserRepository) GetPreferences(ctx context.Context, userID int64) (*UserPreferences, error) {
    query := `SELECT preferences FROM users WHERE id = ?`
    
    var prefsJSON sql.NullString
    err := r.db.QueryRowContext(ctx, query, userID).Scan(&prefsJSON)
    if err != nil {
        return nil, fmt.Errorf("failed to get preferences: %w", err)
    }
    
    if !prefsJSON.Valid {
        return &UserPreferences{}, nil // Return empty preferences
    }
    
    var prefs UserPreferences
    err = json.Unmarshal([]byte(prefsJSON.String), &prefs)
    if err != nil {
        return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
    }
    
    return &prefs, nil
}

// JSON path queries (MySQL 5.7+)
func (r *UserRepository) GetUsersByTheme(ctx context.Context, theme string) ([]*User, error) {
    query := `
        SELECT id, email, username 
        FROM users 
        WHERE JSON_EXTRACT(preferences, '$.theme') = ?
    `
    
    rows, err := r.db.QueryContext(ctx, query, theme)
    if err != nil {
        return nil, fmt.Errorf("failed to query users by theme: %w", err)
    }
    defer rows.Close()
    
    var users []*User
    for rows.Next() {
        var user User
        err := rows.Scan(&user.ID, &user.Email, &user.Username)
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, &user)
    }
    
    return users, nil
}
```

## Advanced UPSERT Patterns

### Smart UPSERT with NULLIF and COALESCE

```go
// Update only changed fields using NULLIF and COALESCE
func (r *UserRepository) SmartUpdate(ctx context.Context, userID int64, updates UserUpdateRequest) error {
    query := `
        INSERT INTO users (id, username, email, first_name, last_name, updated_at) 
        VALUES (?, ?, ?, ?, ?, NOW()) 
        ON DUPLICATE KEY UPDATE 
            username = COALESCE(NULLIF(NULLIF(VALUES(username), username), ''), username),
            email = COALESCE(NULLIF(NULLIF(VALUES(email), email), ''), email),
            first_name = COALESCE(NULLIF(VALUES(first_name), ''), first_name),
            last_name = COALESCE(NULLIF(VALUES(last_name), ''), last_name),
            updated_at = IF(
                VALUES(username) != username OR 
                VALUES(email) != email OR 
                VALUES(first_name) != first_name OR 
                VALUES(last_name) != last_name,
                NOW(),
                updated_at
            )
    `
    
    _, err := r.db.ExecContext(ctx, query, 
        userID, 
        updates.Username, 
        updates.Email, 
        updates.FirstName, 
        updates.LastName,
    )
    
    if err != nil {
        return fmt.Errorf("failed to smart update user: %w", err)
    }
    
    return nil
}

// PostgreSQL UPSERT with conflict handling
func (r *UserRepository) UpsertPostgres(ctx context.Context, user *User) error {
    query := `
        INSERT INTO users (email, username, first_name, last_name, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
        ON CONFLICT (email) 
        DO UPDATE SET
            username = CASE 
                WHEN EXCLUDED.username IS NOT NULL AND EXCLUDED.username != '' 
                THEN EXCLUDED.username 
                ELSE users.username 
            END,
            first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), users.first_name),
            last_name = COALESCE(NULLIF(EXCLUDED.last_name, ''), users.last_name),
            updated_at = NOW()
        WHERE 
            users.username IS DISTINCT FROM EXCLUDED.username OR
            users.first_name IS DISTINCT FROM EXCLUDED.first_name OR
            users.last_name IS DISTINCT FROM EXCLUDED.last_name
        RETURNING id, created_at, updated_at
    `
    
    err := r.db.QueryRowContext(ctx, query, 
        user.Email, 
        user.Username, 
        user.FirstName, 
        user.LastName,
    ).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
    
    if err != nil {
        return fmt.Errorf("failed to upsert user: %w", err)
    }
    
    return nil
}
```

## Transaction Management

### Transaction Helper Functions

```go
type TxFunc func(*sql.Tx) error

func (db *Database) WithTransaction(ctx context.Context, fn TxFunc) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p) // Re-throw panic after rollback
        }
    }()
    
    err = fn(tx)
    if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("transaction error: %v, rollback error: %v", err, rbErr)
        }
        return err
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}

// Usage example
func (s *UserService) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
        // Create user
        result, err := tx.ExecContext(ctx, 
            "INSERT INTO users (email, username) VALUES (?, ?)",
            user.Email, user.Username,
        )
        if err != nil {
            return err
        }
        
        userID, err := result.LastInsertId()
        if err != nil {
            return err
        }
        
        user.ID = userID
        
        // Create profile
        _, err = tx.ExecContext(ctx,
            "INSERT INTO profiles (user_id, first_name, last_name) VALUES (?, ?, ?)",
            userID, profile.FirstName, profile.LastName,
        )
        
        return err
    })
}
```

### Repository with Transaction Support

```go
type UserRepository struct {
    db *sql.DB
    tx *sql.Tx // Optional transaction
}

func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
    return &UserRepository{
        db: r.db,
        tx: tx,
    }
}

func (r *UserRepository) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    if r.tx != nil {
        return r.tx.ExecContext(ctx, query, args...)
    }
    return r.db.ExecContext(ctx, query, args...)
}

func (r *UserRepository) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
    if r.tx != nil {
        return r.tx.QueryRowContext(ctx, query, args...)
    }
    return r.db.QueryRowContext(ctx, query, args...)
}

func (r *UserRepository) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    if r.tx != nil {
        return r.tx.QueryContext(ctx, query, args...)
    }
    return r.db.QueryContext(ctx, query, args...)
}
```

## Error Handling Patterns

### Custom Error Types

```go
type DatabaseError struct {
    Operation string
    Table     string
    Err       error
}

func (e *DatabaseError) Error() string {
    return fmt.Sprintf("database error during %s on %s: %v", e.Operation, e.Table, e.Err)
}

func (e *DatabaseError) Unwrap() error {
    return e.Err
}

// Error type checking helpers
func IsConstraintError(err error) bool {
    var mysqlErr *mysql.MySQLError
    if errors.As(err, &mysqlErr) {
        switch mysqlErr.Number {
        case 1062: // Duplicate entry
            return true
        case 1452: // Foreign key constraint fails
            return true
        }
    }
    
    var pqErr *pq.Error
    if errors.As(err, &pqErr) {
        switch pqErr.Code {
        case "23505": // Unique violation
            return true
        case "23503": // Foreign key violation
            return true
        }
    }
    
    return false
}

func IsConnectionError(err error) bool {
    return errors.Is(err, sql.ErrConnDone) || 
           strings.Contains(err.Error(), "connection refused")
}
```

## Performance Monitoring

### Query Performance Logging

```go
type QueryLogger struct {
    logger *log.Logger
}

func (ql *QueryLogger) LogQuery(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) {
    level := "INFO"
    if err != nil {
        level = "ERROR"
    } else if duration > 100*time.Millisecond {
        level = "WARN"
    }
    
    ql.logger.Printf("[%s] Query took %v: %s (args: %v) error: %v", 
        level, duration, query, args, err)
}

// Wrapper for timed queries
func (r *UserRepository) QueryWithLogging(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    start := time.Now()
    rows, err := r.db.QueryContext(ctx, query, args...)
    duration := time.Since(start)
    
    r.logger.LogQuery(ctx, query, args, duration, err)
    return rows, err
}
```

## Best Practices Summary

### Connection Management
1. **Configure connection pools appropriately** for your environment
2. **Monitor connection statistics** regularly
3. **Use health checks** to detect connection issues early
4. **Handle connection errors gracefully** with retries

### Prepared Statements
1. **Pre-compile all static queries** for better performance
2. **Organize statements by functionality** using enums/constants
3. **Handle preparation errors at startup** to fail fast
4. **Clean up statements properly** when shutting down

### Error Handling
1. **Create specific error types** for different database errors
2. **Log queries and performance metrics** for debugging
3. **Handle constraint violations gracefully** with user-friendly messages
4. **Use proper error wrapping** to maintain error chains

### Performance
1. **Use prepared statements** for repeated queries
2. **Implement query timeouts** to prevent hanging
3. **Monitor slow queries** and optimize them
4. **Use transactions appropriately** for data consistency

## Related Topics

- [Connection Pooling](../performance/connection-pooling.md) - Advanced pooling strategies
- [Database Testing](../operations/testing.md) - Testing database code in Go
- [Error Handling](error-handling.md) - Application error patterns
- [Performance Monitoring](../operations/monitoring.md) - Database performance tracking
- [Migration Tools](../operations/migrations.md) - Schema migration in Go
