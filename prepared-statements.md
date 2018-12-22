## Prepared Statements

- statements are created with placeholders (?), and only the values are sent to the server
- it's unique per connection
- each connection can have a max prepared statement
- it will probably hit the max if the server runs indefinitely (the only way is to restart it, which will terminate the connection and also clear the prepared statement)

## Checking prepared statements

```sql
mysql> SHOW SESSION STATUS LIKE '%prepared%';
+---------------------------------------------+-------+
| Variable_name                               | Value |
+---------------------------------------------+-------+
| Performance_schema_prepared_statements_lost | 0     |
| Prepared_stmt_count                         | 0     |
+---------------------------------------------+-------+
```
