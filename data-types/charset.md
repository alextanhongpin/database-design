# Character Sets and Collations: Complete Guide

Proper character encoding and collation setup is essential for internationalization, data integrity, and application reliability. This guide covers character set configuration, collation choices, and best practices across database systems.

## Table of Contents
- [Character Set Fundamentals](#character-set-fundamentals)
- [MySQL Character Sets](#mysql-character-sets)
- [PostgreSQL Character Encoding](#postgresql-character-encoding)
- [Collation Strategies](#collation-strategies)
- [Migration and Conversion](#migration-and-conversion)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)

## Character Set Fundamentals

### Understanding Character Encoding

```sql
-- Character set vs Collation
-- Character Set: How characters are stored (encoding)
-- Collation: How characters are compared and sorted

-- Example: UTF-8 encoding with different collations
'café' = 'cafe'  -- Depends on collation
'A' = 'a'        -- Depends on collation (case sensitivity)
'ä' = 'a'        -- Depends on collation (accent sensitivity)
```

### Unicode Support Requirements

```sql
-- UTF-8 support requirements:
-- ✅ Support all Unicode characters (emojis, international scripts)
-- ✅ Variable-length encoding (1-4 bytes per character)
-- ✅ Backward compatible with ASCII
-- ✅ Web standard and widely supported

-- Common character sets:
-- utf8mb4: Full UTF-8 support (recommended)
-- utf8mb3/utf8: Limited UTF-8 (3 bytes max, legacy)
-- latin1: Western European languages only
-- ascii: ASCII characters only (7-bit)
```

## MySQL Character Sets

### MySQL 8.0+ (Current Best Practices)

```sql
-- MySQL 8.0 defaults (recommended)
-- Default character set: utf8mb4
-- Default collation: utf8mb4_0900_ai_ci

-- Create database with optimal settings
CREATE DATABASE app_production
CHARACTER SET utf8mb4
COLLATE utf8mb4_0900_ai_ci;

-- Verify database settings
SHOW CREATE DATABASE app_production;

-- Create table with explicit character set
CREATE TABLE users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(320) NOT NULL,
    display_name VARCHAR(100),
    bio TEXT,
    
    -- Indexes work properly with utf8mb4
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email),
    FULLTEXT KEY ft_bio (bio)
) ENGINE=InnoDB 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_0900_ai_ci;
```

### MySQL Legacy Versions (Pre-8.0)

```sql
-- For MySQL 5.7 and earlier, explicitly set utf8mb4
CREATE DATABASE app_legacy
CHARACTER SET utf8mb4
COLLATE utf8mb4_unicode_ci;  -- Best collation for older versions

-- Table creation for legacy MySQL
CREATE TABLE legacy_users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    email VARCHAR(320) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    content TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
) ENGINE=InnoDB 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;
```

### Collation Comparison

```sql
-- Different collation behaviors
CREATE TABLE collation_examples (
    id INT AUTO_INCREMENT PRIMARY KEY,
    
    -- Case-insensitive, accent-insensitive (AI = Accent Insensitive)
    text_ai VARCHAR(100) COLLATE utf8mb4_0900_ai_ci,
    
    -- Case-insensitive, accent-sensitive (AS = Accent Sensitive)  
    text_as VARCHAR(100) COLLATE utf8mb4_0900_as_ci,
    
    -- Case-sensitive, accent-sensitive
    text_cs VARCHAR(100) COLLATE utf8mb4_0900_as_cs,
    
    -- Binary collation (exact byte comparison)
    text_bin VARCHAR(100) COLLATE utf8mb4_bin
);

INSERT INTO collation_examples (text_ai, text_as, text_cs, text_bin) VALUES
('Café', 'Café', 'Café', 'Café'),
('cafe', 'cafe', 'cafe', 'cafe'),
('CAFÉ', 'CAFÉ', 'CAFÉ', 'CAFÉ');

-- Compare collation behaviors
SELECT 'Comparison Results' as test;

-- Case insensitive, accent insensitive
SELECT COUNT(*) as ai_matches FROM collation_examples 
WHERE text_ai = 'cafe';  -- Matches all variants

-- Case insensitive, accent sensitive
SELECT COUNT(*) as as_matches FROM collation_examples 
WHERE text_as = 'cafe';  -- Matches 'cafe' and 'CAFE' but not 'Café'

-- Case sensitive, accent sensitive
SELECT COUNT(*) as cs_matches FROM collation_examples 
WHERE text_cs = 'cafe';  -- Matches only 'cafe'

-- Binary comparison
SELECT COUNT(*) as bin_matches FROM collation_examples 
WHERE text_bin = 'cafe';  -- Matches only exact 'cafe'
```

### Server Configuration

```sql
-- Check current MySQL character set configuration
SHOW VARIABLES WHERE Variable_name LIKE 'character\_set\_%' 
                 OR Variable_name LIKE 'collation%';

-- Optimal MySQL 8.0 configuration should show:
/*
| character_set_client     | utf8mb4            |
| character_set_connection | utf8mb4            |  
| character_set_database   | utf8mb4            |
| character_set_results    | utf8mb4            |
| character_set_server     | utf8mb4            |
| character_set_system     | utf8mb3            |
| collation_connection     | utf8mb4_0900_ai_ci |
| collation_database       | utf8mb4_0900_ai_ci |
| collation_server         | utf8mb4_0900_ai_ci |
*/

-- Set session character set if needed
SET NAMES utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

### Connection String Configuration

```go
// Go MySQL connection with proper charset
import (
    "database/sql"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
)

func connectMySQL() (*sql.DB, error) {
    // Modern MySQL 8.0+ connection
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_0900_ai_ci&loc=UTC",
        user, password, host, port, database)
    
    // Legacy MySQL 5.7 connection
    // dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=UTC",
    //     user, password, host, port, database)
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    // Verify connection and character set
    var charset, collation string
    err = db.QueryRow("SELECT @@character_set_connection, @@collation_connection").Scan(&charset, &collation)
    if err != nil {
        return nil, err
    }
    
    fmt.Printf("Connected with charset: %s, collation: %s\n", charset, collation)
    return db, nil
}
```

```python
# Python MySQL connection with proper charset
import mysql.connector
from mysql.connector import Error

def connect_mysql():
    try:
        # Modern configuration
        connection = mysql.connector.connect(
            host='localhost',
            database='app_production',
            user='username',
            password='password',
            charset='utf8mb4',
            collation='utf8mb4_0900_ai_ci',
            use_unicode=True
        )
        
        if connection.is_connected():
            cursor = connection.cursor()
            cursor.execute("SELECT @@character_set_connection, @@collation_connection")
            charset, collation = cursor.fetchone()
            print(f"Connected with charset: {charset}, collation: {collation}")
            
        return connection
        
    except Error as e:
        print(f"Error connecting to MySQL: {e}")
        return None
```

```javascript
// Node.js MySQL connection with proper charset
const mysql = require('mysql2/promise');

async function connectMySQL() {
    const connection = await mysql.createConnection({
        host: 'localhost',
        user: 'username',
        password: 'password',
        database: 'app_production',
        charset: 'utf8mb4',
        // Modern MySQL 8.0+
        collation: 'utf8mb4_0900_ai_ci',
        // Legacy MySQL 5.7
        // collation: 'utf8mb4_unicode_ci',
        timezone: 'Z'  // UTC timezone
    });
    
    // Verify connection
    const [rows] = await connection.execute(
        'SELECT @@character_set_connection, @@collation_connection'
    );
    console.log(`Connected with charset: ${rows[0]['@@character_set_connection']}, collation: ${rows[0]['@@collation_connection']}`);
    
    return connection;
}
```

## PostgreSQL Character Encoding

### PostgreSQL Encoding Setup

```sql
-- PostgreSQL uses UTF-8 by default (recommended)
-- Check database encoding
SELECT datname, encoding, datcollate, datctype 
FROM pg_database 
WHERE datname = current_database();

-- Create database with explicit encoding
CREATE DATABASE app_production
WITH ENCODING 'UTF8'
LC_COLLATE = 'en_US.UTF-8'
LC_CTYPE = 'en_US.UTF-8';

-- PostgreSQL text types are always Unicode-capable
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    display_name TEXT,
    bio TEXT,
    
    -- Constraints work naturally with UTF-8
    CONSTRAINT uk_username UNIQUE (username),
    CONSTRAINT uk_email UNIQUE (email)
);
```

### PostgreSQL Collation Support

```sql
-- List available collations
SELECT collname, collcollate, collctype 
FROM pg_collation 
WHERE collname LIKE '%utf8%' OR collname LIKE '%UTF%'
ORDER BY collname;

-- Use specific collations for sorting
CREATE TABLE multilingual_content (
    id SERIAL PRIMARY KEY,
    content_en TEXT COLLATE "en_US.UTF-8",
    content_fr TEXT COLLATE "fr_FR.UTF-8", 
    content_de TEXT COLLATE "de_DE.UTF-8",
    content_ja TEXT COLLATE "ja_JP.UTF-8"
);

-- Query with specific collation
SELECT content_en 
FROM multilingual_content 
ORDER BY content_en COLLATE "C";  -- Binary sort

SELECT content_fr 
FROM multilingual_content 
ORDER BY content_fr COLLATE "fr_FR.UTF-8";  -- French-aware sort
```

## Collation Strategies

### Choosing the Right Collation

```sql
-- Collation decision matrix:

-- 1. Case-insensitive, accent-insensitive (most common)
-- Use: utf8mb4_0900_ai_ci (MySQL 8.0+) or utf8mb4_unicode_ci (legacy)
-- Good for: usernames, emails, search functionality
CREATE TABLE case_insensitive (
    username VARCHAR(50) COLLATE utf8mb4_0900_ai_ci
);

-- 2. Case-sensitive, accent-sensitive  
-- Use: utf8mb4_0900_as_cs or utf8mb4_bin
-- Good for: passwords, tokens, exact matching
CREATE TABLE case_sensitive (
    token VARCHAR(255) COLLATE utf8mb4_bin
);

-- 3. Natural language sorting
-- Use: language-specific collations
-- Good for: human-readable content
CREATE TABLE natural_sorting (
    title VARCHAR(255) COLLATE utf8mb4_0900_ai_ci,
    title_german VARCHAR(255) COLLATE utf8mb4_de_pb_0900_ai_ci
);
```

### Collation in Queries

```sql
-- Override table collation in queries
SELECT * FROM users 
WHERE username = 'John' COLLATE utf8mb4_bin;  -- Case-sensitive search

SELECT * FROM users 
ORDER BY username COLLATE utf8mb4_0900_ai_ci;  -- Case-insensitive sort

-- Compare different collations
SELECT 
    'Müller' = 'Mueller' COLLATE utf8mb4_0900_ai_ci as accent_insensitive,
    'Müller' = 'Mueller' COLLATE utf8mb4_0900_as_ci as accent_sensitive,
    'Hello' = 'hello' COLLATE utf8mb4_0900_ai_ci as case_insensitive,
    'Hello' = 'hello' COLLATE utf8mb4_bin as case_sensitive;
```

## Migration and Conversion

### Converting Existing Data

```sql
-- Check current character sets in existing database
SELECT 
    TABLE_SCHEMA,
    TABLE_NAME,
    COLUMN_NAME,
    CHARACTER_SET_NAME,
    COLLATION_NAME
FROM information_schema.COLUMNS 
WHERE TABLE_SCHEMA = 'your_database'
  AND CHARACTER_SET_NAME IS NOT NULL
  AND CHARACTER_SET_NAME != 'utf8mb4';

-- Convert table character set (MySQL)
ALTER TABLE users 
CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Convert specific columns
ALTER TABLE users 
MODIFY COLUMN username VARCHAR(50) 
CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Convert database default
ALTER DATABASE your_database 
CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

### Safe Migration Strategy

```sql
-- 1. Create migration script to backup and convert
-- Step 1: Backup existing data
CREATE TABLE users_backup AS SELECT * FROM users;

-- Step 2: Create new table with correct charset
CREATE TABLE users_new (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
    email VARCHAR(320) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
    -- ... other columns
    
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Step 3: Migrate data with proper conversion
INSERT INTO users_new (id, username, email)
SELECT id, 
       CONVERT(username USING utf8mb4) as username,
       CONVERT(email USING utf8mb4) as email
FROM users;

-- Step 4: Verify data integrity
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM users_new;

-- Step 5: Rename tables (in transaction)
START TRANSACTION;
RENAME TABLE users TO users_old, users_new TO users;
COMMIT;
```

## Performance Considerations

### Index Behavior with Character Sets

```sql
-- Index prefix lengths with utf8mb4
-- utf8mb4 uses up to 4 bytes per character
-- MySQL index key limit: 3072 bytes (InnoDB)

-- Safe index on utf8mb4 VARCHAR
CREATE TABLE indexed_text (
    id INT AUTO_INCREMENT PRIMARY KEY,
    
    -- Full index on short columns (safe)
    short_text VARCHAR(191) CHARACTER SET utf8mb4,  -- 191 * 4 = 764 bytes
    
    -- Prefix index on longer columns
    long_text VARCHAR(1000) CHARACTER SET utf8mb4,
    
    INDEX idx_short (short_text),
    INDEX idx_long_prefix (long_text(191))  -- Prefix index
);

-- Check index usage
SHOW INDEX FROM indexed_text;
```

### Memory and Storage Impact

```sql
-- Storage requirements comparison
CREATE TABLE storage_comparison (
    id INT AUTO_INCREMENT PRIMARY KEY,
    
    -- ASCII text (1 byte per character)
    ascii_text VARCHAR(100) CHARACTER SET ascii,
    
    -- Latin1 text (1 byte per character)  
    latin1_text VARCHAR(100) CHARACTER SET latin1,
    
    -- UTF-8 text (1-4 bytes per character)
    utf8mb4_text VARCHAR(100) CHARACTER SET utf8mb4,
    
    -- Binary storage (exact bytes)
    binary_data VARBINARY(400)
);

-- Actual storage used depends on content
INSERT INTO storage_comparison VALUES
(1, 'Hello World', 'Hello World', 'Hello World', 'Hello World'),
(2, NULL, 'Café français', 'Café français', 'Café français'),  
(3, NULL, NULL, '🚀 Emoji support! 🎉', '🚀 Emoji support! 🎉');

-- Check actual storage usage
SELECT 
    CHAR_LENGTH(utf8mb4_text) as char_length,
    LENGTH(utf8mb4_text) as byte_length,
    utf8mb4_text
FROM storage_comparison 
WHERE utf8mb4_text IS NOT NULL;
```

## Best Practices

### 1. Default Configuration

```sql
-- ✅ Always use utf8mb4 for new projects
CREATE DATABASE new_project
CHARACTER SET utf8mb4
COLLATE utf8mb4_0900_ai_ci;  -- MySQL 8.0+
-- COLLATE utf8mb4_unicode_ci;  -- MySQL 5.7 and earlier

-- ✅ Set at table level for consistency
CREATE TABLE best_practices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT
) ENGINE=InnoDB 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_0900_ai_ci;
```

### 2. Application Configuration

```yaml
# Docker Compose MySQL configuration
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: app_production
      MYSQL_CHARSET: utf8mb4
      MYSQL_COLLATION: utf8mb4_0900_ai_ci
    command: --default-authentication-plugin=mysql_native_password
             --character-set-server=utf8mb4
             --collation-server=utf8mb4_0900_ai_ci
    ports:
      - "3306:3306"
```

### 3. Validation and Testing

```sql
-- Test character set handling
CREATE TABLE charset_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    test_text VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci
);

-- Test with various Unicode characters
INSERT INTO charset_test (test_text) VALUES
('ASCII text'),
('Accented: café, naïve, résumé'),
('Symbols: ©, ®, ™, €, £, ¥'),
('Emoji: 😀, 🚀, 🎉, 👍, ❤️'),
('CJK: 你好, こんにちは, 안녕하세요'),
('Math: ∑, ∞, √, π, α, β, γ'),
('Special: \n\r\t\0');

-- Verify all data is stored correctly
SELECT id, test_text, 
       CHAR_LENGTH(test_text) as char_count,
       LENGTH(test_text) as byte_count,
       HEX(test_text) as hex_representation
FROM charset_test;
```

### 4. Monitoring and Maintenance

```sql
-- Create function to audit character set usage
DELIMITER //
CREATE FUNCTION audit_charset_usage(db_name VARCHAR(64))
RETURNS TEXT
READS SQL DATA
DETERMINISTIC
BEGIN
    DECLARE result TEXT DEFAULT '';
    DECLARE done INT DEFAULT FALSE;
    DECLARE table_name VARCHAR(64);
    DECLARE column_info TEXT;
    
    DECLARE cur CURSOR FOR
        SELECT DISTINCT TABLE_NAME
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = db_name
          AND CHARACTER_SET_NAME IS NOT NULL;
    
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    
    OPEN cur;
    
    read_loop: LOOP
        FETCH cur INTO table_name;
        IF done THEN
            LEAVE read_loop;
        END IF;
        
        SELECT GROUP_CONCAT(
            CONCAT(COLUMN_NAME, ':', CHARACTER_SET_NAME, ':', COLLATION_NAME)
            SEPARATOR '; '
        ) INTO column_info
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = db_name 
          AND TABLE_NAME = table_name
          AND CHARACTER_SET_NAME IS NOT NULL;
        
        SET result = CONCAT(result, table_name, ': ', column_info, '\n');
    END LOOP;
    
    CLOSE cur;
    RETURN result;
END//
DELIMITER ;

-- Use the audit function
SELECT audit_charset_usage('your_database_name');
```

## Conclusion

Proper character set and collation configuration is critical for:

1. **Unicode Support**: Use utf8mb4 for full Unicode compatibility including emojis
2. **Application Reliability**: Prevent encoding-related bugs and data corruption  
3. **International Support**: Handle multiple languages and scripts correctly
4. **Performance**: Choose appropriate collations for your query patterns
5. **Future-Proofing**: utf8mb4 with modern collations supports evolving Unicode standards

### Quick Reference:

**MySQL 8.0+**: `utf8mb4` + `utf8mb4_0900_ai_ci`
**MySQL 5.7**: `utf8mb4` + `utf8mb4_unicode_ci`  
**PostgreSQL**: `UTF8` encoding (default)
**Connection Strings**: Always specify charset and collation
**Indexes**: Be aware of byte limits with multibyte characters

The investment in proper character set configuration pays dividends in application reliability and international compatibility.
