## Getting a random row in Postgres

There are cases where you need to get random rows in postgres (e.g. seeding data):


```sql
SELECT id FROM (
  -- Find the number of rows so that we can randomize a number.
	WITH total_count AS (
		SELECT count(*) AS count
		FROM job
	),
  -- Number each rows by ordering the rows by created_at. NOTE: If you are already using integer id, this won't be a problem.
	numbered_rows AS (
		SELECT *, row_number() OVER (ORDER BY created_at)
		FROM job
	),
	random_row AS (
		SELECT trunc(random() * (SELECT count FROM total_count)) + 1 AS number
	)
	SELECT *
	FROM numbered_rows 
	WHERE row_number = (SELECT number FROM random_row)
) random_job
```

Alternative...aggregate into array, select a position of an item in the array.
