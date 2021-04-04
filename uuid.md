## Serial

Why not auto-increment? It leads to many mistakes with many-to-many relationships, e.g. association the wrong id to another entity.

## In mysql 5.7.8

Creating fallback function for ordered uuid in old mysql version. Example using golang's `goose`:

```sql
--  https://www.percona.com/blog/2014/12/19/store-uuid-optimized-way/

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

To query:

```sql
SELECT FROM hex(uuid) FROM test;
-- 11E92F44A2762A8F83030242AC180002
```

To search:

```sql
SELECT * FROM test WHERE uuid = unhex('11E92F44A2762A8F83030242AC180002')
```

References:
- https://www.percona.com/blog/2014/12/19/store-uuid-optimized-way/


## Generate From Client

MySQL 5.7 uses uuid v1:

```go
package model

import (
	"strings"

	uuid "github.com/satori/go.uuid"
)

func NewOrderedUUID() string {
	//  MySQL 5.7 uses uuid.v1.
	id := uuid.Must(uuid.NewV1())
	output := strings.Split(id.String(), "-")
	part3 := output[0]
	part2 := output[1]
	part1 := output[2]
	part4 := output[3]
	part5 := output[4]
	out := strings.Join([]string{part1, part2, part3, part4, part5}, "")
	return out
}

func OrderedUUID(uuid string) string {
	output := strings.Split(uuid, "-")
	part3 := output[0]
	part2 := output[1]
	part1 := output[2]
	part4 := output[3]
	part5 := output[4]
	out := strings.Join([]string{part1, part2, part3, part4, part5}, "")
	return out
}
```

# Binary UUID MySQL

The correct way of storing a 36-char uuid as binary (16). Supports re-arranging time time component of the uuid to enhance indexing performance (by ordering it sequentially). Only workds for uuid v1.

```sql
INSERT INTO foo (uuid) VALUES (UUID_TO_BIN('3f06af63-a93c-11e4-9797-00505690773f', true));
```


```sql
SELECT BIN_TO_UUID(uuid, true) AS uuid FROM foo;
```
