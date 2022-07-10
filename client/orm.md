## ORM

The question `when/why should I use ORM` should be rephrased as `how to maximize my sql usage`.

The pros of orm:
- don’t have to build query manually (which is also a cons, because in some cases it is easier to build them manually, e.g. function in sql)
- build type-safe sql
- write less codes for common operations (crud). But for uncommon operations (search, union, joins etc) you end up writing more code
- provides support for RAW sql at the same time (but mostly without type-safety)
- you don't need to know sql

The cons of orm:
- database becomes a dumb layer
- can’t utilize database specific features such as functions, triggers, custom datatypes, etc. your database won’t change, so there’s no point of making is swappable
- the idea of swappable database is meant for testing, use pgtap for testing
- poor support for relations (in terms of pagination, nested queries, with CTE statements). I have a use case one where I need to fetch a parent and a paginated child relations, but the ORM ends up fetching all child relations. In another scenario, I need to go down two-nested levels, which is not possible.
- poor aggregation support (mostly, but then ORM should not be used for analytics purposes, use raw SQL for that)
- some ORM like active record provides excellent developer experience (fast), but poor performance (poor query)
