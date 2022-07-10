## Updating check constraints in Postgres

This example demonstrates how to update check constraints in Postgres. Say if we have the following table with a check constraint:
```sql
CREATE TABLE post (
    id uuid DEFAULT gen_random_uuid() UNIQUE,
    type text CHECK (type = ANY (ARRAY['question'::text, 'answer'::text, 'comment'::text])),
    CONSTRAINT post_pkey PRIMARY KEY (id, type)
);
```

Find the name of the check constraint:
```sql
select pgc.conname as constraint_name,
       ccu.table_schema as table_schema,
       ccu.table_name,
       ccu.column_name,
       pgc.conkey as definition
from pg_constraint pgc
join pg_namespace nsp on nsp.oid = pgc.connamespace
join pg_class  cls on pgc.conrelid = cls.oid
left join information_schema.constraint_column_usage ccu
          on pgc.conname = ccu.constraint_name
          and nsp.nspname = ccu.constraint_schema
where contype ='c'
order by pgc.conname;
```
Output:
```
--post_type_check	public	post	type	{2}
```

There are no modify constraint in Postgres, so we need to drop the old constraint by name, and update the new one, preferably with the same name:

```sql
ALTER TABLE POST
DROP CONSTRAINT post_type_check,
ADD CONSTRAINT post_type_check CHECK (type = ANY (ARRAY['question'::text, 'answer'::text, 'comment'::text]));
```


## Limitations

Using `check` constraints to check if the type is one of the values has the same limitation as Postgres's `enum` - we can only add values, we cannot remove an existing value. The example below demonstrates them by creating a table with two enum values `a` and `b` and inserting the values.
```sql
DROP TABLE pg_temp.hello;
CREATE TABLE IF NOT EXISTS pg_temp.hello (
	id int NOT NULL GENERATED ALWAYS AS IDENTITY,
	type text NOT NULL CHECK(type = ANY(ARRAY['a', 'b']))
);
```

If the inserted `type` is not `a` or `b`, an error would be thrown:
```sql
INSERT INTO hello (type) VALUES ('a'), ('b');
INSERT INTO hello (type) VALUES ('c');
--ERROR:  new row for relation "hello" violates check constraint "hello_type_check"
--DETAIL:  Failing row contains (3, c).
```

Once a row with `b` exists, we cannot remove the checking there:
```sql
ALTER TABLE pg_temp.hello
DROP CONSTRAINT hello_type_check,
ADD CONSTRAINT hello_type_check CHECK(type = ANY(ARRAY['a']));
--ERROR:  check constraint "hello_type_check" of relation "hello" is violated by some row
```

However, if we remove the row with type `b`, we can alter the constraint:
```sql
TRUNCATE TABLE pg_temp.hello;

ALTER TABLE pg_temp.hello
DROP CONSTRAINT hello_type_check,
ADD CONSTRAINT hello_type_check CHECK(type = ANY(ARRAY['a']));
```


## Column patterns

- two columns must be null or vice versa (a AND b). A is not null equal B is not null. Alternative is modulo.
- three or more columns must be null or vice versa (create one new column flag to indicate those exists, or better, put them in another table)
- is draft and published at(hint, you only need one)
- active and deleted at (maybe both)
- usecase: publishing products. when is no longer draft, all the columns must be filled, otherwise errors.
- XOR for polymorphic associations, vs table inheritance
- all columns either null (or empty) or filled. We can do the trick by checking the sum of not null modulo length of the column. If we have 5 columns, the modulo of 5 will be zero if we have 0 or 5 empty column.
- xor, one column filled, the other not filled. if there is two column, then A != B. if more than that, sum of not null must be equal 1.


# Row patterns

- min max rows
- at least n row fulfilling condition (usecase: one pending status booking)
- max one row with condition (unique, and partial unique)


## Constraints Usecases


```sql
-- Problem 1: A pair of columns must be null, or not null at the same time.
-- E.g. product price with currency. If a currency is specified, the price must be specified too.
-- If there's no price, the currency should not be there too.
drop table if exists products;
create table if not exists products (
	id int generated always as identity,
	name text not null,

	price int,
	currency varchar(3),

	primary key(id),

	-- Both price and currency must exists together.
    constraint price_with_currency check ((price is null) = (currency is null))
);


-- Problem 2: Two or more columns must either be null, or not null at the same time.
-- E.g. We have additional data associated with a table (they could have been in another table, but without supporting usecases yet, so they are placed in the same table)

drop table if exists idempotency_keys;
create table if not exists idempotency_keys (
	id uuid default gen_random_uuid(),

	idempotency_key text not null,

	-- The three columns below must be either non-null, or null altogether.
	request_params jsonb,
	response_params jsonb,
	status int,

	primary key (id),
	unique (idempotency_key),
	constraint all_or_none check (
		-- ROW(val1, val2, val2) checks if all row is null or otherwise.
		((request_params, response_params, status) is null) or
		((request_params, response_params, status) is not null)
	)
);

-- Alternative, using modulo.
-- 0 % 3 = 0 -- None set.
-- 1 % 3 = 1
-- 2 % 3 = 2
-- 3 % 3 = 0 -- All set.
create table if not exists idempotency_keys (
	id uuid default gen_random_uuid(),

	idempotency_key text not null,

	-- The three columns below must be either non-null, or null altogether.
	request_params jsonb,
	response_params jsonb,
	status int,

	primary key (id),
	unique (idempotency_key),
	constraint all_or_none check (
		-- ROW(val1, val2, val2) checks if all row is null or otherwise.
		num_nonnulls(request_params, response_params, status) % 3 = 0
	)
);


-- Problem 3: XOR columns - For a given pair of columns, if one column is set, the other should not be set.
-- E.g. We have a document that requires approval from either the internal admin, or an external vendor.
-- There could be other way to model this problem, like maybe a single boolean table is sufficient,
-- but we want to store the date it was approved.
drop table if exists document_approvals;
create table if not exists document_approvals (
	id int generated always as identity,

	internal_approved_at timestamptz,
	external_approved_at timestamptz,
	remarks text,

	primary key (id),
	constraint either_internal_or_external_only check (
		(internal_approved_at is null) != (external_approved_at is null)
		-- Alternative
		-- num_nonnulls(internal_approved_at, external_approved_at) = 1
	)
);


-- Problem 4: XOR columns - but for more than 2 columns.
-- E.g. when implementing polymorphism. We have a social media page, and we want to allow users to like posts, comments, photos or videos.
-- For simplicity, we just wanna place the relations in a single table.
drop table if exists likes;
create table if not exists likes (
	id int generated always as identity,

	-- Only one column should be set.
	post_id int,
	comment_id int,
	photo_id int,
	video_id int,

	primary key(id),
	constraint only_one_selected check (
		(post_id is not null)::int +
		(comment_id is not null)::int +
		(photo_id is not null)::int +
		(video_id is not null)::int = 1
	)
);

-- A better version, using num_nonnulls
create table if not exists likes (
	id int generated always as identity,

	-- Only one column should be set.
	post_id int,
	comment_id int,
	photo_id int,
	video_id int,

	primary key(id),
	constraint only_one_selected check (
		num_nonnulls(post_id, comment_id, photo_id, video_id) = 1
	)
);
```

## Constraints Trigger

We can set more advance constraints using triggers

- freeze rows once certain status is achieved
- using WHEN on trigger to check if the primary key is changed
- trigger per column change
- limit number of rows 


The advantage of using trigger vs check constraints is that we can disable/enable it when required.

