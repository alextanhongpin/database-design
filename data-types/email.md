# Email Data Types and Storage

Best practices for storing and validating email addresses in database applications.

## 📚 Table of Contents

- [Storage Strategies](#storage-strategies)
- [Validation Patterns](#validation-patterns)
- [Case Sensitivity](#case-sensitivity)
- [Database Schema Design](#database-schema-design)
- [Performance Considerations](#performance-considerations)
- [Security Considerations](#security-considerations)
- [Application Implementation](#application-implementation)

## Storage Strategies

### Basic Email Storage
```sql
-- PostgreSQL
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(320) UNIQUE NOT NULL,  -- RFC 5321 max length
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- MySQL
CREATE TABLE users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(320) UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Case-Insensitive Storage
```sql
-- PostgreSQL with citext extension (recommended)
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email CITEXT UNIQUE NOT NULL,  -- Case-insensitive text type
    email_verified BOOLEAN DEFAULT FALSE
);

-- Alternative: Functional index for case-insensitive uniqueness
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(320) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE
);

CREATE UNIQUE INDEX idx_users_email_lower 
ON users (LOWER(email));
```

### Normalized Email Storage
```sql
-- Store both original and normalized versions
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email_original VARCHAR(320) NOT NULL,     -- User's original input
    email_normalized VARCHAR(320) UNIQUE NOT NULL,  -- Lowercased, normalized
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Function to normalize email
CREATE OR REPLACE FUNCTION normalize_email(email_input TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(TRIM(email_input));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger to auto-normalize
CREATE OR REPLACE FUNCTION trigger_normalize_email()
RETURNS TRIGGER AS $$
BEGIN
    NEW.email_normalized = normalize_email(NEW.email_original);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER normalize_email_trigger
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION trigger_normalize_email();
```

## Validation Patterns

### Database-Level Validation
```sql
-- PostgreSQL: Email format validation with regex
ALTER TABLE users 
ADD CONSTRAINT valid_email_format 
CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- More comprehensive email validation
ALTER TABLE users 
ADD CONSTRAINT valid_email_comprehensive
CHECK (
    email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'
    AND LENGTH(email) >= 5
    AND LENGTH(email) <= 320
    AND email NOT LIKE '%..%'  -- No consecutive dots
    AND email NOT LIKE '.%'    -- No leading dot
    AND email NOT LIKE '%.'    -- No trailing dot
);
```

### Application-Level Validation
```javascript
// JavaScript email validation
function validateEmail(email) {
    const errors = [];
    
    // Basic format check
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
        errors.push('Invalid email format');
    }
    
    // Length checks
    if (email.length > 320) {
        errors.push('Email too long (max 320 characters)');
    }
    
    if (email.length < 5) {
        errors.push('Email too short (min 5 characters)');
    }
    
    // Local part (before @) validation
    const [localPart, domain] = email.split('@');
    if (localPart && localPart.length > 64) {
        errors.push('Local part too long (max 64 characters)');
    }
    
    // Domain validation
    if (domain && domain.length > 253) {
        errors.push('Domain too long (max 253 characters)');
    }
    
    return {
        isValid: errors.length === 0,
        errors
    };
}

// Usage
const result = validateEmail('user@example.com');
if (!result.isValid) {
    console.log('Validation errors:', result.errors);
}
```

```python
# Python email validation with comprehensive checks
import re
from typing import Dict, List

def validate_email(email: str) -> Dict[str, any]:
    """
    Comprehensive email validation
    Returns: {isValid: bool, errors: List[str], normalized: str}
    """
    errors = []
    
    if not email:
        return {"isValid": False, "errors": ["Email is required"], "normalized": None}
    
    # Normalize email
    normalized = email.strip().lower()
    
    # Length validation
    if len(normalized) > 320:
        errors.append("Email too long (max 320 characters)")
    elif len(normalized) < 5:
        errors.append("Email too short (min 5 characters)")
    
    # Basic format validation
    email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    if not re.match(email_pattern, normalized):
        errors.append("Invalid email format")
    else:
        # Split into local and domain parts
        local_part, domain = normalized.split('@')
        
        # Local part validation
        if len(local_part) > 64:
            errors.append("Local part too long (max 64 characters)")
        
        if local_part.startswith('.') or local_part.endswith('.'):
            errors.append("Local part cannot start or end with a dot")
        
        if '..' in local_part:
            errors.append("Local part cannot contain consecutive dots")
        
        # Domain validation
        if len(domain) > 253:
            errors.append("Domain too long (max 253 characters)")
        
        if domain.startswith('.') or domain.endswith('.'):
            errors.append("Domain cannot start or end with a dot")
        
        if '..' in domain:
            errors.append("Domain cannot contain consecutive dots")
    
    return {
        "isValid": len(errors) == 0,
        "errors": errors,
        "normalized": normalized if len(errors) == 0 else None
    }

# Usage
result = validate_email("User@Example.COM")
print(f"Valid: {result['isValid']}")
print(f"Normalized: {result['normalized']}")
```

```go
// Go email validation
package main

import (
    "errors"
    "fmt"
    "regexp"
    "strings"
)

type EmailValidationResult struct {
    IsValid    bool
    Errors     []string
    Normalized string
}

func ValidateEmail(email string) EmailValidationResult {
    var errors []string
    
    if email == "" {
        return EmailValidationResult{
            IsValid: false,
            Errors:  []string{"email is required"},
        }
    }
    
    // Normalize email
    normalized := strings.ToLower(strings.TrimSpace(email))
    
    // Length validation
    if len(normalized) > 320 {
        errors = append(errors, "email too long (max 320 characters)")
    } else if len(normalized) < 5 {
        errors = append(errors, "email too short (min 5 characters)")
    }
    
    // Format validation
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !emailRegex.MatchString(normalized) {
        errors = append(errors, "invalid email format")
    } else {
        // Additional validation
        parts := strings.Split(normalized, "@")
        if len(parts) == 2 {
            localPart, domain := parts[0], parts[1]
            
            // Local part validation
            if len(localPart) > 64 {
                errors = append(errors, "local part too long (max 64 characters)")
            }
            
            if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
                errors = append(errors, "local part cannot start or end with a dot")
            }
            
            if strings.Contains(localPart, "..") {
                errors = append(errors, "local part cannot contain consecutive dots")
            }
            
            // Domain validation
            if len(domain) > 253 {
                errors = append(errors, "domain too long (max 253 characters)")
            }
        }
    }
    
    return EmailValidationResult{
        IsValid:    len(errors) == 0,
        Errors:     errors,
        Normalized: normalized,
    }
}

// Usage
func main() {
    result := ValidateEmail("User@Example.COM")
    fmt.Printf("Valid: %v\n", result.IsValid)
    fmt.Printf("Normalized: %s\n", result.Normalized)
}
```

## Case Sensitivity

### The Lowercase Question

**Should emails be stored as lowercase?**

**Answer: Yes, always normalize to lowercase for these reasons:**

1. **User Experience**: Users often accidentally capitalize the first character on mobile devices
2. **Industry Practice**: Google, LastPass, and most major services treat emails as case-insensitive
3. **Login Consistency**: Prevents users from being unable to log in due to case differences
4. **Database Efficiency**: Consistent casing enables better indexing and query performance

```sql
-- Application logic for email normalization
-- Always normalize before storing or querying
INSERT INTO users (email) VALUES (LOWER(TRIM('User@Example.COM')));

-- Query normalization
SELECT * FROM users WHERE email = LOWER(TRIM('User@Example.COM'));
```

### Preserving Original Case

If you need to preserve the original case for display purposes:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email_display VARCHAR(320),           -- Original case for display
    email_canonical VARCHAR(320) UNIQUE NOT NULL,  -- Lowercase for uniqueness
    email_verified BOOLEAN DEFAULT FALSE
);

-- Insert preserving both versions
INSERT INTO users (email_display, email_canonical)
VALUES ('User@Example.COM', LOWER('User@Example.COM'));
```

## Database Schema Design

### Email Verification System
```sql
-- Comprehensive email verification schema
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email CITEXT UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE email_verification_tokens (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Only one active token per user
    CONSTRAINT unique_active_token 
    UNIQUE (user_id) WHERE used_at IS NULL
);

-- Index for token lookups
CREATE INDEX idx_verification_tokens_token ON email_verification_tokens(token);
CREATE INDEX idx_verification_tokens_expires ON email_verification_tokens(expires_at);
```

### Email Change Tracking
```sql
-- Track email changes for security
CREATE TABLE email_change_history (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    old_email CITEXT,
    new_email CITEXT NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    changed_by_ip INET,
    changed_by_user_agent TEXT,
    verified_at TIMESTAMPTZ
);

-- Index for security monitoring
CREATE INDEX idx_email_changes_user ON email_change_history(user_id, changed_at);
CREATE INDEX idx_email_changes_email ON email_change_history(old_email, new_email);
```

## Performance Considerations

### Indexing Strategies
```sql
-- Primary email index (already unique)
-- UNIQUE constraint automatically creates index

-- Searching by domain
CREATE INDEX idx_users_email_domain 
ON users (SUBSTRING(email FROM '@(.*)$'));

-- Partial index for unverified emails
CREATE INDEX idx_users_unverified_email 
ON users (email) WHERE email_verified = FALSE;

-- Composite index for common queries
CREATE INDEX idx_users_email_status 
ON users (email, email_verified, created_at);
```

### Query Optimization
```sql
-- Efficient email existence check
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1) as email_exists;

-- Bulk email validation
SELECT email, email_verified 
FROM users 
WHERE email = ANY($1::text[]);  -- Pass array of emails

-- Domain-based queries
SELECT COUNT(*) 
FROM users 
WHERE email LIKE '%@gmail.com';

-- More efficient domain query with functional index
SELECT COUNT(*) 
FROM users 
WHERE SUBSTRING(email FROM '@(.*)$') = 'gmail.com';
```

## Security Considerations

### Email Enumeration Prevention
```sql
-- Don't expose whether email exists during registration
-- Always return success, send email if user exists

-- Rate limiting for email operations
CREATE TABLE email_rate_limits (
    id SERIAL PRIMARY KEY,
    ip_address INET NOT NULL,
    email CITEXT NOT NULL,
    operation_type VARCHAR(50) NOT NULL,  -- 'registration', 'verification', 'reset'
    attempts INT DEFAULT 1,
    window_start TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(ip_address, email, operation_type)
);

-- Clean up old rate limit records
CREATE INDEX idx_rate_limits_cleanup ON email_rate_limits(window_start);
```

### SQL Injection Prevention
```javascript
// Always use parameterized queries
// ❌ Never do this
const query = `SELECT * FROM users WHERE email = '${userInput}'`;

// ✅ Always use parameters
const query = 'SELECT * FROM users WHERE email = $1';
const result = await client.query(query, [userInput]);
```

## Application Implementation

### Email Service Class
```typescript
interface EmailValidationOptions {
    requireVerification?: boolean;
    allowedDomains?: string[];
    blockedDomains?: string[];
    maxAttempts?: number;
}

class EmailService {
    constructor(private db: Database) {}
    
    async validateAndNormalize(
        email: string, 
        options: EmailValidationOptions = {}
    ): Promise<{isValid: boolean, normalized?: string, errors: string[]}> {
        const errors: string[] = [];
        
        if (!email) {
            return {isValid: false, errors: ['Email is required']};
        }
        
        const normalized = email.trim().toLowerCase();
        
        // Basic validation
        const validationResult = this.validateFormat(normalized);
        if (!validationResult.isValid) {
            return validationResult;
        }
        
        // Domain restrictions
        if (options.allowedDomains) {
            const domain = this.extractDomain(normalized);
            if (!options.allowedDomains.includes(domain)) {
                errors.push('Email domain not allowed');
            }
        }
        
        if (options.blockedDomains) {
            const domain = this.extractDomain(normalized);
            if (options.blockedDomains.includes(domain)) {
                errors.push('Email domain is blocked');
            }
        }
        
        return {
            isValid: errors.length === 0,
            normalized: errors.length === 0 ? normalized : undefined,
            errors
        };
    }
    
    async checkExists(email: string): Promise<boolean> {
        const normalized = email.trim().toLowerCase();
        const result = await this.db.query(
            'SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)',
            [normalized]
        );
        return result.rows[0].exists;
    }
    
    private validateFormat(email: string): {isValid: boolean, errors: string[]} {
        const errors: string[] = [];
        
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(email)) {
            errors.push('Invalid email format');
        }
        
        if (email.length > 320) {
            errors.push('Email too long');
        }
        
        return {isValid: errors.length === 0, errors};
    }
    
    private extractDomain(email: string): string {
        return email.split('@')[1] || '';
    }
}
```

## Best Practices Summary

### Storage
1. **Use CITEXT** (PostgreSQL) or functional indexes for case-insensitive storage
2. **Always normalize** emails to lowercase before storage and comparison
3. **Validate length** according to RFC 5321 (320 characters max)
4. **Consider preserving** original case for display if needed

### Validation
1. **Validate at multiple layers** - database constraints and application logic
2. **Use comprehensive regex** patterns but don't over-complicate
3. **Validate local and domain parts** separately for better error messages
4. **Implement rate limiting** to prevent abuse

### Security
1. **Prevent email enumeration** during registration and login flows
2. **Use parameterized queries** always
3. **Implement email verification** for security-sensitive operations
4. **Track email changes** for audit and security monitoring

### Performance
1. **Index strategically** based on query patterns
2. **Use partial indexes** for filtered queries (e.g., unverified emails)
3. **Consider domain-based indexing** for domain analysis queries
4. **Implement efficient existence checks** for registration flows

### User Experience
1. **Provide helpful validation messages** that guide users to correct input
2. **Handle case sensitivity transparently** - users shouldn't worry about it
3. **Implement proper error handling** without exposing sensitive information
4. **Support email updates** with proper verification flows
