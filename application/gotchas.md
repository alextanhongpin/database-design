
# Database Gotchas and Common Pitfalls

Common mistakes, edge cases, and unexpected behaviors when working with databases in application development.

## 📚 Table of Contents

- [PostgreSQL Gotchas](#postgresql-gotchas)
- [MySQL Gotchas](#mysql-gotchas)
- [General SQL Gotchas](#general-sql-gotchas)
- [Application-Level Gotchas](#application-level-gotchas)
- [Performance Gotchas](#performance-gotchas)
- [Schema Design Gotchas](#schema-design-gotchas)
- [Migration Gotchas](#migration-gotchas)
- [Testing Gotchas](#testing-gotchas)

## PostgreSQL Gotchas

### Transaction Timestamp Behavior
```sql
-- ❌ This timestamp remains constant throughout the transaction
BEGIN;
INSERT INTO events (message, created_at) VALUES ('First', NOW());
-- Wait 5 seconds
INSERT INTO events (message, created_at) VALUES ('Second', NOW());
COMMIT;
-- Both records will have the same timestamp!

-- ✅ Use clock_timestamp() for actual current time
BEGIN;
INSERT INTO events (message, created_at) VALUES ('First', clock_timestamp());
-- Wait 5 seconds
INSERT INTO events (message, created_at) VALUES ('Second', clock_timestamp());
COMMIT;
-- Records will have different timestamps
```

### Reserved Keywords
```sql
-- ❌ These are reserved keywords in PostgreSQL
CREATE TABLE user (...);     -- Use 'users' or 'account' instead
CREATE TABLE order (...);    -- Use 'orders' or 'order_item' instead
CREATE TABLE group (...);    -- Use 'groups' or 'user_group' instead

-- ✅ Safe alternatives
CREATE TABLE users (...);
CREATE TABLE orders (...);
CREATE TABLE user_groups (...);
```

### Manual Timestamp Updates
```sql
-- PostgreSQL doesn't auto-update timestamps like MySQL
-- ❌ This won't work
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  title TEXT,
  updated_at TIMESTAMP DEFAULT NOW() ON UPDATE NOW()  -- Invalid syntax
);

-- ✅ Create a trigger instead
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_posts_updated_at 
    BEFORE UPDATE ON posts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

### Case Sensitivity
```sql
-- ❌ Unquoted identifiers are folded to lowercase
CREATE TABLE MyTable (MyColumn TEXT);
SELECT MyColumn FROM MyTable;  -- Error: column "mycolumn" doesn't exist

-- ✅ Use consistent lowercase naming
CREATE TABLE my_table (my_column TEXT);
SELECT my_column FROM my_table;  -- Works

-- ✅ Or quote identifiers (not recommended)
CREATE TABLE "MyTable" ("MyColumn" TEXT);
SELECT "MyColumn" FROM "MyTable";  -- Works but confusing
```

### UUID Performance
```sql
-- ❌ Random UUIDs can cause index fragmentation
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT
);

-- ✅ Consider ordered UUIDs for better performance
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v1mc(),  -- Time-based
  name TEXT
);
```

## MySQL Gotchas

### Silent Data Truncation
```sql
-- ❌ MySQL may silently truncate data in non-strict mode
CREATE TABLE posts (title VARCHAR(10));
INSERT INTO posts VALUES ('This is a very long title');  -- Truncated!

-- ✅ Enable strict mode
SET sql_mode = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
```

### Zero Dates
```sql
-- ❌ MySQL allows invalid dates
INSERT INTO events (date) VALUES ('0000-00-00');  -- Invalid but accepted

-- ✅ Use proper NULL handling
ALTER TABLE events MODIFY date DATE NULL;
INSERT INTO events (date) VALUES (NULL);
```

### AUTO_INCREMENT Gaps
```sql
-- ❌ Assuming sequential AUTO_INCREMENT values
CREATE TABLE orders (
  id INT AUTO_INCREMENT PRIMARY KEY,
  total DECIMAL(10,2)
);

-- Rollbacks, failed inserts, or crashes can create gaps
-- Don't rely on sequential IDs for business logic
```

## General SQL Gotchas

### NULL Comparisons
```sql
-- ❌ NULL comparisons don't work as expected
SELECT * FROM users WHERE age = NULL;     -- Returns no rows
SELECT * FROM users WHERE age != NULL;    -- Returns no rows

-- ✅ Use IS NULL / IS NOT NULL
SELECT * FROM users WHERE age IS NULL;
SELECT * FROM users WHERE age IS NOT NULL;
```

### String Comparisons
```sql
-- ❌ Case sensitivity varies by database
SELECT * FROM users WHERE name = 'John';  -- May or may not match 'john'

-- ✅ Explicit case handling
SELECT * FROM users WHERE LOWER(name) = LOWER('John');
SELECT * FROM users WHERE name ILIKE 'john';  -- PostgreSQL
```

### Date Arithmetic
```sql
-- ❌ Database-specific date functions
SELECT * FROM events WHERE date > NOW() - 7;  -- MySQL days
SELECT * FROM events WHERE date > NOW() - INTERVAL '7 days';  -- PostgreSQL

-- ✅ Use standards-compliant syntax when possible
SELECT * FROM events WHERE date > CURRENT_DATE - INTERVAL '7' DAY;
```

## Application-Level Gotchas

### Connection Pool Exhaustion
```javascript
// ❌ Not releasing connections
async function badQuery() {
  const client = await pool.connect();
  const result = await client.query('SELECT * FROM users');
  // Missing client.release()!
  return result.rows;
}

// ✅ Always release connections
async function goodQuery() {
  const client = await pool.connect();
  try {
    const result = await client.query('SELECT * FROM users');
    return result.rows;
  } finally {
    client.release();
  }
}
```

### N+1 Query Problem
```javascript
// ❌ N+1 queries
async function getPosts() {
  const posts = await db.query('SELECT * FROM posts');
  for (const post of posts) {
    post.author = await db.query('SELECT * FROM users WHERE id = ?', [post.user_id]);
  }
  return posts;
}

// ✅ Use joins or batch queries
async function getPosts() {
  return db.query(`
    SELECT p.*, u.name as author_name, u.email as author_email
    FROM posts p
    JOIN users u ON p.user_id = u.id
  `);
}
```

### SQL Injection
```javascript
// ❌ String concatenation
const query = `SELECT * FROM users WHERE name = '${userInput}'`;

// ✅ Parameterized queries
const query = 'SELECT * FROM users WHERE name = ?';
const result = await db.query(query, [userInput]);
```

### Transaction Scope
```javascript
// ❌ Long-running transactions
async function processOrders() {
  await db.beginTransaction();
  const orders = await db.query('SELECT * FROM orders WHERE status = "pending"');
  
  for (const order of orders) {
    // This could take minutes!
    await processPayment(order);
    await updateInventory(order);
    await sendEmail(order);
  }
  
  await db.commit();  // Locks held too long
}

// ✅ Shorter transaction scope
async function processOrders() {
  const orders = await db.query('SELECT * FROM orders WHERE status = "pending"');
  
  for (const order of orders) {
    await db.beginTransaction();
    try {
      await updateOrderStatus(order.id, 'processing');
      await db.commit();
      
      // Do external operations outside transaction
      await processPayment(order);
      await sendEmail(order);
    } catch (error) {
      await db.rollback();
      throw error;
    }
  }
}
```

## Performance Gotchas

### Missing Indexes
```sql
-- ❌ Querying without appropriate indexes
SELECT * FROM orders WHERE customer_id = 123 AND status = 'pending';
-- Slow if no index on (customer_id, status)

-- ✅ Create composite indexes for common queries
CREATE INDEX idx_orders_customer_status ON orders (customer_id, status);
```

### Inefficient Pagination
```sql
-- ❌ OFFSET becomes slow with large offsets
SELECT * FROM posts ORDER BY id LIMIT 20 OFFSET 100000;  -- Very slow

-- ✅ Use cursor-based pagination
SELECT * FROM posts WHERE id > 1000 ORDER BY id LIMIT 20;
```

### Function Calls in WHERE
```sql
-- ❌ Functions prevent index usage
SELECT * FROM users WHERE LOWER(email) = 'john@example.com';

-- ✅ Use functional indexes or store lowercase values
CREATE INDEX idx_users_email_lower ON users (LOWER(email));
-- Or store normalized data
```

## Schema Design Gotchas

### Premature Normalization
```sql
-- ❌ Over-normalized for simple use cases
CREATE TABLE user_addresses (
  id SERIAL PRIMARY KEY,
  user_id INT REFERENCES users(id),
  street TEXT,
  city TEXT,
  country TEXT
);

-- ✅ Sometimes denormalization is better
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name TEXT,
  address_street TEXT,
  address_city TEXT,
  address_country TEXT
);
```

### Inadequate Data Types
```sql
-- ❌ Using wrong data types
CREATE TABLE products (
  price VARCHAR(10),  -- Should be DECIMAL
  quantity VARCHAR(5), -- Should be INTEGER
  is_active VARCHAR(5) -- Should be BOOLEAN
);

-- ✅ Use appropriate data types
CREATE TABLE products (
  price DECIMAL(10,2),
  quantity INTEGER,
  is_active BOOLEAN
);
```

## Migration Gotchas

### Non-Atomic Migrations
```sql
-- ❌ Multiple operations that could fail partially
ALTER TABLE users ADD COLUMN email VARCHAR(255);
UPDATE users SET email = 'unknown@example.com' WHERE email IS NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;

-- ✅ Break into separate migrations or use transactions
BEGIN;
ALTER TABLE users ADD COLUMN email VARCHAR(255);
UPDATE users SET email = 'unknown@example.com' WHERE email IS NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
COMMIT;
```

### Blocking Operations
```sql
-- ❌ Operations that lock tables for too long
ALTER TABLE large_table ADD COLUMN new_col TEXT DEFAULT 'default_value';

-- ✅ Add column first, then set default separately
ALTER TABLE large_table ADD COLUMN new_col TEXT;
-- Update in batches
UPDATE large_table SET new_col = 'default_value' WHERE id BETWEEN 1 AND 1000;
-- Continue in batches...
```

## Testing Gotchas

### Shared Test Database
```javascript
// ❌ Tests interfering with each other
describe('User tests', () => {
  test('creates user', async () => {
    await db.query("INSERT INTO users (name) VALUES ('John')");
    const users = await db.query("SELECT * FROM users");
    expect(users).toHaveLength(1);  // Fails if other tests ran first
  });
});

// ✅ Clean database state between tests
beforeEach(async () => {
  await db.query('TRUNCATE TABLE users RESTART IDENTITY CASCADE');
});
```

### Time-Dependent Tests
```javascript
// ❌ Tests that depend on current time
test('creates user with current timestamp', async () => {
  const user = await createUser({name: 'John'});
  expect(user.created_at).toBe(new Date());  // Likely to fail due to timing
});

// ✅ Mock time or use ranges
test('creates user with current timestamp', async () => {
  const before = new Date();
  const user = await createUser({name: 'John'});
  const after = new Date();
  
  expect(user.created_at).toBeGreaterThanOrEqual(before);
  expect(user.created_at).toBeLessThanOrEqual(after);
});
```

## Best Practices to Avoid Gotchas

### 1. Use Static Analysis
- Enable SQL linting tools
- Use type-safe query builders
- Enable strict mode in your database

### 2. Comprehensive Testing
- Test edge cases and error conditions
- Use database-specific test utilities
- Mock external dependencies

### 3. Monitoring and Observability
- Log slow queries
- Monitor connection pool usage
- Track transaction durations

### 4. Documentation
- Document database-specific behaviors
- Maintain migration rollback procedures
- Keep schema change logs

### 5. Code Reviews
- Review database interactions carefully
- Check for common anti-patterns
- Validate migration safety

## Quick Reference

### PostgreSQL-Specific
- Use `clock_timestamp()` for real current time in transactions
- Avoid reserved keywords: `user`, `order`, `group`
- Create triggers for auto-updating timestamps
- Quote identifiers only when necessary

### MySQL-Specific
- Enable strict mode to prevent silent data truncation
- Don't rely on sequential AUTO_INCREMENT values
- Handle zero dates properly

### General
- Always use parameterized queries
- Release database connections properly
- Use appropriate data types
- Design indexes for your query patterns
- Keep transactions short and focused
