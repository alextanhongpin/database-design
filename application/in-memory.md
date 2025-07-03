# In-Memory Databases

In-memory database solutions for testing, development, and high-performance applications.

## 📚 Table of Contents

- [Overview](#overview)
- [Testing Use Cases](#testing-use-cases)
- [In-Memory Database Options](#in-memory-database-options)
- [Implementation Patterns](#implementation-patterns)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Overview

In-memory databases provide fast data access by storing data in RAM rather than on disk. They're particularly useful for:

- **Testing**: Fast, isolated test environments
- **Caching**: High-speed data retrieval
- **Development**: Quick setup without external dependencies
- **Analytics**: Fast processing of temporary datasets
- **Session Storage**: Temporary user state management

## Testing Use Cases

### Unit Testing Benefits
```javascript
// Fast test setup with in-memory database
describe('User Repository', () => {
  let db;
  
  beforeEach(async () => {
    db = new InMemoryDatabase();
    await db.migrate();  // Fast schema setup
  });
  
  afterEach(async () => {
    await db.close();  // Instant cleanup
  });
  
  test('creates user successfully', async () => {
    const user = await db.users.create({
      name: 'John Doe',
      email: 'john@example.com'
    });
    
    expect(user.id).toBeDefined();
    expect(user.name).toBe('John Doe');
  });
});
```

### Integration Testing
```go
// Go example with in-memory SQLite
func TestUserService(t *testing.T) {
    // Setup in-memory database
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    defer db.Close()
    
    // Run migrations
    err = runMigrations(db)
    require.NoError(t, err)
    
    // Test service with real database interactions
    service := NewUserService(db)
    user, err := service.CreateUser("John", "john@example.com")
    
    assert.NoError(t, err)
    assert.Equal(t, "John", user.Name)
}
```

## In-Memory Database Options

### SQLite In-Memory
```go
// Go with SQLite in-memory
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

func setupTestDB() (*sql.DB, error) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        return nil, err
    }
    
    // Create schema
    _, err = db.Exec(`
        CREATE TABLE users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
    
    return db, err
}
```

### H2 Database (Java)
```java
// Java with H2 in-memory database
@DataJpaTest
@TestPropertySource(properties = {
    "spring.datasource.url=jdbc:h2:mem:testdb",
    "spring.datasource.driver-class-name=org.h2.Driver",
    "spring.jpa.hibernate.ddl-auto=create-drop"
})
class UserRepositoryTest {
    
    @Autowired
    private UserRepository userRepository;
    
    @Test
    void shouldCreateUser() {
        User user = new User("John", "john@example.com");
        User saved = userRepository.save(user);
        
        assertThat(saved.getId()).isNotNull();
        assertThat(saved.getName()).isEqualTo("John");
    }
}
```

### DoltHub Go MySQL Server
Provides MySQL-compatible in-memory database for Go applications:

```go
import "github.com/dolthub/go-mysql-server"

// Create in-memory MySQL-compatible server
func createInMemoryMySQL() *sqle.Engine {
    engine := sqle.NewDefault(
        sql.NewDatabaseProvider(
            memory.NewDatabase("test"),
        ),
    )
    
    return engine
}
```

### Redis for Caching
```javascript
// Redis as in-memory cache
const redis = require('redis');
const client = redis.createClient();

class UserCache {
    async getUser(id) {
        const cached = await client.get(`user:${id}`);
        if (cached) {
            return JSON.parse(cached);
        }
        
        // Fetch from database
        const user = await database.getUser(id);
        
        // Cache for 1 hour
        await client.setex(`user:${id}`, 3600, JSON.stringify(user));
        
        return user;
    }
    
    async invalidateUser(id) {
        await client.del(`user:${id}`);
    }
}
```

### Apache Derby (Java)
```java
// Embedded Derby database
public class DerbyTestUtils {
    public static Connection createInMemoryConnection() throws SQLException {
        String url = "jdbc:derby:memory:testdb;create=true";
        return DriverManager.getConnection(url);
    }
    
    public static void setupSchema(Connection conn) throws SQLException {
        try (Statement stmt = conn.createStatement()) {
            stmt.execute("""
                CREATE TABLE users (
                    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
                    name VARCHAR(255) NOT NULL,
                    email VARCHAR(255) UNIQUE NOT NULL,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """);
        }
    }
}
```

## Implementation Patterns

### Repository Pattern with In-Memory Backend
```python
# Python abstract repository with in-memory implementation
from abc import ABC, abstractmethod
from typing import Dict, List, Optional

class UserRepository(ABC):
    @abstractmethod
    def create(self, user: User) -> User:
        pass
    
    @abstractmethod
    def find_by_id(self, user_id: int) -> Optional[User]:
        pass
    
    @abstractmethod
    def find_by_email(self, email: str) -> Optional[User]:
        pass

class InMemoryUserRepository(UserRepository):
    def __init__(self):
        self._users: Dict[int, User] = {}
        self._next_id = 1
    
    def create(self, user: User) -> User:
        user.id = self._next_id
        self._next_id += 1
        self._users[user.id] = user
        return user
    
    def find_by_id(self, user_id: int) -> Optional[User]:
        return self._users.get(user_id)
    
    def find_by_email(self, email: str) -> Optional[User]:
        for user in self._users.values():
            if user.email == email:
                return user
        return None

# Usage in tests
def test_user_creation():
    repo = InMemoryUserRepository()
    user = User(name="John", email="john@example.com")
    
    created = repo.create(user)
    found = repo.find_by_id(created.id)
    
    assert found.name == "John"
    assert found.email == "john@example.com"
```

### Test Database Factory
```typescript
// TypeScript test database factory
interface TestDatabase {
  query<T>(sql: string, params?: any[]): Promise<T[]>;
  close(): Promise<void>;
}

class InMemoryTestDatabase implements TestDatabase {
  private sqlite: Database;
  
  constructor() {
    this.sqlite = new Database(':memory:');
  }
  
  async query<T>(sql: string, params: any[] = []): Promise<T[]> {
    return new Promise((resolve, reject) => {
      this.sqlite.all(sql, params, (err, rows) => {
        if (err) reject(err);
        else resolve(rows as T[]);
      });
    });
  }
  
  async migrate(): Promise<void> {
    await this.query(`
      CREATE TABLE users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `);
  }
  
  async close(): Promise<void> {
    return new Promise((resolve) => {
      this.sqlite.close(resolve);
    });
  }
}

// Test factory
export function createTestDatabase(): TestDatabase {
  const db = new InMemoryTestDatabase();
  return db;
}
```

### Data Seeding
```go
// Go test data seeding
type TestData struct {
    Users []User
    Posts []Post
}

func SeedTestData(db *sql.DB) (*TestData, error) {
    data := &TestData{}
    
    // Create users
    users := []User{
        {Name: "Alice", Email: "alice@example.com"},
        {Name: "Bob", Email: "bob@example.com"},
    }
    
    for _, user := range users {
        err := db.QueryRow(
            "INSERT INTO users (name, email) VALUES (?, ?) RETURNING id",
            user.Name, user.Email,
        ).Scan(&user.ID)
        
        if err != nil {
            return nil, err
        }
        
        data.Users = append(data.Users, user)
    }
    
    // Create posts
    posts := []Post{
        {UserID: data.Users[0].ID, Title: "First Post", Content: "Hello World"},
        {UserID: data.Users[1].ID, Title: "Second Post", Content: "Testing"},
    }
    
    for _, post := range posts {
        err := db.QueryRow(
            "INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?) RETURNING id",
            post.UserID, post.Title, post.Content,
        ).Scan(&post.ID)
        
        if err != nil {
            return nil, err
        }
        
        data.Posts = append(data.Posts, post)
    }
    
    return data, nil
}
```

## Performance Considerations

### Memory Usage
```javascript
// Monitor memory usage in long-running tests
class MemoryMonitor {
    private initialMemory: number;
    
    constructor() {
        this.initialMemory = process.memoryUsage().heapUsed;
    }
    
    checkMemoryLeak(threshold: number = 50 * 1024 * 1024) {  // 50MB
        const currentMemory = process.memoryUsage().heapUsed;
        const difference = currentMemory - this.initialMemory;
        
        if (difference > threshold) {
            console.warn(`Potential memory leak detected: ${difference / 1024 / 1024}MB increase`);
        }
        
        return difference;
    }
}

// Usage in tests
describe('Long running tests', () => {
    let monitor: MemoryMonitor;
    
    beforeEach(() => {
        monitor = new MemoryMonitor();
    });
    
    afterEach(() => {
        monitor.checkMemoryLeak();
    });
});
```

### Connection Pooling
```java
// Java connection pooling for in-memory tests
@Configuration
@TestProfile
public class TestDatabaseConfig {
    
    @Bean
    @Primary
    public DataSource testDataSource() {
        HikariConfig config = new HikariConfig();
        config.setJdbcUrl("jdbc:h2:mem:testdb;DB_CLOSE_DELAY=-1");
        config.setDriverClassName("org.h2.Driver");
        config.setMaximumPoolSize(10);  // Limit connections for tests
        config.setMinimumIdle(2);
        config.setConnectionTimeout(5000);
        
        return new HikariDataSource(config);
    }
}
```

### Bulk Operations
```python
# Efficient bulk operations for test data
class BulkDataLoader:
    def __init__(self, connection):
        self.conn = connection
    
    def bulk_insert_users(self, users: List[Dict]) -> List[int]:
        cursor = self.conn.cursor()
        
        # Use executemany for bulk inserts
        cursor.executemany(
            "INSERT INTO users (name, email) VALUES (?, ?)",
            [(user['name'], user['email']) for user in users]
        )
        
        # Get generated IDs
        cursor.execute(
            "SELECT id FROM users WHERE email IN ({})".format(
                ','.join('?' * len(users))
            ),
            [user['email'] for user in users]
        )
        
        return [row[0] for row in cursor.fetchall()]
```

## Best Practices

### 1. Test Isolation
```typescript
// Ensure tests don't interfere with each other
describe('User Service', () => {
    let database: TestDatabase;
    
    beforeEach(async () => {
        database = createTestDatabase();
        await database.migrate();
    });
    
    afterEach(async () => {
        await database.close();  // Clean slate for each test
    });
    
    test('should not see data from other tests', async () => {
        const users = await database.query('SELECT * FROM users');
        expect(users).toHaveLength(0);  // Always starts empty
    });
});
```

### 2. Schema Versioning
```go
// Version your test schemas
type Migration struct {
    Version int
    SQL     string
}

var migrations = []Migration{
    {1, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`},
    {2, `ALTER TABLE users ADD COLUMN email TEXT`},
    {3, `CREATE INDEX idx_users_email ON users(email)`},
}

func MigrateToVersion(db *sql.DB, targetVersion int) error {
    for _, migration := range migrations {
        if migration.Version <= targetVersion {
            _, err := db.Exec(migration.SQL)
            if err != nil {
                return fmt.Errorf("migration %d failed: %w", migration.Version, err)
            }
        }
    }
    return nil
}
```

### 3. Configuration Management
```yaml
# test-config.yml
test:
  database:
    type: "in-memory"
    provider: "sqlite"
    url: ":memory:"
    pool_size: 5
    timeout: 5s
    
  fixtures:
    auto_load: true
    path: "./fixtures"
    
  cleanup:
    strategy: "truncate"  # or "drop_recreate"
    parallel: true
```

### 4. Fixture Management
```python
# Reusable test fixtures
import pytest
from typing import Generator

@pytest.fixture
def in_memory_db() -> Generator[Database, None, None]:
    db = create_in_memory_database()
    db.migrate()
    yield db
    db.close()

@pytest.fixture
def sample_users(in_memory_db) -> List[User]:
    users = [
        User(name="Alice", email="alice@test.com"),
        User(name="Bob", email="bob@test.com"),
    ]
    
    for user in users:
        in_memory_db.create_user(user)
    
    return users

def test_user_count(sample_users):
    assert len(sample_users) == 2
```

### 5. Error Simulation
```javascript
// Simulate database errors in tests
class ControllableInMemoryDB {
    private shouldFailNext = false;
    private failureType = 'connection';
    
    simulateFailure(type: 'connection' | 'timeout' | 'constraint') {
        this.shouldFailNext = true;
        this.failureType = type;
    }
    
    async query(sql: string, params: any[] = []) {
        if (this.shouldFailNext) {
            this.shouldFailNext = false;
            
            switch (this.failureType) {
                case 'connection':
                    throw new Error('Connection lost');
                case 'timeout':
                    throw new Error('Query timeout');
                case 'constraint':
                    throw new Error('Constraint violation');
            }
        }
        
        return this.realQuery(sql, params);
    }
}

// Test error handling
test('handles connection errors gracefully', async () => {
    const db = new ControllableInMemoryDB();
    db.simulateFailure('connection');
    
    await expect(userService.createUser('John')).rejects.toThrow('Connection lost');
});
```

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Tests with In-Memory DB
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v2
    
    - name: Setup Node.js
      uses: actions/setup-node@v2
      with:
        node-version: '16'
    
    - name: Install dependencies
      run: npm ci
    
    - name: Run tests with in-memory database
      run: npm test
      env:
        NODE_ENV: test
        DB_TYPE: memory
```

## Quick Reference

### When to Use In-Memory Databases
- ✅ Unit and integration testing
- ✅ Development environments
- ✅ Temporary data processing
- ✅ Caching layers
- ✅ CI/CD pipelines

### When NOT to Use
- ❌ Production data storage
- ❌ Data that must survive restarts
- ❌ Large datasets that exceed RAM
- ❌ Multi-process shared data
- ❌ ACID compliance requirements

### Popular Options
- **SQLite** (`:memory:`): SQL-compatible, lightweight
- **H2**: Java-based, MySQL/PostgreSQL compatible
- **DoltHub go-mysql-server**: MySQL-compatible for Go
- **Redis**: Key-value store with persistence options
- **Apache Derby**: Pure Java embedded database
