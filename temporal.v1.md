# Temporal with single date column


```sql
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
	price int not null,

	effective_at timestamptz not null default now(),
	created_at timestamptz not null default now(),

	primary key (id),
	foreign key (product_id) references products(id),
	unique (product_id, effective_at)
);

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

select timestamptz 'yesterday';
select timestamptz 'today';
select timestamptz 'tomorrow';
select now() + interval '1 hour';

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
select *, active_period @> now() as is_current
from product_price_history
where product_id = 1
order by active_period asc
-- and active_period @> timestamptz 'tomorrow';
-- and active_period @> now();
;

--- With multiple constraints

drop table if exists product_price_tiers cascade;

-- TODO: Create trigger to disallow editing once the the current time is greater than the effective at date, since price that is already active should not longer be editable.
create table if not exists product_price_tiers (
	id int generated always as identity,

	product_id int not null,
	min_quantity int not null,
	price int, -- Price is set to be nullable, which means it's deleted

	effective_at timestamptz not null default now(),
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

```
