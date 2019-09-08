## Things I learn about postgres

Tools
- using DBeaver for GUI visualization on MacOS

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
