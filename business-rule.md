# Should business rules be stored in the database?

Constraints are part of business rules. Foreign keys relationship, and also declaring a NOT NULL is part of business rules.
But placing business rules in database makes testing harder - most of the time, we want to test against the business rules logic, not the database. In a distributed system, it might be advantageous (?) though to store the business rules in the database to ensure they are honored.

Some examples of business rules that can be (or should be) excluded from the database is count/datetime comparison. It might be better to query the values and compare them instead, e.g.

```
const count = repository.getOrderCount()
if (count > 3) ....

// rather than
count hasThreeOrders = repository.hasThreeOrders() 
// what if the requirements changed? or if we need to have different conditions for different count?
// if count > 3, if count > 5 ...
```


## Business logic in database

Yes, add it where appropriate. It can be a performance improvement. However, testing becomes more complicated. How do you test if the data in the database is accurate? You still have to seed data to test out the results.

- https://martinfowler.com/articles/dblogic.html
- https://tapoueh.org/blog/2017/06/sql-and-business-logic/

## Business logic in application layer
There are many good reasons why you should keep business logic in application layer

- no full acccess to db to create table/functions etc (though unlikely)
- you only have a single instancethat connects tothe db, thus limiting access to you only as the sole owner of the db
- full access to progamming capability
- able to design more complex rules to apply on thedata

But there are more reasons why keeping it in the database makes sense

- single source of truth. Business rule defined in db must be followed by all client connecting to the db. When done in application layer, multiple instance running multiple logic might lead to data inconsistency. Also, insert or update done directly through postgres cli has no knowledge of business logic in application
- guard against data race or unwanted insert. A db specifying unique column guarantes that, while application is not data aware.
- transaction isolation and guarantee when running triggers
- performant. Finding count doesnt require you to load everything in memory. Same goes for sorting, filtering.
- one may argue when sharded, you dont get the benefits, because data is now scattered across different db, which is true




## implementing rule engine

operators and value

- rules table with operator, expected, and column
- join table to rules

# domain type
http://www.postgresonline.com/journal/archives/205-Using-Domains-to-Enforce-Business-Rules.html

# check constraint

# enums

# rule vs trigger

# functions

# why business rule in db versus application

# storing dynamic ui
# access control, roles and permission

# postgres check constraint involving another table
- usecase, ensuring uniqness from another table
- usecase, inserting to another table for polymorphic association

http://tdan.com/modeling-business-rules-data-driven-business-rules/5227
https://databasemanagement.fandom.com/wiki/Business_Rules
https://martinfowler.com/bliki/RulesEngine.html
https://en.wikipedia.org/wiki/Semantic_reasoner
https://en.wikipedia.org/wiki/Rete_algorithm
https://en.wikipedia.org/wiki/Business_rules_engine
http://www.databaseanswers.org/data_models/rules_engines/index.htm
https://www.quora.com/What-are-the-business-rules-in-a-database-What-are-some-examples
