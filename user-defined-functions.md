## Functions best practices

- use functions to simplify database operations, e.g. generating custom ids, slugifying url, updating timestamps
- functions can be used to enforce business rules quite efficiently
- create functions under `pg_temp` for testing, it is done per session, and will be removed once the session is disconnected. Don't use this for production! The point of having functions in the database is so that every application has access to them


## Creating temporary functions for development

Instead of creating actual tables and functions, add the path `pg_temp` for testing. Example of temporary functions:
```sql
CREATE OR REPLACE FUNCTION pg_temp.sum() RETURNS int AS $$
BEGIN
	RETURN 1 + 1;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

SELECT pg_temp.sum();
```

Example of temporary tables:

```sql
CREATE TABLE IF NOT EXISTS pg_temp.account_type (
	id int GENERATED ALWAYS AS IDENTITY,
	
	PRIMARY KEY (id)
);

DROP TABLE pg_temp.account_type;
```

# UDF

```sql
-- +goose Up
DROP FUNCTION IF EXISTS fn_test;
-- +goose StatementBegin
-- SQL in this section is executed when the migration is applied.
CREATE FUNCTION fn_test(a int, b int) RETURNS int 
DETERMINISTIC
NO SQL
BEGIN
	DECLARE result int default 0;
	SET result = a + b;
	RETURN result;
END;
-- +goose StatementEnd
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP FUNCTION IF EXISTS fn_test;
```


## Find all functions in Postgres

```sql
select n.nspname as function_schema,
       p.proname as function_name,
       l.lanname as function_language,
       case when l.lanname = 'internal' then p.prosrc
            else pg_get_functiondef(p.oid)
            end as definition,
       pg_get_function_arguments(p.oid) as function_arguments,
       t.typname as return_type
from pg_proc p
left join pg_namespace n on p.pronamespace = n.oid
left join pg_language l on p.prolang = l.oid
left join pg_type t on t.oid = p.prorettype 
where n.nspname not in ('pg_catalog', 'information_schema')
AND p.proname like '%auth%'
order by function_schema,
         function_name;
```


## Lookup Functions
Use lookup functions for cleaner code. Performance-wise, it is not so bad, and it reduces a lot of complexity (joining on application side):
```sql
-- With Joins.
EXPLAIN ANALYZE
SELECT *,
	et.name AS employment_type,
	c.name AS salary_currency
FROM job
JOIN employment_type et ON (et.id = employment_type_id)
JOIN currency c ON (c.id = salary_currency_id);

-- With functions.
-- Clean SQL
-- Lookup can be improved by using covering index to include the columns when querying.
EXPLAIN ANALYZE
SELECT *,
	lookup_employment_type_name_from_id(employment_type_id) AS employment_type,
	lookup_currency_code_from_id(salary_currency_id) AS salary_currency 
FROM job;
```

## Function enumify Postgres

Simple function to enumify the column:

```sql
CREATE OR REPLACE FUNCTION enumify (_txt text) RETURNS text AS $$
	WITH uppercased AS (
		SELECT upper(_txt) AS txt
	), 
	alphabets_only AS (
		SELECT regexp_replace((SELECT txt FROM uppercased), '[^A-Z]', '_', 'g') AS txt
	),
	single_separator AS (
		SELECT regexp_replace((SELECT txt FROM alphabets_only), '_+', '_', 'g') AS txt
	)
	SELECT txt FROM single_separator;
$$ LANGUAGE SQL IMMUTABLE; -- Must be immutable

SELECT enumify('hello world'); -- HELLO_WORLD
```

### Three different ways to add enum column

Check validation to ensure the format matches:
```sql
DROP TABLE IF EXISTS test;
CREATE TABLE IF NOT EXISTS test (
	id uuid DEFAULT gen_random_uuid(),
	name text NOT NULL,
	enum text NOT NULL CHECK (enum = enumify(name)), -- No control over enum naming.
	PRIMARY KEY (id),
	UNIQUE (enum) -- Name will be unique too. This will not help if we want non-unique name but unique enum.
);

INSERT INTO test (name, enum) VALUES ('hello world', 'HELLO_WORLD');
```

Postgres generated column to create the `enum` column from the `name` column:
```sql
DROP TABLE IF EXISTS test;
CREATE TABLE IF NOT EXISTS test (
	id uuid DEFAULT gen_random_uuid(),
	name text NOT NULL,
	-- No control over enum naming. Note that the function used must be IMMUTABLE.
	-- Redundant column for either name or enum, because only one will be used. But for presentation ui purposes, name will be displayed.
	enum text GENERATED ALWAYS AS (enumify(name)) STORED, 
	PRIMARY KEY (id),
	UNIQUE (enum)
);

INSERT INTO test (name) VALUES ('hello world');
```

Use `citext`, case insensitive extension for the `name`. So `hello` and `HELLO` is the same when querying:
```sql
CREATE EXTENSION IF NOT EXISTS citext;
DROP TABLE IF EXISTS test;
CREATE TABLE IF NOT EXISTS test (
	id uuid DEFAULT gen_random_uuid(),
	-- Single enum column. No duplicates. But performance may vary.
	name citext NOT NULL,
	PRIMARY KEY (id)
);

INSERT INTO test (name) VALUES ('hello world');
```

Other approaches:
- use trigger on insert/update (lack visibility, hidden business rule)
- use ENUM type (too static, lack visibility also, prefer to put them in a table, more dynamic and easier to insert/update/delete)
- use another table for the type, might not work for unique columns that is more dynamic (like slug generation). In other words, the approach above works for reference tables where the number of types are limited.

