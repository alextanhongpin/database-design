# faker

How to populate	a table with fake data:

```sql
insert into towns (
    code, article, name, department
)
select
    left(md5(i::text), 10),
    md5(random()::text),
    md5(random()::text),
    left(md5(random()::text), 4)
from generate_series(1, 1000000) s(i)
```

## Generate a random string with min 5 and max 10 characters

```sql
select left(md5(i::text), round(random() * 5 + 5)::int) from generate_series(1, 100) i;
```

## Setting seed

```sql
-- Ensure reproducibility.
SELECT setseed(0.42);
```
