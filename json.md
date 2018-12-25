# JSON

- by default, the field is not null

## To update an existing field

```sql
UPDATE table SET json_data = JSON_REPLACE(json_data, "$.field", value) WHERE...;
```

## To query a json field

```sql
SELECT json_data -> "$.field_name" FROM table;
```

## To get rows with json array

```sql
SELECT * FROM table WHERE JSON_TYPE(json_field) = 'ARRAY';
```

## To aggregate the json array from all rows where the column is not null

```sql
SELECT JSON_ARRAYAGG(organizations) FROM user WHERE JSON_TYPE(organizations) = 'ARRAY';
```
## Privilege for functions 

```sql
mysql -u USERNAME -p
set global log_bin_trust_function_creators=1;
```
