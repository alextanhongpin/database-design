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


## Converting JSON to a database row (Postgres)

For single row:
```sql
SELECT * 
FROM json_populate_record(null::account, '{"email": "john.doe@mail.com"}');
```


For multiple rows:

```sql
SELECT * 
FROM json_populate_recordset(null::account, '[{"email": "john.doe@mail.com"}, {"email": "janedoe@mail.com"}]');
```


To build it from a dynamic list:

```sql
-- json_populate_record(record, json) <- convert the jsonb format to json. Merge with || only works with jsonb.
SELECT * FROM json_populate_record(null::account, ('{"email": "john.doe@mail.com"}'::jsonb || '{"token": "hello"}')::json);
SELECT * FROM json_populate_record(null::account, ('{"email": "john.doe@mail.com"}'::jsonb || json_build_object('token', 'hello')::jsonb)::json);
SELECT * FROM json_populate_recordset(null::account, '[{"email": "john.doe@mail.com"}, {"email": "janedoe@mail.com"}]');
```

## Building json object (Postgres)

The merge only works for `jsonb`, not `json`:

```sql
SELECT '{"email": "john.doe@mail.com"}'::jsonb || '{"token": "hello"}'; -- {"email": "john.doe@mail.com", "token": "hello"}
SELECT '{"email": "john.doe@mail.com"}'::json || '{"token": "hello"}'; -- {"email": "john.doe@mail.com"}{"token": "hello"}
```


## Insert json into table (Postgres)

Some limitations - if the field value is not provided in json, it will be treated as null. So for strings, it will throw an error if there is a text column with `not null` constraint.
```sql
  INSERT INTO pg_temp.person (name, picture, display_name)
  -- Don't include fields like ids.
  SELECT name, picture, display_name
    FROM json_populate_record(
      null::pg_temp.person, 
      (_extra::jsonb || json_build_object('name', _name, 'display_name', _display_name)::jsonb)::json
    )
  RETURNING *
```

## Check if a json field exists (Postgres)

```sql
SELECT '{"name": "a", "age": 10}' ? 'age';
```

## Aggregating rows as json in (Postgres)

There are times we want to aggregate a row as json, so that we can deserialize it back to full objects at the application layer:

```sql
-- NOTE: The table name (notification) must be specified before the .*
SELECT array_agg(to_json(notification.*)), subscriber_id, count(*)
FROM notification
GROUP BY subscriber_id;
```
