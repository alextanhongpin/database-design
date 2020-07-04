## Dynamic logic without reference table - use functions

```sql
CREATE OR REPLACE FUNCTION is_admin(email text) RETURNS boolean AS $$
	SELECT email in ('john.doe@mail.com', 'jane.doe@mail.com');
$$ LANGUAGE SQL;


select is_admin('john.doe@mail.com'); -- Returns true.
select is_admin('jane@mail.com'); -- Returns false.
```

## Best practices

- use enum if the value is fixed (gender etc)
- use reference table if the values are dynamic
- use `serial` id for reference tables, don't need to use UUID which complicates thing. Use UUID only for entities created by users that exists in large numbers.
