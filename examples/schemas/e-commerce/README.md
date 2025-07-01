# E-Commerce Database Schema

A comprehensive, production-ready database schema for an e-commerce platform supporting products, orders, payments, inventory, and customer management.

## 🎯 Business Requirements

### Core Features
- **Product Catalog** - Complex product hierarchies with variants, options, and pricing
- **Inventory Management** - Real-time stock tracking with multi-warehouse support
- **Order Processing** - Complete order lifecycle from cart to delivery
- **Payment Processing** - Multiple payment methods with fraud detection
- **Customer Management** - User accounts, profiles, and address management
- **Multi-Tenant** - Support for multiple sellers/vendors
- **Reviews & Ratings** - Product reviews with moderation
- **Promotions** - Coupons, discounts, and promotional campaigns

### Scale Requirements
- **Products**: 1M+ products with variants
- **Orders**: 10K+ orders per day
- **Customers**: 100K+ active customers
- **Inventory**: Real-time updates with 99.9% accuracy
- **Search**: Sub-second product search and filtering

## 🗄️ Database Schema

### Core Domain Types
```sql
-- Business-specific domain types for data validation
CREATE DOMAIN email AS TEXT 
CHECK (VALUE ~* '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$');

CREATE DOMAIN currency_code AS CHAR(3) 
CHECK (VALUE ~ '^[A-Z]{3}$');

CREATE DOMAIN product_sku AS TEXT
CHECK (
    LENGTH(VALUE) BETWEEN 6 AND 20 AND
    VALUE ~ '^[A-Z0-9-]+$'
);

CREATE DOMAIN phone_number AS TEXT
CHECK (VALUE ~ '^\+?[1-9]\d{1,14}$');

CREATE DOMAIN postal_code AS TEXT
CHECK (LENGTH(VALUE) BETWEEN 3 AND 10);

-- Price stored as cents to avoid floating point issues
CREATE DOMAIN price_cents AS INTEGER
CHECK (VALUE >= 0 AND VALUE <= 999999999); -- Max $9.9M

CREATE DOMAIN percentage AS DECIMAL(5,2)
CHECK (VALUE >= 0.00 AND VALUE <= 100.00);

CREATE DOMAIN weight_grams AS INTEGER
CHECK (VALUE > 0 AND VALUE <= 100000); -- Max 100kg
```

### User Management
```sql
-- Core user account
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email email NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    
    -- Account status
    status TEXT NOT NULL DEFAULT 'active' CHECK (
        status IN ('active', 'suspended', 'deactivated', 'deleted')
    ),
    
    -- Verification
    email_verified_at TIMESTAMP,
    phone_verified_at TIMESTAMP,
    
    -- Security
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    
    -- Timestamps
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- User profiles (separate for performance)
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    phone phone_number,
    date_of_birth DATE,
    
    -- Preferences
    preferred_currency currency_code DEFAULT 'USD',
    preferred_language CHAR(5) DEFAULT 'en-US',
    timezone TEXT DEFAULT 'UTC',
    
    -- Marketing preferences
    email_notifications BOOLEAN DEFAULT TRUE,
    sms_notifications BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- User addresses with validation
CREATE TABLE user_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Address components
    type TEXT NOT NULL CHECK (type IN ('shipping', 'billing', 'both')),
    company_name TEXT,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    address_line1 TEXT NOT NULL,
    address_line2 TEXT,
    city TEXT NOT NULL,
    state_province TEXT NOT NULL,
    postal_code postal_code NOT NULL,
    country_code CHAR(2) NOT NULL CHECK (country_code ~ '^[A-Z]{2}$'),
    
    -- Metadata
    is_default BOOLEAN DEFAULT FALSE,
    phone phone_number,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure only one default address per type per user
    UNIQUE (user_id, type, is_default) WHERE is_default = TRUE
);

-- Indexes for user management
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_status ON users (status) WHERE status != 'deleted';
CREATE INDEX idx_user_addresses_user ON user_addresses (user_id);
```

### Product Catalog
```sql
-- Product categories (nested hierarchy)
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES categories(id),
    
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    
    -- Display settings
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- SEO
    meta_title TEXT,
    meta_description TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Prevent cycles in hierarchy
    CHECK (parent_id IS NULL OR parent_id != id)
);

-- Brands/Manufacturers
CREATE TABLE brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    logo_url TEXT,
    website_url TEXT,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Main products table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku product_sku NOT NULL UNIQUE,
    
    -- Basic info
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    short_description TEXT,
    
    -- Categorization
    category_id UUID NOT NULL REFERENCES categories(id),
    brand_id UUID REFERENCES brands(id),
    
    -- Pricing (base price, variants can override)
    base_price_cents price_cents NOT NULL,
    compare_at_price_cents price_cents, -- For showing discounts
    cost_cents price_cents, -- For profit calculations
    
    -- Physical properties
    weight_grams weight_grams,
    dimensions_cm JSONB, -- {"length": 10, "width": 5, "height": 2}
    
    -- Status and visibility
    status TEXT NOT NULL DEFAULT 'draft' CHECK (
        status IN ('draft', 'active', 'archived', 'deleted')
    ),
    
    -- Inventory tracking
    track_inventory BOOLEAN DEFAULT TRUE,
    allow_backorder BOOLEAN DEFAULT FALSE,
    
    -- SEO and marketing
    meta_title TEXT,
    meta_description TEXT,
    search_keywords TEXT[], -- For search optimization
    
    -- Timestamps
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Business rules
    CONSTRAINT valid_pricing CHECK (
        compare_at_price_cents IS NULL OR 
        compare_at_price_cents > base_price_cents
    )
);

-- Product variants (color, size, etc.)
CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    
    sku product_sku NOT NULL UNIQUE,
    
    -- Variant attributes
    title TEXT NOT NULL, -- "Red / Large"
    option1_name TEXT, -- "Color"
    option1_value TEXT, -- "Red" 
    option2_name TEXT, -- "Size"
    option2_value TEXT, -- "Large"
    option3_name TEXT, -- "Material" 
    option3_value TEXT, -- "Cotton"
    
    -- Variant-specific pricing (overrides product base price)
    price_cents price_cents,
    compare_at_price_cents price_cents,
    cost_cents price_cents,
    
    -- Physical properties
    weight_grams weight_grams,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure unique combinations per product
    UNIQUE (product_id, option1_value, option2_value, option3_value)
);

-- Product images
CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
    
    url TEXT NOT NULL,
    alt_text TEXT,
    display_order INTEGER DEFAULT 0,
    
    -- Image metadata
    width INTEGER,
    height INTEGER,
    file_size INTEGER, -- bytes
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Images can be associated with product or specific variant
    CHECK (
        (product_id IS NOT NULL AND variant_id IS NULL) OR
        (product_id IS NOT NULL AND variant_id IS NOT NULL)
    )
);

-- Product search and filtering optimization
CREATE INDEX idx_products_category ON products (category_id) WHERE status = 'active';
CREATE INDEX idx_products_brand ON products (brand_id) WHERE status = 'active';
CREATE INDEX idx_products_price ON products (base_price_cents) WHERE status = 'active';
CREATE INDEX idx_products_search ON products USING gin(search_keywords) WHERE status = 'active';
CREATE INDEX idx_products_published ON products (published_at DESC) WHERE status = 'active';

-- Full-text search
CREATE INDEX idx_products_fts ON products USING gin(
    to_tsvector('english', name || ' ' || COALESCE(description, '') || ' ' || COALESCE(short_description, ''))
) WHERE status = 'active';
```

### Inventory Management
```sql
-- Warehouses/Locations
CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    
    -- Address
    address_line1 TEXT NOT NULL,
    address_line2 TEXT,
    city TEXT NOT NULL,
    state_province TEXT NOT NULL,
    postal_code postal_code NOT NULL,
    country_code CHAR(2) NOT NULL,
    
    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Only one default warehouse
    UNIQUE (is_default) WHERE is_default = TRUE
);

-- Inventory levels per variant per warehouse
CREATE TABLE inventory_levels (
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    
    -- Stock levels
    quantity_available INTEGER NOT NULL DEFAULT 0 CHECK (quantity_available >= 0),
    quantity_reserved INTEGER NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    quantity_incoming INTEGER NOT NULL DEFAULT 0 CHECK (quantity_incoming >= 0),
    
    -- Reorder settings
    reorder_point INTEGER DEFAULT 0,
    reorder_quantity INTEGER DEFAULT 0,
    
    -- Cost tracking
    average_cost_cents price_cents DEFAULT 0,
    
    updated_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (variant_id, warehouse_id)
);

-- Inventory movements/transactions
CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    
    -- Movement details
    type TEXT NOT NULL CHECK (
        type IN ('purchase', 'sale', 'adjustment', 'transfer_in', 'transfer_out', 'return')
    ),
    quantity_change INTEGER NOT NULL, -- Positive or negative
    
    -- Reference to source transaction
    reference_type TEXT, -- 'order', 'purchase_order', 'adjustment'
    reference_id UUID,
    
    -- Cost information
    unit_cost_cents price_cents,
    total_cost_cents price_cents,
    
    -- Metadata
    notes TEXT,
    performed_by UUID,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure non-zero movements
    CHECK (quantity_change != 0)
);

-- Function to update inventory levels
CREATE OR REPLACE FUNCTION update_inventory_level()
RETURNS TRIGGER AS $$
BEGIN
    -- Update the inventory level based on movement
    INSERT INTO inventory_levels (variant_id, warehouse_id, quantity_available)
    VALUES (NEW.variant_id, NEW.warehouse_id, 
            CASE 
                WHEN NEW.type IN ('purchase', 'adjustment', 'transfer_in', 'return') THEN NEW.quantity_change
                ELSE 0
            END)
    ON CONFLICT (variant_id, warehouse_id)
    DO UPDATE SET
        quantity_available = inventory_levels.quantity_available + 
            CASE 
                WHEN NEW.type IN ('purchase', 'adjustment', 'transfer_in', 'return') THEN NEW.quantity_change
                WHEN NEW.type IN ('sale', 'transfer_out') THEN -ABS(NEW.quantity_change)
                ELSE 0
            END,
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER inventory_movement_trigger
    AFTER INSERT ON inventory_movements
    FOR EACH ROW EXECUTE FUNCTION update_inventory_level();

-- Indexes for inventory
CREATE INDEX idx_inventory_levels_variant ON inventory_levels (variant_id);
CREATE INDEX idx_inventory_levels_warehouse ON inventory_levels (warehouse_id);
CREATE INDEX idx_inventory_movements_variant ON inventory_movements (variant_id, created_at);
CREATE INDEX idx_inventory_movements_reference ON inventory_movements (reference_type, reference_id);
```

### Shopping Cart & Orders
```sql
-- Shopping carts (guest and registered users)
CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    
    -- For guest checkout
    session_id TEXT,
    
    -- Cart metadata
    currency_code currency_code NOT NULL DEFAULT 'USD',
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Either user_id or session_id must be present
    CHECK (user_id IS NOT NULL OR session_id IS NOT NULL)
);

-- Cart items
CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    
    -- Capture price at time of adding to cart
    unit_price_cents price_cents NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Unique variant per cart
    UNIQUE (cart_id, variant_id)
);

-- Orders
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number TEXT NOT NULL UNIQUE, -- Human-readable order number
    user_id UUID REFERENCES users(id),
    
    -- Order status with proper state machine
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'confirmed', 'processing', 'shipped', 'delivered', 
                  'cancelled', 'refunded', 'partially_refunded')
    ),
    
    -- Financial information
    currency_code currency_code NOT NULL DEFAULT 'USD',
    subtotal_cents price_cents NOT NULL,
    tax_cents price_cents NOT NULL DEFAULT 0,
    shipping_cents price_cents NOT NULL DEFAULT 0,
    discount_cents price_cents NOT NULL DEFAULT 0,
    total_cents price_cents NOT NULL,
    
    -- Customer information (snapshot at time of order)
    customer_email email NOT NULL,
    customer_phone phone_number,
    
    -- Addresses (stored as JSONB for historical accuracy)
    billing_address JSONB NOT NULL,
    shipping_address JSONB NOT NULL,
    
    -- Shipping information
    shipping_method TEXT,
    tracking_number TEXT,
    carrier TEXT,
    
    -- Important timestamps
    confirmed_at TIMESTAMP,
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    
    -- Metadata
    notes TEXT,
    tags TEXT[],
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Business rules
    CONSTRAINT valid_total CHECK (
        total_cents = subtotal_cents + tax_cents + shipping_cents - discount_cents
    ),
    CONSTRAINT positive_subtotal CHECK (subtotal_cents > 0)
);

-- Order items (snapshot of product at time of order)
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Product information (snapshot)
    product_id UUID NOT NULL, -- Reference for reporting, don't FK to allow product deletion
    variant_id UUID NOT NULL, -- Reference for reporting
    sku product_sku NOT NULL,
    name TEXT NOT NULL,
    variant_title TEXT,
    
    -- Pricing and quantity
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents price_cents NOT NULL,
    total_price_cents price_cents NOT NULL,
    
    -- Physical properties for shipping
    weight_grams weight_grams,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure total is correct
    CONSTRAINT valid_item_total CHECK (total_price_cents = quantity * unit_price_cents)
);

-- Order status history for audit trail
CREATE TABLE order_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    from_status TEXT,
    to_status TEXT NOT NULL,
    
    -- Context
    changed_by UUID,
    reason TEXT,
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Function to generate order numbers
CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS TEXT AS $$
DECLARE
    new_number TEXT;
    counter INTEGER;
BEGIN
    -- Generate order number like ORD-2025001234
    SELECT COALESCE(MAX(
        CAST(SUBSTRING(order_number FROM 9) AS INTEGER)
    ), 0) + 1 INTO counter
    FROM orders 
    WHERE order_number LIKE 'ORD-' || EXTRACT(YEAR FROM NOW()) || '%';
    
    new_number := 'ORD-' || EXTRACT(YEAR FROM NOW()) || LPAD(counter::TEXT, 6, '0');
    
    RETURN new_number;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-generate order numbers
CREATE OR REPLACE FUNCTION set_order_number()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.order_number IS NULL THEN
        NEW.order_number := generate_order_number();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER order_number_trigger
    BEFORE INSERT ON orders
    FOR EACH ROW EXECUTE FUNCTION set_order_number();

-- Indexes for orders
CREATE INDEX idx_orders_user ON orders (user_id);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_created ON orders (created_at DESC);
CREATE INDEX idx_orders_email ON orders (customer_email);
CREATE INDEX idx_order_items_order ON order_items (order_id);
```

### Payment Processing
```sql
-- Payment methods
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Payment method type
    type TEXT NOT NULL CHECK (type IN ('credit_card', 'debit_card', 'paypal', 'bank_transfer')),
    
    -- For cards - store safely tokenized data only
    card_brand TEXT, -- 'visa', 'mastercard', etc.
    card_last_four CHAR(4),
    card_exp_month INTEGER CHECK (card_exp_month BETWEEN 1 AND 12),
    card_exp_year INTEGER CHECK (card_exp_year >= EXTRACT(YEAR FROM NOW())),
    
    -- External payment processor data
    processor TEXT NOT NULL, -- 'stripe', 'paypal', etc.
    processor_payment_method_id TEXT NOT NULL,
    
    -- Metadata
    is_default BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Only one default payment method per user
    UNIQUE (user_id, is_default) WHERE is_default = TRUE
);

-- Payment transactions
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    
    -- Payment details
    amount_cents price_cents NOT NULL,
    currency_code currency_code NOT NULL,
    
    -- Payment method info
    payment_method_id UUID REFERENCES payment_methods(id),
    payment_method_type TEXT NOT NULL,
    
    -- External processor information
    processor TEXT NOT NULL,
    processor_transaction_id TEXT NOT NULL,
    processor_payment_intent_id TEXT,
    
    -- Status tracking
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'processing', 'succeeded', 'failed', 'cancelled', 'refunded')
    ),
    
    -- Failure information
    failure_code TEXT,
    failure_message TEXT,
    
    -- Important timestamps
    processed_at TIMESTAMP,
    
    -- Metadata
    metadata JSONB,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Refunds
CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    
    -- Refund details
    amount_cents price_cents NOT NULL,
    reason TEXT,
    
    -- Processing info
    processor_refund_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'succeeded', 'failed', 'cancelled')
    ),
    
    -- Who issued the refund
    issued_by UUID,
    
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Can't refund more than original payment
    CONSTRAINT valid_refund_amount CHECK (
        amount_cents <= (SELECT amount_cents FROM payments WHERE id = payment_id)
    )
);

-- Payment indexes
CREATE INDEX idx_payments_order ON payments (order_id);
CREATE INDEX idx_payments_status ON payments (status);
CREATE INDEX idx_payments_processor ON payments (processor, processor_transaction_id);
CREATE INDEX idx_refunds_payment ON refunds (payment_id);
```

### Reviews & Ratings
```sql
-- Product reviews
CREATE TABLE product_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    user_id UUID NOT NULL REFERENCES users(id),
    order_id UUID REFERENCES orders(id), -- Link to purchase
    
    -- Review content
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title TEXT,
    content TEXT,
    
    -- Moderation
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'approved', 'rejected', 'flagged')
    ),
    moderated_by UUID,
    moderated_at TIMESTAMP,
    moderation_reason TEXT,
    
    -- Helpfulness tracking
    helpful_count INTEGER DEFAULT 0,
    total_votes INTEGER DEFAULT 0,
    
    -- Media attachments
    images JSONB, -- Array of image URLs
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- One review per product per user
    UNIQUE (product_id, user_id)
);

-- Review helpfulness votes
CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES product_reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    is_helpful BOOLEAN NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (review_id, user_id)
);

-- Function to update review helpfulness counts
CREATE OR REPLACE FUNCTION update_review_helpfulness()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE product_reviews SET
        helpful_count = (
            SELECT COUNT(*) FROM review_votes 
            WHERE review_id = COALESCE(NEW.review_id, OLD.review_id) 
            AND is_helpful = TRUE
        ),
        total_votes = (
            SELECT COUNT(*) FROM review_votes 
            WHERE review_id = COALESCE(NEW.review_id, OLD.review_id)
        )
    WHERE id = COALESCE(NEW.review_id, OLD.review_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER review_vote_trigger
    AFTER INSERT OR UPDATE OR DELETE ON review_votes
    FOR EACH ROW EXECUTE FUNCTION update_review_helpfulness();

-- Indexes for reviews
CREATE INDEX idx_reviews_product ON product_reviews (product_id, status);
CREATE INDEX idx_reviews_user ON product_reviews (user_id);
CREATE INDEX idx_reviews_rating ON product_reviews (rating) WHERE status = 'approved';
```

### Promotions & Discounts
```sql
-- Discount codes/coupons
CREATE TABLE discount_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    
    -- Discount details
    type TEXT NOT NULL CHECK (type IN ('percentage', 'fixed_amount', 'free_shipping')),
    value_cents price_cents, -- NULL for free shipping
    percentage percentage, -- NULL for fixed amount
    
    -- Usage limits
    usage_limit INTEGER, -- NULL for unlimited
    usage_count INTEGER DEFAULT 0,
    usage_limit_per_customer INTEGER DEFAULT 1,
    
    -- Validity period
    starts_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    -- Conditions
    minimum_order_cents price_cents DEFAULT 0,
    applies_to TEXT CHECK (applies_to IN ('all', 'category', 'product')),
    applies_to_ids UUID[], -- Category or product IDs
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Metadata
    description TEXT,
    created_by UUID,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Validation
    CHECK (
        (type = 'percentage' AND percentage IS NOT NULL AND value_cents IS NULL) OR
        (type = 'fixed_amount' AND value_cents IS NOT NULL AND percentage IS NULL) OR
        (type = 'free_shipping' AND value_cents IS NULL AND percentage IS NULL)
    )
);

-- Track discount code usage
CREATE TABLE discount_code_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discount_code_id UUID NOT NULL REFERENCES discount_codes(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    user_id UUID REFERENCES users(id),
    
    amount_discounted_cents price_cents NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Prevent duplicate usage per order
    UNIQUE (discount_code_id, order_id)
);

-- Update usage count trigger
CREATE OR REPLACE FUNCTION update_discount_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE discount_codes 
    SET usage_count = usage_count + 1
    WHERE id = NEW.discount_code_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER discount_usage_trigger
    AFTER INSERT ON discount_code_usage
    FOR EACH ROW EXECUTE FUNCTION update_discount_usage();
```

## 🔍 Key Views for Application Usage

### Product Catalog Views
```sql
-- Products with computed rating and inventory
CREATE VIEW products_catalog AS
SELECT 
    p.*,
    b.name as brand_name,
    c.name as category_name,
    
    -- Review statistics
    COALESCE(r.avg_rating, 0) as average_rating,
    COALESCE(r.review_count, 0) as review_count,
    
    -- Inventory information
    COALESCE(i.total_available, 0) as total_inventory,
    (COALESCE(i.total_available, 0) > 0) as in_stock,
    
    -- Pricing
    CASE 
        WHEN p.compare_at_price_cents IS NOT NULL 
        THEN ROUND((p.compare_at_price_cents - p.base_price_cents) * 100.0 / p.compare_at_price_cents, 2)
        ELSE 0 
    END as discount_percentage

FROM products p
LEFT JOIN brands b ON b.id = p.brand_id
LEFT JOIN categories c ON c.id = p.category_id
LEFT JOIN (
    SELECT 
        product_id,
        ROUND(AVG(rating)::numeric, 2) as avg_rating,
        COUNT(*) as review_count
    FROM product_reviews 
    WHERE status = 'approved'
    GROUP BY product_id
) r ON r.product_id = p.id
LEFT JOIN (
    SELECT 
        pv.product_id,
        SUM(il.quantity_available) as total_available
    FROM product_variants pv
    JOIN inventory_levels il ON il.variant_id = pv.id
    WHERE pv.is_active = TRUE
    GROUP BY pv.product_id
) i ON i.product_id = p.id

WHERE p.status = 'active';
```

### Order Management Views
```sql
-- Orders with customer and status information
CREATE VIEW orders_management AS
SELECT 
    o.*,
    up.first_name || ' ' || up.last_name as customer_name,
    
    -- Order summary
    (SELECT COUNT(*) FROM order_items WHERE order_id = o.id) as item_count,
    
    -- Payment status
    p.status as payment_status,
    p.processed_at as payment_processed_at,
    
    -- Shipping timeline
    CASE 
        WHEN o.delivered_at IS NOT NULL THEN 'delivered'
        WHEN o.shipped_at IS NOT NULL THEN 'shipped'  
        WHEN o.confirmed_at IS NOT NULL THEN 'confirmed'
        ELSE 'pending'
    END as fulfillment_status

FROM orders o
LEFT JOIN users u ON u.id = o.user_id
LEFT JOIN user_profiles up ON up.user_id = u.id
LEFT JOIN payments p ON p.order_id = o.id AND p.status = 'succeeded'
ORDER BY o.created_at DESC;
```

## 📊 Analytics & Reporting

### Sales Analytics
```sql
-- Daily sales summary
CREATE VIEW daily_sales AS
SELECT 
    DATE(created_at) as sale_date,
    COUNT(*) as orders_count,
    SUM(total_cents) as total_revenue_cents,
    AVG(total_cents) as avg_order_value_cents,
    COUNT(DISTINCT user_id) as unique_customers
FROM orders 
WHERE status NOT IN ('cancelled', 'refunded')
GROUP BY DATE(created_at)
ORDER BY sale_date DESC;

-- Product performance
CREATE VIEW product_performance AS
SELECT 
    p.id,
    p.name,
    p.sku,
    c.name as category_name,
    
    -- Sales metrics
    COALESCE(s.total_sold, 0) as units_sold,
    COALESCE(s.revenue_cents, 0) as total_revenue_cents,
    COALESCE(s.orders_count, 0) as orders_count,
    
    -- Inventory metrics
    COALESCE(i.total_available, 0) as current_stock,
    
    -- Performance ratios
    CASE 
        WHEN COALESCE(i.total_available, 0) + COALESCE(s.total_sold, 0) > 0
        THEN ROUND(COALESCE(s.total_sold, 0) * 100.0 / (COALESCE(i.total_available, 0) + COALESCE(s.total_sold, 0)), 2)
        ELSE 0
    END as sell_through_rate

FROM products p
LEFT JOIN categories c ON c.id = p.category_id
LEFT JOIN (
    SELECT 
        oi.product_id,
        SUM(oi.quantity) as total_sold,
        SUM(oi.total_price_cents) as revenue_cents,
        COUNT(DISTINCT oi.order_id) as orders_count
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE o.status NOT IN ('cancelled', 'refunded')
    GROUP BY oi.product_id
) s ON s.product_id = p.id
LEFT JOIN (
    SELECT 
        pv.product_id,
        SUM(il.quantity_available) as total_available
    FROM product_variants pv
    JOIN inventory_levels il ON il.variant_id = pv.id
    GROUP BY pv.product_id
) i ON i.product_id = p.id

WHERE p.status = 'active'
ORDER BY COALESCE(s.revenue_cents, 0) DESC;
```

## 🚀 Performance Optimizations

### Essential Indexes
```sql
-- Product search and filtering
CREATE INDEX CONCURRENTLY idx_products_search_combo ON products (category_id, status, base_price_cents);
CREATE INDEX CONCURRENTLY idx_products_brand_status ON products (brand_id, status);

-- Order processing
CREATE INDEX CONCURRENTLY idx_orders_user_status ON orders (user_id, status);
CREATE INDEX CONCURRENTLY idx_orders_timeline ON orders (created_at DESC, status);

-- Inventory management
CREATE INDEX CONCURRENTLY idx_inventory_low_stock ON inventory_levels (quantity_available) 
WHERE quantity_available <= reorder_point;

-- Analytics queries
CREATE INDEX CONCURRENTLY idx_order_items_product_date ON order_items (product_id, created_at);
CREATE INDEX CONCURRENTLY idx_payments_date_status ON payments (created_at, status);
```

### Partitioning Strategy
```sql
-- Partition large tables by date for better performance
-- This would typically be done for tables like order_status_history, inventory_movements, etc.

-- Example: Partition inventory movements by month
CREATE TABLE inventory_movements_2025_01 PARTITION OF inventory_movements
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE inventory_movements_2025_02 PARTITION OF inventory_movements  
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
```

## 🔒 Security Considerations

### Row-Level Security
```sql
-- Enable RLS for multi-tenant data
ALTER TABLE user_profiles ENABLE ROW LEVEL SECURITY;

-- Users can only see their own profile
CREATE POLICY user_profiles_self_policy ON user_profiles
FOR ALL TO application_user
USING (user_id = current_setting('app.current_user_id')::UUID);

-- Orders visibility policy
CREATE POLICY orders_customer_policy ON orders
FOR SELECT TO application_user
USING (
    user_id = current_setting('app.current_user_id')::UUID OR
    current_setting('app.user_role') = 'admin'
);
```

### Data Encryption
```sql
-- Encrypt sensitive data at rest
-- Credit card data should be tokenized, not stored directly
-- Use application-level encryption for PII data

-- Example: Encrypt customer email for analytics while preserving searchability
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Hash email for analytics (one-way)
ALTER TABLE users ADD COLUMN email_hash TEXT 
GENERATED ALWAYS AS (encode(digest(email, 'sha256'), 'hex')) STORED;

-- Create index on hash for analytics queries
CREATE INDEX idx_users_email_hash ON users (email_hash);
```

## 📈 Scaling Considerations

### Read Replicas
```sql
-- Separate read-heavy queries to read replicas
-- Use connection pooling and query routing

-- Example: Analytics queries go to read replica
-- SELECT /* read_replica */ * FROM product_performance;
```

### Caching Strategy
```sql
-- Materialized views for expensive aggregations
CREATE MATERIALIZED VIEW category_stats AS
SELECT 
    c.id,
    c.name,
    COUNT(p.id) as product_count,
    AVG(p.base_price_cents) as avg_price_cents
FROM categories c
LEFT JOIN products p ON p.category_id = c.id AND p.status = 'active'
GROUP BY c.id, c.name;

-- Refresh strategy
CREATE OR REPLACE FUNCTION refresh_category_stats()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY category_stats;
END;
$$ LANGUAGE plpgsql;

-- Schedule refresh every hour
-- SELECT cron.schedule('refresh-category-stats', '0 * * * *', 'SELECT refresh_category_stats();');
```

This e-commerce schema provides a solid foundation for a production-ready online store with proper normalization, constraints, indexes, and scaling considerations. It handles complex scenarios like inventory management, order processing, and payment flows while maintaining data integrity and performance.
