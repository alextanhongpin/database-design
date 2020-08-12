## Tracking history changes

This is useful if the item does not change too frequently, else the column will grow large. 

Create a table for testing:
```sql
CREATE TABLE IF NOT EXISTS pg_temp.historical (
	id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	history jsonb NOT NULL DEFAULT '[]'::jsonb,
	validity tstzrange NOT NULL DEFAULT tstzrange(current_timestamp, NULL),
	name text NOT NULL
);
```

Create the trigger to push the old changes with the metadata to the `jsonb` history column:
```sql
CREATE OR REPLACE FUNCTION pg_temp.history() 
RETURNS TRIGGER AS $$
	DECLARE
		validity_changed bool = false;
		is_deleting bool = false;
		is_restored bool = false;
	BEGIN
		SELECT OLD.validity IS DISTINCT FROM NEW.validity INTO validity_changed;
		IF validity_changed THEN
			SELECT upper(OLD.validity) IS NULL AND upper(NEW.validity) IS NOT NULL INTO is_deleting; 
			SELECT upper(OLD.validity) IS NOT NULL AND upper(NEW.validity) IS NULL INTO is_restored; 
		END IF;
			
		-- If the item is no longer valid (the current timestamp is not between the lower and upper bound of validity)
		-- skip the update.
		IF NOT is_restored AND NOT now() <@ OLD.validity THEN
			RAISE EXCEPTION 'Cannot update expired rows'
			USING HINT = 'The validity has expired';
		END IF;

		-- If the history has not yet been set, set it to an empty array.
		IF NEW.history IS NULL THEN
			NEW.history := '[]'::jsonb;
		END IF;
		
		IF validity_changed THEN
			IF is_restored THEN
				-- Lower range: the date when the item was deleted, upper range: the date when it is restored.
				OLD.validity = tstzrange(upper(OLD.validity), now());
				NEW.validity = tstzrange(now(), null);
			ELSIF is_deleting THEN
				OLD.validity = tstzrange(lower(OLD.validity), now());
				NEW.validity = tstzrange(lower(OLD.validity), now());
			END IF;
		ELSE
			OLD.validity = tstzrange(lower(OLD.validity), now());
		END IF;
		
	
		-- We convert the old row to json, remove the 'history' field, and append it to the array of history.
    -- In addition, we add additional meta fields, prefixed with "pg."
		NEW.history = NEW.history::jsonb || (row_to_json(OLD)::jsonb - 'history' || format('{"pg.modified_by": "%s", "pg.modified_at": "%s", "pg.status": "%s"}', user, now(), (SELECT CASE WHEN is_restored THEN 'RESTORED' WHEN is_deleting THEN 'DELETED' ELSE 'UPDATED' END))::jsonb);

		RETURN NEW;
	END;
$$ LANGUAGE plpgsql;
```

Create a trigger to capture update, only if there are changes in the record. This trigger requires the column `validity` and `history` to be present:

```sql
CREATE TRIGGER historical_history 
BEFORE UPDATE ON historical
FOR EACH ROW
WHEN (OLD.* IS DISTINCT FROM NEW.*)
EXECUTE PROCEDURE pg_temp.history();
```

Perform some operation to see the changes:

```sql
TRUNCATE TABLE historical;

INSERT INTO historical (name) values ('John Doe');
UPDATE historical SET name = 'John Doe (edited)';
UPDATE historical SET name = 'John Doe (edited)';
UPDATE historical SET name = 'John Doe';
UPDATE historical SET name = 'Alice';

-- To expire records.
UPDATE historical SET validity = tstzrange(lower(validity), now());

-- To restore record.
UPDATE historical SET validity = tstzrange(lower(validity), null);

-- To select valid records.
SELECT * FROM historical WHERE now() <@ validity;
SELECT *, jsonb_array_length(history), jsonb_pretty(history) FROM historical;
```

Output:
```json
[
    {
        "id": 7,
        "name": "John Doe",
        "validity": "[\"2020-08-12 10:38:19.829765+00\",\"2020-08-12 10:38:20.720551+00\")",
        "pg.status": "UPDATED",
        "pg.modified_at": "2020-08-12 10:38:20.720551+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "John Doe (edited)",
        "validity": "[\"2020-08-12 10:38:19.829765+00\",\"2020-08-12 10:38:22.262249+00\")",
        "pg.status": "UPDATED",
        "pg.modified_at": "2020-08-12 10:38:22.262249+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "John Doe",
        "validity": "[\"2020-08-12 10:38:19.829765+00\",\"2020-08-12 10:38:23.123769+00\")",
        "pg.status": "UPDATED",
        "pg.modified_at": "2020-08-12 10:38:23.123769+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "Alice",
        "validity": "[\"2020-08-12 10:38:19.829765+00\",\"2020-08-12 10:38:24.564872+00\")",
        "pg.status": "DELETED",
        "pg.modified_at": "2020-08-12 10:38:24.564872+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "Alice",
        "validity": "[\"2020-08-12 10:38:24.564872+00\",\"2020-08-12 10:38:25.640635+00\")",
        "pg.status": "RESTORED",
        "pg.modified_at": "2020-08-12 10:38:25.640635+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "Alice",
        "validity": "[\"2020-08-12 10:38:25.640635+00\",\"2020-08-12 10:38:28.494331+00\")",
        "pg.status": "UPDATED",
        "pg.modified_at": "2020-08-12 10:38:28.494331+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "John Doe",
        "validity": "[\"2020-08-12 10:38:25.640635+00\",\"2020-08-12 10:38:30.113996+00\")",
        "pg.status": "UPDATED",
        "pg.modified_at": "2020-08-12 10:38:30.113996+00",
        "pg.modified_by": "john"
    },
    {
        "id": 7,
        "name": "John Doe (edited)",
        "validity": "[\"2020-08-12 10:38:25.640635+00\",\"2020-08-12 10:38:32.933882+00\")",
        "pg.status": "DELETED",
        "pg.modified_at": "2020-08-12 10:38:32.933882+00",
        "pg.modified_by": "john"
    }
]
```

See also: 

- [read-insert-only-table](https://github.com/alextanhongpin/database-design/blob/master/read-insert-only-table.md) 
- [timerange](https://github.com/alextanhongpin/database-design/blob/master/timerange.md)
