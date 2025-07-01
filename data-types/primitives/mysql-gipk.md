## Explore GIPK

GIPK stands for Generated Invisible Primary Key.

```sql
-- Default is off
SELECT @@sql_generate_invisible_primary_key;

SET sql_generate_invisible_primary_key=ON;
SELECT @@sql_generate_invisible_primary_key;
```

Sample table:

```sql
CREATE TABLE users (
	uuid binary(16) unique,
	name varchar(255) not null
);
```

The generated primary key column will be named `my_row_id`:
```sql
show create table users;
| users | CREATE TABLE `users` (
  `my_row_id` bigint unsigned NOT NULL AUTO_INCREMENT /*!80023 INVISIBLE */,
  `uuid` binary(16) DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  PRIMARY KEY (`my_row_id`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
```

Inserting sample data:

```sql
insert into users(uuid, name) values (uuid_to_bin(uuid_v4()), 'john');

-- This does not use primary key.
explain select * from users where uuid = uuid_to_bin('10cc2990-a1eb-4348-9e8d-a5ac892567fc');

-- This uses primary key.
explain select * from users where my_row_id = 1;
```

Make GIPK visible:
```sql
ALTER TABLE users ALTER COLUMN my_row_id SET VISIBLE;
```

Can we use GIPK as foreign keys?

```sql
create table accounts (
	uuid binary(16) unique,
	name varchar(255) not null,
	user_uuid binary(16) not null,
	user_my_row_id bigint references users(my_row_id),
	foreign key (user_uuid) references users(uuid)
);
ALTER TABLE accounts ALTER COLUMN my_row_id SET VISIBLE;

INSERT INTO accounts (uuid, name, user_uuid, user_my_row_id) select uuid_to_bin(uuid_v4()), 'john''s account', uuid, my_row_id from users where name = 'john';

-- This is not using primary key.
explain select * from users u join accounts a on (u.uuid = a.user_uuid) \G;

-- This is using primary key
explain select * from users u join accounts a on (u.my_row_id = a.user_my_row_id) \G;
```
