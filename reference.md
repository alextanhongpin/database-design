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


## Using int vs text for reference keys

We often see primary int key being used for reference table. While it is usually not obvious what an id of 1, 2 or 3 means, it is usually a better practice to use `int` id as the primary key instead of `text`.
```
id | person_name | country_id
1. | john        | 1
```


Sure, `text` makes it a possibility that you don't need to join the table to infer what the foreign key means:
```
id | person_name | country_id
1. | john        | my
```

But there are reasons why using `int` id is a better choice:
- varchars use more space than ints
- you are more likely to have to update a varchar value than an int value, causing cascading updates
- might not be appropriate in internationalized applications (i.e. different values for different languages)


If joins is not your thing, you can always write a lookup function to map id to values and vice versa:

```sql
CREATE OR REPLACE FUNCTION lookup_country_iso_from_name(_name citext) RETURNS text AS $country_iso$
	SELECT iso
	FROM country
	WHERE name = _name
$country_iso$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION lookup_country_name_from_iso(_iso citext) RETURNS text AS $country_name$
	SELECT name
	FROM country
	WHERE iso = _iso
$country_name$ LANGUAGE SQL STABLE;
```

Then, we can use it when reading and writing:
```sql
-- Reading person table, and mapping the country_iso back to the desired text label.
SELECT id, name, lookup_country_name_from_iso(country_iso) AS country
FROM person;

-- Compared to using join. Imagine you have 10 columns that you need to map, which means 10 table joins.
-- In terms of performance, joins will definitely be faster, but this approach keeps the code cleaner (subject to individual).
-- Also, we can index the country table with INCLUDE columns for index-only scan.
SELECT id, name, country.name AS country
FROM person
JOIN country ON (country_iso = country.iso)
JOIN ...
JOIN ...
JOIN ...
JOIN ...
JOIN ...
JOIN ...
```

```sql
-- Writing to person table, and providing the raw text value which will be mapped back to id
-- in a single statement.
INSERT INTO person(name, country_iso) VALUES 
('John Doe', lookup_country_iso_from_name('Malaysia'))
```

Some other thoughts:
- when joining reference table, and when the value is small, why not "preload" the data with the CTE statement, and join them?


## Hardcoding reference data in client, vs storing in separate table and returning in API

There are always cases where we need to return some reference data, and the question is whether to hardcode it on a client, vs storing it in database and returning it through API. There are several pros and cons, let's explore them:


Types of data
- static 
	- this data is mostly static, and the values are known ahead of time
	- this data can normally be hardcoded, since it will almost not change
	- e.g. gender (male, female, others), marital status (single, married, divorced), application type (web, mobile), country/currency list
- "mostly" static
	- this data are mostly static, but if the ids are generated to our application (e.g. surrogate keys, auto-incremented keys), we might still want to return them the list
	- e.g. domain specific types, e.g. employment type in job domain
- dynamic
	- the values almost always change, can be added/removed
	- e.g. status types, category types, dynamic forms
	
Pros of hardcoding:
- simpler on the server side, less endpoints, less service (but not necessarily better)
- simpler on the client side, don't have to asynchronously fetch the data to populate component (select, dropdown)
- automatic allowlist - just add what is needed by the application, don't have to validate on the server (?! which is wrong)

Cons of hardcoding:
- must be updated manually on all the clients that called the data (e.g. mobile, web, other backends) whenever a value change/added/removed
- denylist means updating the application to remove the options available
- more validation is necessary, since we can't trust client's input, especially if there's no referential integrity (in NoSQL, no foreign key)
- return types might be different for different clients (e.g. return allowlist category list a for tenant-a, category list b for tenant-b)


When in doubt, return from the server.
