## Updating check constraints in Postgres

This example demonstrates how to update check constraints in Postgres. Say if we have the following table with a check constraint:
```sql
CREATE TABLE post (
    id uuid DEFAULT gen_random_uuid() UNIQUE,
    type text CHECK (type = ANY (ARRAY['question'::text, 'answer'::text, 'comment'::text])),
    CONSTRAINT post_pkey PRIMARY KEY (id, type)
);
```

Find the name of the check constraint:
```sql
select pgc.conname as constraint_name,
       ccu.table_schema as table_schema,
       ccu.table_name,
       ccu.column_name,
       pgc.conkey as definition
from pg_constraint pgc
join pg_namespace nsp on nsp.oid = pgc.connamespace
join pg_class  cls on pgc.conrelid = cls.oid
left join information_schema.constraint_column_usage ccu
          on pgc.conname = ccu.constraint_name
          and nsp.nspname = ccu.constraint_schema
where contype ='c'
order by pgc.conname;
```
Output:
```
--post_type_check	public	post	type	{2}
```

There are no modify constraint in Postgres, so we need to drop the old constraint by name, and update the new one, preferably with the same name:

```sql
ALTER TABLE POST 
DROP CONSTRAINT post_type_check,
ADD CONSTRAINT post_type_check CHECK (type = ANY (ARRAY['question'::text, 'answer'::text, 'comment'::text]));
```
