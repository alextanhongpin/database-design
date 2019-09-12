# Update if value is set

This will cause empty updates though when the values are all null, which is unlikely. To prevent that, see below.
```sql
UPDATE some_table SET
  column_1 = COALESCE(param_1, column_1),
  column_2 = COALESCE(param_2, column_2),
  column_3 = COALESCE(param_3, column_3),
  column_4 = COALESCE(param_4, column_4),
  column_5 = COALESCE(param_5, column_5)
WHERE id = some_id;
```

By adding "param_x IS NOT NULL", we avoid empty updates:
```sql
UPDATE some_table SET
    column_1 = COALESCE(param_1, column_1),
    column_2 = COALESCE(param_2, column_2),
    ...
WHERE id = some_id
AND  (param_1 IS NOT NULL AND param_1 IS DISTINCT FROM column_1 OR
      param_2 IS NOT NULL AND param_2 IS DISTINCT FROM column_2 OR
     ...
 );
```
