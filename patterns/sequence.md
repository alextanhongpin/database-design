# Postgres Serial Type

- use generated always as identity
- when an operation fails, the serial number will be used

## Find all existing sequence name
```sql
SELECT sequence_schema, sequence_name
FROM information_schema.sequences;
```

## Restart sequence

```sql
ALTER SEQUENCE seq RESTART;
```


