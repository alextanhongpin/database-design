## In mysql 5.7.8

Creating fallback function for ordered uuid in old mysql version. Example using golang's `goose`:

```sql
-- +goose Up
-- SQL in this section is executed when the migration is applied.
DROP FUNCTION IF EXISTS ordered_uuid;

-- +goose StatementBegin
CREATE FUNCTION ordered_uuid(uuid BINARY(36))
RETURNS binary(16) DETERMINISTIC
RETURN UNHEX(CONCAT(SUBSTR(uuid, 15, 4),SUBSTR(uuid, 10, 4),SUBSTR(uuid, 1, 8),SUBSTR(uuid, 20, 4),SUBSTR(uuid, 25)));
-- +goose StatementEnd

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP FUNCTION ordered_uuid;

```

We can then store the uuid as `binary(16)`. This does not work, since default values must be `constant`.

```
sql
CREATE TABLE test (
  uuid binary(16) NOT NULL DEFAULT uuid();
)
```

We have to insert them manually:

```sql
INSERT INTO test (uuid) VALUES (ordered_uuid(uuid())
```



References:
- https://www.percona.com/blog/2014/12/19/store-uuid-optimized-way/
