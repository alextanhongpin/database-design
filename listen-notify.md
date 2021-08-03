# Pub/Sub with postgres


Usecases
- broadcasting events on table changes
- can be used on smaller table, e.g. config tables to propagate real-time changes to application. When config is inserted, updated or deleted, the applications that listens to the channel will update the values in the application in a thread-safe manner
- useful for reference tables, or rule/permission tables



## Basic

```sql
listen hello;
notify hello, 'this is good';
unlisten hello;
```

## Using default tcn module

Only useful if you want to propagate the primary key, as the body or the changes will not be notified.

```sql
drop table users cascade;
create table if not exists users (
	id uuid default gen_random_uuid(),
	name text not null,
	
	primary key (id)
);

-- This only broadcasts the primary key, not body.
create extension tcn;

-- ERROR:  triggered_change_notification: must be called on a table with a primary key
create trigger users_tcn
after insert or update or delete on users
for each row execute function
triggered_change_notification('users_tcn');
drop trigger users_tcn on users;

listen users_tcn;
insert into users(name) values ('john');
update users set name = 'jane';
delete from users;
table users;
```


## Listens to all changes for a given table as json string

```sql

create table if not exists users (
	id uuid default gen_random_uuid(),
	name text not null,
	
	primary key (id)
);
insert into users(name) values ('john');
select row_to_json(users.*) from users;


create or replace function triggered_change_notification_jsonb() returns trigger as $$
	begin
		IF TG_ARGV[0] IS NULL THEN
			RAISE EXCEPTION 'triggered_change_notification_jsonb requires channel name as the input';
		END IF;
		perform pg_notify(
			TG_ARGV[0], -- channel name.
			json_build_object(
				'new', NEW,
				'old', OLD,
				'tg_name', TG_NAME,
				'tg_when', TG_WHEN,
				'tg_level', TG_LEVEL,
				'tg_op', TG_OP,
				'tg_relid', TG_RELID,
				'tg_relname', TG_RELNAME,
				'tg_table_name', TG_TABLE_NAME,
				'tg_table_schema', TG_TABLE_SCHEMA,
				'tg_nargs', TG_NARGS,
				'tg_argv', TG_ARGV
			)::text
		);
		return null;
	end;
$$ language plpgsql;


drop trigger users_tcn on users;
create trigger users_tcn 
after update on users
for each row
when (old.* is distinct from new.*) -- Two separate trigger required to enable this
execute function triggered_change_notification_jsonb('users_tcn');

create trigger users_tcn_insert_delete
after insert or delete on users
for each row
execute function triggered_change_notification_jsonb('users_tcn');

unlisten users_tcn;
listen users_tcn;
insert into users(name) values ('john');
update users set name = 'jane';
delete from users;
```
