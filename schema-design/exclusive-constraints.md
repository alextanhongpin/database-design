# XOR Column

__Use Case__: If column type is 'A', then 'A value' is required. Else if column type is 'B', then 'A value' must be null. If the `type` is `market`, the `limit_price` is not required. But if the `type` is `limit` then `limit_price` is required.


```sql
-- Table name needs to be quoted because order is a reserved keyword.
CREATE TABLE "order" (
	id int GENERATED ALWAYS AS IDENTITY,
	type text NOT NULL CHECK (type IN ('market', 'limit')),
	limit_price decimal(13, 4) NULL CHECK ((type = 'limit' AND limit_price IS NOT NULL) OR (type = 'market' AND limit_price IS NULL))
);


INSERT INTO "order" (type, limit_price) VALUES ('market', 12.3); -- Fail
INSERT INTO "order" (type, limit_price) VALUES ('market', NULL);
INSERT INTO "order" (type, limit_price) VALUES ('limit', NULL); -- Fail
INSERT INTO "order" (type, limit_price) VALUES ('limit', 13.4);

TABLE "order";
DROP TABLE "order";
```

Another approach using `CASE WHEN ... THEN ... END`:
```sql
CREATE TABLE "order" (
	id int GENERATED ALWAYS AS IDENTITY,
	type text NOT NULL CHECK (type IN ('market', 'limit')),
	limit_price decimal(13, 4) NULL CHECK (CASE WHEN type = 'limit' THEN limit_price IS NOT NULL ELSE limit_price IS NULL END)
);

INSERT INTO "order" (type, limit_price) VALUES ('market', 12.3); -- Fail
INSERT INTO "order" (type, limit_price) VALUES ('market', NULL);
INSERT INTO "order" (type, limit_price) VALUES ('limit', NULL); -- Fail
INSERT INTO "order" (type, limit_price) VALUES ('limit', 13.4);
```


Alternatives:

```sql
DROP TABLE IF EXISTS entity_1;
CREATE TABLE IF NOT EXISTS entity_1 (
	id uuid DEFAULT gen_random_uuid(),
	name text NOT NULL,
	PRIMARY KEY (id)
);

DROP TABLE IF EXISTS entity_2;
CREATE TABLE IF NOT EXISTS entity_2 (
	id uuid DEFAULT gen_random_uuid(),
	name text NOT NULL,
	PRIMARY KEY (id)
);
```
If we are only working with two conditions, this is a good approach:

```sql
DROP TABLE IF EXISTS test;
CREATE TABLE IF NOT EXISTS test (
	id uuid DEFAULT gen_random_uuid(),

	entity_1_id uuid NULL,
	entity_2_id uuid NULL,

	PRIMARY KEY (id),
	FOREIGN KEY (entity_1_id) REFERENCES entity_1(id),
	FOREIGN KEY (entity_2_id) REFERENCES entity_2(id),
	CHECK ((entity_1_id IS NULL) != (entity_2_id IS NULL))
);
```

Otherwise, for a growing number of conditions, this is a better approach:
```sql
DROP TABLE IF EXISTS test;
CREATE TABLE IF NOT EXISTS test (
	id uuid DEFAULT gen_random_uuid(),

	entity_1_id uuid NULL,
	entity_2_id uuid NULL,

	PRIMARY KEY (id),
	FOREIGN KEY (entity_1_id) REFERENCES entity_1(id),
	FOREIGN KEY (entity_2_id) REFERENCES entity_2(id),
	CHECK (
		(entity_1_id IS NULL)::int +
		(entity_2_id IS NULL)::int = 1)
);
```


```
INSERT INTO entity_1(name) VALUES ('a');
INSERT INTO entity_2(name) VALUES ('b');
INSERT INTO test (entity_1_id, entity_2_id) VALUES
((SELECT id FROM entity_1), (SELECT id FROM entity_2));
INSERT INTO test (entity_1_id) VALUES
((SELECT id FROM entity_1));
INSERT INTO test (entity_2_id) VALUES
((SELECT id FROM entity_2));
```

## To check if both column is null/not null

```sql
CHECK(ROW(col1, col2) IS NOT NULL)
CHECK(ROW(col1, col2) IS NULL)
```
