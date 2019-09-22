# Generating ordered UUID v1

Random produces very fragmented inserts that destroy tables. Use `uuid_generate_v1mc()` [instead] ... the keys are seq because they're time based. So all inserts go to the same data page without random io.

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS "uuid-ossp";

SELECT uuid_generate_v1();
SELECT uuid_generate_v4();

-- Recommended!
SELECT uuid_generate_v1mc();

-- Not recommended!
CREATE EXTENSION pgcrypto;
select gen_random_uuid();
```

## References

- https://www.2ndquadrant.com/en/blog/sequential-uuid-generators/
- https://stackoverflow.com/questions/34230208/uuid-primary-key-in-postgres-what-insert-performance-impact
- https://dzone.com/articles/store-uuid-optimized-way
- https://starkandwayne.com/blog/uuid-primary-keys-in-postgresql/
