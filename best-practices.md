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
