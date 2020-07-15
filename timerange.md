## Setting default timestamp with range

This will produce default constant value:
```diff sql
	- validity tstzrange NOT NULL DEFAULT '[now,)',
	+ validity tstzrange NOT NULL DEFAULT tstzrange(now(), null, ‘[)’),
```

## Using timestamp range in Postgres
```sql
CREATE TABLE IF NOT EXISTS hello (
	id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	validity tstzrange NOT NULL DEFAULT tstzrange(NOW(), NULL, '[)'),
	name text
)
;
INSERT INTO hello (name) VALUES ('john');

SELECT 
	lower_inc(validity), 
	lower(validity), 
	upper(validity),
	upper_inc(validity),
	validity,
	upper(validity) - lower(validity)
FROM hello;

UPDATE hello 
SET validity = tstzrange(
	lower(validity), now(), '[)'
);
```
