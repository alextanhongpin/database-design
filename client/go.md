# Setting connection limits for mysql library

```go
db.SetMaxOpenConns(5)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```


# Using JSON

```go
	var b []byte
	stmt := `
		SELECT JSON_OBJECT(
			"id", HEX(id),
			"email", email,
			"email_verified", IF(email_verified = 1, true, false) IS true
		) FROM employee
	`
	err = db.QueryRow(stmt).Scan(&b)
	if err != nil {
		log.Fatal(err)
	}
	var m model.Employee
	if err := json.Unmarshal(b, &m); err != nil {
		log.Fatal(err)
	}
	log.Println(m)
```


## Use Prepared statements

One of the advantages not mentioned is that this works as a warning whenever there's an sql syntax error, as opposed when calling it during `db.Query`. The warning will be thrown during compilation, and not during runtime, which can be helpful to detect what errors occurs.
```go
package main

import (
	"fmt"
)

type Stmt int

const (
	createBookStmt = iota
	deleteBookStmt
	updateBookStmt
)

var stmts = []struct {
	id   Stmt
	stmt string
}{
	{
		createBookStmt,
		`CREATE BOOK WHERE`,
	},
}

var statements map[Stmt]*sql.Stmt
func main() {
	for _, i := range stmts {
		fmt.Println(i.id, i.stmt)
		stmt, err := db.Prepare(i.stmt)
		if err != nil {
			log.Fatal(err)
		}
		statements[i.id] = stmt
	}
}
```

## On Duplicate Key Update alternative

Problem: need to build dynamic query depending on the number of params that are passed in. Disadvantage is:
- possible syntax error when building query dynamically
- cannot be prepared


The NULLIF() function compares two expressions and returns NULL if they are equal. Otherwise, the first expression is returned.

COALESCE returns the first non-null value in a list.


To perform an update only if the value is different, and the value is not empty ("", or 0):

```sql
insert into user 
	(id, name, age) 
values (1, "jessie", 20) 
on duplicate key 
update 
	name = COALESCE(NULLIF(NULLIF("jessie", name), ''), name),
	age = COALESCE(NULLIF(NULLIF(20, age), 0), age)
```
