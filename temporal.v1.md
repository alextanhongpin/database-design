# Temporal with single date column


- History table captures only events that happened in the past.
- In short, we are not allowed to modify the past.
- We can only add new events.
- effective at is the date when the fact is valid.
- If we add new entries that are effective in the future, as long as it has not happened yet, we can delete it.
- There is no concept of delete in history, we can however set the entry to `null` as a tombstone entry, to indicate it is no longer effective. We can however resume it in the future.
- For entries that requires correction in the past, use [facts table](temporal.facts.md).
- If the previous active value is the same as the new value, we should skip the update.
- Note the boundary, `[)` the `)` is exclusive.


```sql
drop table if exists products;
create table if not exists products (
	id int generated always as identity,

	name text not null,
	primary key (id)
);

drop table if exists product_prices cascade;

-- TODO: Create trigger to disallow editing once the the current time is greater than the effective at date, since price that is already active should not longer be editable.
create table if not exists product_prices (
	id int generated always as identity,

	product_id int not null,
	price int not null, -- If we want to allow

	effective_at timestamptz not null default now(),
	-- See below if you need to prepopulate the data in the past.
	-- effective_at timestamptz not null default now() CHECK (effective_at >= now()),
	created_at timestamptz not null default now(),

	primary key (id),
	foreign key (product_id) references products(id),
	unique (product_id, effective_at)
);

-- Seed data.
insert into products(name) values ('chair');
insert into product_prices (product_id, price) values (1, 100);
insert into product_prices (product_id, price) values (1, 250);
insert into product_prices (product_id, price, effective_at) values (1, 175, now() + interval '1 hour');
insert into product_prices (product_id, price, effective_at) values (1, 200, timestamptz 'tomorrow');

insert into products(name) values ('table');
insert into product_prices (product_id, price) values (2, 100);
insert into product_prices (product_id, price) values (2, 250);
insert into product_prices (product_id, price, effective_at) values (2, 175, now() + interval '1 hour');
insert into product_prices (product_id, price, effective_at) values (2, 200, timestamptz 'tomorrow');


-- Refresher on timestamptz operations.
select timestamptz 'yesterday';
select timestamptz 'today';
select timestamptz 'tomorrow';
select now() + interval '1 hour';


-- Once we seeded data in the past, we can add the constraint to allow only future data.
-- The `not valid` does not check against past data, but new data be validated.
alter table product_prices add constraint allow_only_future_events check (effective_at >= now()) not valid;
-- This will now fail.
insert into product_prices (product_id, price, effective_at) values (1, 175, timestamptz 'yesterday');


select * from product_prices;

select
	id,
	product_id,
	price,
	effective_at,
			lead(effective_at, 1) over (partition by product_id order by effective_at asc), -- end_date
		tstzrange(
			effective_at, -- start date
			lead(effective_at, 1) over (partition by product_id order by effective_at asc), -- end_date
			'[)'
		) as active_period
from product_prices;

drop view product_price_history;
create or replace view product_price_history as (
	select
		id,
		product_id,
		price,
		tstzrange(
			effective_at, -- start date
			lead(effective_at, 1) over (partition by product_id order by effective_at asc), -- end_date
			'[)'
		) as active_period,
		created_at
	from product_prices
	order by effective_at asc
);


select * from product_price_history;
select *, active_period @> now() as is_current, (id, product_id, price) is not distinct from (5, 2, 100)
from product_price_history
where product_id = 2
order by active_period asc;

-- We do not want to insert the new values if the value is the same as the current active price.
CREATE OR REPLACE FUNCTION skip_insert_if_unchanged() RETURNS TRIGGER AS $$
DECLARE
	unchanged bool;
BEGIN
	SELECT price = NEW.price
	INTO unchanged
	FROM product_price_history
	WHERE product_id = NEW.product_id
	AND active_period @> NEW.effective_at;

	IF unchanged THEN
		RAISE EXCEPTION 'product_id % with price % already exists at %', NEW.product_id, NEW.price, NEW.effective_at
		USING HINT = 'Price exists at the given effective period';
		-- RETURN NULL;
	END IF;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER skip_insert_if_unchanged
	BEFORE INSERT ON product_prices
	FOR EACH ROW
	EXECUTE PROCEDURE skip_insert_if_unchanged();

CREATE OR REPLACE FUNCTION freeze_product_price() RETURNS TRIGGER AS $$
BEGIN
	RAISE EXCEPTION 'Cannot perform % on table product_prices', TG_OP
	USING HINT = 'Run ALTER TABLE DISABLE TRIGGER freeze_product_price()';
	RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER freeze_product_price
	BEFORE UPDATE OR DELETE ON product_prices
	FOR EACH ROW
	EXECUTE PROCEDURE freeze_product_price();

```

## With Multiple Constraints


Similar to the above, but with compound primary key.

```sql
--- With multiple constraints

drop table if exists product_price_tiers cascade;

-- TODO: Create trigger to disallow editing once the the current time is greater than the effective at date, since price that is already active should not longer be editable.
create table if not exists product_price_tiers (
	id int generated always as identity,

	product_id int not null,
	min_quantity int not null,
	price int, -- Price is set to be nullable, which means it's deleted

	effective_at timestamptz not null default now() CHECK (effective_at >= now()), -- Can only insert future data
	created_at timestamptz not null default now(),

	primary key (id),
	foreign key (product_id) references products(id),
	unique (product_id, min_quantity, effective_at)
);


insert into products(name) values ('chair');
insert into product_price_tiers (product_id, min_quantity, price, effective_at) values
	(1, 1, 100, clock_timestamp()),
	(1, 1, 250, clock_timestamp()),
	(1, 1, 175, now() + interval '1 hour'),
	(1, 1, 200, timestamptz 'tomorrow'),
	(1, 1, null, now() + interval '2 hour'),
	(1, 1, 250, now() + interval '3 hour'),
	(1, 2, 100, clock_timestamp()),
	(1, 2, 250, clock_timestamp()),
	(1, 2, 175, now() + interval '1 hour'),
	(1, 2, 200, timestamptz 'tomorrow'),
	(2, 1, 100, clock_timestamp()),
	(2, 1, 250, clock_timestamp()),
	(2, 1, 175, now() + interval '1 hour'),
	(2, 1, 200, timestamptz 'tomorrow'),
	(2, 2, 100, clock_timestamp()),
	(2, 2, 250, clock_timestamp()),
	(2, 2, 175, now() + interval '1 hour'),
	(2, 2, 200, timestamptz 'tomorrow');


select * from product_price_tiers;

select
	id,
	product_id,
	min_quantity,
	price,
	effective_at,
	lead(effective_at, 1) over (partition by product_id, min_quantity order by effective_at asc), -- end_date
	tstzrange(
		effective_at, -- start date
		lead(effective_at, 1) over (partition by product_id, min_quantity order by effective_at asc), -- end_date
		'[)'
	) as active_period
from product_price_tiers;

drop view product_price_history;
create or replace view product_price_tier_history as (
	select
		id,
		product_id,
		price,
		min_quantity,
		tstzrange(
			effective_at, -- start date
			lead(effective_at, 1) over (partition by product_id, min_quantity order by effective_at asc), -- end_date
			'[)'
		) as active_period,
		created_at
	from product_price_tiers
	order by effective_at asc
);



select * from product_price_tier_history;

select *, active_period @> now() as is_current, price is null as inactive
from product_price_tier_history
where
	product_id = 1
	and min_quantity = 1;


-- How do we represent deleted at?
-- Allow setting the price to null, and filtering it later. So this creates a tombstone.


create or replace function freeze_past_product_price () returns trigger as $$
begin
	IF OLD.effective_at < now() THEN
		RAISE EXCEPTION 'ProductPrice<id=%, price=%, min_quantity=%, effective_at=%> is effective', OLD.id, OLD.price, OLD.min_quantity, OLD.effective_at
		USING HINT = 'Cannot change product price that is already effective';
		RETURN NULL;
	END IF;

	IF TG_OP = 'DELETE' THEN
		RETURN OLD;
	END IF;

	RETURN NEW;
end;
$$ language plpgsql;

CREATE TRIGGER freeze_past_product_price
BEFORE UPDATE OR DELETE ON product_price_tiers
FOR EACH ROW
EXECUTE PROCEDURE freeze_past_product_price();

select * from product_price_tiers;
delete from product_price_tiers where id = 2;
```
