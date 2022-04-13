# MySQL

Store the uuids as order binary ids to save space and improve performance, as well as sortability.

# General

- use uuid for generated rows
- use int id for reference table (countries, currency etc), but mask the id when returning to client (e.g. using uuid v5)

## Postgres

The most common question that a database designer will ask themselves over and over - integer id or uuid? I've seen common solution like

- use uuid as primary key (always the best choice for UGC)
- use int id for entries that don't grow fast (e.g. products table, you probably won't have millions of products), but then hash the id using solutions like hash id (not preferable, uuid v5 is probably much better, since you can generated it in SQL)
- create two columns, one for int key and another for uuid


For the last scenario, if you choose that, use the uuid column as the primary key, not the other way around. There are a few advantages

- it's easier to read the int id (hey can you take a look at product id 44)
- the int column is unique and easily sortable too (unlike timestamp, where you can create products in a transaction and all of them will have the same sort key)
- easier to do pagination

The design decision is not purely technical, but based more on _empathy_. Prefer a solution that is kind to your users.


```sql
create table if not exists users (
	id uuid default gen_random_uuid(),
	
	name text not null,
	sort int generated always as identity,
	
	primary key(id)
);

insert into users (name) values 
('alice'),
('bob');

table users;

id	name	sort
df8c515a-ec2a-483e-83e0-a3ce68d56add	alice	1
3a1d641a-5a31-4e5f-b895-ad955eb351df	bob	2
;
```


