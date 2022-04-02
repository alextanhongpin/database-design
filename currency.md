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
