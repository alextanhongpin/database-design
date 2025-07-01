
# Featured column

This fulfils the constraints for when we want to have only one featured product for a given category.

```sql
DROP TABLE products;
CREATE TABLE IF NOT EXISTs products (
	id serial,
	featured boolean NOT NULL DEFAULT false,
	category text NOT NULL,
	name text NOT NULL,
	
	PRIMARY KEY (id)
);


-- One category can only have one featured product.
CREATE UNIQUE INDEX ON products (category) WHERE (featured IS TRUE);
DROP INDEX products_name_category_idx;

INSERT INTO products(name, category, featured) VALUES 
('Apple', 'FRUIT', true),
('Banana', 'FRUIT', false);

BEGIN;
UPDATE products SET featured = false;
UPDATE products SET featured = true WHERE name = 'Banana';
COMMIT;
```
