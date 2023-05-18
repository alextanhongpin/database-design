## Thoughts

- use comments for columns
- use reference table instead of enums even when the number of references are small (this prevents mistakes when creating the enum like wrong spelling, or non-existing enums)
- unique constraints only works on the first 64 characters
- use ordered uuid instead of integer (they are equally performant, but at the same time prevent users from crawling the database)
- keep history in another table for slowly changing dimension

https://thedailywtf.com/articles/Database-Changes-Done-Right
https://blog.bluzelle.com/things-you-should-know-about-database-caching-2e8451656c2d


## Should database contain business logic?

It depends. Having complicated business logic can be troublesome - it's hard to know what's happening unless there are clear documentation or there's a central team to manage them.

Here's when to add logic into database:
- pros: single source of truth, code must obey the database. If we have multiple backends (this is a micro-service anti-pattern btw), then they will all have to follow the rules set in the database.
- pros: Centralization of business logic;
- pros: independency of application type, programming language, OS, etc;
- unique constraints (single column or compound) - handling this at application level is not thread-safe (concurrent entry)
- length limit etc

Here's when not to add logic into database:
- your business logic will break another person's logic
- short term storage (tokens etc, can be stored in distributed cache)
- client ips etc, there's no reason to store it here, get them from the log
- cached dynamic values (e.g. average rating, item count can be delegated to distributed cache, unless sorting is required, which is make complicated with external cache if we are going to take into account pagination, sorting, filtering etc)
- avoid stored procedures

## Log as external database?

The single source of truth. But somehow this is not analysed at all.

# storage
you can save storage by sorting the columns, aka column tetris https://stackoverflow.com/questions/2966524/calculating-and-saving-space-in-postgresql



## Column naming

- keep it short
- add units to your column, e.g. `weight_g`, `length_cm`
- use the smallest unit of measurement, e.g. `weight_g` instead of `weight_kg`, cents for currency etc
- avoid floats, use integer instead, applies to money and unit of measurement

## Column to avoid

- computed column (with exception), store raw column
- exception is perhaps when we want to have a unique column instead of setting constraints on multiple columns, which can lead to larger index. For this, we create a computed column that hashes multiple column values, and set a unique index on that column


## Reference table
- avoid storing status etc as integer in the column
- use reference table instead, and apply foreign key association
- enum sounds like a good choice - only if the values do not change, e.g. days `Sunday`, `Monday`...
- if the values do change, don't use enum. Use a reference table


## Documentation
- mermaid erd diagram (recommended, because you can embed it as code snippet in Github/Gitlab for rendering)
- https://dbdiagram.io/home
- https://dbml.dbdiagram.io/home/
- https://dbdocs.io/

## BI Tools

-  Metabase (I've used this many times)
-  Looker Studio (looks promising, haven't explored)
- [DuckDB](https://duckdb.org/), supposedly super fast
- https://duckdb.org/2022/09/30/postgres-scanner.html

## Diff tools
- https://pypi.org/project/migra/ (for postgres)
- https://www.skeema.io/ (for mysql, used by Github)
- https://atlasgo.io/
- https://schemahero.io/

## Viewer
- https://www.tadviewer.com/
- https://www.bytebase.com/
- https://www.dbvis.com/


## Data transformation

- https://meltano.com/
- https://sqlmesh.readthedocs.io/en/stable/quick_start/
- https://prql-lang.org/
- https://www.benthos.dev/
- https://clickhouse.com/docs/knowledgebase/postgresql-to-parquet-csv-json (to convert postgres to parquet)
- https://github.com/rilldata/rill-developer
- https://github.com/malloydata/malloy
- https://github.com/erezsh/Preql
- https://logica.dev/


## Cli tools

- https://github.com/danvergara/dblab, tried, but doesn't beat psql
