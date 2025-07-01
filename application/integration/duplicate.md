## nodejs

With `mysql2` module.

```js
try {
  await db.query(stmt)
} catch (error) {
  if (error.code === 'ER_DUP_ENTRY') {
    console.log('found duplicate entry')
  }
}
```

## go
With Mysql:
```go
err := db.QueryRow(stmt).Scan(&res)
// https://dev.mysql.com/doc/refman/5.7/en/server-error-reference.html
if mysqlError, ok := err.(*mysql.MySQLError); ok {
  if mysqlError.Number == 1062 {
    // Duplicate key.
  }
}
```

Alternatively, use this package to avoid hardcoding `https://github.com/VividCortex/mysqlerr/blob/master/mysqlerr.go`:
```go
if mysqlError, ok := err.(*mysql.MySQLError); ok {
  if mysqlError.Number == mysqlerr.ER_DUP_KEY {
    // Duplicate key.
  }
}
```

With Postgres:
```go
package database

import (
	"errors"

	"github.com/lib/pq"
)

const DuplicatePrimaryKeyViolation = "23505"

var ErrDuplicatePrimaryKey = errors.New("Duplicate entity error")

func IsDuplicate(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == DuplicatePrimaryKeyViolation
	}
	return false
}
```

## Duplicate Key

```sql
INSERT INTO table (id, name, age) VALUES(1, "A", 19) ON DUPLICATE KEY UPDATE name="A", age=19
```
