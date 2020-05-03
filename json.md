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


## Thoughts on storing json data as object vs array


with object:
- we probably need to create a static struct to manage the growing keys
- no identity on the kind of data (unless determined through column name)
- once unmarshalled, the values can be used straight away

with array:
- more generic approach
- need to loop through each key value pairs to get the data 
- easier to extend in the future

```js
{
  "a": "1",
  "b": "2"
}

// vs
{
  "data": [{"key": "a", "value": "1"}]
}
```

## Json or not?

Don’t use json
- no protection against referential integrity (if something gets deleted etc)
- no sorting
- no joining
- no constraints (uniqueness)
