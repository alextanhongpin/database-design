# Update if value is set

This will cause empty updates though when the values are all null, which is unlikely. To prevent that, see below.
```sql
UPDATE some_table SET
  column_1 = COALESCE(param_1, column_1),
  column_2 = COALESCE(param_2, column_2),
  column_3 = COALESCE(param_3, column_3),
  column_4 = COALESCE(param_4, column_4),
  column_5 = COALESCE(param_5, column_5)
WHERE id = some_id;
```

By adding "param_x IS NOT NULL", we avoid empty updates:
```sql
UPDATE some_table SET
    column_1 = COALESCE(param_1, column_1),
    column_2 = COALESCE(param_2, column_2),
    ...
WHERE id = some_id
AND  (param_1 IS NOT NULL AND param_1 IS DISTINCT FROM column_1 OR
      param_2 IS NOT NULL AND param_2 IS DISTINCT FROM column_2 OR
     ...
 );
```

# Update and return the changes (only if there are changes)

If there are changes, the values will be returned. 

```sql
create table if not exists foo(
	id serial,
	name text not null,
	age int not null,
	bio text not null,
	primary key(id),
	unique(name)
);

-- Perform an upsert
-- on conflict and return the 
-- updated values only if they are different
insert into foo(id, name, age, bio) 
values (1, 'john', 14, 'hello')
on conflict (name)
do update 
set (id, name, age, bio) = ROW(excluded.*)
where foo is distinct from excluded
returning *;
```
