## Modelling Tags with Postgres Array


```sql
CREATE TABLE IF NOT EXISTS tags (
	tags text[] not null default '{}'
);

-- Index for performance.
CREATE INDEX ON tags USING gin(tags);

INSERT INTO tags (tags) VALUES (ARRAY['hello', 'world']);

SELECT * FROM tags WHERE tags @> ARRAY['hello'];
SELECT * FROM tags WHERE tags @> ARRAY['haha'];
```

Pros:
- save storage, no additional table required
- no table joins, fast query
- fast lookup with index (some benchmark shows that the performance is faster than using table joins)
- fast save

Cons:
- same value can appear in the array
- values may be duplicated (Job, Jobs, job)
- while it is fast to search tags for a document, searching for most popular tag can be hard (need to aggregate all tags values first, then count corresponding documents with tags)
- update/delete is not reflected in the array
  - we can store the ids of the tags instead of string, but this requires another lookup)
  - if tags are arbitrary readonly data, then we don't need to enforce such constraints)
- saving an empty array can clear all tags

Using table joins has its pros and cons too

Pros:
- easier for analytics (finding most popular tags, and getting tags for a particular document is easier)
- documents count can be stored alongside the tags table, to reduce query
- updates/deletes in the tags will be reflected (though by right they should be not deleted)

Cons:
- joins can be slow
- saving tags can be painful, mostly requires deleting all old tags, and recreating the associations, unless we do upsert only. If we want to save tags that consist of some added and deleted tags, this may be complicated to perform)


