# Date

## Timestamps

It is common to have the following columns in your db:

```sql
created_at timestamptz not null,
updated_at timestamptz not null,
deleted_at timestamptz
```

The `deleted_at` is optional is commonly used to implement soft-delete.

## Valid from

Sometimes you want to flag a row as valid only from a certain time that is different from the created at. For example, a blog post is only visible once it is published:

```sql
create table posts (
    -- REDACTED
    published_at timestamptz    
)
```

This is preferable over setting a boolean flag `active/is_active`, since we also have information about the date.

## Valid till

For rows that have a limited lifespan, you can add a date to mark the end date. For example

- a campaign is only active until the end of the month

- ticket sales ends at the certain period

- invoice expires after 2 hour

For the last example particularly, it is preferable to store the absolute time rather than the duration (e.g. 2 hours). Snapshotting final value is an important concept to avoid recomputing them again (doing so means the database is business-logic aware, which should be avoided since clients may interpret and implement them differently).

## Active Period

When we combined valid from and valid till, we have an active period. This is commonly used in temporal table to mark validity of the row data. 



Other examples includes:

- appointment/booking systems with start and end date

- slowly changing dimension (SDC) like product prices etc

- campaigns/banners/promos that lasts



The active period can have different behaviours for different applications. For example, we may want it to be 

- continuous, from the beginning of time: for marking a person's address since birth for example.

- continuous, from the date it is created: for product prices. There cannot be a gap between the date the price changes, and hence continuous.

- overlapping: for calendar application. You may have multiple events that is happening at the same time or period

- non-overlapping, with gaps: for marking events



## History

History/audit tables are another example of temporal tables that is continuous. If the entries is strictly continuous, then using a single timestamp column is enough to indicate that. A view can be created to construct the active period, to make querying simpler. There can be two views

-  one view for displaying the full history

- another view for displaying the present active row



If we don't need control over the dates, then we can just create two separate tables, one to store the original, another to capture the modification of the rows.

## Schedule

The difference between this and a history table is that the data in the history table is immutable - they are created through triggers whenever the row is created/updated or deleted, and hence the time will always be the same as the transaction time.



Tables that requires scheduling however allows the client to specify the date range, and depending on the usecase, it can be for a date range in the past or future.



We can use either 

- two columns to indicate start and end date

- one tstzrange column, which is better since it also captures information about whether the start/end is inclusive or exclusive, something that two timestamp columns couldn't





```sql
-- Product price depends on multiple variables.
create type product_price_variables as (
    min_quantity int,
    country text,
    supplier text
);

create table product_price_tiers (
    id int generated always as identity,
    product_id int not null,
    variables product_price_variables not null,
    valid_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    primary key (id),
    foreign key (product_id) references products(id),
    unique (product_id, variables)
);
```






