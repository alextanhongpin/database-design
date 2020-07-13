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
