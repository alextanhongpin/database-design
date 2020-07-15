## Read Insert Only Table

Sometimes we only want tables that can we can `insert` or `read` values, but not `update` or `delete` values.
This is useful when modelling data, can in real-life, most data are not deleted, but they expire. Think events and pricing table, an iPhone 6 cost $799 today, but could be $599 in the future, and it may cease to be sold, but it still exists once at a point in time.
Aside from historical data, we can also model versioning. 

- Historical: currency changes, events, promotions. Allow us to perform "AS OF" query, aka time travelling. e.g. What is the price of iPhone 6 in 2017, what is the price in 2019?
- Versioning

In short, when combined with temporal data (valid from, valid to), we can create a model a lot of real-world application.

There are several approach:
1. use roles to restrict access
2. use triggers


## Using roles

```sql
DROP TABLE read_insert_only;
CREATE TABLE IF NOT EXISTS read_insert_only (
	id serial NOT NULL PRIMARY KEY,
	identity int GENERATED ALWAYS AS IDENTITY,
	name text NOT NULL
);

REVOKE ALL ON TABLE read_insert_only FROM public;
GRANT SELECT, INSERT ON TABLE read_insert_only TO public;
-- User also need the USAGE privilege for the SEQUENCE attached to the serial column.
-- ERROR:  permission denied for sequence read_insert_only_id_seq
GRANT USAGE ON SEQUENCE read_insert_only_id_seq TO public;

CREATE ROLE jessie;
SET ROLE jessie;

INSERT INTO read_insert_only (name) VALUES ('john');
TABLE read_insert_only;
DELETE FROM read_insert_only;
UPDATE read_insert_only SET name = 'jessie';

RESET ROLE;
```


## Using Triggers

```sql
CREATE EXTENSION btree_gist;

DROP TABLE IF EXISTS reservation CASCADE;
CREATE TABLE IF NOT EXISTS reservation (
	id int NOT NULL,
	room text NOT NULL,
	period tstzrange NOT NULL,
	EXCLUDE USING GIST (room WITH =, period WITH &&)
);

INSERT INTO reservation(id, room, period) VALUES 
(1, 'room 1', tstzrange(now() - interval '2 day', now() - interval '1 day'));
INSERT INTO reservation(id, room, period) VALUES 
(1, 'room 1', tstzrange(now(), now() + interval '1 day'));

TABLE reservation;

-- Show only active reservation.
CREATE VIEW valid_reservation AS (
	SELECT * 
	FROM reservation
	WHERE now() <@ period
);

TABLE valid_reservation;

SELECT now();
SET clock.as_of_time = '2020-07-14 17:19:05.890707+00';

-- To customize the date.
CREATE VIEW valid_reservation_var AS (
	SELECT * 
	FROM reservation
	WHERE current_setting('clock.as_of_time')::timestamptz <@ period
);

TABLE valid_reservation_var;

CREATE OR REPLACE FUNCTION version_trigger() RETURNS TRIGGER AS $$
BEGIN
	IF TG_OP = 'UPDATE' THEN
		IF NEW.id <> OLD.id
		THEN 
			RAISE EXCEPTION 'the id must not be changed';
		END IF;
		
		UPDATE reservation
		SET period = tstzrange(lower(period), current_timestamp)
		WHERE id = NEW.id
		AND current_timestamp <@ period;
		
		IF NOT FOUND THEN
			RETURN NULL;
		END IF;
	END IF;
	
	IF TG_OP IN ('INSERT', 'UPDATE') THEN 
		INSERT INTO reservation (id, room, period)
		VALUES (NEW.id, NEW.room, tstzrange(current_timestamp, null));
		
		RETURN NEW;
	END IF;
	
	IF TG_OP = 'DELETE' THEN
		UPDATE reservation
		SET period = tstzrange(lower(period), current_timestamp)
		WHERE id = OLD.id
		AND current_timestamp <@ period;
		
		IF FOUND THEN
			RETURN OLD;
		ELSE
			RETURN NULL;
		END IF;
	END IF;
END;
$$ LANGUAGE plpgsql;

-- We can only apply this on the view.
CREATE TRIGGER version_trigger
INSTEAD OF INSERT OR UPDATE OR DELETE
ON valid_reservation
FOR EACH ROW 
EXECUTE PROCEDURE version_trigger();


UPDATE valid_reservation SET room = 'updated name';
DELETE FROM valid_reservation;
```
