## Check if a row exists

Setting `LIMIT 1` makes a huge difference, especially when they can have multiple values. 

```sql
SELECT EXISTS(SELECT 1 FROM test2 WHERE id ='321321' LIMIT 1)
```

References:
- https://stackoverflow.com/questions/1676551/best-way-to-test-if-a-row-exists-in-a-mysql-table
