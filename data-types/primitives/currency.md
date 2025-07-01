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

In SQL, it is hard to achieve this. Note that using `ceil/floor` instead of `round` will lead to largely imbalance distribution. Rounding seems to ideal for most cases. For more advance usecases, also look at [banker's rounding](https://en.wikipedia.org/wiki/Rounding#Round_half_to_even).


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
