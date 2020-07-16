## Setting default timestamp with range

This will produce default constant value:
```diff sql
	-validity tstzrange NOT NULL DEFAULT '[now,)',
	+validity tstzrange NOT NULL DEFAULT tstzrange(now(), null, ‘[)’),
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

## Temporal Data

Useful for data with the following characteristic:
- cannot be deleted once created
- can only be `effective`, which means it has not expired
- no overlapping data with the same `effective` period

E.g. room reservation. There can be only one person making a reservation for a hotel room at a given period. 
```sql
CREATE EXTENSION btree_gist;
CREATE TABLE IF NOT EXISTS reservation (
	room text NOT NULL,
	period tstzrange NOT NULL,
	EXCLUDE USING GIST (room WITH =, period WITH &&)
);
INSERT INTO reservation(room, period) VALUES 
('1', tstzrange(now(), now() + interval '1 day'));

TABLE reservation;
INSERT INTO reservation(room, period) VALUES 
('1', tstzrange(now() + interval '1 day', now() + interval '2 day'));
```

## Null

```sql
select tstzrange(now(), null); -- ["2020-07-15 17:26:58.875943+00",)
select tstzrange(now(), timestamptz 'infinity'); -- ["2020-07-15 17:27:09.516742+00",infinity)
select now() + interval '1 day' <@ tstzrange(now(), null); -- t
select now() + interval '1 day' <@ tstzrange(now(), timestamptz 'infinity'); -- t
select upper_inf(tstzrange(now(), null)); -- t
select upper_inf(tstzrange(now(), timestamptz 'infinity')); --f

select '[,)'::int4range;
```
