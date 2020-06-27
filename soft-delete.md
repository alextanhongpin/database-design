## Using postgres rule to perform soft delete

Disadvantage: needs to be performed for each table.

```sql
CREATE OR REPLACE RULE delete_venue AS
  ON DELETE TO venues DO INSTEAD
    UPDATE venues
    SET active = 'f'
    WHERE venues.venue_id = old.venue_id;
```



## Soft delete vs hard delete
- how does it work for nested entities?

hard delete
- on delete cascade will remove all child entity with the parent id
soft delete
- it will not allow delete (how to handle the error message?)
- soft delete parent will not automatically soft delete the children. Needs to be performed manually in the reverse order
- but normally soft deleting child means there’s a possibility you won’t be able to access the children anymore from the ui
- if there are unique constraint, then when handling constraint, we need to set deleted at to null
- need to add filter for deleted at is null for each queries (solution: use view)
- need to add handling for unique constraint for create

other alternatives?
- hard delete, but copy the row to another table for auditing
- this can be done through listener in the application (not safe), or database triggers


**References**:
http://abstraction.blog/2015/06/28/soft-vs-hard-delete
https://docs.sourcegraph.com/dev/postgresql


## Soft delete in real world

In real world, delete is rarely practical. Say we have a product, iphone in our list of database. Then it is discontinued and we decided not to show them anymore. But we should not delete the product, because it exists. Instead of setting it to deleted at only (which does not provide any context to the users viewing the database), it is better to add a status, e.g. `DISCONTINUED` to indicate that the product is not longer manufactured. Also, if the product has a specific lifespan, we can also provide a more precise daate range (e.g. **valid_from** and **valid_till** columns) in which the product was valid (e.g. events, product pricing, promotions, sales all share this quality). 

From the UI perspective, we have two options:

1. don't show the product anymore
2. show the product with the discontinued label


## Using rule vs trigger

https://www.postgresql.org/docs/8.2/rules-triggers.html#:~:text=A%20trigger%20is%20fired%20for,execute%20its%20operations%20many%20times.


## Soft delete pattern

Why do people use `id SERIAL PRIMARY KEY NOT NULL` when primary key can never be null?

Create a sample table:
```sql
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT NOT NULL
);
```

Create another table with additional `deleted_at` field.
The `INCLUDING ALL` copies all table definitions of the original table to the newly created table. Note that if we change the original table, we need to update this table too.
```sql
CREATE SCHEMA deleted;
CREATE TABLE deleted.posts (
  deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  LIKE posts INCLUDING ALL
)
```

If we want to query both tables (yes, we can join across different schemas):

```sql
CREATE SCHEMA combined;
CREATE VIEW combined.posts AS 
  SELECT null AS deleted_at, * FROM posts
  UNION ALL
  SELECT * FROM deleted.posts;
```


To perform delete, copy the row from the posts table to the deleted posts table and then delete the original (use trigger/rules for this?).

```sql
INSERT INTO deleted.posts
  SELECT NOW() AS deleted_at, * FROM posts
  WHERE posts.id = 2;

DELETE FROM posts
  WHERE posts.id = 2;
```
