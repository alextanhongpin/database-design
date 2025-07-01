# Business Rules in Database Design

Business rules are the policies, constraints, and logic that govern how a business operates. The decision of where to implement these rules - in the database, application layer, or both - is crucial for data integrity, performance, and maintainability.

## 🎯 What Are Business Rules?

### Types of Business Rules

**Data Integrity Rules**
- Required fields (`NOT NULL`)
- Unique constraints (`UNIQUE`)
- Referential integrity (`FOREIGN KEY`)
- Value ranges (`CHECK` constraints)

**Business Logic Rules**
- Pricing calculations
- Workflow transitions
- Access permissions
- Validation rules

**Temporal Rules**
- Date ranges (start_date < end_date)
- Business hours constraints
- Expiration policies

## ⚖️ Database vs Application Layer

### 🏛️ Database Layer Benefits

#### Guarantees Data Integrity
```sql
-- Price validation - ALWAYS enforced
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price_cents INTEGER NOT NULL CHECK (price_cents > 0),
    discount_percentage DECIMAL(5,2) CHECK (
        discount_percentage >= 0 AND discount_percentage <= 100
    ),
    
    -- Business rule: Sale price must be less than original price
    sale_price_cents INTEGER CHECK (
        sale_price_cents IS NULL OR sale_price_cents < price_cents
    )
);

-- This constraint CANNOT be bypassed
INSERT INTO products (name, price_cents, sale_price_cents) 
VALUES ('Widget', 1000, 1200); -- ERROR: violates check constraint
```

#### Single Source of Truth
```sql
-- Order validation that applies to ALL applications
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled')
    ),
    total_cents INTEGER NOT NULL CHECK (total_cents > 0),
    
    -- Business rule: Can't ship without confirming
    confirmed_at TIMESTAMP,
    shipped_at TIMESTAMP,
    
    CONSTRAINT shipping_requires_confirmation CHECK (
        shipped_at IS NULL OR confirmed_at IS NOT NULL
    ),
    
    -- Business rule: Delivered orders must be shipped first
    delivered_at TIMESTAMP,
    
    CONSTRAINT delivery_requires_shipping CHECK (
        delivered_at IS NULL OR shipped_at IS NOT NULL
    )
);

-- These rules apply whether data comes from:
-- - Web application
-- - Mobile app  
-- - Admin interface
-- - Data import scripts
-- - Direct database access
```

#### Performance & Atomicity
```sql
-- Complex business rule implemented efficiently in database
CREATE OR REPLACE FUNCTION apply_volume_discount()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate volume discount based on order quantity
    IF NEW.quantity >= 100 THEN
        NEW.unit_price_cents := NEW.unit_price_cents * 0.8; -- 20% discount
    ELSIF NEW.quantity >= 50 THEN  
        NEW.unit_price_cents := NEW.unit_price_cents * 0.9; -- 10% discount
    ELSIF NEW.quantity >= 10 THEN
        NEW.unit_price_cents := NEW.unit_price_cents * 0.95; -- 5% discount
    END IF;
    
    -- Update total
    NEW.total_cents := NEW.quantity * NEW.unit_price_cents;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER volume_discount_trigger
    BEFORE INSERT OR UPDATE ON order_items
    FOR EACH ROW EXECUTE FUNCTION apply_volume_discount();
```

### 💻 Application Layer Benefits

#### Complex Logic & External Dependencies
```javascript
// Business rules requiring external APIs or complex calculations
class OrderService {
    async calculateShipping(order, address) {
        // Call external shipping API
        const rates = await shippingAPI.getRates(order.items, address);
        
        // Complex business logic
        const freeShippingThreshold = await this.getFreeShippingThreshold(order.customer);
        
        if (order.total >= freeShippingThreshold) {
            return 0;
        }
        
        // Apply customer-specific shipping discounts
        const discount = await this.getShippingDiscount(order.customer);
        return Math.max(0, rates.standard * (1 - discount));
    }
    
    async validateInventory(orderItems) {
        // Check inventory across multiple warehouses
        // Consider reserved stock, incoming shipments, etc.
        const availability = await inventoryService.checkAvailability(orderItems);
        
        // Complex allocation logic
        return this.optimizeInventoryAllocation(availability, orderItems);
    }
}
```

#### Flexibility & Testing
```javascript
// Easy to test and modify business rules
describe('Order validation', () => {
    test('should reject orders over credit limit', () => {
        const customer = { creditLimit: 1000, currentBalance: 800 };
        const order = { total: 300 };
        
        expect(() => validateOrder(order, customer))
            .toThrow('Order exceeds available credit');
    });
    
    test('should allow orders within credit limit', () => {
        const customer = { creditLimit: 1000, currentBalance: 700 };
        const order = { total: 200 };
        
        expect(validateOrder(order, customer)).toBe(true);
    });
});
```

## 🏗️ Hybrid Approach: Best of Both Worlds

### Critical Rules in Database
```sql
-- Non-negotiable business rules in database
CREATE TABLE bank_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_number TEXT NOT NULL UNIQUE,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    
    -- CRITICAL: Account balance cannot go negative (overdraft protection)
    CONSTRAINT no_negative_balance CHECK (balance_cents >= 0),
    
    -- CRITICAL: Account numbers follow specific format
    CONSTRAINT valid_account_format CHECK (
        account_number ~ '^[0-9]{10,12}$'
    )
);

-- Transaction logging for audit (database ensures this ALWAYS happens)
CREATE OR REPLACE FUNCTION log_balance_changes()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO account_transaction_log (
        account_id, 
        old_balance_cents, 
        new_balance_cents, 
        change_cents,
        operation,
        created_at
    ) VALUES (
        COALESCE(NEW.id, OLD.id),
        COALESCE(OLD.balance_cents, 0),
        COALESCE(NEW.balance_cents, 0),
        COALESCE(NEW.balance_cents, 0) - COALESCE(OLD.balance_cents, 0),
        TG_OP,
        NOW()
    );
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER account_balance_log_trigger
    AFTER INSERT OR UPDATE OR DELETE ON bank_accounts
    FOR EACH ROW EXECUTE FUNCTION log_balance_changes();
```

### Business Logic in Application
```javascript
// Complex business logic in application layer
class TransferService {
    async transfer(fromAccountId, toAccountId, amountCents, memo) {
        return await db.transaction(async (trx) => {
            // 1. Validate transfer limits (complex business logic)
            await this.validateTransferLimits(fromAccountId, amountCents);
            
            // 2. Check for fraud patterns (external service)
            await fraudService.validateTransfer(fromAccountId, toAccountId, amountCents);
            
            // 3. Apply fees (complex calculation)
            const fees = await this.calculateTransferFees(fromAccountId, toAccountId, amountCents);
            
            // 4. Execute transfer (database constraints ensure integrity)
            const fromAccount = await trx('bank_accounts')
                .where('id', fromAccountId)
                .forUpdate()
                .first();
            
            if (fromAccount.balance_cents < amountCents + fees) {
                throw new Error('Insufficient funds');
            }
            
            // Database constraints will prevent negative balances
            await trx('bank_accounts')
                .where('id', fromAccountId)
                .decrement('balance_cents', amountCents + fees);
            
            await trx('bank_accounts')
                .where('id', toAccountId)
                .increment('balance_cents', amountCents);
            
            // 5. Record transaction details
            await this.recordTransfer(trx, fromAccountId, toAccountId, amountCents, fees, memo);
        });
    }
}
```

## 🌍 Real-World Examples

### E-Commerce Platform Rules

#### Database Layer (Non-Negotiable)
```sql
-- Product pricing rules
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price_cents INTEGER NOT NULL CHECK (price_cents > 0),
    
    -- Business rule: Cannot have more than 99% discount
    discount_percentage DECIMAL(5,2) CHECK (
        discount_percentage IS NULL OR 
        (discount_percentage >= 0 AND discount_percentage < 99)
    ),
    
    -- Business rule: Sale price must be reasonable
    sale_price_cents INTEGER CHECK (
        sale_price_cents IS NULL OR 
        (sale_price_cents > 0 AND sale_price_cents <= price_cents)
    )
);

-- Inventory rules
CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_variant_id UUID NOT NULL,
    movement_type TEXT NOT NULL CHECK (
        movement_type IN ('purchase', 'sale', 'adjustment', 'return')
    ),
    quantity_change INTEGER NOT NULL CHECK (quantity_change != 0),
    
    -- Business rule: Sales must be negative quantities
    CONSTRAINT sales_are_negative CHECK (
        movement_type != 'sale' OR quantity_change < 0
    ),
    
    -- Business rule: Purchases must be positive quantities  
    CONSTRAINT purchases_are_positive CHECK (
        movement_type != 'purchase' OR quantity_change > 0
    )
);

-- Order constraints
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'cart' CHECK (
        status IN ('cart', 'pending', 'confirmed', 'shipped', 'delivered', 'cancelled')
    ),
    
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    
    -- Business rule: Confirmed orders must have payment
    payment_id UUID,
    confirmed_at TIMESTAMP,
    
    CONSTRAINT confirmed_orders_require_payment CHECK (
        confirmed_at IS NULL OR payment_id IS NOT NULL
    ),
    
    -- Business rule: Shipped orders must be confirmed
    shipped_at TIMESTAMP,
    
    CONSTRAINT shipped_orders_must_be_confirmed CHECK (
        shipped_at IS NULL OR confirmed_at IS NOT NULL
    )
);
```

#### Application Layer (Business Logic)
```javascript
class ECommerceBusinessRules {
    // Dynamic pricing based on customer segment, time, inventory, etc.
    async calculatePrice(productId, customerId, quantity = 1) {
        const product = await Product.findById(productId);
        const customer = await Customer.findById(customerId);
        
        let price = product.price_cents;
        
        // Volume discounts
        if (quantity >= 10) price *= 0.95;
        if (quantity >= 50) price *= 0.90;
        if (quantity >= 100) price *= 0.85;
        
        // Customer segment pricing
        if (customer.segment === 'premium') {
            price *= 0.9; // 10% discount for premium customers
        }
        
        // Time-based pricing
        const hour = new Date().getHours();
        if (hour >= 2 && hour <= 6) { // Early morning discount
            price *= 0.95;
        }
        
        // Inventory clearance
        const inventory = await this.getInventoryLevel(productId);
        if (inventory < 10) {
            price *= 1.1; // 10% markup for low stock
        }
        
        return Math.round(price);
    }
    
    // Complex shipping calculation
    async calculateShipping(order, address) {
        const items = await order.getItems();
        
        // Free shipping thresholds by customer type
        const freeShippingThreshold = {
            'premium': 5000, // $50
            'regular': 7500, // $75
            'new': 10000     // $100
        };
        
        const customer = await order.getCustomer();
        const threshold = freeShippingThreshold[customer.type] || 10000;
        
        if (order.total_cents >= threshold) {
            return 0;
        }
        
        // Calculate shipping based on weight, destination, speed
        const weight = items.reduce((sum, item) => sum + item.weight_grams, 0);
        const distance = await shippingService.calculateDistance(
            await this.getWarehouseAddress(items), 
            address
        );
        
        return shippingService.calculateRate(weight, distance);
    }
}
```

### SaaS Platform Rules

#### Database Layer (Limits & Constraints)
```sql
-- Tenant resource limits
CREATE TABLE tenant_subscriptions (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id),
    plan_name TEXT NOT NULL CHECK (
        plan_name IN ('free', 'starter', 'professional', 'enterprise')
    ),
    
    -- Hard limits enforced by database
    max_users INTEGER NOT NULL CHECK (max_users > 0),
    max_projects INTEGER NOT NULL CHECK (max_projects > 0),
    max_storage_gb INTEGER NOT NULL CHECK (max_storage_gb > 0),
    
    -- Usage tracking
    current_users INTEGER DEFAULT 0 CHECK (current_users >= 0),
    current_projects INTEGER DEFAULT 0 CHECK (current_projects >= 0),
    current_storage_gb DECIMAL(10,2) DEFAULT 0 CHECK (current_storage_gb >= 0),
    
    -- Business rule: Cannot exceed limits
    CONSTRAINT user_limit CHECK (current_users <= max_users),
    CONSTRAINT project_limit CHECK (current_projects <= max_projects),
    CONSTRAINT storage_limit CHECK (current_storage_gb <= max_storage_gb),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Usage enforcement triggers
CREATE OR REPLACE FUNCTION enforce_tenant_limits()
RETURNS TRIGGER AS $$
DECLARE
    subscription RECORD;
BEGIN
    -- Get tenant subscription
    SELECT * INTO subscription 
    FROM tenant_subscriptions 
    WHERE tenant_id = NEW.tenant_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'No subscription found for tenant %', NEW.tenant_id;
    END IF;
    
    -- Enforce limits based on what's being created
    IF TG_TABLE_NAME = 'tenant_users' THEN
        IF subscription.current_users >= subscription.max_users THEN
            RAISE EXCEPTION 'User limit exceeded. Current: %, Max: %', 
                subscription.current_users, subscription.max_users;
        END IF;
        
        -- Update usage counter
        UPDATE tenant_subscriptions 
        SET current_users = current_users + 1 
        WHERE tenant_id = NEW.tenant_id;
        
    ELSIF TG_TABLE_NAME = 'projects' THEN
        IF subscription.current_projects >= subscription.max_projects THEN
            RAISE EXCEPTION 'Project limit exceeded. Current: %, Max: %', 
                subscription.current_projects, subscription.max_projects;
        END IF;
        
        UPDATE tenant_subscriptions 
        SET current_projects = current_projects + 1 
        WHERE tenant_id = NEW.tenant_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enforce_user_limits
    BEFORE INSERT ON tenant_users
    FOR EACH ROW EXECUTE FUNCTION enforce_tenant_limits();

CREATE TRIGGER enforce_project_limits
    BEFORE INSERT ON projects  
    FOR EACH ROW EXECUTE FUNCTION enforce_tenant_limits();
```

#### Application Layer (Feature Access)
```javascript
class SaaSBusinessRules {
    // Feature access control
    async checkFeatureAccess(tenantId, featureName) {
        const subscription = await TenantSubscription.findByTenantId(tenantId);
        
        const featureMatrix = {
            'free': ['basic_projects', 'basic_support'],
            'starter': ['basic_projects', 'basic_support', 'advanced_analytics'],
            'professional': ['basic_projects', 'basic_support', 'advanced_analytics', 'api_access', 'priority_support'],
            'enterprise': ['*'] // All features
        };
        
        const allowedFeatures = featureMatrix[subscription.plan_name] || [];
        
        return allowedFeatures.includes('*') || allowedFeatures.includes(featureName);
    }
    
    // Dynamic limit checking for soft limits
    async checkSoftLimits(tenantId, resourceType, increment = 1) {
        const subscription = await TenantSubscription.findByTenantId(tenantId);
        const usage = await this.getCurrentUsage(tenantId);
        
        // Soft limits with warnings
        const softLimitThreshold = 0.8; // 80% of hard limit
        
        let currentUsage, maxLimit, limitName;
        
        switch (resourceType) {
            case 'api_calls':
                currentUsage = usage.monthly_api_calls;
                maxLimit = subscription.max_monthly_api_calls;
                limitName = 'monthly API calls';
                break;
            case 'storage':
                currentUsage = usage.storage_gb;
                maxLimit = subscription.max_storage_gb;
                limitName = 'storage';
                break;
        }
        
        const newUsage = currentUsage + increment;
        const softLimit = maxLimit * softLimitThreshold;
        
        if (newUsage >= maxLimit) {
            throw new Error(`${limitName} limit exceeded`);
        }
        
        if (newUsage >= softLimit) {
            // Send warning notification
            await notificationService.sendLimitWarning(tenantId, {
                resourceType,
                currentUsage: newUsage,
                maxLimit,
                percentageUsed: Math.round((newUsage / maxLimit) * 100)
            });
        }
        
        return true;
    }
}
```

## 🛠️ Implementation Patterns

### 1. Layered Business Rules
```sql
-- Layer 1: Database constraints (always enforced)
CREATE TABLE financial_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount_cents BIGINT NOT NULL,
    
    -- CRITICAL: Amounts cannot be zero
    CONSTRAINT non_zero_amount CHECK (amount_cents != 0),
    
    -- CRITICAL: Debits are negative, credits are positive
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('debit', 'credit')),
    
    CONSTRAINT amount_sign_consistency CHECK (
        (transaction_type = 'debit' AND amount_cents < 0) OR
        (transaction_type = 'credit' AND amount_cents > 0)
    )
);

-- Layer 2: Application business logic (flexible)
class TransactionRules {
    validateTransactionLimits(userId, amountCents) {
        // Complex rules that can change without database migrations
        const user = await User.findById(userId);
        const dailyLimit = this.getDailyLimit(user.tier);
        const todayTotal = await this.getTodayTransactions(userId);
        
        if (Math.abs(amountCents) > dailyLimit) {
            throw new Error('Transaction exceeds daily limit');
        }
        
        if (todayTotal + Math.abs(amountCents) > dailyLimit) {
            throw new Error('Transaction would exceed daily limit');
        }
    }
}
```

### 2. Rule Engine Pattern
```sql
-- Flexible rule storage for complex business logic
CREATE TABLE business_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name TEXT NOT NULL UNIQUE,
    description TEXT,
    
    -- Rule conditions and actions stored as JSON
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    
    -- Rule metadata
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Applicability
    applies_to_table TEXT,
    applies_to_operation TEXT CHECK (
        applies_to_operation IN ('INSERT', 'UPDATE', 'DELETE')
    ),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Example rules
INSERT INTO business_rules (rule_name, description, conditions, actions, applies_to_table) VALUES
(
    'high_value_order_approval',
    'Orders over $10,000 require manager approval',
    '{"total_cents": {"operator": "gt", "value": 1000000}}',
    '{"set_status": "pending_approval", "notify": ["managers"]}',
    'orders'
),
(
    'volume_discount',
    'Apply 10% discount for orders with 50+ items',
    '{"item_count": {"operator": "gte", "value": 50}}',
    '{"apply_discount": {"type": "percentage", "value": 10}}',
    'orders'
);

-- Rule evaluation function
CREATE OR REPLACE FUNCTION evaluate_business_rules()
RETURNS TRIGGER AS $$
DECLARE
    rule RECORD;
    conditions_met BOOLEAN;
BEGIN
    -- Evaluate applicable rules
    FOR rule IN 
        SELECT * FROM business_rules 
        WHERE is_active = TRUE 
        AND applies_to_table = TG_TABLE_NAME
        AND applies_to_operation = TG_OP
        ORDER BY priority DESC
    LOOP
        -- Evaluate conditions (simplified - real implementation would be more complex)
        conditions_met := evaluate_rule_conditions(rule.conditions, row_to_json(NEW));
        
        IF conditions_met THEN
            -- Apply actions
            NEW := apply_rule_actions(rule.actions, NEW);
        END IF;
    END LOOP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### 3. Temporal Business Rules
```sql
-- Rules that change over time
CREATE TABLE pricing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_category TEXT NOT NULL,
    customer_segment TEXT NOT NULL,
    
    -- Rule definition
    rule_type TEXT NOT NULL CHECK (rule_type IN ('discount', 'markup', 'fixed_price')),
    value_percentage DECIMAL(5,2),
    value_fixed_cents INTEGER,
    
    -- Time validity
    effective_from TIMESTAMP NOT NULL DEFAULT NOW(),
    effective_until TIMESTAMP,
    
    -- Days of week (bit mask: Sunday=1, Monday=2, etc.)
    applicable_days INTEGER DEFAULT 127, -- All days
    
    -- Time ranges
    applicable_time_start TIME,
    applicable_time_end TIME,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure rule has either percentage or fixed value
    CONSTRAINT valid_rule_value CHECK (
        (rule_type = 'discount' AND value_percentage IS NOT NULL) OR
        (rule_type = 'markup' AND value_percentage IS NOT NULL) OR
        (rule_type = 'fixed_price' AND value_fixed_cents IS NOT NULL)
    )
);

-- Function to get applicable pricing rules
CREATE OR REPLACE FUNCTION get_applicable_pricing_rules(
    p_product_category TEXT,
    p_customer_segment TEXT,
    p_check_time TIMESTAMP DEFAULT NOW()
) RETURNS SETOF pricing_rules AS $$
BEGIN
    RETURN QUERY
    SELECT pr.*
    FROM pricing_rules pr
    WHERE pr.product_category = p_product_category
    AND pr.customer_segment = p_customer_segment
    AND pr.is_active = TRUE
    AND pr.effective_from <= p_check_time
    AND (pr.effective_until IS NULL OR pr.effective_until > p_check_time)
    -- Check day of week
    AND (pr.applicable_days & (1 << EXTRACT(DOW FROM p_check_time)::INTEGER)) > 0
    -- Check time of day
    AND (
        pr.applicable_time_start IS NULL OR 
        pr.applicable_time_end IS NULL OR
        p_check_time::TIME BETWEEN pr.applicable_time_start AND pr.applicable_time_end
    )
    ORDER BY pr.effective_from DESC;
END;
$$ LANGUAGE plpgsql;
```

## 📊 Monitoring & Analytics

### Rule Effectiveness Tracking
```sql
-- Track business rule violations and effectiveness
CREATE TABLE rule_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    record_id UUID,
    
    violation_type TEXT NOT NULL CHECK (
        violation_type IN ('constraint_violation', 'business_rule_failure', 'warning')
    ),
    
    violation_details JSONB,
    severity TEXT NOT NULL DEFAULT 'medium' CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    ),
    
    -- Context
    user_id UUID,
    session_id TEXT,
    ip_address INET,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Rule performance analytics
CREATE VIEW business_rule_analytics AS
SELECT 
    rule_name,
    violation_type,
    severity,
    COUNT(*) as violation_count,
    COUNT(DISTINCT user_id) as affected_users,
    COUNT(DISTINCT DATE(created_at)) as affected_days,
    
    MIN(created_at) as first_violation,
    MAX(created_at) as last_violation,
    
    -- Trend analysis
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as violations_last_24h,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '7 days') as violations_last_7d
    
FROM rule_violations
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY rule_name, violation_type, severity
ORDER BY violation_count DESC;
```

## 🚨 Common Pitfalls & Solutions

### Pitfall 1: All Rules in Application
```javascript
// ❌ Problem: No data integrity protection
class OrderService {
    async createOrder(orderData) {
        // Application validation only
        if (orderData.total <= 0) {
            throw new Error('Invalid order total');
        }
        
        // But direct database access bypasses this
        return db.orders.insert(orderData);
    }
}

// ✅ Solution: Critical rules in database
CREATE TABLE orders (
    total_cents INTEGER NOT NULL CHECK (total_cents > 0) -- Always enforced
);
```

### Pitfall 2: All Rules in Database
```sql
-- ❌ Problem: Inflexible, hard to test
CREATE OR REPLACE FUNCTION complex_pricing_logic()
RETURNS TRIGGER AS $$
BEGIN
    -- 200 lines of complex business logic in database
    -- Hard to test, debug, and modify
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ✅ Solution: Keep complex logic in application
-- Database: Basic constraints
-- Application: Complex business logic
```

### Pitfall 3: Inconsistent Rule Enforcement
```javascript
// ❌ Problem: Rules enforced inconsistently
function processOrder(order) {
    if (order.total > 10000) {
        // Sometimes remembered, sometimes forgotten
        order.requiresApproval = true;
    }
}

// ✅ Solution: Centralized rule engine
class BusinessRuleEngine {
    evaluateOrderRules(order) {
        const rules = this.getApplicableRules('order', order);
        return rules.reduce((result, rule) => {
            return rule.evaluate(result);
        }, order);
    }
}
```

## 💡 Best Practices

1. **Critical Rules in Database** - Data integrity, referential constraints
2. **Complex Logic in Application** - Business calculations, external dependencies
3. **Document Everything** - Rules change; documentation prevents confusion
4. **Version Control Rules** - Track changes to business logic
5. **Test Thoroughly** - Both database constraints and application logic
6. **Monitor Violations** - Track when rules are violated and why
7. **Plan for Change** - Business rules evolve; design for flexibility
8. **Performance First** - Database rules are faster for data validation
9. **Separation of Concerns** - Don't mix data integrity with business logic
10. **Fail Fast** - Validate early, fail with clear error messages

## 🔄 Evolution Strategy

### Phase 1: Start Simple
```sql
-- Begin with basic constraints
CREATE TABLE products (
    id UUID PRIMARY KEY,
    price_cents INTEGER NOT NULL CHECK (price_cents > 0)
);
```

### Phase 2: Add Business Logic
```javascript
// Add complex rules in application
class PricingService {
    calculatePrice(product, customer, context) {
        // Complex pricing logic
    }
}
```

### Phase 3: Optimize Performance
```sql
-- Move performance-critical rules to database
CREATE OR REPLACE FUNCTION calculate_discount(base_price INTEGER, quantity INTEGER)
RETURNS INTEGER AS $$
BEGIN
    RETURN CASE 
        WHEN quantity >= 100 THEN base_price * 0.8
        WHEN quantity >= 50 THEN base_price * 0.9
        ELSE base_price
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

### Phase 4: Add Flexibility
```sql
-- Configurable rules for changing requirements
CREATE TABLE business_rule_config (
    rule_name TEXT PRIMARY KEY,
    parameters JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);
```

Remember: **Business rules are not just technical constraints - they encode the essential logic that makes your business unique**. Choose the right layer for each rule based on criticality, complexity, and performance requirements.
