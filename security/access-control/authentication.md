## Performing authentication in Postgres

Pros:
- language independent
- business logic will always be applied (no tampering, no forgetting to apply encryption when using triggers)
- reusable module when starting new projects, don't have to look for encryption library and redoing them
- almost static business logic

Cons:
- can slow the db server (high CPUs)
- global single point of failure
- poor documentation? (triggers are magical)
- when business rules requires changes, there might be complexity (mostly FUD)
- passing the password to db may cause it to be logged as plaintext (?)

Thoughts:
- the main difference is, do you want to put the business logic on application layer or database layer?


```sql
-- Required for encryption functions.
CREATE EXTENSION pgcrypto;

CREATE TABLE users (
	email text not null unique,
	encrypted_password text not null
);
DROP TABLE users;
```

Without triggers (using plain functions):
```sql
WITH inserted_user AS (
	INSERT INTO users (email, encrypted_password) 
	VALUES ('alice@mail.com', crypt('12345678', gen_salt('bf', 12)))
	RETURNING *
)
SELECT 
	-- Convert the payload to json.
	-- Remove the field "encrypted_password" from the json object.
	-- Add a field "event" with value "created" to the json object.
	pg_notify('user_created', (row_to_json(inserted_user)::jsonb - 'encrypted_password' || jsonb '{"event": "created"}')::text),
	inserted_user.email
FROM inserted_user;
```

Pros of the approach above:
- flexible, can bypass auth mechanism if we need to apply a new one (e.g. using argon2)
- if there's a need, can always fallback to encryption on application layer

Cons of the approach above:
- user may forget to apply the encryption step


Let's use a trigger instead, which will encrypt the password when
- a new user is created
- when user updates the password 
  - and the password is not the same as the old password
  
```sql
CREATE OR REPLACE FUNCTION encrypt_password()
  RETURNS TRIGGER AS
$$
DECLARE
BEGIN
	IF OLD.encrypted_password = crypt(NEW.encrypted_password, OLD.encrypted_password) THEN
		RAISE EXCEPTION 'New password cannot be the same as old password'
		USING hint = 'Please use a different password';
	END IF;
	
	
	-- When updating, we do not want to encrypt the password again, unless it has change.
	-- The NULLIF checks if the new and old hash is the same, and returns null if they are.
	-- If the hash is not the same, then the new password is a plaintext, which we want to encrypt.
	IF NULLIF(NEW.encrypted_password, OLD.encrypted_password) IS NOT NULL THEN
		-- Encryption with blowfish algorithm, with cost set as 12 (better?) 2 ^ 12 = 4096 iterations.
		-- There's limitation when using bf, the max password length is 72 characters.
		-- To slow down the process even more, set a timeout at application level HAHA.
		NEW.encrypted_password := crypt(NEW.encrypted_password, gen_salt('bf', 12));
	END IF;
 	RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER encrypt_password
BEFORE INSERT OR UPDATE ON users
FOR EACH ROW
EXECUTE PROCEDURE encrypt_password();

DROP TRIGGER encrypt_password ON users;
```

The query is now free from the implementation details:
```sql
WITH inserted_user AS (
	INSERT INTO users (email, encrypted_password) 
	VALUES ('a@mail.com', '12345678')
	returning *
)
SELECT pg_notify('user_created', (row_to_json(inserted_user)::jsonb - 'encrypted_password' || jsonb '{"event": "created"}')::text), *
FROM inserted_user;
```

To find users with the given password:
```sql
-- This actually finds all the user with that password. HAHA sucks.
SELECT * FROM users WHERE encrypted_password = crypt('12345678', encrypted_password);
```

Find by email first (else the db will be doing a lot of work encrypting):
```sql
SELECT * 
FROM users 
WHERE email = 'a@mail.com'
AND encrypted_password = crypt('12345678', encrypted_password);
```

Updating the password will trigger the encryption too:
```sql
UPDATE users 
SET encrypted_password = '123456789'
WHERE email = 'a@mail.com';
```

When updating with the old password:
```
ERROR:  New password cannot be the same as old password
HINT:  Please use a different password
CONTEXT:  PL/pgSQL function encrypt_password() line 5 at RAISE
```
