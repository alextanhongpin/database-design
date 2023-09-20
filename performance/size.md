# Calculate Table Size

## Postgres

```sql
SELECT pg_size_pretty(sum(pg_column_size(id))) from users;
SELECT pg_size_pretty(pg_total_relation_size('users'));
SELECT pg_size_pretty(pg_database_size('dbname'));
```


## MySQL

```
SELECT
  TABLE_NAME AS `Table`,
  ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024) AS `Size (MB)`
FROM
  information_schema.TABLES
WHERE
  TABLE_SCHEMA = "bookstore"
ORDER BY
  (DATA_LENGTH + INDEX_LENGTH)
DESC;
```
