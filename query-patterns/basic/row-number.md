# Row Number

## MySql 5.7


There is no `row_number` function in mysql 5.7, only 8.0. To emulate it, we can use session variables.

However, it might not work perfectly.

```sql
SELECT
    (@row_number := @row_number + 1) AS rnk, points
FROM yourTable,
(SELECT @row_number := 0) AS x
ORDER BY points DESC;
```
