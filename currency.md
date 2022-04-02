# How to store currency in Postgres?


- most suggest to store them in the [smallest currency unit, e.g. cents](https://news.ycombinator.com/item?id=20575702)
- so for example, if you have MYR 123.45, then you will store it as 12345 in the database as integer
- this is [how Stripe does it](https://stripe.com/docs/api/charges/object#charge_object-amount)
- Google stores them as [micros](https://developers.google.com/standard-payments/reference/glossary#micros), which seems a little overkill
- what are the disadvantages of this? client now needs to convert the amount to the smallest currency unit, which can be confusing. If the client forgets to handle the conversion, it will be charged less - e.g.g MYR 1.23 instead of MYR 123.45
- it is also confusing, when the client is sending MYR, but the expected amount is in cents
- there are suggested libraries like dinero.js


## Using numeric

```sql
CREATE TABLE IF NOT EXISTS product_prices (
	id int GENERATED ALWAYS AS IDENTITY,

	price numeric(15,4) NOT NULL,

	PRIMARY KEY (id)
);


INSERT INTO product_prices(price) VALUES
(1234.99999),
(1234.56789),
(1234.56789123),
(1234.5679987654321); -- The numbers will be rounded if it's over 4 decimal places.

SELECT * FROM product_prices;
```

## Dealing with division between currency.


Say if you have MYR 100 to be divided equally between 3 people, you will soon face the issue with decimal.

The proper solution is to divide the amount between two people first, with the values rounded (up or down, it depends on your impementation), and take  the total minus the amount already disbursed.

```
A, B and C
A: received 33
B: received 33
C: received (100 - 33 - 33) = 34
Total disbursed: 100
```

In SQL, it is hard to achieve this.

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
