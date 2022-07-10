# Estimation

```sql
SELECT pg_size_pretty(sum(pg_column_size(id))) from users;
SELECT pg_size_pretty(pg_total_relation_size('users'));
SELECT pg_size_pretty(pg_database_size('dbname'));
```
