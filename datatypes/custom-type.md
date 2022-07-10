# Postgres CREATE TYPE and CREATE DOMAIN

## Basic create type

```sql
DROP TYPE employee_contact;
CREATE TYPE employee_contact AS (
	name text,
--   NOTE: constraints does not work.
--	 name text not null,
--	 name text check (name ~ '\s'),
	phone_number text
);

select ('john', '12345678')::employee_contact;
```

## Adding validation


`CREATE TYPE` does not allow constraints. To do so, we use `CREATE DOMAIN` to either add field level validation, or type level validation.

```sql
-- Adding validation on a single field.
DROP DOMAIN check_name CASCADE;
CREATE DOMAIN check_name AS TEXT NOT NULL CHECK (VALUE ~ '^\w([\w\s]+\w)?$');


DROP TYPE employee_contact;
CREATE TYPE employee_contact AS (
	name check_name,
	phone_number text
);

select ''::check_name;
select '(,12345678)'::employee_contact;
select '( ,12345678)'::employee_contact;
select '(hello,12345678)'::employee_contact;
select ('(hello,12345678)'::employee_contact).*;
select ('(hello,12345678)'::employee_contact).name;
select ('(hello,12345678)'::employee_contact).phone_number;

-- Adding validation on type level.
-- Postgres don't have unsigned integer.
CREATE DOMAIN uint2 AS int4
   CHECK(VALUE >= 0 AND VALUE < 65536);

SELECT (-1)::uint2;
SELECT -1::uint2;

DROP TYPE IF EXISTS dimensions;
CREATE TYPE dimensions AS (
	width uint2,
	height uint2,
	length uint2
);

-- Adding type-level validation.
DROP DOMAIN IF EXISTS check_dimensions;
CREATE DOMAIN check_dimensions AS dimensions NOT NULL
	CHECK (
		((VALUE).width, (VALUE).height, (VALUE).length) IS NOT NULL
	);


select '(1,1,)'::dimensions;
select '(1,1,)'::check_dimensions;
select '(1,1,-1)'::check_dimensions;
```


## List all domain types

```sql
SELECT n.nspname AS schema
     , t.typname AS name
     , pg_catalog.format_type(t.typbasetype, t.typtypmod) AS underlying_type
     , t.typnotnull AS not_null

     , (SELECT c.collname
        FROM   pg_catalog.pg_collation c, pg_catalog.pg_type bt
        WHERE  c.oid = t.typcollation AND bt.oid = t.typbasetype AND t.typcollation <> bt.typcollation) AS collation
     , t.typdefault AS default
     , pg_catalog.array_to_string(ARRAY(SELECT pg_catalog.pg_get_constraintdef(r.oid, TRUE) FROM pg_catalog.pg_constraint r WHERE t.oid = r.contypid), ' ') AS check_constraints
FROM   pg_catalog.pg_type t
LEFT   JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
WHERE  t.typtype = 'd'  -- domains
AND    n.nspname <> 'pg_catalog'
AND    n.nspname <> 'information_schema'
AND    pg_catalog.pg_type_is_visible(t.oid)
ORDER  BY 1, 2;
```
