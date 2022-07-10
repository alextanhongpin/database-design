## LIMIT ... WITH TIES

Allows the last few rows to have the same values based on the `order by` key.
```sql
SELECT * FROM employees
ORDER BY salary DESC
FETCH FIRST 2 ROWS WITH TIES
OFFSET 3;
```

Output:
```
NAME	SALARY	DEPARTMENT
Falsa Tortuga	1400	marketing
Duquesa	1300	sales
Liebre de Marzo	1300	engineering
```


## Limit at most N rows per category

Use cases:
- limiting user from creating multiple entries
- used to prevent coupon codes from beeing redeemed when it reaches the limit



```sql
DROP TABLE coupon CASCADE;
CREATE TABLE IF NOT EXISTS coupon (
	id bigint GENERATED ALWAYS AS IDENTITY,
  
	code text NOT NULL,
  	-- Use the check constraint to apply the business logic.
	max_redemption int NOT NULL CHECK (max_redemption > 0),
	redempted int NOT NULL DEFAULT 0 CHECK (redempted > -1 AND redempted <= max_redemption),
	
	PRIMARY KEY (id),
	UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS  "user"  (
	id bigint GENERATED ALWAYS AS IDENTITY,
	name text NOT NULL,
	
	PRIMARY KEY (id)
);


CREATE TABLE IF NOT EXISTS  user_coupon (
	id bigint GENERATED ALWAYS AS IDENTITY,
  
  	-- Foreign keys.
	user_id bigint NOT NULL,
	coupon_id bigint NOT NULL,
	
  	-- Constraints.
	PRIMARY KEY (id),
	UNIQUE (user_id, coupon_id), -- Each user can only redeem once.
	FOREIGN KEY (user_id) REFERENCES "user"(id),
	FOREIGN KEY (coupon_id) REFERENCES coupon(id)
);
```

Aside from database trigger, normal transactions should work well:

```sql
BEGIN;
  -- Row-level locking.
	SELECT id
	FROM coupon 
	WHERE code = 'JOHN'
	LIMIT 1
	FOR UPDATE;

	-- Create the user coupon.
	INSERT INTO user_coupon (user_id, coupon_id)
	VALUES (2, (SELECT id FROM coupon WHERE code = 'JOHN'));
	
	-- Update the count.
	UPDATE coupon 
	SET redempted = redempted + 1 
	WHERE id = (SELECT id FROM coupon WHERE code = 'JOHN');
COMMIT;
```

Using `WITH` CTE:

```sql
WITH coupon_locked AS (
	SELECT id
	FROM coupon 
	WHERE code = 'JOHN'
	LIMIT 1
	FOR UPDATE
), redeemed AS (
	UPDATE coupon 
	SET redempted = redempted + 1
	WHERE id = (SELECT id FROM coupon_locked)
)
INSERT INTO user_coupon (user_id, coupon_id)
VALUES (1, (SELECT id FROM coupon_locked));
```

Using function (provides flexibility, and does not tie the business logic to database unlike triggers):

```sql
DROP FUNCTION redeem_coupon(text, int);

-- Updates the coupon by the code, and returning the coupon id.
CREATE OR REPLACE FUNCTION redeem_coupon(_code text, _count int DEFAULT 1) RETURNS bigint AS $$
	UPDATE coupon SET redempted = redempted+_count 
	WHERE code = _code
	RETURNING id;
$$ LANGUAGE sql; 

-- Use a coupon.
INSERT INTO user_coupon(user_id, coupon_id)
VALUES (1, redeem_coupon('JOHN'));
```

A more generic design - if the threshold is set, then the counter cannot be more than the threshold:
```sql
CREATE TABLE IF NOT EXISTS counter (
	id bigint GENERATED ALWAYS AS IDENTITY,
  
	code text NOT NULL,
	-- Use the check constraint to apply the business logic.
	threshold int NOT NULL DEFAULT 0,
	counter int NOT NULL DEFAULT 0 CHECK (counter > -1 AND (CASE WHEN threshold = 0 THEN true ELSE counter <= threshold END)),
	
	PRIMARY KEY (id),
	UNIQUE (code)
);
```
