## Query rows from the last nth minute

```sql
CREATE TABLE test (date datetime);
INSERT INTO test (date) VALUES (CURRENT_TIMESTAMP());

-- Query item from the last minute.
SELECT * FROM test WHERE date > date_sub(now(), interval 1 minute);
```
