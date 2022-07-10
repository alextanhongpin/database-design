# JSON

- by default, the field is not null
- use `jsonb` instead of `json`
- note that `jsonb` strips all white spaces

## To update an existing field

```sql
UPDATE table SET json_data = JSON_REPLACE(json_data, "$.field", value) WHERE...;
```

## To query a json field

```sql
SELECT json_data -> "$.field_name" FROM table;
```

## To get rows with json array

```sql
SELECT * FROM table WHERE JSON_TYPE(json_field) = 'ARRAY';
```

## To aggregate the json array from all rows where the column is not null

```sql
SELECT JSON_ARRAYAGG(organizations) FROM user WHERE JSON_TYPE(organizations) = 'ARRAY';
```

## Privilege for functions

```sql
mysql -u USERNAME -p
set global log_bin_trust_function_creators=1;
```

## Thoughts on storing json data as object vs array

with object:

- we probably need to create a static struct to manage the growing keys
- no identity on the kind of data (unless determined through column name)
- once unmarshalled, the values can be used straight away

with array:

- more generic approach
- need to loop through each key value pairs to get the data 
- easier to extend in the future

```js
{
  "a": "1",
  "b": "2"
}

// vs
{
  "data": [{"key": "a", "value": "1"}]
}
```

## Json or not?

Don’t use json

- no protection against referential integrity (if something gets deleted etc)
- no sorting
- no joining
- no constraints (uniqueness)

## Converting JSON to a database row (Postgres)

For single row:

```sql
SELECT * 
FROM json_populate_record(null::account, '{"email": "john.doe@mail.com"}');
```

For multiple rows:

```sql
SELECT * 
FROM json_populate_recordset(null::account, '[{"email": "john.doe@mail.com"}, {"email": "janedoe@mail.com"}]');
```

To build it from a dynamic list:

```sql
-- json_populate_record(record, json) <- convert the jsonb format to json. Merge with || only works with jsonb.
SELECT * FROM json_populate_record(null::account, ('{"email": "john.doe@mail.com"}'::jsonb || '{"token": "hello"}')::json);
SELECT * FROM json_populate_record(null::account, ('{"email": "john.doe@mail.com"}'::jsonb || json_build_object('token', 'hello')::jsonb)::json);
SELECT * FROM json_populate_recordset(null::account, '[{"email": "john.doe@mail.com"}, {"email": "janedoe@mail.com"}]');
```

## Building json object (Postgres)

The merge only works for `jsonb`, not `json`:

```sql
SELECT '{"email": "john.doe@mail.com"}'::jsonb || '{"token": "hello"}'; -- {"email": "john.doe@mail.com", "token": "hello"}
SELECT '{"email": "john.doe@mail.com"}'::json || '{"token": "hello"}'; -- {"email": "john.doe@mail.com"}{"token": "hello"}
```

## Insert json into table (Postgres)

Some limitations - if the field value is not provided in json, it will be treated as null. So for strings, it will throw an error if there is a text column with `not null` constraint.

```sql
  INSERT INTO pg_temp.person (name, picture, display_name)
  -- Don't include fields like ids.
  SELECT name, picture, display_name
    FROM json_populate_record(
      null::pg_temp.person, 
      (_extra::jsonb || json_build_object('name', _name, 'display_name', _display_name)::jsonb)::json
    )
  RETURNING *
```

## Check if a json field exists (Postgres)

```sql
SELECT '{"name": "a", "age": 10}' ? 'age';
```

## Aggregating rows as json in (Postgres)

There are times we want to aggregate a row as json, so that we can deserialize it back to full objects at the application layer:

```sql
-- NOTE: The table name (notification) must be specified before the .*
SELECT array_agg(to_json(notification.*)), subscriber_id, count(*)
FROM notification
GROUP BY subscriber_id;
```

## Update json data with jsonb set (Postgres)

Idempotent update of a json object counter.

```sql
select jsonb_set(
    '{"video": 1}'::jsonb, 
    '{video}', 
    (SELECT (SELECT '{"video": 1}'::jsonb-> 'video')::int + 1)::text::jsonb
);
```

## Convert row to json, and add additional fields

```sql
SELECT row_to_json(reservation_created.*)::jsonb || json_build_object('start_date', lower(validity), 'end_date', upper(validity))::jsonb
FROM reservation_created
```

## Prettify JSONB array column

```sql
SELECT id, jsonb_pretty(log::jsonb) FROM saga, UNNEST(logs) AS log;
```



## Using custom type vs JSON

If the shape is known, store a custom type instead of json.

```sql
create type translations as (
	en text,
	ms text
);


create table products (
	id int generated always as identity,
	
	name text not null,
	translations translations not null,
	primary key (id)
);

insert into products (name, translations) values
('test', '(en-test,ms-test)'::translations);
table products;
alter type translations drop attribute ms; -- This will drop the data.
alter type translations add attribute ms text;

select *, (translations).en, (translations).ms from products;

-- Updating doesn't require the column.
update products set translations.ms = 'ms-test';
update products set translations.ms = null;

select *, (translations).en, (translations).ms from products;

-- However, we are unable to enforce the constraint. Let's add a domain type that is derived from the base type with some constraint - both values must be either set or null.
create domain app_translations as translations check (
	((value).en, (value).ms) is not null
);

drop table products;
create table products (
	id int generated always as identity,
	
	name text not null,
	translations app_translations not null,
	primary key (id)
);

insert into products (name, translations) values
('test', '(en-test,ms-test)'::translations);
update products set translations.ms = null;
select *, (translations).en, (translations).ms from products;



drop table currencies;
create table currencies (
	name text not null,
	primary key (name)
);

insert into currencies (name) values ('sgd');

create type money2 as (
	currency text,
	amount int
);
drop type money;
select '(idr,100)'::money2;

create table products (
	id int generated always as identity,
	price money2,
	
	primary key (id),
	foreign key (price.currency) references currencies (name) -- NOT POSSIBLE
);

drop table products;

-- We can however, make use of generated columns to enforce foreign key.
create table products (
	id int generated always as identity,
	price money2 not null,
	currency text generated always as ((price).currency) stored,
	
	
	primary key (id),
	foreign key (currency) references currencies (name)
);
insert into products (price) values ('(sgd,1000)'::money2); -- Works
insert into products (price) values ('(idr,1000)'::money2); -- Fails

table products;
```

Updating constraints for custom types is easy. For the `app_translations` example, say if we want to add a new translation `id`.



```sql
-- Add a new translation `id`
alter type translations add attribute id text;

-- Update existing data first.
update products set translations.id = 'some value';

SELECT * FROM pg_catalog.pg_constraint where conname = 'app_translations_check';
-- Drop the existing constraint.
alter domain app_translations drop constraint app_translations_check;

-- Now make the the field `id` mandatory.
alter domain app_translations add constraint app_translations_check check (
	((value).en, (value).ms, (value).id) is not null
);
```
