# Sorting with uuid

Sorting with uuid is slow, and most of the time the sorting is not time-based.

Using int/bigint primary key as sort key is good, but most of the time you dont want to use a numeric primary key.
The solution is to use uuid as primary key, and an additional column for sorting.

Alternatively, use uuid v7/uuid v8, which is sortable. Ideally the generation of the primary key should be done in the database.


https://encore.dev/blog/go-1.18-generic-identifiers
