## Functions best practices

- use functions to simplify database operations, e.g. generating custom ids, slugifying url, updating timestamps
- functions can be used to enforce business rules quite efficiently
- create functions under `pg_temp` for testing, it is done per session, and will be removed once the session is disconnected. Don't use this for production! The point of having functions in the database is so that every application has access to them


## Creating temporary functions for development

```sql
CREATE OR REPLACE FUNCTION pg_temp.sum() RETURNS int AS $$
BEGIN
	RETURN 1 + 1;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

SELECT pg_temp.sum();
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
