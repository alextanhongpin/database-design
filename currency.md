# How to store currency in Postgres?


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
