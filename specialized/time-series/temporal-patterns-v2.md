# Temporal with two columns.

```sql
create table if not exists products (
	id int generated always as identity,
	name text not null,

	primary key (id)
);

select now();

create extension btree_gist;
drop table if exists product_prices;
create table if not exists product_prices (
	id int generated always as identity,

	price int not null,
	active_period tstzrange not null,

	product_id int not null,

	primary key(id),
	foreign key(product_id) references products(id),
	-- No overlapping active period for the same product.
	-- Deferrable allows execution of the constraint after one or more rows are inserted.
	EXCLUDE USING gist (product_id WITH =, active_period WITH &&) DEFERRABLE INITIALLY DEFERRED
);

-- We first use this to check continuous tstzrange.
-- We will create a custom aggregate function later.
create or replace function check_continuous_tstzrange() returns trigger as $$
declare
	_product_id int;
begin
	IF TG_OP = 'DELETE' THEN
		_product_id = OLD.product_id;
	ELSE
		_product_id = NEW.product_id;
	END IF;

	-- Skip if price is unchanged.
	-- Don't allow past price to be inserted.

	if not (select every(matched) from (
		select
			tstzrange(
				lower(active_period),
				coalesce(
					lower(
						lead(active_period, 1) over (partition by product_id order by active_period)
					),
					upper(active_period)
				),
				'[)'
			) = active_period as matched
		from product_prices
		where product_id = _product_id
		order by active_period
	) t) THEN
		RAISE EXCEPTION 'tstzrange not continuous';
	END IF;
	RETURN NULL;
end;
$$ language plpgsql;


select
	tstzrange(
		lower(active_period),
		coalesce(
			lower(
				lead(active_period, 1) over (partition by product_id order by active_period)
			),
			upper(active_period)
		),
		'[)'
	) = active_period as matched
from product_prices
where product_id = 1
order by active_period;

drop trigger check_continuous_tstzrange on product_prices;

create constraint trigger check_continuous_tstzrange
	after insert or update or delete on product_prices
	-- For batch operations, it only executes once.
	-- This only works for `constraint` trigger + after statement.
	-- It won't work with before statement.
	deferrable initially deferred
	for each row
	execute procedure check_continuous_tstzrange();


insert into products(name) values ('table'), ('chair');
table products;
table product_prices;

insert into product_prices(product_id, price, active_period) values
	(1, 100, tstzrange(timestamptz 'yesterday', now(), '[)')),
	(1, 200, tstzrange(now(), timestamptz 'tomorrow', '[)'))
;
insert into product_prices(product_id, price, active_period) values
(1, 200, tstzrange(timestamptz 'tomorrow', null, '[)'));

truncate table product_prices;

select *
from product_prices
order by product_id, active_period;
delete from product_prices where id = 2;

with new_data as (
	select
		timestamptz 'tomorrow' + interval '20 hour' as effective_at,
		250 as target_price,
		1 as target_product_id
),
old_data as (
	select
		id as old_id,
		lower(active_period) as lower_active_period,
		upper(active_period) as upper_active_period,
		t.*
	from product_prices, lateral (select * from new_data) t
	where product_id = t.target_product_id
	and active_period @> t.effective_at
	and price <> t.target_price
),
update_old as (
	update product_prices
	set active_period = tstzrange(
		t.lower_active_period,
		t.effective_at,
		'[)'
	)
	from (select * from old_data) t
	where id = t.old_id
	returning product_prices.*
),
insert_new as (
	insert into product_prices (product_id, price, active_period)
	select target_product_id, target_price, tstzrange(effective_at, upper_active_period, '[)')
	from old_data
	returning *
)
select * from update_old
union
select * from insert_new;
```

## Custom aggregate function for continuous tstzrange.

```sql
create type continuous_tstzrange AS (
	value tstzrange,
	valid bool
);
select (tstzrange(now(), null, '[)'), true)::continuous_tstzrange;


-- sfunc is the state transition function.
-- This is executed for each row.
-- It is similar to JavaScript's reduce function.
-- If the state is null, it is initialized with the first row.
-- Otherwise, we check if the new tstzrange is continuous with the previous one.
-- If the previous result is invalid, we exit.
create or replace function tstzrange_adjacent_sfunc(agg_state continuous_tstzrange, el tstzrange)
	returns continuous_tstzrange
	immutable
	language plpgsql
	as $$
begin
	-- RAISE NOTICE 'got state %', agg_state;
	IF agg_state IS NULL THEN
		RETURN (el, true)::continuous_tstzrange;
	ELSE
		IF (agg_state).valid AND (agg_state).value -|- el THEN
			RETURN (el, true)::continuous_tstzrange;
		ELSE
			RETURN ((agg_state).value, false)::continuous_tstzrange;
		END IF;
	END IF;
end;
$$;

create function tstzrange_adjacent_finalfunc(agg_state continuous_tstzrange)
returns bool
immutable
strict
language plpgsql
as $$
begin
  return (agg_state).valid;
end;
$$;


create or replace aggregate check_adjacent_tstzrange (tstzrange)
(
    sfunc = tstzrange_adjacent_sfunc,
    stype = continuous_tstzrange,
    finalfunc = tstzrange_adjacent_finalfunc
);


-- This fails when the ordering is not in ascending
select check_adjacent_tstzrange(active_period)
from product_prices
where product_id = 1
-- order by active_period -- custom ordering is not allowed ...
;

-- This works..
-- NOET: Always order by active_period.
select check_adjacent_tstzrange(active_period order by active_period)
from product_prices
where product_id = 1;

-- This works because the ordering is honoured.
with target as (
	select *
	from product_prices
	where product_id = 1
	order by active_period
)
select check_adjacent_tstzrange(active_period)
from target;

select *
from product_prices
where product_id = 1
order by product_id, active_period;
```

We can now use a new trigger with the custom aggregate function to validate the continuous tstzrange.

```sql
create or replace function check_continuous_tstzrange() returns trigger as $$
declare
	_product_id int;
begin
	IF TG_OP = 'DELETE' THEN
		_product_id = OLD.product_id;
	ELSE
		_product_id = NEW.product_id;
	END IF;

	-- Skip if price is unchanged.
	-- Don't allow past price to be inserted.

	if not (
		select check_adjacent_tstzrange(active_period order by active_period)
		from product_prices
		where product_id = _product_id
	) THEN
		RAISE EXCEPTION 'tstzrange not continuous';
	END IF;
	RETURN NULL;
end;
$$ language plpgsql;
```
