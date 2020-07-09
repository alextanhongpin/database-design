## Passing in current context to postgres local session

This technique is useful if you want to apply Row Level Security, but using external users intead of internal users.


```sql
begin;
	-- Variables are only local to transaction.
	set local my.name to 'john.doe';
	select current_setting('my.name');
commit;
```

The following implementation below has been tested and it does not work (shame on CTE!):
```sql
-- Does not work.
select current_setting('my.name');

-- Does not work.
with user_config as (
	select set_config('my.name', 'john.doe', true)
)
select current_setting('my.name', true);

-- Does not work.
select set_config('my.name', 'john.doe', true), set_config('my.age', '10', true);
select current_setting('my.name', true);

-- Does not work.
begin;
	select set_config('my.name', 'john.doe', true);
	select current_setting('my.name', true);
commit;
```
