## To update an existing field

```sql
UPDATE table SET json_data = JSON_REPLACE(json_data, "$.field", value) WHERE...;
```

## To query a json field

```sql
SELECT json_data -> "$.field_name" FROM table;
```
