Should business rules be stored in the database?

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
