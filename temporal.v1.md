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
	foreign key (product_id) references products(id)
);

insert into products(name) values ('chair');
insert into product_prices (product_id, price) values (1, 100);
insert into product_prices (product_id, price) values (1, 250);
insert into product_prices (product_id, price, effective_at) values (1, 175, now() + interval '1 hour');
insert into product_prices (product_id, price, effective_at) values (1, 200, timestamptz 'tomorrow');

select timestamptz 'yesterday';
select timestamptz 'today';
select timestamptz 'tomorrow';
select now() + interval '1 hour';

select * from product_prices;
drop view product_price_history;
create or replace view product_price_history as (
	select
		id,
		product_id,
		price,
		tstzrange(
			effective_at, -- start date
			lead(effective_at, 1) over (order by effective_at asc), -- end_date
			'[)'
		) as active_period,
		created_at
	from product_prices
	order by effective_at asc
);


select *, active_period @> now() as is_current
from product_price_history
where product_id = 1
order by active_period asc
-- and active_period @> timestamptz 'tomorrow';
-- and active_period @> now();
;
```
