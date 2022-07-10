# Range

```sql
CREATE TABLE prices (
	item text,
	price int4range 
);


INSERT INTO prices
VALUES
('mouse', '[10,16)'),
('keyboard', '[20,31)'),
('mouse', '[35,56)');

-- Find all items with price range $15 to $30 with && (range overlap);
SELECT * 
FROM prices
WHERE price && int4range(15, 30);
```
