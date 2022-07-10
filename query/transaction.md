# Isolation level

MySQL default isolation level is `repeatable read`:
https://dev.mysql.com/doc/refman/5.6/en/set-transaction.html

Postgres default isolation level is `Read committed`:
https://www.postgresql.org/docs/current/sql-set-transaction.html


# Sample transaction with Node.js

```js
async function main() {
  const db = await mysql.createPool({
    database: config.database,
    host: config.host,
    password: config.password,
    user: config.user
  })

  const conn = await db.getConnection()
  try {
    const stmt = `
      INSERT INTO ()...
    `
    await conn.query('START TRANSACTION')
    await conn.execute(stmt, [])
    // If all is successful until this point, commit the 
    // transaction.
    await conn.query('COMMIT')
  } catch (error) {
    // Perform rollback when an error occurred.
    await conn.query('ROLLBACK')
  } finally {
    // Release the connection at the end to save resources.
    await conn.release()
  }
} 
```
# Sample result from nodejs

```js
// This will be returned when running execute.
ResultSetHeader {
  fieldCount: 0,
  affectedRows: 1,
  insertId: 0,
  info: 'Rows matched: 1  Changed: 1  Warnings: 0',
  serverStatus: 2,
  warningStatus: 0,
  changedRows: 1 
}

// To check if the row is updated.
const isUpdated = !!result.changedRows

// To get the last id created (int, auto-incremented primary key)
const id = result.insertId
```


## Transaction trap with golang

```go
package main

import (
	"database/sql"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Name string
	Age  int
}

func main() {
	db, err := sql.Open("mysql", "john:123456@/test?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		err := transactionUpdate(db)
		if err != nil {
			log.Println(err)
		}
	}()
	go func() {
		defer wg.Done()

		err := transactionUpdate(db)
		if err != nil {
			log.Println(err)
		}
	}()
	// If start age is 1, the end age is 2, not 3.
	wg.Wait()
}

func transactionUpdate(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	var u User
	// Either use `SELECT name, age FROM user LIMIT FOR UPDATE`
	err = tx.QueryRow(`SELECT name, age FROM user LIMIT 1`).Scan(
		&u.Name,
		&u.Age,
	)
	log.Printf("got user: %+v\n", u)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	time.Sleep(1 * time.Second)

	// Atomic-safe: `UPDATE user SET age = age + 1 WHERE name = ?`
	res, err := tx.Exec(`UPDATE user SET age = ? WHERE name = ?`, u.Age+1, u.Name)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	log.Printf("%+v\n", res)
	_ = tx.Commit()
	return nil
}
```

## Using transaction correctly

- use only in the scenario when an action fails, all previous action would be reverted as well. In other words, when the states are dependent on one another.
- don't use it when each item state is independent of one another. Update them separately to reduce errors. If one of the query fails, the rest should still be updated. There was once a scenario in keeping records, when 10 claims have been approved and the entry needs to be inserted into the database. The dev use transaction in order to query and update the data, but this is exactly the wrong usage of transaction, because their states (claims) are independent on one another). The failure in updating one should not affect the remaining transaction.


## Setting transaction isolation level in Postgres
```sql
begin;
	set transaction isolation level repeatable read;
	// Do stuff...
commit;
```
