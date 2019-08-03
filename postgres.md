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
