# Currency Storage: Complete Guide

Storing monetary values correctly is critical for financial applications. This guide covers best practices for currency storage, arithmetic operations, and handling multi-currency scenarios.

## Table of Contents
- [Storage Strategies](#storage-strategies)
- [Currency Arithmetic](#currency-arithmetic)
- [Multi-Currency Support](#multi-currency-support)
- [Rounding and Precision](#rounding-and-precision)
- [Distribution and Splitting](#distribution-and-splitting)
- [Common Pitfalls](#common-pitfalls)
- [Best Practices](#best-practices)

## Storage Strategies

### 1. Integer Storage (Recommended)

Store currency in the smallest unit (cents, pence, etc.) as integers to avoid floating-point precision issues.

```sql
-- Store amounts in cents as INTEGER
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    amount_cents INTEGER NOT NULL, -- $123.45 stored as 12345
    currency_code CHAR(3) NOT NULL DEFAULT 'USD',
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure positive amounts for certain transaction types
    CONSTRAINT positive_amount CHECK (amount_cents >= 0)
);

-- Examples of storage
INSERT INTO transactions (amount_cents, currency_code, description) VALUES
(12345, 'USD', 'Product purchase: $123.45'),
(9999, 'EUR', 'Service fee: €99.99'),
(500, 'USD', 'Tip: $5.00');
```

**Advantages:**
- No floating-point precision errors
- Exact arithmetic operations
- Consistent with payment processors (Stripe, PayPal)
- Fast integer operations

**Considerations:**
- Application must handle conversion to/from display format
- Clear documentation needed for API consumers

### 2. DECIMAL Storage (Alternative)

Use DECIMAL type when you need to store the actual currency value directly.

```sql
-- Store amounts as DECIMAL with appropriate precision
CREATE TABLE product_prices (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    price DECIMAL(15,4) NOT NULL, -- 15 digits total, 4 decimal places
    currency_code CHAR(3) NOT NULL DEFAULT 'USD',
    effective_date DATE NOT NULL,
    
    -- Ensure reasonable price ranges
    CONSTRAINT reasonable_price CHECK (price >= 0 AND price <= 999999999.9999)
);

-- Examples
INSERT INTO product_prices (product_id, price, currency_code) VALUES
(1, 123.4500, 'USD'), -- $123.45
(2, 99.9900, 'EUR'),  -- €99.99
(3, 0.0100, 'USD');   -- $0.01
```

**When to use DECIMAL:**
- Direct currency value storage needed
- Integration with systems expecting decimal values
- Regulatory requirements for decimal representation
- Complex calculations requiring decimal precision

### 3. Currency-Specific Considerations

```sql
-- Handle currencies with different decimal places
CREATE TABLE currency_definitions (
    code CHAR(3) PRIMARY KEY,
    name TEXT NOT NULL,
    decimal_places SMALLINT NOT NULL,
    smallest_unit_name TEXT,
    examples TEXT[],
    
    CONSTRAINT valid_decimal_places CHECK (decimal_places >= 0 AND decimal_places <= 4)
);

INSERT INTO currency_definitions VALUES
('USD', 'US Dollar', 2, 'cent', '{"$1.23", "$0.01"}'),
('JPY', 'Japanese Yen', 0, 'yen', '{"¥123", "¥1"}'),
('BHD', 'Bahraini Dinar', 3, 'fils', '{"BD 1.234", "BD 0.001"}'),
('CLF', 'Chilean Unit of Account', 4, 'clf', '{"CLF 1.2345"}'),
('IDR', 'Indonesian Rupiah', 0, 'rupiah', '{"Rp 12345"}'');

-- Flexible amount storage
CREATE TABLE flexible_amounts (
    id SERIAL PRIMARY KEY,
    amount_minor_units BIGINT NOT NULL, -- Amount in smallest currency unit
    currency_code CHAR(3) REFERENCES currency_definitions(code),
    
    -- Computed column for display amount
    amount_display AS (
        amount_minor_units::DECIMAL / (10 ^ (
            SELECT decimal_places FROM currency_definitions 
            WHERE code = currency_code
        ))
    ) STORED
);
```

## Currency Arithmetic

### Safe Addition and Subtraction

```sql
-- Account balance operations with integer amounts
CREATE TABLE account_balances (
    account_id INTEGER PRIMARY KEY,
    balance_cents INTEGER NOT NULL DEFAULT 0,
    currency_code CHAR(3) NOT NULL DEFAULT 'USD',
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT non_negative_balance CHECK (balance_cents >= 0)
);

-- Safe balance updates with explicit locking
CREATE OR REPLACE FUNCTION update_account_balance(
    p_account_id INTEGER,
    p_amount_cents INTEGER,
    p_description TEXT
) RETURNS INTEGER AS $$
DECLARE
    v_new_balance INTEGER;
BEGIN
    -- Lock the account row to prevent concurrent modifications
    UPDATE account_balances 
    SET balance_cents = balance_cents + p_amount_cents,
        last_updated = NOW()
    WHERE account_id = p_account_id
    RETURNING balance_cents INTO v_new_balance;
    
    -- Log the transaction
    INSERT INTO balance_transactions (
        account_id, amount_cents, description, balance_after
    ) VALUES (
        p_account_id, p_amount_cents, p_description, v_new_balance
    );
    
    RETURN v_new_balance;
END;
$$ LANGUAGE plpgsql;
```

### Percentage Calculations

```sql
-- Calculate percentages safely with proper rounding
CREATE OR REPLACE FUNCTION calculate_percentage(
    amount_cents INTEGER,
    percentage_rate DECIMAL(5,4), -- e.g., 0.0750 for 7.5%
    rounding_mode TEXT DEFAULT 'round' -- 'round', 'floor', 'ceil'
) RETURNS INTEGER AS $$
DECLARE
    calculated_amount DECIMAL;
    result_cents INTEGER;
BEGIN
    calculated_amount := amount_cents * percentage_rate;
    
    CASE rounding_mode
        WHEN 'floor' THEN result_cents := FLOOR(calculated_amount);
        WHEN 'ceil' THEN result_cents := CEIL(calculated_amount);
        ELSE result_cents := ROUND(calculated_amount);
    END CASE;
    
    RETURN result_cents;
END;
$$ LANGUAGE plpgsql;

-- Usage examples
SELECT 
    calculate_percentage(10000, 0.0750, 'round') as tax_amount, -- $100 * 7.5% = $7.50 (750 cents)
    calculate_percentage(10033, 0.0750, 'round') as tax_rounded, -- $100.33 * 7.5% = $7.52 (752 cents)
    calculate_percentage(10033, 0.0750, 'floor') as tax_floor;   -- $100.33 * 7.5% = $7.52 (752 cents)
```

## Multi-Currency Support

### Currency Conversion

```sql
-- Exchange rates table
CREATE TABLE exchange_rates (
    id SERIAL PRIMARY KEY,
    from_currency CHAR(3) NOT NULL,
    to_currency CHAR(3) NOT NULL,
    rate DECIMAL(20,10) NOT NULL,
    effective_date DATE NOT NULL,
    expires_date DATE,
    source TEXT, -- e.g., 'ECB', 'Fed', 'manual'
    
    UNIQUE(from_currency, to_currency, effective_date),
    CONSTRAINT positive_rate CHECK (rate > 0),
    CONSTRAINT different_currencies CHECK (from_currency != to_currency)
);

-- Get current exchange rate
CREATE OR REPLACE FUNCTION get_exchange_rate(
    p_from_currency CHAR(3),
    p_to_currency CHAR(3),
    p_date DATE DEFAULT CURRENT_DATE
) RETURNS DECIMAL(20,10) AS $$
DECLARE
    v_rate DECIMAL(20,10);
BEGIN
    -- Direct rate
    SELECT rate INTO v_rate
    FROM exchange_rates
    WHERE from_currency = p_from_currency
      AND to_currency = p_to_currency
      AND effective_date <= p_date
      AND (expires_date IS NULL OR expires_date > p_date)
    ORDER BY effective_date DESC
    LIMIT 1;
    
    -- If no direct rate, try inverse
    IF v_rate IS NULL THEN
        SELECT 1.0 / rate INTO v_rate
        FROM exchange_rates
        WHERE from_currency = p_to_currency
          AND to_currency = p_from_currency
          AND effective_date <= p_date
          AND (expires_date IS NULL OR expires_date > p_date)
        ORDER BY effective_date DESC
        LIMIT 1;
    END IF;
    
    RETURN v_rate;
END;
$$ LANGUAGE plpgsql;

-- Convert currency amounts
CREATE OR REPLACE FUNCTION convert_currency(
    p_amount_cents INTEGER,
    p_from_currency CHAR(3),
    p_to_currency CHAR(3),
    p_date DATE DEFAULT CURRENT_DATE
) RETURNS INTEGER AS $$
DECLARE
    v_rate DECIMAL(20,10);
    v_converted_amount DECIMAL;
    v_from_decimals INTEGER;
    v_to_decimals INTEGER;
BEGIN
    -- Return original amount if same currency
    IF p_from_currency = p_to_currency THEN
        RETURN p_amount_cents;
    END IF;
    
    -- Get exchange rate
    v_rate := get_exchange_rate(p_from_currency, p_to_currency, p_date);
    IF v_rate IS NULL THEN
        RAISE EXCEPTION 'No exchange rate found for % to % on %', 
            p_from_currency, p_to_currency, p_date;
    END IF;
    
    -- Get decimal places for each currency
    SELECT decimal_places INTO v_from_decimals
    FROM currency_definitions WHERE code = p_from_currency;
    
    SELECT decimal_places INTO v_to_decimals
    FROM currency_definitions WHERE code = p_to_currency;
    
    -- Convert: amount_in_major_units * rate * to_currency_multiplier
    v_converted_amount := (p_amount_cents::DECIMAL / (10 ^ v_from_decimals)) 
                         * v_rate 
                         * (10 ^ v_to_decimals);
    
    RETURN ROUND(v_converted_amount);
END;
$$ LANGUAGE plpgsql;
```

## Rounding and Precision

### Banker's Rounding Implementation

```sql
-- Banker's rounding (round half to even) for fair distribution
CREATE OR REPLACE FUNCTION bankers_round(value DECIMAL) RETURNS INTEGER AS $$
DECLARE
    truncated INTEGER;
    fractional DECIMAL;
BEGIN
    truncated := FLOOR(value);
    fractional := value - truncated;
    
    -- If fractional part is exactly 0.5, round to even
    IF ABS(fractional - 0.5) < 0.0000001 THEN
        RETURN CASE WHEN truncated % 2 = 0 THEN truncated ELSE truncated + 1 END;
    ELSE
        RETURN ROUND(value);
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Example usage
SELECT 
    bankers_round(2.5) as round_25,  -- 2 (even)
    bankers_round(3.5) as round_35,  -- 4 (even)
    bankers_round(4.5) as round_45,  -- 4 (even)
    bankers_round(5.5) as round_55;  -- 6 (even)
```

## Distribution and Splitting

### Fair Distribution Algorithm

```sql
-- Distribute amount fairly among recipients with minimal remainder
CREATE OR REPLACE FUNCTION distribute_amount(
    total_amount_cents INTEGER,
    recipient_count INTEGER,
    distribution_weights INTEGER[] DEFAULT NULL -- Optional weights
) RETURNS INTEGER[] AS $$
DECLARE
    base_amount INTEGER;
    remainder INTEGER;
    result INTEGER[];
    i INTEGER;
    weight_sum INTEGER;
    weighted_amount DECIMAL;
    allocated_total INTEGER := 0;
BEGIN
    IF recipient_count <= 0 THEN
        RAISE EXCEPTION 'Recipient count must be positive';
    END IF;
    
    -- Initialize result array
    result := array_fill(0, ARRAY[recipient_count]);
    
    -- Simple equal distribution if no weights provided
    IF distribution_weights IS NULL THEN
        base_amount := total_amount_cents / recipient_count;
        remainder := total_amount_cents % recipient_count;
        
        -- Give base amount to everyone
        FOR i IN 1..recipient_count LOOP
            result[i] := base_amount;
        END LOOP;
        
        -- Distribute remainder to first recipients
        FOR i IN 1..remainder LOOP
            result[i] := result[i] + 1;
        END LOOP;
    ELSE
        -- Weighted distribution
        IF array_length(distribution_weights, 1) != recipient_count THEN
            RAISE EXCEPTION 'Weights array length must match recipient count';
        END IF;
        
        weight_sum := (SELECT SUM(w) FROM unnest(distribution_weights) w);
        
        -- Calculate weighted amounts
        FOR i IN 1..recipient_count LOOP
            weighted_amount := (total_amount_cents::DECIMAL * distribution_weights[i]) / weight_sum;
            result[i] := ROUND(weighted_amount);
            allocated_total := allocated_total + result[i];
        END LOOP;
        
        -- Adjust for rounding differences (give/take from largest allocation)
        IF allocated_total != total_amount_cents THEN
            -- Find index of maximum allocation
            SELECT array_position(result, max_val) INTO i
            FROM (SELECT MAX(val) as max_val FROM unnest(result) val) t;
            
            result[i] := result[i] + (total_amount_cents - allocated_total);
        END IF;
    END IF;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Example: Distribute $100.00 among 3 people
SELECT distribute_amount(10000, 3); -- Returns {3334, 3333, 3333}

-- Example: Weighted distribution (2:1:1 ratio)
SELECT distribute_amount(10000, 3, ARRAY[2, 1, 1]); -- Returns {5000, 2500, 2500}
```

### Practical Distribution Example

```sql
-- Split restaurant bill with tips
CREATE TABLE bill_splits (
    id SERIAL PRIMARY KEY,
    bill_amount_cents INTEGER NOT NULL,
    tip_percentage DECIMAL(5,4) NOT NULL,
    participant_count INTEGER NOT NULL,
    splits INTEGER[] NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION split_restaurant_bill(
    bill_amount_cents INTEGER,
    tip_percentage DECIMAL(5,4),
    participant_count INTEGER
) RETURNS TABLE(
    participant INTEGER,
    bill_portion_cents INTEGER,
    tip_portion_cents INTEGER,
    total_portion_cents INTEGER
) AS $$
DECLARE
    tip_amount_cents INTEGER;
    bill_splits INTEGER[];
    tip_splits INTEGER[];
    i INTEGER;
BEGIN
    -- Calculate tip
    tip_amount_cents := ROUND(bill_amount_cents * tip_percentage);
    
    -- Distribute bill and tip separately for fairness
    bill_splits := distribute_amount(bill_amount_cents, participant_count);
    tip_splits := distribute_amount(tip_amount_cents, participant_count);
    
    -- Return results
    FOR i IN 1..participant_count LOOP
        participant := i;
        bill_portion_cents := bill_splits[i];
        tip_portion_cents := tip_splits[i];
        total_portion_cents := bill_splits[i] + tip_splits[i];
        RETURN NEXT;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Example: $127.50 bill with 18% tip, split 4 ways
SELECT * FROM split_restaurant_bill(12750, 0.18, 4);
```

## Common Pitfalls

### 1. Floating Point Arithmetic

```sql
-- ❌ DON'T: Use floating point for currency
CREATE TABLE bad_prices (
    price REAL -- Precision errors guaranteed!
);

-- Example of precision loss
SELECT 
    0.1::REAL + 0.2::REAL = 0.3::REAL as should_be_true_but_false,
    0.1::REAL + 0.2::REAL as actual_result; -- 0.30000001
```

### 2. Currency Mixing Without Validation

```sql
-- ❌ DON'T: Mix currencies without explicit conversion
CREATE TABLE bad_transactions (
    amount_cents INTEGER,
    -- Missing currency code - dangerous!
);

-- ✅ DO: Always specify currency and validate
CREATE TABLE good_transactions (
    amount_cents INTEGER NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    CONSTRAINT valid_currency 
    CHECK (currency_code IN ('USD', 'EUR', 'GBP', 'JPY', 'etc'))
);
```

### 3. Inconsistent Rounding

```sql
-- ❌ DON'T: Inconsistent rounding in calculations
CREATE OR REPLACE FUNCTION bad_calculate_tax(amount_cents INTEGER)
RETURNS INTEGER AS $$
BEGIN
    -- Sometimes ROUND, sometimes FLOOR - inconsistent!
    IF amount_cents > 10000 THEN
        RETURN FLOOR(amount_cents * 0.075);
    ELSE
        RETURN ROUND(amount_cents * 0.075);
    END IF;
END;
$$ LANGUAGE plpgsql;

-- ✅ DO: Consistent rounding strategy
CREATE OR REPLACE FUNCTION good_calculate_tax(
    amount_cents INTEGER,
    rate DECIMAL(5,4),
    rounding_strategy TEXT DEFAULT 'bankers'
) RETURNS INTEGER AS $$
BEGIN
    CASE rounding_strategy
        WHEN 'bankers' THEN RETURN bankers_round(amount_cents * rate);
        WHEN 'up' THEN RETURN CEIL(amount_cents * rate);
        WHEN 'down' THEN RETURN FLOOR(amount_cents * rate);
        ELSE RETURN ROUND(amount_cents * rate);
    END CASE;
END;
$$ LANGUAGE plpgsql;
```

## Best Practices

### 1. Always Store Currency Code

```sql
-- Every monetary amount should have an associated currency
CREATE TABLE monetary_amounts (
    id SERIAL PRIMARY KEY,
    amount_cents INTEGER NOT NULL,
    currency_code CHAR(3) NOT NULL,
    
    -- Enforce valid ISO currency codes
    CONSTRAINT valid_currency_code 
    CHECK (currency_code ~ '^[A-Z]{3}$')
);
```

### 2. Use Appropriate Data Types

```sql
-- For high-precision requirements or very large amounts
CREATE TABLE high_precision_amounts (
    id SERIAL PRIMARY KEY,
    amount_minor_units BIGINT NOT NULL, -- Use BIGINT for large amounts
    currency_code CHAR(3) NOT NULL,
    precision_factor INTEGER NOT NULL DEFAULT 100 -- Track precision used
);
```

### 3. Document Rounding Strategies

```sql
-- Document rounding decisions in schema comments
COMMENT ON FUNCTION calculate_percentage IS 
'Calculates percentage amounts using specified rounding strategy.
Default strategy is standard rounding (0.5 rounds up).
Use banker''s rounding for fair distribution scenarios.';

COMMENT ON TABLE exchange_rates IS
'Exchange rates are stored with 10 decimal places for maximum precision.
Rates are applied using banker''s rounding to minimize bias in conversions.';
```

### 4. Validate Currency Operations

```sql
-- Prevent mixing currencies in calculations
CREATE OR REPLACE FUNCTION validate_same_currency(
    currency1 CHAR(3),
    currency2 CHAR(3),
    operation_name TEXT
) RETURNS VOID AS $$
BEGIN
    IF currency1 != currency2 THEN
        RAISE EXCEPTION 'Cannot perform % with different currencies: % and %',
            operation_name, currency1, currency2;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Use in operations
CREATE OR REPLACE FUNCTION add_amounts(
    amount1_cents INTEGER, currency1 CHAR(3),
    amount2_cents INTEGER, currency2 CHAR(3)
) RETURNS INTEGER AS $$
BEGIN
    PERFORM validate_same_currency(currency1, currency2, 'addition');
    RETURN amount1_cents + amount2_cents;
END;
$$ LANGUAGE plpgsql;
```

### 5. Audit Currency Calculations

```sql
-- Log all currency calculations for audit trails
CREATE TABLE currency_calculation_audit (
    id SERIAL PRIMARY KEY,
    operation_type TEXT NOT NULL,
    input_amounts JSONB NOT NULL,
    output_amount_cents INTEGER NOT NULL,
    output_currency CHAR(3) NOT NULL,
    calculation_method TEXT,
    performed_by TEXT,
    performed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Example audit logging
INSERT INTO currency_calculation_audit (
    operation_type, input_amounts, output_amount_cents, 
    output_currency, calculation_method
) VALUES (
    'currency_conversion',
    '{"from_amount": 10000, "from_currency": "USD", "to_currency": "EUR", "rate": 0.85}',
    8500,
    'EUR',
    'ECB_daily_rate'
);
```

## Conclusion

Proper currency handling requires:

1. **Storage**: Use integers for exact arithmetic, DECIMAL when necessary
2. **Precision**: Understand currency-specific decimal places and rounding rules
3. **Validation**: Always validate currency codes and prevent mixing currencies
4. **Documentation**: Clearly document rounding strategies and precision decisions
5. **Testing**: Thoroughly test edge cases, especially distribution and conversion scenarios

The key is consistency in your approach and clear communication with stakeholders about how monetary calculations are performed.


Square also does [banker's rounding](https://squareup.com/help/us/en/article/5092-rounding#:~:text=When%20using%20standard%20rounding%2C%20you,closer%20to%20the%20actual%20amount.).


```sql
drop table product_prices;

CREATE TABLE IF NOT EXISTS product_prices (
	id int GENERATED ALWAYS AS IDENTITY,

	price int NOT NULL,

	PRIMARY KEY (id)
);
truncate table product_prices;
SELECT setseed(0.42);
INSERT INTO product_prices(price)
SELECT round(100*random())
FROM generate_series(1, 10);


-- Underflow: When total is 591, and you have 595 to disburse, due to rounding, only 593 is disbursed.
-- Overflow: When total is 700, the values round up to 701, but you can only disburse 700.

with data as (
	select
		sum(price) 	as total_price,
		max(id) 	as last_id,
		700  		as amount_to_divide_by
	from product_prices
),
projected as (
	select *,
		round(price / total_price::numeric * 100, 2) as ratio,
		round(price / total_price::numeric * amount_to_divide_by) as projected_amount
	from product_prices,
	lateral (select * from data) t
),
corrected as (
	select *,
		sum(projected_amount) over (order by id) as projected_cumulative_amount,
		case when id = last_id
			then amount_to_divide_by
			else least(sum(projected_amount) over (order by id), amount_to_divide_by)
		end as corrected_cumulative_amount
	from projected
)
SELECT *,
	projected_amount - (projected_cumulative_amount - corrected_cumulative_amount) as distributed,
	(projected_amount - (projected_cumulative_amount - corrected_cumulative_amount)) - price as gain,
	sum(projected_amount - (projected_cumulative_amount - corrected_cumulative_amount)) over (order by id) as cumulative_distributed
FROM corrected;
```

# Dealing with currencies in applications


1. not storing currencies as big int
2. not knowing there exists smaller denominators
3. displaying the wrong discount label (due to conversion between discounts and currency)
4. online vs offline store calculation
5. splitting currency equally
6. rounding currencies in operations (issue with 1 cent discrepancy)
7. not storing discounted amount in unit cents (using percent is dangerous)


## not storing currencies as big int


When storing money in the databse, it is preferable to store it in the base units, which is cents. This is how Stripe does it too. So MYR 100.50 will be stored as 10050 in the database. There are libraries for the Frontend that handles the presentation of the currency such as [Dinero.js](https://dinerojs.com/).


There are certainly exceptions, such as when you need to store charges at rates per minutes. If you business has usecases that charges users at rates per minutes (like AWS instances), then there are two options - storing it as decimal, or instead of cents, store it as microcents (x1e6). However, in the end,it is still not possible to charge users fractions of cents. So the solution is to just store the number of minutes the users are charged, and the rates per minutes. Then the chargeable amount will be truncated and still stored as cents.


## not knowing there exists smaller denominators

What is the similarity between Japanese yen and Indonesian rupiah? The answer is both doesn't have unit cents. However, the smallest unit for Indonesian Rupiah is not 1 Rp, it is safer to say it is 500 Rp [^1]. This affects a lot of things, particularly pricing related to offline sales, since you cannot charge users 1111 Rp. Most digital payments however accepts that during transfer, though it is questionable how they could allow withdrawal of such sums later on. So even though you are mostly running online stores that accepts digital payments, you should also take that into consideration so as to not confuse your offline buyers.


## discount labels

Again, due to fractional units and the lowest denominator that varies per currency, the conversion from discount to unit cents and vice versa might not always lead to the same result.


When it comes to rounding, you can either round the values up (called ceiling) or down (called floor) or whichever is nearer (0.7 becomes 1, 0.3 becomes 0). There is also Banker's rounding.

When dealing with discounts, it is probably better to round up the fractional units, aka giving users more discounts, but keep the discount percent.

For example, given a product costing MYR 8.80 with 10% discount, the discount should be 0.88 cents. However, the smallest unit for MYR is 5 cents, so we round the discount to 0.90 cents. However, after rounding up, the discount would actually be more than 10%. But this is still okay, since we can't collect 2 cents from the users. The other option is to just not show the percent, but only the absolute value (e.g. 90cents discount etc).

## splitting currencies

Given MYR 100, how do you split it equally among 3 people? This again, depends on the context. For anything involving money, instead of thinking them as just involving numbers, think of the problem as having to split paper notes.

If I have 100 notes each value at MYR 1, then I can only possibly split it to a ratio of 33, 33 and 34. On the other hand, if the smallest note is MYR 5, then the only possible option is  6, 7, 7 notes each. In your application, when performing the split, ensure that the each split is reproducible - especially when splitting the remainders. Whether to give the first few users or the last few users the remainders, this might impact the calculation a lot.


## not storing discount as cents

Lastly, business rules such as discounts can be stored as percent. However, the actual discount given should be stored as absolute value in cents, so that there's no second guessing how the values are derived.

The db users does not need to know how the discounts are calculated, they just need to refer to the absolute value stored. So any changes in the code in the future (such as mistakes in percent calculation) does not affect the db users, they can just refer to the absolute discount charged.



[^1]: https://en.wikipedia.org/wiki/Indonesian_rupiah#:~:text=The%20rupiah%20is%20divided%20into,banknotes%20denominated%20in%20sen%20obsolete.&text=The%20language(s)%20of%20this,have%20a%20morphological%20plural%20distinction.&text=)%20The%20subunit%20sen%20is%20no%20longer%20in%20practical%20use.
