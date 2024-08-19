## Reset template database
```
UPDATE pg_database SET datallowconn = TRUE WHERE datname = 'template0';
\c template0
UPDATE pg_database SET datistemplate = FALSE WHERE datname = 'template1';
DROP DATABASE template1;
CREATE DATABASE template1 WITH TEMPLATE = 'template0';
\c template1
UPDATE pg_database SET datistemplate = TRUE WHERE datname = 'template1';
UPDATE pg_database SET datallowconn = FALSE WHERE datname = 'template0';
```

tips for speeding up ci/cd
- use postgres template instead of running migrations multiple times
- rollback after executing logic (doesn't always work, e.g. frozen timestamp, triggers logic)

https://supabase.com/blog/postgresql-templates



# the common technique for speeding postgres test
- disable fsync
- use transaction and rollback
- template database (it is faster to drop and recreate database)

They are all outlined here https://www.maragu.dk/blog/speeding-up-postgres-integration-tests-in-go/
