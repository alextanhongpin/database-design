# Serial vs Identity vs UUID

- TLDR, stick to uuid to avoid issues
- use identity only for reference tables, e.g. category, roles, permissions. They usually have a limited number of rows, and they don't expose vulnerability when the ids are exposed to public
- some pitfals when using int id, the foreign key/primary key may clash, resulting in wrong references, e.g. user_id can be passed in admin_id due to confusion, but with uuid this will almost never happen


## How to reset sequence?

```sql
ALTER SEQUENCE your_sequence_name RESTART WITH 1;

-- To find your sequence name
SELECT * FROM information_schema.sequences;
```
