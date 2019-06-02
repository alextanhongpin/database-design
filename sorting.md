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
