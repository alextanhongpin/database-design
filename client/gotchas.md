
## Postgres

- when running in transaction, the timestamp is not updated, so when dealing with time range, it can be confusing
- "user" and "order" is reserved keyword, so don't use it for table name
- you need to a trigger to update the timestamp yourself
