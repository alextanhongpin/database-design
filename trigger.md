# Postgres Triggers

- triggers attached to tables are removed when the table is dropped, so we do not need to include it in the migration down file

## Finding all Database Triggers


```sql
select event_object_schema as table_schema,
       event_object_table as table_name,
       trigger_schema,
       trigger_name,
       string_agg(event_manipulation, ',') as event,
       action_timing as activation,
       action_condition as condition,
       action_statement as definition
from information_schema.triggers
group by 1,2,3,4,6,7,8
order by table_schema,
         table_name;
```

## Example Trigger

```sql
CREATE TABLE users (
	username text NOT NULL PRIMARY KEY
);

-- An audit log.
CREATE TABLE audit_log (
	at timestamptz NOT NULL DEFAULT now(),
	description text NOT NULL
);

-- The actual function that is executed per insert.
CREATE FUNCTION on_user_added() RETURNS TRIGGER AS $$
BEGIN
	IF (TG_OP = 'INSERT') THEN
		-- Add an entry into the audit log.
		INSERT INTO audit_log(description)
			VALUES ('new user created, username is ' || NEW.username);

		-- Sends a notification.
		PERFORM pg_notify('usercreated', NEW.username);
	END IF;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Set the function as an insert trigger.
CREATE TRIGGER on_user_added
AFTER INSERT ON users
FOR EACH ROW 
EXECUTE PROCEDURE on_user_added();

INSERT INTO users VALUES ('car');

LISTEN usercreated;
NOTIFY usercreated, 'hello';
UNLISTEN usercreated;
```


## Trigger to update timestamp
	
__Use Case:__ Allowing flexibility to update timestamp is essential, especially when working on moderations/internal admin tasks. Say we have a comment website where admins can moderate by editing comments/correcting grammar/removing promo codes or links etc. If the comments are sorted by updated at date, then whenever the admin modify the comment, the date would be updated. The date should not be updated in that case, and therefore we have to disable the trigger. The same goes when there is a need to add new column and populating the data - the updated date should not be modified.

We can disable the update timestamp trigger by setting the `session_replication_role` to `local`. This is useful for bulk updating data, and leaving the timestamp untouched.

```sql
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
	IF current_setting('session_replication_role') = 'local' THEN
		RETURN NEW;
	END IF;

	NEW.updated_at = now();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

This is how you change the session replication role. The default session replication role is `origin`.
```sql
SHOW session_replication_role; -- Defaults to origin

BEGIN;
SET LOCAL session_replication_role = 'local';
UPDATE tag SET name = 'hello name';
COMMIT;
```

Alternatively, we can set the condition during trigger. Note that the disadvantage is that we have to define it every single time. Defining the condition in function is easier, since it is applied to all triggers (this can be pros or cons):

```sql
CREATE TRIGGER update_timestamp
BEFORE UPDATE ON table_name
FOR EACH ROW
WHEN (current_setting('session_replication_role') <> 'local')
EXECUTE PROCEDURE update_timestamp();
```

Alternatively, we can also use a custom config name, e.g. `application_name = skiptrig`.

References:
https://www.endpoint.com/blog/2015/07/15/selectively-firing-postgres-triggers

