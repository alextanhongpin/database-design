## Getting a random row in Postgres

There are cases where you need to get random rows in postgres (e.g. seeding data):


```sql
SELECT id FROM (
  -- Find the number of rows so that we can randomize a number.
	WITH total_count AS (
		SELECT count(*) AS count
		FROM job
	),
  -- Number each rows by ordering the rows by created_at. NOTE: If you are already using integer id, this won't be a problem.
	numbered_rows AS (
		SELECT *, row_number() OVER (ORDER BY created_at)
		FROM job
	),
	random_row AS (
		SELECT trunc(random() * (SELECT count FROM total_count)) + 1 AS number
	)
	SELECT *
	FROM numbered_rows 
	WHERE row_number = (SELECT number FROM random_row)
) random_job
```

Alternative...aggregate into array, select a position of an item in the array.


## Alternatively use table sample

https://www.postgresql.org/docs/9.6/tsm-system-rows.html

Note that tablesample is just a probablity. So if you want to randomise the whole table, it is not possible.
```sql
drop table users cascade;
create table if not exists users (
	id uuid default gen_random_uuid(),
	name text not null,
	
	primary key (id)
);

insert into users (name) values 
('john'), ('jane'), ('jessie'), ('alpha'), ('beta'), ('boy');
table users;

create extension tsm_system_rows;
select * 
from users
tablesample bernoulli(100) -- 100% sample does not work, it will just return the sorting as it is.
repeatable (10);
```
