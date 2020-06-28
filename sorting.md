## Sorting Integer String (Natural Sort)

```sql
-- Casting the type back to integer, slow for large table. Index the column for better performance.
SELECT * FROM table ORDER BY CAST(column AS UNSIGNED) DESC

-- Pad the left side of the strings with "0". Need to know the total length of the string.
SELECT * FROM table ORDER BY LPAD(column, 255, "0") DESC

-- Probably might not work for decimal.
SELECT * FROM table ORDER BY column * 1 DESC

-- Order by length of the string first, then the string itself. Benchmark if this is the fastest.
SELECT * FROM table ORDER BY ORDER BY LENGTH(column), column DESC
```

# Sorting by Secondary Column

When you sort by secondary sort with the same name, it will not override the first one. This is useful when you only want to make the primary sort criteria dynamic, and when the values are the same, then sort by the secondary/tertiary ones. Secondary sort is important so that we can rank the items correctly. 

```sql
select * from users
order by name desc, name asc.
[result => sorted by name desc]

select * from users
order by name asc, name asc.
[result => sorted by name asc]
```


## Sorting alphabetically case-insensitive

Postgres sorting is by collation, and is case-sensitive by default:
```sql
select * from sorting 
order by name;
```

The output might not be desirable. Output:
```
Andrew
Johb
alex
john
```

To solves this, we can apply `lower` function, but this may not be performant:
```sql
select * from sorting 
order by LOWER(name);
```

Explain statement:
```
Sort  (cost=97.78..101.18 rows=1360 width=64)
  Sort Key: (lower(name))
  ->  Seq Scan on sorting  (cost=0.00..27.00 rows=1360 width=64)
```

In postgres 12, we can create custom collation:

```sql
CREATE COLLATION case_insensitive (provider = icu, locale = 'und-u-ks-level2', deterministic = false);
```

The query will now produce the desired result:
```sql
select * from sorting 
order by name collate case_insensitive;
```

Output:
```
alex
Andrew
Johb
john
```

Explain:

```
Sort  (cost=94.38..97.78 rows=1360 width=64)
  Sort Key: name COLLATE case_insensitive
  ->  Seq Scan on sorting  (cost=0.00..23.60 rows=1360 width=64)
```
