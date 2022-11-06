What are the goals and values you would bring when learning good database design?


- more future proof database schema
- understand the tradeoffs between different design, and applying the right one for your usecase
- reduce cost (although storage is free, larger tables gets slower over time)
- design schemas that models you business logic well
- querying tables becomes more natural (when joining etc)
- better analytics


Tradeoffs between rows and columns
- using columns allows the usage of constraints, something that is hard to be done with rows (can only be done using triggers)
- e.g. modelling can only have two rows could also be done by using two columns, the second is non nullable
- we don't have to count the rows that way

