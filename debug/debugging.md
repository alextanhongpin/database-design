# When you are new to a project

## Find columns of a specific table.
```sql
SELECT *
  FROM information_schema.columns
 WHERE table_schema = 'public'
   AND table_name   = 'products';
```
## Find specific columns
```sql
SELECT *
  FROM information_schema.columns
 WHERE table_schema = 'public' AND 
 	column_name LIKE '%flow%';
```
