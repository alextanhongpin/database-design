In clean architecture for Backend, there is usually a separation between presentation layer, domain layer and repository layer.

Can we further include another table for presentation layer in our database? 

This is different than a database view, because there may be additional (mostly computed) columns that are UI-specific. For example in an e-commerce database:

- computed machine learning ranking result for product recommendation
- flags/toggles on whether to show a product on the UI (this should not be placed in the original `products` table), that is usually an override of already existing `available/active` flag
- flag/toggles to _bump/highlight_ certain products
- text-search columns (?)
- more specific ranking columns (e.g. sort by out of stock last, sort by newest, sort by cheapest price etc), that is usually expensive to compute. This can be done using materialized view.


In most of the scenarios above, we want to avoid touching the primary table mainly because
- those UI/feature-specific columns should not belong in the original table, additional migration etc will only cause downtime when the table gets too huge
- potentially null columns for older data whenever new columns are added


They should be either separated in another table, or be placed in database that are more performant for querying (e.g. cassandra)
