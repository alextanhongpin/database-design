# Search

```sql
ALTER TABLE configs ADD COLUMN ts tsvector
    GENERATED ALWAYS AS (to_tsvector('english', key)) STORED;

CREATE INDEX CONCURRENTLY ts_idx ON configs USING GIN (ts);

-- prefix search
select * from configs where ts @@ to_tsquery('english', 'ke:*');;
```


If you are using `ilike %hello%` often, create a trigram index:

```sql
create extension pg_trgm;
create index trgm_idx
on configs
using gin (key gin_trgm_ops);
```
