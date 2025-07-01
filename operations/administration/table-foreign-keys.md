# SQL to find tables and foreign keys


For LLM usage.

```sql
create table if not exists users (
	id uuid default gen_random_uuid(),
	name text not null check(length(name) > 0),
	age int not null check(age > 0),
	created_at timestamptz not null default current_timestamp,
	updated_at timestamptz not null default current_timestamp,
	primary key (id)
);


create table if not exists accounts (
	id uuid default gen_random_uuid(),
	provider text not null,
	user_id uuid,
	created_at timestamptz not null default current_timestamp,
	updated_at timestamptz not null default current_timestamp,
	primary key(id),
	foreign key (user_id) references users(id)
);

SELECT *
FROM information_schema.tables
WHERE table_schema = 'public';

SELECT
    kcu.table_name AS foreign_table,
    rel_kcu.table_name AS primary_table,
    kcu.column_name AS foreign_column,
    rel_kcu.column_name AS primary_column
FROM
    information_schema.table_constraints AS tc
JOIN
    information_schema.key_column_usage AS kcu
ON
    tc.constraint_name = kcu.constraint_name
JOIN
    information_schema.key_column_usage AS rel_kcu
ON
    kcu.ordinal_position = rel_kcu.ordinal_position
    AND tc.constraint_name = rel_kcu.constraint_name
WHERE
    tc.constraint_type = 'FOREIGN KEY';
```
