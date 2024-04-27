# Search

```sql
ALTER TABLE configs ADD COLUMN ts tsvector
    GENERATED ALWAYS AS (to_tsvector('english', key)) STORED;

CREATE INDEX CONCURRENTLY ts_idx ON configs USING GIN (ts);

-- prefix search
select * from configs where ts @@ to_tsquery('english', 'ke:*');;
```
