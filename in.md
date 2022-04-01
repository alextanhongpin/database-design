https://stackoverflow.com/questions/24647503/performance-issue-in-update-query

Join to values instead of using IN for large values of IN


```sql
CREATE TABLE IF NOT EXISTS words (
	id int GENERATED ALWAYS AS IDENTITY,

	word TEXT NOT NULL,

	PRIMARY KEY (id)
);

-- Takes ~30s
INSERT INTO words(word)
SELECT md5(i::text)
FROM generate_series(1, 1000000) i;
```


## Sampling repeatable data for testing

```sql
-- Take 0.1% (1,000 rows with 1,000,000 dataset) of the data with the same seed.
SELECT *
FROM words
WHERE word IN (
	SELECT word
	FROM words
	TABLESAMPLE BERNOULLI(0.1) REPEATABLE(0)
);
```

## Using IN (values)

```sql
-- Take 0.1% (1,000 rows with 1,000,000 dataset) of the data with the same seed.
-- Example 1
SELECT string_agg(''''||word||'''', ', ')
FROM words
TABLESAMPLE BERNOULLI(0.1) REPEATABLE(0);


EXPLAIN ANALYZE
SELECT *
FROM words
WHERE word IN (
	-- Copy values from Example 1.
);
```

Output:

```sql
Gather  (cost=1002.50..15686.71 rows=1002 width=37) (actual time=6.590..202.280 rows=1002 loops=1)
  Workers Planned: 2
  Workers Launched: 2
  ->  Parallel Seq Scan on words  (cost=2.50..14586.51 rows=418 width=37) (actual time=0.236..58.900 rows=334 loops=3)
      Filter: (word = ANY ('{...the long uuid}'))
        Rows Removed by Filter: 332999
Planning Time: 0.932 ms
Execution Time: 202.519 ms
```

## Using IN (VALUES (), ()...)

```sql
-- Example 2
SELECT string_agg('(' || ''''||word||''''||')', ', ')
FROM words
TABLESAMPLE BERNOULLI(0.1) REPEATABLE(0);


EXPLAIN ANALYZE
SELECT *
FROM words
JOIN (VALUES
	-- Copy values from Example 2
) vals (v)
ON (word = v);
```

Output:
```sql
Gather  (cost=1025.05..15192.60 rows=1002 width=69) (actual time=13.278..286.691 rows=1002 loops=1)
  Workers Planned: 2
  Workers Launched: 2
  ->  Hash Join  (cost=25.05..14092.40 rows=418 width=69) (actual time=2.112..161.689 rows=334 loops=3)
        Hash Cond: (words.word = "*VALUES*".column1)
        ->  Parallel Seq Scan on words  (cost=0.00..12500.67 rows=416667 width=37) (actual time=0.012..62.213 rows=333333 loops=3)
        ->  Hash  (cost=12.53..12.53 rows=1002 width=32) (actual time=0.595..0.596 rows=1002 loops=3)
              Buckets: 1024  Batches: 1  Memory Usage: 72kB
              ->  Values Scan on "*VALUES*"  (cost=0.00..12.53 rows=1002 width=32) (actual time=0.003..0.234 rows=1002 loops=3)
Planning Time: 1.005 ms
Execution Time: 287.476 ms
```

## Using Subquery


```sql
EXPLAIN ANALYZE
SELECT *
FROM words
WHERE word IN (
	SELECT word
	FROM words
	TABLESAMPLE BERNOULLI(0.1) REPEATABLE(0)
);
```

Output:

```sql
Gather  (cost=9356.50..23055.56 rows=1000 width=37) (actual time=30.848..298.949 rows=1002 loops=1)
  Workers Planned: 2
  Workers Launched: 2
  ->  Hash Semi Join  (cost=8356.50..21955.56 rows=417 width=37) (actual time=35.210..181.711 rows=334 loops=3)
        Hash Cond: (words.word = words_1.word)
        ->  Parallel Seq Scan on words  (cost=0.00..12500.67 rows=416667 width=37) (actual time=0.011..45.760 rows=333333 loops=3)
        ->  Hash  (cost=8344.00..8344.00 rows=1000 width=33) (actual time=32.240..32.241 rows=1002 loops=3)
              Buckets: 1024  Batches: 1  Memory Usage: 72kB
              ->  Sample Scan on words words_1  (cost=0.00..8344.00 rows=1000 width=33) (actual time=1.606..31.397 rows=1002 loops=3)
                    Sampling: bernoulli ('0.1'::real) REPEATABLE ('0'::double precision)
Planning Time: 0.146 ms
Execution Time: 299.197 ms
```
