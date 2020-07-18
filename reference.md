## Dynamic logic without reference table - use functions

```sql
CREATE OR REPLACE FUNCTION is_admin(email text) RETURNS boolean AS $$
	SELECT email in ('john.doe@mail.com', 'jane.doe@mail.com');
$$ LANGUAGE SQL;


select is_admin('john.doe@mail.com'); -- Returns true.
select is_admin('jane@mail.com'); -- Returns false.
```

## Best practices

- use enum if the value is fixed (gender etc)
- use reference table if the values are dynamic
- use `serial` id for reference tables, don't need to use UUID which complicates thing. Use UUID only for entities created by users that exists in large numbers.


## Caching Strategy

Often, the reference table remains static and we want to cache them. The problem is, how long should we cache them? In case they get updated, we want to ensure that the old cached is purged. 

```sql
WITH json_data AS (
	SELECT json_agg(country.*) AS data
	FROM country
)
SELECT md5(data::text), data 
FROM json_data
```

Hypothesis: We can check the changes in content by md5-ing the whole data. If there are just a slight change in data, we can refresh the cache.

Here's how we can implement them:
- when the server starts and db connection has been establish, make a query to get the md5 of the content
- set redis key cache to the `md5-ied` value, and the redis value to the `json` data
- when client requests the data, we can now cached it locally
- whenever the client connects, compare the data and refetch when necessary (example using service-worker)
