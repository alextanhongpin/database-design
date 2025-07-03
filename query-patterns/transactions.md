# Database Transactions

Complete guide to database transactions, isolation levels, concurrency control, and best practices for maintaining data consistency.

## Table of Contents

1. [Transaction Fundamentals](#transaction-fundamentals)
2. [Isolation Levels](#isolation-levels)
3. [Transaction Patterns](#transaction-patterns)
4. [Concurrency Control](#concurrency-control)
5. [Performance Considerations](#performance-considerations)
6. [Error Handling](#error-handling)
7. [Best Practices](#best-practices)
8. [Anti-Patterns](#anti-patterns)

## Transaction Fundamentals

### ACID Properties

**Atomicity**: All operations in a transaction succeed or all fail
**Consistency**: Database remains in a valid state
**Isolation**: Concurrent transactions don't interfere with each other
**Durability**: Committed transactions persist even after system failure

### Basic Transaction Syntax

```sql
-- Standard SQL transaction
BEGIN TRANSACTION;
    UPDATE accounts SET balance = balance - 100 WHERE id = 1;
    UPDATE accounts SET balance = balance + 100 WHERE id = 2;
COMMIT;

-- PostgreSQL syntax
BEGIN;
    -- operations
COMMIT;

-- MySQL syntax
START TRANSACTION;
    -- operations
COMMIT;

-- Rollback on error
BEGIN;
    UPDATE accounts SET balance = balance - 100 WHERE id = 1;
    -- If something goes wrong
ROLLBACK;
```

## Isolation Levels

### Default Isolation Levels

- **MySQL**: `REPEATABLE READ`
- **PostgreSQL**: `READ COMMITTED`
- **SQL Server**: `READ COMMITTED`
- **Oracle**: `READ COMMITTED`

### Isolation Level Details

#### READ UNCOMMITTED
```sql
-- Can read uncommitted changes from other transactions
SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
BEGIN;
    SELECT balance FROM accounts WHERE id = 1;  -- May see dirty reads
COMMIT;
```

#### READ COMMITTED
```sql
-- Can only read committed data (most common)
SET TRANSACTION ISOLATION LEVEL READ COMMITTED;
BEGIN;
    SELECT balance FROM accounts WHERE id = 1;  -- Always sees committed data
    -- May see different values if another transaction commits
    SELECT balance FROM accounts WHERE id = 1;
COMMIT;
```

#### REPEATABLE READ
```sql
-- Same reads return same results within transaction
SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;
BEGIN;
    SELECT balance FROM accounts WHERE id = 1;  -- Returns 1000
    -- Another transaction updates this row and commits
    SELECT balance FROM accounts WHERE id = 1;  -- Still returns 1000
COMMIT;
```

#### SERIALIZABLE
```sql
-- Highest isolation, prevents all phenomena
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
BEGIN;
    SELECT COUNT(*) FROM orders WHERE status = 'pending';
    -- No new pending orders can be created by other transactions
    INSERT INTO orders (status) VALUES ('pending');
COMMIT;
```

### PostgreSQL Isolation Examples

```sql
-- Transaction-specific isolation level
BEGIN;
    SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;
    SELECT * FROM products WHERE category = 'electronics';
    -- Operations within this transaction see consistent snapshot
COMMIT;

-- Session-level isolation level
SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL SERIALIZABLE;
```

## Transaction Patterns

### Financial Transfers

```sql
-- Safe money transfer with proper error handling
CREATE OR REPLACE FUNCTION transfer_money(
    from_account_id INTEGER,
    to_account_id INTEGER,
    transfer_amount DECIMAL(10,2)
) RETURNS BOOLEAN AS $$
DECLARE
    from_balance DECIMAL(10,2);
BEGIN
    -- Start transaction is implicit in function
    
    -- Lock and verify source account
    SELECT balance INTO from_balance
    FROM accounts 
    WHERE id = from_account_id 
    FOR UPDATE;
    
    -- Check sufficient funds
    IF from_balance < transfer_amount THEN
        RAISE EXCEPTION 'Insufficient funds: % available, % requested', 
            from_balance, transfer_amount;
    END IF;
    
    -- Debit source account
    UPDATE accounts 
    SET balance = balance - transfer_amount,
        updated_at = NOW()
    WHERE id = from_account_id;
    
    -- Credit destination account
    UPDATE accounts 
    SET balance = balance + transfer_amount,
        updated_at = NOW()
    WHERE id = to_account_id;
    
    -- Log transaction
    INSERT INTO transaction_log (from_account, to_account, amount, created_at)
    VALUES (from_account_id, to_account_id, transfer_amount, NOW());
    
    RETURN TRUE;
    
EXCEPTION WHEN OTHERS THEN
    -- Error is automatically rolled back
    RAISE;
END;
$$ LANGUAGE plpgsql;
```

### Inventory Management

```sql
-- Safe inventory update with reservation
CREATE OR REPLACE FUNCTION reserve_inventory(
    product_id INTEGER,
    quantity_needed INTEGER,
    customer_id INTEGER
) RETURNS INTEGER AS $$  -- Returns reservation_id
DECLARE
    available_quantity INTEGER;
    reservation_id INTEGER;
BEGIN
    -- Lock inventory record
    SELECT available_stock INTO available_quantity
    FROM inventory 
    WHERE product_id = product_id 
    FOR UPDATE;
    
    -- Check availability
    IF available_quantity < quantity_needed THEN
        RAISE EXCEPTION 'Insufficient inventory: % available, % requested', 
            available_quantity, quantity_needed;
    END IF;
    
    -- Create reservation
    INSERT INTO inventory_reservations (product_id, customer_id, quantity, expires_at)
    VALUES (product_id, customer_id, quantity_needed, NOW() + INTERVAL '30 minutes')
    RETURNING id INTO reservation_id;
    
    -- Update available stock
    UPDATE inventory 
    SET available_stock = available_stock - quantity_needed,
        reserved_stock = reserved_stock + quantity_needed,
        updated_at = NOW()
    WHERE product_id = product_id;
    
    RETURN reservation_id;
END;
$$ LANGUAGE plpgsql;
```

### Batch Operations

```sql
-- Process batch of orders atomically
CREATE OR REPLACE FUNCTION process_order_batch(order_ids INTEGER[])
RETURNS TABLE(order_id INTEGER, success BOOLEAN, error_message TEXT) AS $$
DECLARE
    current_order_id INTEGER;
    order_total DECIMAL(10,2);
    customer_balance DECIMAL(10,2);
BEGIN
    FOREACH current_order_id IN ARRAY order_ids LOOP
        BEGIN
            -- Get order details
            SELECT total_amount, customer_id INTO order_total, customer_id
            FROM orders WHERE id = current_order_id FOR UPDATE;
            
            -- Check customer balance
            SELECT balance INTO customer_balance
            FROM customers WHERE id = customer_id FOR UPDATE;
            
            IF customer_balance >= order_total THEN
                -- Process order
                UPDATE orders SET status = 'paid' WHERE id = current_order_id;
                UPDATE customers SET balance = balance - order_total WHERE id = customer_id;
                
                -- Return success
                order_id := current_order_id;
                success := TRUE;
                error_message := NULL;
                RETURN NEXT;
            ELSE
                -- Insufficient funds
                order_id := current_order_id;
                success := FALSE;
                error_message := 'Insufficient funds';
                RETURN NEXT;
            END IF;
            
        EXCEPTION WHEN OTHERS THEN
            -- Handle individual order error
            order_id := current_order_id;
            success := FALSE;
            error_message := SQLERRM;
            RETURN NEXT;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

## Concurrency Control

### Optimistic Locking

```sql
-- Version-based optimistic locking
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200),
    content TEXT,
    version INTEGER DEFAULT 1,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Update with version check
UPDATE documents 
SET title = @new_title,
    content = @new_content,
    version = version + 1,
    updated_at = NOW()
WHERE id = @document_id 
    AND version = @expected_version;

-- Check if update succeeded (no concurrent modification)
GET DIAGNOSTICS updated_rows = ROW_COUNT;
IF updated_rows = 0 THEN
    RAISE EXCEPTION 'Document was modified by another user';
END IF;
```

### Pessimistic Locking

```sql
-- Explicit row locking
BEGIN;
    -- Lock specific rows for update
    SELECT * FROM accounts 
    WHERE id IN (1, 2) 
    FOR UPDATE;
    
    -- Perform updates knowing rows are locked
    UPDATE accounts SET balance = balance - 100 WHERE id = 1;
    UPDATE accounts SET balance = balance + 100 WHERE id = 2;
COMMIT;

-- Different lock types
SELECT * FROM products WHERE category = 'electronics' FOR SHARE;     -- Shared lock
SELECT * FROM products WHERE id = 1 FOR UPDATE NOWAIT;               -- Fail if can't lock
SELECT * FROM products WHERE id = 1 FOR UPDATE SKIP LOCKED;          -- Skip locked rows
```

### Deadlock Prevention

```sql
-- Always lock resources in consistent order to prevent deadlocks
CREATE OR REPLACE FUNCTION safe_transfer_ordered_locks(
    account_id_1 INTEGER,
    account_id_2 INTEGER,
    amount DECIMAL(10,2)
) RETURNS BOOLEAN AS $$
DECLARE
    from_id INTEGER;
    to_id INTEGER;
BEGIN
    -- Always lock accounts in ascending ID order
    IF account_id_1 < account_id_2 THEN
        from_id := account_id_1;
        to_id := account_id_2;
    ELSE
        from_id := account_id_2;
        to_id := account_id_1;
    END IF;
    
    -- Lock in consistent order
    PERFORM * FROM accounts WHERE id = from_id FOR UPDATE;
    PERFORM * FROM accounts WHERE id = to_id FOR UPDATE;
    
    -- Perform transfer logic
    UPDATE accounts SET balance = balance - amount WHERE id = account_id_1;
    UPDATE accounts SET balance = balance + amount WHERE id = account_id_2;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
```

## Performance Considerations

### Transaction Size and Duration

```sql
-- Bad: Long-running transaction
BEGIN;
    -- This locks too many rows for too long
    UPDATE large_table SET processed = TRUE WHERE created_at < '2024-01-01';
    -- Other operations...
COMMIT;

-- Good: Batch processing
DO $$
DECLARE
    batch_size INTEGER := 1000;
    affected_rows INTEGER;
BEGIN
    LOOP
        -- Process in small batches
        UPDATE large_table 
        SET processed = TRUE 
        WHERE id IN (
            SELECT id FROM large_table 
            WHERE processed = FALSE 
            ORDER BY id 
            LIMIT batch_size
        );
        
        GET DIAGNOSTICS affected_rows = ROW_COUNT;
        EXIT WHEN affected_rows = 0;
        
        -- Commit each batch
        COMMIT;
    END LOOP;
END $$;
```

### Read-Only Transactions

```sql
-- Mark read-only transactions for better performance
BEGIN READ ONLY;
    SELECT COUNT(*) FROM orders WHERE created_at >= '2024-01-01';
    SELECT AVG(amount) FROM orders WHERE status = 'completed';
COMMIT;

-- PostgreSQL: Set transaction characteristics
BEGIN;
    SET TRANSACTION READ ONLY;
    -- Read operations only
COMMIT;
```

## Error Handling

### Application-Level Transaction Handling

#### Node.js Example

```javascript
async function transferMoney(fromAccountId, toAccountId, amount) {
    const client = await pool.connect();
    
    try {
        await client.query('BEGIN');
        
        // Check source account balance
        const sourceResult = await client.query(
            'SELECT balance FROM accounts WHERE id = $1 FOR UPDATE',
            [fromAccountId]
        );
        
        if (sourceResult.rows[0].balance < amount) {
            throw new Error('Insufficient funds');
        }
        
        // Perform transfer
        await client.query(
            'UPDATE accounts SET balance = balance - $1 WHERE id = $2',
            [amount, fromAccountId]
        );
        
        await client.query(
            'UPDATE accounts SET balance = balance + $1 WHERE id = $2',
            [amount, toAccountId]
        );
        
        // Log transaction
        await client.query(
            'INSERT INTO transaction_log (from_account, to_account, amount) VALUES ($1, $2, $3)',
            [fromAccountId, toAccountId, amount]
        );
        
        await client.query('COMMIT');
        return { success: true };
        
    } catch (error) {
        await client.query('ROLLBACK');
        return { success: false, error: error.message };
    } finally {
        client.release();
    }
}
```

#### Go Example

```go
func transferMoney(db *sql.DB, fromID, toID int, amount decimal.Decimal) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            tx.Rollback()
        }
    }()
    
    // Lock and check source account
    var balance decimal.Decimal
    err = tx.QueryRow(
        "SELECT balance FROM accounts WHERE id = ? FOR UPDATE", 
        fromID,
    ).Scan(&balance)
    if err != nil {
        return err
    }
    
    if balance.LessThan(amount) {
        return errors.New("insufficient funds")
    }
    
    // Perform transfer
    _, err = tx.Exec(
        "UPDATE accounts SET balance = balance - ? WHERE id = ?", 
        amount, fromID,
    )
    if err != nil {
        return err
    }
    
    _, err = tx.Exec(
        "UPDATE accounts SET balance = balance + ? WHERE id = ?", 
        amount, toID,
    )
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

### Retry Logic for Serialization Failures

```sql
-- Function with retry logic for serialization failures
CREATE OR REPLACE FUNCTION retry_transfer(
    from_id INTEGER,
    to_id INTEGER,
    amount DECIMAL(10,2),
    max_retries INTEGER DEFAULT 3
) RETURNS BOOLEAN AS $$
DECLARE
    retry_count INTEGER := 0;
    success BOOLEAN := FALSE;
BEGIN
    WHILE retry_count < max_retries AND NOT success LOOP
        BEGIN
            -- Attempt transfer
            PERFORM transfer_money(from_id, to_id, amount);
            success := TRUE;
            
        EXCEPTION 
            WHEN serialization_failure OR deadlock_detected THEN
                retry_count := retry_count + 1;
                IF retry_count >= max_retries THEN
                    RAISE;
                END IF;
                
                -- Wait before retry (exponential backoff)
                PERFORM pg_sleep(0.1 * (2 ^ retry_count));
        END;
    END LOOP;
    
    RETURN success;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Keep Transactions Short

```sql
-- ✅ Good: Short, focused transaction
BEGIN;
    UPDATE inventory SET quantity = quantity - 1 WHERE product_id = 123;
    INSERT INTO order_items (order_id, product_id, quantity) VALUES (456, 123, 1);
COMMIT;

-- ❌ Bad: Long-running transaction
BEGIN;
    -- Complex calculations
    -- File I/O operations
    -- External API calls
    -- Multiple unrelated updates
COMMIT;
```

### 2. Handle Exceptions Properly

```sql
-- PostgreSQL function with proper exception handling
CREATE OR REPLACE FUNCTION safe_order_process(order_id INTEGER)
RETURNS BOOLEAN AS $$
BEGIN
    -- Process order logic
    UPDATE orders SET status = 'processing' WHERE id = order_id;
    UPDATE inventory SET quantity = quantity - 1 WHERE product_id = (
        SELECT product_id FROM order_items WHERE order_id = order_id
    );
    
    RETURN TRUE;
    
EXCEPTION 
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'Access denied for order %', order_id;
        RETURN FALSE;
    WHEN check_violation THEN
        RAISE NOTICE 'Constraint violation for order %', order_id;
        RETURN FALSE;
    WHEN OTHERS THEN
        RAISE NOTICE 'Unexpected error for order %: %', order_id, SQLERRM;
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql;
```

### 3. Use Savepoints for Partial Rollbacks

```sql
BEGIN;
    INSERT INTO orders (customer_id, total) VALUES (123, 100.00);
    
    SAVEPOINT before_items;
    
    BEGIN
        INSERT INTO order_items (order_id, product_id, quantity) 
        VALUES (LAST_INSERT_ID(), 456, 1);
    EXCEPTION WHEN OTHERS THEN
        ROLLBACK TO before_items;
        -- Continue with order creation even if items fail
    END;
    
    UPDATE customers SET last_order_date = NOW() WHERE id = 123;
COMMIT;
```

## Anti-Patterns

### Common Transaction Mistakes

```sql
-- ❌ Bad: Autocommit disabled without proper transaction management
SET autocommit = 0;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
-- Forgot to COMMIT - change is lost!

-- ❌ Bad: Transaction for independent operations
BEGIN;
    UPDATE user_profile SET last_login = NOW() WHERE user_id = 1;
    UPDATE user_profile SET last_login = NOW() WHERE user_id = 2;
    UPDATE user_profile SET last_login = NOW() WHERE user_id = 3;
COMMIT;
-- These updates are independent and don't need to be atomic

-- ❌ Bad: Long transaction with external calls
BEGIN;
    UPDATE orders SET status = 'processing';
    -- Call external payment API (takes 30 seconds)
    -- Send email notification (takes 5 seconds)
    UPDATE orders SET status = 'completed';
COMMIT;

-- ❌ Bad: Reading uncommitted data without understanding implications
SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
SELECT balance FROM accounts WHERE id = 1;  -- May see dirty data

-- ❌ Bad: Not handling deadlocks
-- Two transactions updating same tables in different order
-- Transaction 1: UPDATE table_a, then table_b
-- Transaction 2: UPDATE table_b, then table_a
-- Results in deadlock
```

### Performance Anti-Patterns

```sql
-- ❌ Bad: Unnecessary locking
SELECT * FROM products FOR UPDATE;  -- Locks all products unnecessarily

-- ✅ Good: Specific locking
SELECT * FROM products WHERE id = 123 FOR UPDATE;

-- ❌ Bad: Wrong isolation level
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
-- Using highest isolation when READ COMMITTED would suffice

-- ❌ Bad: Hot spot contention
-- Multiple transactions updating same counter
UPDATE global_stats SET page_views = page_views + 1;

-- ✅ Good: Batch counter updates or use separate counting system
```

## Related Patterns

- [Locking Patterns](locks.md)
- [Concurrency Patterns](README.md)
- [Error Handling](error-handling.md)
- [Performance Optimization](../performance/README.md)
