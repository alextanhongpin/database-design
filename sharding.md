
## Database.Sharding
https://medium.com/@jeeyoungk/how-sharding-works-b4dec46b3f6


## Database
Binary handling https://stackoverflow.com/questions/5801352/storing-binary-string-in-mysql
which database to choose for analytics https://www.linkedin.com/pulse/what-database-do-you-choose-analytics-shankar-meganatha
https://docs.microsoft.com/en-us/dotnet/standard/microservices-architecture/architect-microservice-container-applications/data-sovereignty-per-microservice
https://www.holistics.io/blog/should-you-use-mongodb-or-sql-databases-for-analytics/


## the cost of sharding

- Joins are less efficient. If you are building a social media network, and you want to fetch your friends on different shards, the cost of joining in db is greater than fetching in application. 
- Associations are now scattered across different shards.
- filtering is more complex
- sorting is a pain


## how to shard correctly?

https://stackoverflow.com/questions/6716351/application-level-join-with-where-and-order-by-on-n-postgresql-shards

hash key is user id so that all associations stays on the same db, however, fetching user to user association suffers still.


https://www.quora.com/Why-is-it-not-possible-to-do-a-JOIN-on-a-sharded-database


## NoSQL vs Postgres

NoSQL is highly misunderstood, cause you won't need it if you don't have terabytes of data.


The core strength for them is usually sharding. When your data no longer fits on one database, you have to plan to split them into several, with some algo to determine how they are going to be split. But usually that is an afterthought, when you reach that stage only you need it, and by that time, it's probably hard to make changes.

Rdms like postgres doesn't support sharding natively. And even when they do, you will see why it's not much difference from NoSQL


1) referential integrity. So, you sharded your postgres to 5 separate database. How can you reference a foreign key on another table? You can't, so you drop it. Referential integrity gone.
2) how about joins? You don't need foreign keys to do so, you can join by matching data. But how do you do joins across different database if the data is scattered? There are postgres foreign data wrapper to do that, but joining now suffers from network latency, and usually at that stage, you will just do joins at apllication layer.
3) how do you do ordering, searching, aggregation, indexing for uniqueness across different database? You can't, that's why you don't. Everything is done at the application layer.
4) schema migration. NoSQL is schemaless, it was intentional. When you have separate postgres instance, how do you ensure migrations is run correctly on all (in some cases you don't want to). Or if you are designing multi tenant software, same principles apply. 

In short, at scale, all the benefits you get from rdbms becomes a weakness. Facebook doesn't do joins or stuff also, the data store is just a dumb storage (see their usage of Cassandra(

Also by the time you reach that scale, I assume the schema is mostly fixed already. So the data you entered in NoSQL would most likely be consistent also

They try to sell it wrong by saying it allows you to do schemaless and improve agility when you are just prototyping. I think when you prototype, using rdbms and strict schema is more useful.
