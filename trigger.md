# Postgres Triggers

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
