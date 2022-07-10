# Useful validation patterns

## Using `create domain` in postgres

https://begriffs.com/posts/2017-10-21-sql-domain-integrity.html

## Using `check`
http://shuber.io/porting-activerecord-validations-to-postgres/


## Email Validation
Regex is from [emailregex.com[(https://emailregex.com/):

```sql
CREATE DOMAIN email AS text CHECK(value ~* $$(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\])$$);
DROP DOMAIN email;

-- Invalid.
select 'mail.com'::email;
select email('mail.com');

-- Valid.
select 'john.doe@mail.com'::email;
```

## Safely migration domain constraints

Say if we already have a domain type, and we want to update the constraints without downtime, or without applying it to the old values (Explanation: there are old data in the table that does not fulfill the constraint, and we want to ignore them, but at the same time apply the new constraints on newly entered data). 

```sql
-- Check the value matches the regex, which means only alphanumeric, space, dot and dash is allowed.
-- ~* means case-insensitive.
CREATE DOMAIN username AS text CHECK (VALUE ~* '^[a-z0-9 .-]$');
```

With this, we already have an implied constraint check called `username_check`. Let's test it:

```sql
-- Type casting.
SELECT 'john'::username;

-- Like a function.
SELECT username('john');
```

Unfortunately, we forgot to allow more than one character, so the constraint is actually broken. Let's update it.

```sql
-- This will create another constraint username_check1, if the naming is important, use the method below to define new constraint name.
-- NOT VALID means the constraint does not apply on old values, only new values. This way, we can safely apply the new constraints on new values only.
ALTER DOMAIN username ADD CHECK (VALUE ~* '^[a-z0-9 .-]+$') NOT VALID;

-- Alternative: Adding a constraint, but with the flexibility of providing a name.
ALTER DOMAIN username ADD CONSTRAINT username_check_new CHECK (VALUE ~* '^[a-z0-9]+$') NOT VALID;
```

You can check the constraints for the given domain with this query:
```sql
SELECT conname
FROM pg_constraint
WHERE contypid = 'username'::regtype;
```

Output:
```
username_check
username_check1
```

As you can see, we now have two constraints applied for the `username` domain. Let's remove the old constraint.

Dropping old constraints:
```sql
ALTER DOMAIN username DROP CONSTRAINT username_check;
```

Let's also rename the new constraint (optional):
```sql
ALTER DOMAIN username RENAME CONSTRAINT username_check1 TO username_check;
```

This will now work correctly:
```sql
SELECT username('john');
SELECT username('john*');
```

Using on a table:
```sql
CREATE TABLE IF NOT EXISTS users (
  id serial PRIMARY KEY,
  name username not null
);
```

## Adding error messages using comment

The error message shown when the constraint is not valid is not that helpful. There are several ways to improve the error message. The example below uses the comment:
```sql
CREATE DOMAIN user_password AS text CHECK(LENGTH(VALUE) > 7);
COMMENT ON DOMAIN user_password IS 'password must be 8 characters';

CREATE OR REPLACE FUNCTION check_user_password( cmd TEXT ) RETURNS user_password AS $$
DECLARE
  dom text;
  friendly text;
  retval user_password;
BEGIN
  -- attempt to run original command
  select user_password(cmd) INTO retval;
  RETURN retval;
EXCEPTION WHEN check_violation THEN
  -- extract the relevant data type from the exception
  GET STACKED DIAGNOSTICS dom = PG_DATATYPE_NAME;

  -- look for a user comment on that type
  SELECT pg_catalog.obj_description(oid)
    FROM pg_catalog.pg_type
   WHERE typname = dom
    INTO friendly;

  IF friendly IS NULL THEN
    -- if there is no comment, throw original exception
    RAISE;
  ELSE
    -- otherwise throw a revised exception with better message
    RAISE check_violation USING message = friendly;
  END IF;
END;
$$ language plpgsql;
```

Test:
```sql
SELECT check_user_password('12');
```

Output:
```
ERROR:  password must be 8 characters
CONTEXT:  PL/pgSQL function check_user_password(text) line 25 at RAISE
```


## Thoughts on using Custom Domain on Table datatype

- does the language support casting the type to the primitives? If we create the `username` type, does it get converted back to type `text`?
- are there complexity in updating the constraints? (no, but requires proper version control, as well as documentation)
- we don't have to use the `domain` type in tables, we can always just use it for validation by casting the types back, e.g. `select 'text'::username`

## Getting all UDF (user-defined functions)

```sql
SELECT * 
FROM information_schema.routines 
WHERE routine_type='FUNCTION' 
  AND specific_schema='public'
  AND data_type = 'USER-DEFINED'
  AND external_language = 'PLPGSQL';
```

## Better error message with custom domain type

```sql
CREATE OR REPLACE FUNCTION check_username(name TEXT) RETURNS username AS $$
DECLARE
  result username;
BEGIN
  SELECT name::username INTO result;
  RETURN result;
EXCEPTION WHEN check_violation THEN
--	RAISE SQLSTATE '23514'
--	RAISE check_violation
	RAISE
		USING HINT = 'Please check your username for invalid characters',
		MESSAGE=format('Invalid username: %s', name),
		DETAIL=format('name %I contains invalid character', name);

END;
$$ LANGUAGE plpgsql;
```

Test:
```sql
select check_username('johnhleo*');
```

Output:
```sql
ERROR:  Invalid username: johnhleo*
DETAIL:  name "johnhleo*" contains invalid character
HINT:  Please check your username for invalid characters
CONTEXT:  PL/pgSQL function check_username(text) line 10 at RAISE
```
