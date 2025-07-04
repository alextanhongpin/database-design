# Data Sanitization

Comprehensive guide to sanitizing and validating data before database storage to prevent security vulnerabilities and ensure data integrity.

## 🎯 Overview

Data sanitization is crucial for:
- **Security** - Preventing SQL injection, XSS, and other attacks
- **Data Integrity** - Ensuring consistent, valid data formats
- **Performance** - Optimizing storage and queries
- **Compliance** - Meeting regulatory requirements

## 🛡️ SQL Injection Prevention

### Input Validation

```sql
-- PostgreSQL function for safe column name validation
CREATE OR REPLACE FUNCTION validate_column_name(column_name TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- Only allow alphanumeric characters and underscores
    RETURN column_name ~ '^[a-zA-Z_][a-zA-Z0-9_]*$' 
           AND length(column_name) <= 63; -- PostgreSQL identifier limit
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT validate_column_name('user_id'); -- Returns TRUE
SELECT validate_column_name('user-id'); -- Returns FALSE
SELECT validate_column_name('1user');   -- Returns FALSE
```

### Safe Dynamic Query Construction

```sql
-- PostgreSQL function for safe ORDER BY clause
CREATE OR REPLACE FUNCTION safe_order_by(
    table_name TEXT,
    column_name TEXT,
    direction TEXT DEFAULT 'ASC'
) RETURNS TEXT AS $$
DECLARE
    safe_query TEXT;
BEGIN
    -- Validate table name
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = $1
    ) THEN
        RAISE EXCEPTION 'Invalid table name: %', $1;
    END IF;
    
    -- Validate column name
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = $1 AND column_name = $2
    ) THEN
        RAISE EXCEPTION 'Invalid column name: %', $2;
    END IF;
    
    -- Validate direction
    IF upper($3) NOT IN ('ASC', 'DESC') THEN
        RAISE EXCEPTION 'Invalid sort direction: %', $3;
    END IF;
    
    -- Construct safe query
    safe_query := format('SELECT * FROM %I ORDER BY %I %s', 
                        $1, $2, upper($3));
    
    RETURN safe_query;
END;
$$ LANGUAGE plpgsql;
```

## 🧹 Text Sanitization

### Email Sanitization

```sql
-- Email cleaning and validation
CREATE OR REPLACE FUNCTION sanitize_email(email TEXT)
RETURNS TEXT AS $$
DECLARE
    cleaned_email TEXT;
BEGIN
    -- Remove leading/trailing whitespace and convert to lowercase
    cleaned_email := trim(lower(email));
    
    -- Basic email format validation
    IF NOT (cleaned_email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$') THEN
        RAISE EXCEPTION 'Invalid email format: %', email;
    END IF;
    
    -- Remove consecutive dots
    cleaned_email := regexp_replace(cleaned_email, '\.{2,}', '.', 'g');
    
    -- Remove dots from username part (Gmail style)
    -- This is optional based on your requirements
    -- cleaned_email := regexp_replace(cleaned_email, '\.(?=.*@)', '', 'g');
    
    RETURN cleaned_email;
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT sanitize_email('  User.Name+tag@GMAIL.COM  ');
-- Returns: user.name+tag@gmail.com
```

### Phone Number Sanitization

```sql
-- Phone number cleaning
CREATE OR REPLACE FUNCTION sanitize_phone(phone TEXT)
RETURNS TEXT AS $$
DECLARE
    cleaned_phone TEXT;
BEGIN
    -- Remove all non-digit characters
    cleaned_phone := regexp_replace(phone, '[^0-9]', '', 'g');
    
    -- Handle international format
    IF length(cleaned_phone) > 10 AND substring(cleaned_phone, 1, 1) = '1' THEN
        cleaned_phone := substring(cleaned_phone, 2);
    END IF;
    
    -- Validate US phone number format
    IF length(cleaned_phone) != 10 THEN
        RAISE EXCEPTION 'Invalid phone number format: %', phone;
    END IF;
    
    -- Format as (XXX) XXX-XXXX
    RETURN format('(%s) %s-%s',
                  substring(cleaned_phone, 1, 3),
                  substring(cleaned_phone, 4, 3),
                  substring(cleaned_phone, 7, 4));
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT sanitize_phone('1-800-555-0123');
-- Returns: (800) 555-0123
```

### URL Sanitization

```sql
-- URL cleaning and validation
CREATE OR REPLACE FUNCTION sanitize_url(url TEXT)
RETURNS TEXT AS $$
DECLARE
    cleaned_url TEXT;
BEGIN
    -- Remove leading/trailing whitespace
    cleaned_url := trim(url);
    
    -- Add protocol if missing
    IF NOT (cleaned_url ~ '^https?://') THEN
        cleaned_url := 'https://' || cleaned_url;
    END IF;
    
    -- Basic URL validation
    IF NOT (cleaned_url ~ '^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}') THEN
        RAISE EXCEPTION 'Invalid URL format: %', url;
    END IF;
    
    -- Remove trailing slash
    cleaned_url := regexp_replace(cleaned_url, '/$', '');
    
    RETURN cleaned_url;
END;
$$ LANGUAGE plpgsql;
```

## 🔤 Character Encoding

### UTF-8 Validation

```sql
-- Ensure proper UTF-8 encoding
CREATE OR REPLACE FUNCTION sanitize_utf8(input_text TEXT)
RETURNS TEXT AS $$
DECLARE
    sanitized_text TEXT;
BEGIN
    -- Remove or replace invalid UTF-8 characters
    sanitized_text := convert_from(convert_to(input_text, 'UTF8'), 'UTF8');
    
    -- Remove control characters except tab, newline, carriage return
    sanitized_text := regexp_replace(sanitized_text, '[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]', '', 'g');
    
    -- Normalize whitespace
    sanitized_text := regexp_replace(sanitized_text, '\s+', ' ', 'g');
    sanitized_text := trim(sanitized_text);
    
    RETURN sanitized_text;
END;
$$ LANGUAGE plpgsql;
```

### HTML Entity Sanitization

```sql
-- HTML entity encoding for safe display
CREATE OR REPLACE FUNCTION sanitize_html(input_text TEXT)
RETURNS TEXT AS $$
DECLARE
    sanitized_text TEXT;
BEGIN
    sanitized_text := input_text;
    
    -- Replace HTML entities
    sanitized_text := replace(sanitized_text, '&', '&amp;');
    sanitized_text := replace(sanitized_text, '<', '&lt;');
    sanitized_text := replace(sanitized_text, '>', '&gt;');
    sanitized_text := replace(sanitized_text, '"', '&quot;');
    sanitized_text := replace(sanitized_text, '''', '&#39;');
    
    RETURN sanitized_text;
END;
$$ LANGUAGE plpgsql;
```

## 🔢 Numeric Data Sanitization

### Currency Sanitization

```sql
-- Currency value cleaning
CREATE OR REPLACE FUNCTION sanitize_currency(amount TEXT)
RETURNS DECIMAL(15,2) AS $$
DECLARE
    cleaned_amount TEXT;
    numeric_amount DECIMAL(15,2);
BEGIN
    -- Remove currency symbols and formatting
    cleaned_amount := regexp_replace(amount, '[$,\s]', '', 'g');
    
    -- Handle negative values in parentheses
    IF cleaned_amount ~ '^\(.*\)$' THEN
        cleaned_amount := '-' || substring(cleaned_amount, 2, length(cleaned_amount) - 2);
    END IF;
    
    -- Convert to numeric
    BEGIN
        numeric_amount := cleaned_amount::DECIMAL(15,2);
    EXCEPTION
        WHEN invalid_text_representation THEN
            RAISE EXCEPTION 'Invalid currency format: %', amount;
    END;
    
    RETURN numeric_amount;
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT sanitize_currency('$1,234.56');  -- Returns: 1234.56
SELECT sanitize_currency('($500.00)');  -- Returns: -500.00
```

### ID Sanitization

```sql
-- UUID validation and formatting
CREATE OR REPLACE FUNCTION sanitize_uuid(uuid_text TEXT)
RETURNS UUID AS $$
DECLARE
    cleaned_uuid TEXT;
BEGIN
    -- Remove hyphens and convert to lowercase
    cleaned_uuid := lower(regexp_replace(uuid_text, '-', '', 'g'));
    
    -- Validate length
    IF length(cleaned_uuid) != 32 THEN
        RAISE EXCEPTION 'Invalid UUID length: %', uuid_text;
    END IF;
    
    -- Validate hex characters
    IF NOT (cleaned_uuid ~ '^[0-9a-f]{32}$') THEN
        RAISE EXCEPTION 'Invalid UUID format: %', uuid_text;
    END IF;
    
    -- Format as proper UUID
    RETURN format('%s-%s-%s-%s-%s',
                  substring(cleaned_uuid, 1, 8),
                  substring(cleaned_uuid, 9, 4),
                  substring(cleaned_uuid, 13, 4),
                  substring(cleaned_uuid, 17, 4),
                  substring(cleaned_uuid, 21, 12))::UUID;
END;
$$ LANGUAGE plpgsql;
```

## 🧼 Data Cleaning Pipelines

### Batch Data Sanitization

```sql
-- Comprehensive data cleaning procedure
CREATE OR REPLACE FUNCTION sanitize_user_data()
RETURNS VOID AS $$
DECLARE
    rec RECORD;
    cleaned_email TEXT;
    cleaned_phone TEXT;
    errors_count INTEGER := 0;
BEGIN
    -- Create temporary table for error logging
    CREATE TEMP TABLE IF NOT EXISTS sanitization_errors (
        user_id UUID,
        field_name TEXT,
        error_message TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    -- Process each user record
    FOR rec IN SELECT id, email, phone, name FROM users WHERE sanitized_at IS NULL LOOP
        BEGIN
            -- Sanitize email
            BEGIN
                cleaned_email := sanitize_email(rec.email);
                UPDATE users SET email = cleaned_email WHERE id = rec.id;
            EXCEPTION
                WHEN OTHERS THEN
                    INSERT INTO sanitization_errors VALUES (rec.id, 'email', SQLERRM);
                    errors_count := errors_count + 1;
            END;
            
            -- Sanitize phone
            BEGIN
                cleaned_phone := sanitize_phone(rec.phone);
                UPDATE users SET phone = cleaned_phone WHERE id = rec.id;
            EXCEPTION
                WHEN OTHERS THEN
                    INSERT INTO sanitization_errors VALUES (rec.id, 'phone', SQLERRM);
                    errors_count := errors_count + 1;
            END;
            
            -- Mark as sanitized
            UPDATE users SET sanitized_at = CURRENT_TIMESTAMP WHERE id = rec.id;
            
        EXCEPTION
            WHEN OTHERS THEN
                INSERT INTO sanitization_errors VALUES (rec.id, 'general', SQLERRM);
                errors_count := errors_count + 1;
        END;
    END LOOP;
    
    RAISE NOTICE 'Sanitization complete. Errors: %', errors_count;
END;
$$ LANGUAGE plpgsql;
```

## 🔍 Validation Rules

### Application-Level Validation

```go
// Go example for input validation
package main

import (
    "fmt"
    "regexp"
    "strings"
    "unicode"
)

// ValidateColumnName ensures safe column names for dynamic queries
func ValidateColumnName(column string) bool {
    // Only allow alphanumeric characters and underscores
    matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, column)
    return matched && len(column) <= 63
}

// SanitizeInput removes dangerous characters from user input
func SanitizeInput(input string) string {
    // Remove control characters
    result := strings.Map(func(r rune) rune {
        if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
            return -1 // Remove character
        }
        return r
    }, input)
    
    // Normalize whitespace
    result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
    return strings.TrimSpace(result)
}

// ValidateEmail performs basic email validation
func ValidateEmail(email string) bool {
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(strings.ToLower(strings.TrimSpace(email)))
}

func main() {
    // Example usage
    fmt.Println(ValidateColumnName("user_id"))    // true
    fmt.Println(ValidateColumnName("user-id"))    // false
    fmt.Println(SanitizeInput("Hello\x00World"))  // "Hello World"
    fmt.Println(ValidateEmail("user@example.com")) // true
}
```

## 🎯 Best Practices

### Input Validation Guidelines

1. **Always Validate at Multiple Levels**
   - Application layer validation
   - Database constraint validation
   - API input validation

2. **Use Parameterized Queries**
   ```sql
   -- ✅ Good: Parameterized query
   SELECT * FROM users WHERE email = $1;
   
   -- ❌ Bad: String concatenation
   SELECT * FROM users WHERE email = '" + userEmail + "';
   ```

3. **Sanitize Before Storage**
   ```sql
   -- Add sanitization triggers
   CREATE TRIGGER sanitize_user_data
   BEFORE INSERT OR UPDATE ON users
   FOR EACH ROW
   EXECUTE FUNCTION sanitize_user_row();
   ```

4. **Log Sanitization Errors**
   ```sql
   -- Error logging table
   CREATE TABLE sanitization_logs (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       table_name TEXT NOT NULL,
       column_name TEXT NOT NULL,
       original_value TEXT,
       error_message TEXT,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```

### Security Checklist

- [ ] Validate all user inputs
- [ ] Use parameterized queries
- [ ] Sanitize data before storage
- [ ] Implement rate limiting
- [ ] Log suspicious activities
- [ ] Regular security audits
- [ ] Keep validation rules updated
- [ ] Test with malicious inputs

## 🔄 Monitoring and Maintenance

### Sanitization Metrics

```sql
-- Track sanitization effectiveness
CREATE VIEW sanitization_metrics AS
SELECT 
    table_name,
    column_name,
    COUNT(*) as error_count,
    COUNT(DISTINCT error_message) as unique_errors,
    MIN(created_at) as first_error,
    MAX(created_at) as last_error
FROM sanitization_logs
GROUP BY table_name, column_name
ORDER BY error_count DESC;
```

This comprehensive guide provides robust data sanitization strategies to protect your database from security vulnerabilities while maintaining data integrity and performance.
