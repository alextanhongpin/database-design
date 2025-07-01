## Things I learn about postgres

Tools
- ~using DBeaver for GUI visualization on MacOS~
- use Postico insteand

Query
- somehow table name must be surrounded by double quote `"`
- somehow string values must be surrounded by single quote `'`, and not double-quote `"`

## Difference between mysql

- does no support update timestamp, need to create a trigger
- uuid does not need to be ordered
- no need to set utf8 mb4 
- use `serial` or `bigserial` for postgres, use `auto_increment` for mysql.

postgres:
```
CREATE TABLE fruits(
   id SERIAL PRIMARY KEY,
   name VARCHAR NOT NULL
);
CREATE EXTENSIONS IF NOT EXISTS “uuid-ossp”;

CREATE TABLE IF NOT EXISTS user (
	id UUID PRIMARY KEY default uuid_generate_v4(),
	name TEXT NOT NULL DEFAULT ‘’,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL 	DEFAULT current_timestamp,
)
```
mysql:
```
CREATE TABLE IF NOT EXISTS user (
	id BIGINT UNSIGNED AUTO_INCREMENT,
 	email VARCHAR(255) NOT NULL UNIQUE,
	created_at DATETIME NOT NULL DEFAULT current_timestamp,
	updated_at DATETIME NOT NULL DEFAULT current_timestamp ON UPDATE current_timestamp,
	PRIMARY KEY (id)
)
```

## Useful commands

Equivalent of `show tables` in MySQL:

```postgres
SELECT * 
FROM  pg_catalog.pg_tables 
WHERE schemaname != 'pg_catalog' 
AND   schemaname != 'information_schema';
```

## Primary key in postgres

```sql
-- Posgresql
id serial PRIMARY KEY

-- Mysql
id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY
```
## Text or varchar

In postgres, use TEXT for everything, it should be equally performant as VARCHAR. Use VARCHAR only to limit the characters if you need to validate the length.


## Miscellaneous
- Postgres does not have timestamp on update, you need to manually implement a trigger
- Postgres does not allow table name to be user, use account instead
- In node js, selecting a column as a name must be lowercase, not camecase. In node library it is automatically converted to lowercase even if we set it as camelcase.


## UUID

http://www.postgresqltutorial.com/postgresql-uuid/
```
SELECT * FROM pg_available_extensions;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```


## On Conflict Update
```sql
INSERT INTO product (user_id, name) 
VALUES ('fa633b7c-f865-11e9-9537-7b2b141b65eb', 'iphone')
ON CONFLICT(name) DO UPDATE SET updated_at = now()
RETURNING id;
```

## Golang Errors

## postgres pqerror 
Constraint unique column.

```go
var pqerr *pq.Error
if errors.As(err, &pqerr) {
	fmt.Printf("%#v\n", pqerr)
}
```

Output:

```
&pq.Error{Severity:"ERROR", Code:"23505", Message:"duplicate key value violates unique constraint \"product_name_key\"", Detail:"Key (name)=(ENkZWiilbpvnSWOnPBYRkrqRJ) already exists.", Hint:"", Position:"", InternalPosition:"", InternalQuery:"", Where:"", Schema:"public", Table:"product", Column:"", DataTypeName:"", Constraint:"product_name_key", File:"nbtinsert.c", Line:"570", Routine:"_bt_check_unique"}
```

Handling duplicate:
```go
const DuplicatePrimaryKeyViolation = "23505"

...

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code == DuplicatePrimaryKeyViolation {
			return false, shorturl.ErrAlreadyExists
		}
	}
```

## Useful Postgres query

```sql

-- Apply ranking to all rows, and find the current rank of the existing user.
SELECT 	login, rank FROM (
	SELECT 	login, rank() OVER (ORDER BY updated_at) AS rank 
	FROM 	login
) ranked 
WHERE login = 'alextanhongpin';

-- Select the rows where the column contains the prefix word jav*.
SELECT 	distinct(trim(primary_language)) 
FROM 	repository 
WHERE 	to_tsvector(primary_language) @@ to_tsquery('jav:*');

-- Select the rows where the column contains either the word first or word with prefix sec...
SELECT 	* 
FROM 	login 
WHERE 	to_tsvector(company) @@ to_tsquery('first & sec:*');

-- Select the rows where the JSON array contains the search text.
SELECT 	login, organizations 
FROM 	login 
WHERE 	organizations ? 'text'::text;

-- Concat JSON array, but only take the distinct values.
SELECT 	JSON_AGG(distinct trim(value)) 
FROM 	login, jsonb_array_elements_text(organizations) 
WHERE 	organizations <> 'null';
```


## Installing a new postgres version

To debug issue:
```
$ tail -n 10 /usr/local/var/log/postgres.log
```
```
2019-11-25 16:47:19.036 +08 [3896] FATAL:  database files are incompatible with server
2019-11-25 16:47:19.036 +08 [3896] DETAIL:  The data directory was initialized by PostgreSQL version 11, which is not compatible with this version 12.1.
2019-11-25 16:47:29.061 +08 [3899] FATAL:  database files are incompatible with server
2019-11-25 16:47:29.061 +08 [3899] DETAIL:  The data directory was initialized by PostgreSQL version 11, which is not compatible with this version 12.1.
2019-11-25 16:47:33.625 +08 [4067] FATAL:  database files are incompatible with server
2019-11-25 16:47:33.625 +08 [4067] DETAIL:  The data directory was initialized by PostgreSQL version 11, which is not compatible with this version 12.1.
2019-11-25 16:47:43.648 +08 [4264] FATAL:  database files are incompatible with server
2019-11-25 16:47:43.648 +08 [4264] DETAIL:  The data directory was initialized by PostgreSQL version 11, which is not compatible with this version 12.1.
2019-11-25 16:47:53.671 +08 [4265] FATAL:  database files are incompatible with server
2019-11-25 16:47:53.671 +08 [4265] DETAIL:  The data directory was initialized by PostgreSQL version 11, which is not compatible with this version 12.1.
```

Solution:
```
$ brew postgresql-upgrade-database
```


## Show all tables in Postgres

```sql
select table_name from information_schema.tables where table_schema = 'public' and table_type = 'BASE TABLE';
```

## Check connections to database

```sql
select * from pg_stat_activity;
```

## Check number of prepared statements for the given connection
```sql
select * from pg_prepared_statements;
```

## Ordering result by the given id

```sql
select * 
from product_items 
join unnest('{223, 212, 2, 226, 333, 3, 1, 224, 8, 281}'::int[]) with ordinality t(id, ord)
using (id)
order by t.ord;
```

## Lateral Join for Postgres

What is the use case?

