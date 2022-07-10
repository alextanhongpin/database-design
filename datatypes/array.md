# Postgres Array

## What

An array data type. Postgres array can be a real anti-pattern when used wrongly. This clearly go against the `1st Normal Form (1NF)`, that states:

> Here, atomicity states that a single cell cannot hold multiple values. It must hold only a single-valued attribute.

However, arrays can be very practical when used correctly.

## Why

When there's a need to store multiple values in a column.

## When

Similar to JSON, use Array only if we do not need to enforce any rules on the items at database layer. That means

-  there's no need for uniqueness/sortable - it's an array, not a set (though you can write a function that does that)

- the values are independent from other rows.

- the items does not require referencing values from another table

- when the whole array needs to change together. Having a solution with a separate table means diffing the change before/after and adding/removing entries. With array, it is as simple as updating one column.



Some good examples includes storing tags/labels (e.g. in Github, you can add tags to your repository for more meaningful description). 



Some anti-pattern is to store `mentions` . This values are dependent on other rows, and the value may change when the username change. Instead of storing the `mentions` as an array, we could probably just store a reference to the mentioned user and query the `mention` real-time.



## How

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

## Performance

Performance for array can be further optimized by using Gin index.
