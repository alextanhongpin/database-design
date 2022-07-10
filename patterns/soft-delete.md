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
- if there are unique constraint, then when handling constraint, we need to set deleted at to null (or set partial unique constraints on where deleted_at is null). NOTE: When handling on conflict with partial constraints, we also need to specify the condition set for the partial unique constraint
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

## Using triggers for soft delete (as opposed to rule)

Create a user table:
```sql
CREATE TABLE users (  
  id        serial PRIMARY KEY,
  username  text NOT NULL
);

INSERT INTO users (username) VALUES ('sean'), ('sam'), ('doug');  
```

Create a deleted at column, and also index the deleted at column only if it is not null to speed up queries:
```
ALTER TABLE users ADD COLUMN deleted_at timestamptz;
CREATE INDEX not_deleted ON users WHERE deleted_at IS NULL;
```

```sql
CREATE FUNCTION soft_delete()  
  RETURNS trigger AS $$
    DECLARE
      command text := ' SET deleted_at = current_timestamp WHERE id = $1';
    BEGIN
      EXECUTE 'UPDATE ' || TG_TABLE_NAME || command USING OLD.id;
      RETURN NULL;
    END;
  $$ LANGUAGE plpgsql;
```

```sql
CREATE TRIGGER soft_delete_user  
  BEFORE DELETE ON users
  FOR EACH ROW EXECUTE PROCEDURE soft_delete();
```

Create view to query only non-deleted users:

```sql
CREATE VIEW users_without_deleted AS  
  SELECT * FROM users WHERE deleted_at IS NULL;
```


Alter table and views name (since we will be working with `users` table without soft deleted users most of the time):

```sql
ALTER TABLE users RENAME TO users_with_deleted;  
ALTER VIEW users_without_deleted RENAME TO users;  
```

Remove trigger from previous table and assigned it to the `users_with_deleted`:
```sql
ALTER TABLE users_with_deleted DROP TRIGGER soft_delete_user;

CREATE TRIGGER soft_delete_user  
  INSTEAD OF DELETE ON users
  FOR EACH ROW EXECUTE PROCEDURE soft_delete();
```

This means now we can
1. soft delete users on `users` table
2. perform hard delete users on `users_with_deleted` table

What if we want to add a `unique` index to username, but only for the non-deleted ones?

```sql
CREATE UNIQUE INDEX unique_username ON users_with_deleted (username) WHERE deleted_at IS NULL;
```

tldr

```sql
CREATE TABLE users_with_deleted (  
  id          serial PRIMARY KEY,
  username    text NOT NULL,
  deleted_at  timestamptz
);

CREATE INDEX not_deleted ON users_with_deleted WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX unique_username ON users_with_deleted (username) WHERE deleted_at IS NULL;

CREATE VIEW users AS  
  SELECT * FROM users_with_deleted WHERE deleted_at IS NULL;

CREATE FUNCTION soft_delete()  
  RETURNS trigger AS $$
    DECLARE
      command text := ' SET deleted_at = current_timestamp WHERE id = $1';
    BEGIN
      EXECUTE 'UPDATE ' || TG_TABLE_NAME || command USING OLD.id;
      RETURN NULL;
    END;
  $$ LANGUAGE plpgsql;

CREATE TRIGGER soft_delete_user  
  INSTEAD OF DELETE ON users
  FOR EACH ROW EXECUTE PROCEDURE soft_delete();
```

References:
- http://shuber.io/porting-activerecord-soft-delete-behavior-to-postgres/
