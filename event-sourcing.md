# Implementing basic event sourcing tables with Postgres

There are three main tables `aggregate`, `event` and `snapshot`.


```sql
DROP TYPE aggregate_type;
CREATE TYPE aggregate_type AS ENUM ('person', 'organization');


DROP TABLE IF EXISTS aggregate CASCADE;
CREATE TABLE IF NOT EXISTS aggregate (
	aggregate_id uuid DEFAULT gen_random_uuid(),
	type aggregate_type NOT NULL,
	version int NOT NULL DEFAULT 0,
	
	PRIMARY KEY (aggregate_id)
);

COMMENT ON COLUMN aggregate.aggregate_id IS 'a unique identifier';
COMMENT ON COLUMN aggregate.type IS 'the type of aggregrate. In an object-oriented event sourcing implementation, this is the aggregate''s class name, e.g. Person';
COMMENT ON COLUMN aggregate.version IS 'current version of the aggregate. Incremented every time an event is applied to the aggregate and indicates how many events have been applied to it since it was first created';

DROP TABLE IF EXISTS event CASCADE;
CREATE TABLE IF NOT EXISTS event (
	aggregate_id uuid NOT NULL,
	version int NOT NULL DEFAULT 1,
	event_data jsonb NOT NULL DEFAULT '{}'::jsonb,
	
	PRIMARY KEY (aggregate_id, version),
	FOREIGN KEY (aggregate_id) REFERENCES aggregate(aggregate_id)
);

COMMENT ON COLUMN event.aggregate_id IS 'a foreign key pointing to the aggregate table';
COMMENT ON COLUMN event.version IS 'incremented version number';
COMMENT ON COLUMN event.event_data IS 'the actual event';

INSERT INTO aggregate(type) VALUES ('person') RETURNING *;

CREATE OR REPLACE FUNCTION add_event(
	_aggregate_id uuid, 
	_event_data jsonb
) RETURNS event AS $$
DECLARE 
	_aggregate_version int;
	_event event;
BEGIN
	
	UPDATE aggregate
	SET version = version + 1
	RETURNING version 
	INTO STRICT _aggregate_version;

	INSERT INTO event(aggregate_id, version, event_data)
	VALUES (_aggregate_id, _aggregate_version, _event_data)
	RETURNING * 
	INTO STRICT _event;

	RETURN _event;
END
$$ language plpgsql;

TABLE aggregate;

SELECT * FROM add_event('6deed889-d5d1-47ec-80ac-b55731a86f1d', '{"type": "person_created", "name": "pelle"}');
SELECT * FROM add_event('6deed889-d5d1-47ec-80ac-b55731a86f1d', '{"type": "name_was_changed", "name": "kalle"}');
SELECT * FROM add_event('6deed889-d5d1-47ec-80ac-b55731a86f1d', '{"type": "age_was_changed", "new_age": 26}');

TABLE event;

SELECT * FROM event;

-- Custom aggregate functions to merge json.
CREATE OR REPLACE AGGREGATE jsonb_merge_agg(jsonb)
(
    sfunc = jsonb_concat,
    stype = jsonb,
    initcond = '{}'
);

CREATE OR REPLACE FUNCTION jsonb_concat(a jsonb, b jsonb) RETURNS jsonb
    AS 'select $1 || $2'
    language sql
    immutable
    parallel safe
;

SELECT jsonb_merge_agg(event_data) - 'type'
FROM event 
GROUP BY aggregate_id;
-- {"name": "kalle", "type": "age_was_changed", "new_age": 26}

CREATE TABLE IF NOT EXISTS snapshot(
	aggregate_id uuid NOT NULL,
	version int NOT NULL,
	serialized_data jsonb,
	
	PRIMARY KEY (aggregate_id),
	FOREIGN KEY (aggregate_id) REFERENCES aggregate(aggregate_id)
);


CREATE OR REPLACE FUNCTION add_event_with_snapshot(
	_aggregate_id uuid, 
	_event_data jsonb
) RETURNS event AS $$
DECLARE 
	_aggregate_version int;
	_event event;
BEGIN
	
	UPDATE aggregate
	SET version = version + 1
	RETURNING version 
	INTO STRICT _aggregate_version;

	INSERT INTO event(aggregate_id, version, event_data)
	VALUES (_aggregate_id, _aggregate_version, _event_data)
	RETURNING * 
	INTO STRICT _event;
	
	INSERT INTO snapshot(aggregate_id, version, serialized_data)
		SELECT 
			_aggregate_id AS aggregate_id,
			_aggregate_version AS version,
			jsonb_merge_agg(event_data) - 'type' AS serialized_data
		FROM event 
		GROUP BY aggregate_id
	ON CONFLICT (aggregate_id) DO UPDATE SET
		version = EXCLUDED.version,
		serialized_data = EXCLUDED.serialized_data;

	RETURN _event;
END
$$ language plpgsql;

TRUNCATE TABLE aggregate CASCADE;
TRUNCATE TABLE event CASCADE;
TABLE aggregate;
TABLE event;
TABLE snapshot;

INSERT INTO aggregate(type) VALUES ('person') RETURNING *;
SELECT * FROM add_event_with_snapshot((SELECT aggregate_id FROM aggregate), '{"type": "person_created", "name": "pelle"}');
SELECT * FROM add_event_with_snapshot((SELECT aggregate_id FROM aggregate), '{"type": "name_was_changed", "name": "kalle"}');
SELECT * FROM add_event_with_snapshot((SELECT aggregate_id FROM aggregate), '{"type": "age_was_changed", "new_age": 26}');

-- Snapshot Table:
-- aggregate_id: 531a3d37-4617-4359-a9c6-a7592c9e616d
-- version: 3
-- serialized_data: {"name": "kalle", "new_age": 26}


CREATE OR REPLACE FUNCTION add_event_with_snapshot_enhanced(
	_aggregate_id uuid, 
	_event_data jsonb
) RETURNS event AS $$
DECLARE 
	_aggregate_version int;
	_event event;
BEGIN
	
	UPDATE aggregate
	SET version = version + 1
	RETURNING version 
	INTO STRICT _aggregate_version;

	INSERT INTO event(aggregate_id, version, event_data)
	VALUES (_aggregate_id, _aggregate_version, _event_data)
	RETURNING * 
	INTO STRICT _event;
	
	INSERT INTO snapshot(aggregate_id, version, serialized_data)
	VALUES (_aggregate_id, _aggregate_version, _event_data)
	ON CONFLICT (aggregate_id) DO UPDATE SET
		version = EXCLUDED.version,
		serialized_data = (snapshot.serialized_data || EXCLUDED.serialized_data) - 'type';
		-- NOTE: When concatenating the json, the right value must be newer (EXCLUDED is the new value).

	RETURN _event;
END
$$ language plpgsql;

TRUNCATE TABLE aggregate CASCADE;
TRUNCATE TABLE event CASCADE;
TABLE aggregate;
TABLE event;
TABLE snapshot;

INSERT INTO aggregate(type) VALUES ('person') RETURNING *;
SELECT * FROM add_event_with_snapshot_enhanced((SELECT aggregate_id FROM aggregate), '{"type": "person_created", "name": "pelle"}');
SELECT * FROM add_event_with_snapshot_enhanced((SELECT aggregate_id FROM aggregate), '{"type": "name_was_changed", "name": "kalle"}');
SELECT * FROM add_event_with_snapshot_enhanced((SELECT aggregate_id FROM aggregate), '{"type": "age_was_changed", "new_age": 26}');
```
