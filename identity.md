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


## Sortable uuid 

Rearranging v1 uuid in mysql to make it sortable by time
https://mysqlserverteam.com/mysql-8-0-uuid-support/
https://github.com/uuidjs/uuid/issues/75

v6 uuid
http://gh.peabody.io/uuidv6/

## Human readable uuid, base32 

https://github.com/solsson/uuid-base32
https://connect2id.com/blog/how-to-generate-human-friendly-identifiers
