## Query rows from the last nth minute

```sql
CREATE TABLE test (date datetime);
INSERT INTO test (date) VALUES (CURRENT_TIMESTAMP());

-- Query item from the last minute.
SELECT * FROM test WHERE date > date_sub(now(), interval 1 minute);
```

## Get rows that is today

```sql
SELECT users.id, DATE_FORMAT(users.signup_date, '%Y-%m-%d') 
FROM users 
WHERE DATE(signup_date) = CURDATE()
```

## Format to dd-mm-yyyy

```sql
SELECT date_format(NOW(), '%d-%m-%Y') as ddmmyyyy;
```
