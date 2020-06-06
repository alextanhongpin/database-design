
## Add new enum values

Postgres 9.1 and above:
```sql
ALTER TYPE enum_type ADD VALUE 'new_value'; -- appends to list
ALTER TYPE enum_type ADD VALUE 'new_value' BEFORE 'old_value';
ALTER TYPE enum_type ADD VALUE 'new_value' AFTER 'old_value';
```

## Display enum list in Postgres:

```sql
select enum_range(null::<your_enum>);
```
