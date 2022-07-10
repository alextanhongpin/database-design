
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


 	
## Rename enum postgres> 9.6

```sql
ALTER TYPE status_enum RENAME VALUE 'waiting' TO 'blocked';
```


## enum postgres

Enums is represented as integer internally. So you have both the benefits of performance and also readability when using enum.

One common issue when designing reference table is there's always a hunger for performance. What it means is we want to avoid joins, but at the same need the additional columns for clarity.


We can have both now

1. Create the enum type
2. Create another table and use that enum type as the primary key.

This way, you have readability on your table that references the enum, as well as performance, referential integrity and an option to query the reference table as options for the drop-down on the user interface 
