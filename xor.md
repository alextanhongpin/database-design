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
