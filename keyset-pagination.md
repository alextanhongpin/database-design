# Keyset-pagination

When dealing with keyset pagination for multiple columns (especially when you are using uuid as a primary key, and have to resort to other columns), it's probably easier to have a single int column used for ranking instead.

We can do this by
- using a unix timestamp column instead of series (which defeats the purpose)
- creating a new view with the custom sorting and join back to the original
- using materialized view if the sorting does not change often
