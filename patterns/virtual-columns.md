# Virtual columns

When to store values in column? Database columns should not contain computed values - there are exceptions: 
- caching association counts can improve query performance, especially if the count does not need to be accurate
- e.g. users has posts, reviews, and comments. Rather than performing the query everytime to get those counts, we store them in the table
- computed columns can be sorted - we can now sort the rows by the reviews count, comments count, ranking etc
- we can use a prefix to distinguish the normal columns from virtual columns, e.g. `cached_comments_counts`, `cached_posts_count`

Store in column pros
- Can sort
- Values are easy to query

Store in columns cons
- Need to update
- Values might not be accurate over time
- Additional column for every other field
- that column might not be needed at all (optional).

Store in redis pros
- Fast
- Can be decentralised
- Store only what you need, reduce storage on database
- Precomputed value


## Using postgres generated column

Usecases
- for data that is precomputed from the same table
- mostly static, does not (actually it cannot) require references to another table
- reduce the need for additional computation. Column computed once, but requires additional storage
- column is mandatory, and the logic is fixed (hard to change after it is there)

```sql
create table if not exists users (
	id int generated always as identity,
	first_name text,
	last_name text,
	full_name text generated always as (first_name || ' ' || last_name) stored
);

insert into users(first_name, last_name) 
values ('john', 'doe');
table users;

-- Whenever the first_name change, the computed column name also reflects the changes.
update users set first_name = 'jane';
table users;
```


## Using lookup function for table

Usecases
- mostly static, but precomputed everytime it is invoked
- does not require additional storage
- easier to change implementation, changes are applied to function instead of table
- column can be treated as optional (only called when required) instead of making it permanent

```sql
create table if not exists users (
	id int generated always as identity,
	first_name text,
	last_name text
);

create or replace function users_full_name(tbl users) returns text AS $$
	select tbl.first_name || ' ' || tbl.last_name;
$$ language sql;

insert into users(first_name, last_name) 
values ('john', 'doe');

select first_name, last_name, users.users_full_name as full_name from users;
```

## Virtual column based on another

There are few usecases where you want to compute the column of a table based on that of another.

However, you do not want to do it in application code, because calling a list of rows leads to n+1 problem where you need to fetch and compute the column value for each row. 

Doing in application also has one issue where the business logic is not shared across if you have more than one application referencing the same database (e.g. for analytics etc). Doing the computation for each different application can lead to inconsistent business logic, where it is changed in one but not updated in another.

```sql
create table if not exists questions (
	id uuid default gen_random_uuid(),
	title text not null,
	primary key (id)
);

create table if not exists answers (
	id uuid default gen_random_uuid(),
	title text not null,
	question_id uuid not null references questions(id),
	primary key (id)
);
drop table answers;
drop table questions;

-- The answers count are precomputed.
create or replace function questions_answers_count(tbl questions) returns int as $$
	select count(*) from answers where question_id = tbl.id;
$$ language sql;

insert into questions (title) values ('hello?');
insert into answers (title, question_id) values ('world!', (select id from questions limit 1));
insert into answers (title, question_id) values ('not world!', (select id from questions limit 1));

insert into questions (title) values ('foo?');
insert into answers (title, question_id) values ('world!', (select id from questions limit 1 offset 1));
table questions;
table answers;

-- Performance seems to be the best too (for unknown reasons)
explain analyze select *, questions.questions_answers_count as total
from questions 
order by total desc;

explain analyze select *, (select count(*) from answers where question_id = questions.id) as total
from questions
order by total desc;

explain analyze select *
from questions, lateral (select count(*) as total from answers where question_id = id) as total
order by total desc;
```

## Random in vs = performance

Postgres analyzer seems to generate the same optimized query.

```sql
explain analyze select * from questions where id = '3d2c651c-a1da-4d2e-a44d-6cab24019752';
explain analyze select * from questions where id in ('3d2c651c-a1da-4d2e-a44d-6cab24019752');
```
