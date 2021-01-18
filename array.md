# Postgres Array

```sql
CREATE TABLE posts (
	title text NOT NULL PRIMARY KEY,
	tags text[]
);

INSERT INTO posts (title, tags) 
VALUES ('Hello world', '{postgres, triggers, go}');

SELECT * 
FROM posts 
WHERE tags @> '{"go"}';
```


## Postgres Slice Array

NOTE: The index starts from 1, not 0:

```sql
SELECT (ARRAY[1, 2, 3, 4, 5])[0]; -- NULL (index starts from 1)
SELECT (ARRAY[1, 2, 3, 4, 5])[1]; -- {1}
SELECT (ARRAY[1, 2, 3, 4, 5])[1:1]; -- {1}
SELECT (ARRAY[1, 2, 3, 4, 5])[1:2]; -- {1,2}
SELECT (ARRAY[1, 2, 3, 4, 5])[1:10]; -- {1,2,3,4,5}
```

Some useful applications is when aggregating rows as json, we can slice only selected rows:
```sql
SELECT (array_agg(to_json(notification.*)))[1:3], subscriber_id, count(*)
FROM notification
GROUP BY subscriber_id;
```

Alternatively, we can slice on update:
```sql
DROP TABLE notification;
CREATE TABLE IF NOT EXISTS notification(
	id bigint GENERATED ALWAYS AS IDENTITY,
	type text NOT NULL,
	history jsonb[] NOT NULL DEFAULT '{}',
	PRIMARY KEY (id),
	UNIQUE (type)
);


-- Index starts from 1. The last index is excluded. Below, we slice only three items at index, 1, 2 and 3.
INSERT INTO notification(type, history) VALUES ('question_answered', ARRAY['{"id": "3"}']::jsonb[])
ON CONFLICT (type) DO UPDATE 
SET history = (array_prepend('{"id": "3"}'::jsonb, notification.history))[1:4];

TABLE notification;
```
