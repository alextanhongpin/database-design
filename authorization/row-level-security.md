## Row Level Security

Setup role level security with external data:

```sql
--https://www.postgresql.org/docs/current/sql-createrole.html
create role app_authenticator with login password 'password here' noinherit;
create role app_visitor;
-- Grant the group role of app_visitor to app_authenticator.
grant app_visitor to app_authenticator;
revoke app_visitor from app_authenticator;

-- CRUD roles.
CREATE ROLE anon;
SET ROLE anon;
RESET ROLE;
SHOW ROLE;
DROP ROLE anon;

-- Starting as superuser.
RESET role;
CREATE TABLE IF NOT EXISTS comments (
	id serial PRIMARY KEY,
	user_id int NOT NULL,
	body text NOT NULL
);
DROP TABLE comments CASCADE;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;

INSERT INTO comments (user_id, body) 
VALUES 
(1, 'hello world'),
(2, 'hello world');

-- Superuser can view all.
SELECT * FROM comments;

CREATE OR REPLACE FUNCTION current_user_id() RETURNS integer AS $$
  select nullif(current_setting('my.user_id', true), '')::integer;
$$ language sql stable;

-- Add an update only policy for owners.
CREATE POLICY update_if_owner
  ON comments
  FOR update
  USING ("user_id" = current_user_id())
  WITH CHECK ("user_id" = current_user_id());

DROP POLICY update_if_owner ON comments;

-- Add a read only policy for owners.
CREATE POLICY read_if_owner
ON comments
FOR SELECT
USING (user_id = current_user_id());

DROP POLICY read_if_owner ON comments;

  
-- Create and set role to anon;
CREATE ROLE anon;
SET ROLE anon;

-- Anon cannot access any tables yet.
TABLE comments;


-- Switch back to superuser to create rules.
RESET ROLE;
GRANT
	SELECT,
	INSERT (body, user_id),
	UPDATE (body),
	DELETE
ON comments 
TO anon;

-- Now, switch back to anon;
SET ROLE anon;
SHOW ROLE;

-- Can access, but not visible due to row level permissions.
TABLE comments;


-- Setting user id per transaction.
BEGIN;
	SET LOCAL my.user_id TO 1;
	TABLE comments;
COMMIT;

BEGIN;
	SET LOCAL my.user_id TO 2;
	TABLE comments;
COMMIT;

-- This does not work...
WITH user_config AS (
	SELECT set_config('my.user_id', '1', true)
)
TABLE comments;

-- Cannot update unless owner is set.
UPDATE comments SET body = 'haha';

BEGIN;
	SET LOCAL my.user_id TO 2;
	UPDATE comments 
	SET body = 'edited'
	RETURNING *;
COMMIT;
```

## Advance

How to prevent users for abusing the connection and overriding the row-level-security for those who have SQL access:

https://www.2ndquadrant.com/en/blog/application-users-vs-row-level-security/
