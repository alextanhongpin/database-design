## Things I learn about postgres

Tools
- ~using DBeaver for GUI visualization on MacOS~
- use Postico insteand

Query
- somehow table name must be surrounded by double quote `"`
- somehow string values must be surrounded by single quote `'`, and not double-quote `"`

## Useful commands

Equivalent of `show tables` in MySQL:

```postgres
SELECT * 
FROM  pg_catalog.pg_tables 
WHERE schemaname != 'pg_catalog' 
AND   schemaname != 'information_schema';
```

## Primary key in postgres

```sql
-- Posgresql
id serial PRIMARY KEY

-- Mysql
id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY
```
## Text or varchar

In postgres, use TEXT for everything, it should be equally performant as VARCHAR. Use VARCHAR only to limit the characters if you need to validate the length.


## Miscellaneous
- Postgres does not have timestamp on update, you need to manually implement a trigger
- Postgres does not allow table name to be user, use account instead
- In node js, selecting a column as a name must be lowercase, not camecase. In node library it is automatically converted to lowercase even if we set it as camelcase.


## UUID

http://www.postgresqltutorial.com/postgresql-uuid/
```
SELECT * FROM pg_available_extensions;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```


## On Conflict Update
```sql
INSERT INTO product (user_id, name) 
VALUES ('fa633b7c-f865-11e9-9537-7b2b141b65eb', 'iphone')
ON CONFLICT(name) DO UPDATE SET updated_at = now()
RETURNING id;
```

## Golang Errors

## postgres pqerror 
Constraint unique column.

```go
var pqerr *pq.Error
if errors.As(err, &pqerr) {
	fmt.Printf("%#v\n", pqerr)
}
```

Output:

```
&pq.Error{Severity:"ERROR", Code:"23505", Message:"duplicate key value violates unique constraint \"product_name_key\"", Detail:"Key (name)=(ENkZWiilbpvnSWOnPBYRkrqRJ) already exists.", Hint:"", Position:"", InternalPosition:"", InternalQuery:"", Where:"", Schema:"public", Table:"product", Column:"", DataTypeName:"", Constraint:"product_name_key", File:"nbtinsert.c", Line:"570", Routine:"_bt_check_unique"}
```

