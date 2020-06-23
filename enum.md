
## Add new enum values

Postgres 9.1 and above:
```sql
-- ALTER TYPE name ADD VALUE new_enum_value [ { BEFORE | AFTER } existing_enum_value ]

ALTER TYPE enum_type ADD VALUE 'new_value'; -- appends to list
ALTER TYPE enum_type ADD VALUE 'new_value' BEFORE 'old_value';
ALTER TYPE enum_type ADD VALUE 'new_value' AFTER 'old_value';
```

## Display enum list in Postgres:

```sql
select enum_range(null::<your_enum>);
```

## Example usage

```sql
CREATE TYPE action_type AS ENUM ('commented_by', 'answered_by', 'reacted_by');
select enum_range(null::action_type);

CREATE TYPE subscription_type AS ENUM ('question', 'answer', 'comment');
select enum_range(null::subscription_type);
ALTER TYPE subscription_type ADD VALUE 'reaction' AFTER 'answer';
```
