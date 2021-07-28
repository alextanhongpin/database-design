
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



https://www.quora.com/Why-is-it-not-possible-to-do-a-JOIN-on-a-sharded-database
