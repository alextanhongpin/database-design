In most application, you would need a database. Some apps just need to connect and read data, others need to perform migration. For testing, you might want to work with the actual database too, rather than mocking them. 

You can opt to use an ORM, or just separate the implementation too query data through repository pattern.

You will also most likely deal with dynamic and static query, we will take a look at the differences later.

Learning to setup transaction is super useful too. This will avoid unnecessary refactoring later, should you choose to implement repository pattern, and also rolling back of statements in your integration testing.

Let's talk a look a a of the few basic recipe.

1. Connecting to db. Most client comes with basic configuration to allow you to connect to a database. You just need to provide the connection string. Most of the time, it is just enough. But for a more production ready application, you might want to setup a connection pool, and configuring statement timeouts.
2. Setup connection pool. Connection pool asked you to reuse connection, without exhausting them. There's a limited amount of connection that can be made to the application, but creating a connection is not cheap. There are some cases where you might exhaust the connections. One common mistake is to create transaction without committing them. This locks the current connection and the next call requires a new connection to be created. To prevent this, we can rely on testing, as well as seeing a timeout for your statement.
3. Setup timeouts. There are several timeouts that can be configured and it depends on the database driver your are using. We don't want to setup this for every application though. If you application is expected to perform long queries, say for analytics, this will terminate the connection before your query can return you data if it is set too low. For client facing applications, this is especially useful.
4. Migrations. We use migrations to bring our database to the next state by creating/altering tables, adding functions etc. Usually using SQL is preferable over ORM, for you have the fullest control over what you want to perform regardless of what technology stack you are using (SQL is the already generic across all languages, ORMs are not). There are several approach for running migration, externally controlled (usually through CLI) or internally (when the application starts, run the migration). There a pros and cons of doing so, which you should identify in your workflowans organisation structure. Do note that if you are running multiple microservices, having the migration running when the app start may cause deadlock to the db (worst case data inconsistency). 
5. Seeding. Is it recommended to separate populating the data from migrations, even though it's possible to do so in migrations. This allows you to seed data separately for production and development environment (as well as testing). Most of the time, reference data like countries too can be seeded in a separate migration. Perhaps it is even better to have separate workflows for separating application data Vs reference data, especially when the sequences matters)
6. Prepare statements (for non dynamic query), preparing statement allows the server to check if the statement is correct, though not 100%, as it will not work with dynamic queries. The best way to check if your queries are correct (right number of columns, correct data passed in etc) is to run a full Integration test against the database.
7. Close connection when application stops
8. Testing. There are several approaches to testing from postgres. You can always setup docker for testing. Alternatively you can use pgtap for testing too, which is more powerful. Usually there will be a combination of both. For dynamic queries where the statement might be constructed on the application side (better way is to have two separate queries for different conditions though), you might still working to test it in your application. If your queries are mostly static, and can be extracted from the application (normally SQL are written inline, but to test using pgtap we need to load the SQL, or resort to duplication) then pgtap can make your testing much easier. If you deal with a lot of stored procedures, user roles, rules, triggers or functions too, testing it with pgtap is much easier. 

Testing approach:
Global setup when testing database. We only want to run this once to perform migration before running our integration test. Once the is done, we can just setup the database connection before the test that needs to connect to the database. You might want to distinguish your unit test and integration test since most unit test won't need to connect to the database. Also, it is recommended to hardcode your database connection to those of docker to avoid dropping actual production database 
1. Setup connection to primary table
2. Drop all connection except this to perform drop database on template
3. Create template database
4. Connect to template database
5. Run migration on template database
6. Run seed on template database
7. For each worker, create a database that inherits template database. In this scenario, worker refers to the number of parallel tests we are running. Running parallel tests through separate database allows one to reduce the time to to run tests

When testing mutation
1.Setup transaction
2. Run test
3. Assert
4. Rollback

When testing read
1. Just perform query

Do I need to test the database?
Most microservices tend to use repository layers just as separation of concern.
The idea of mocking is just to mock one layer below. So if we have service, repository, we mock the repository. But if we are testing repository, we do not want to mock the client, it is pointless. 

The most common issue is wrong arguments, invalid column name, too much, too little columns/values provided, as well as wrong coercion types.

There are several approach to testing database.

1) test in application layer
2) test externally, e.g. using pgtap
3) ironically, you can always choose the typesafe approach too, which is using tools to generate the applicationcode from your SQL queries. E.g. using pgtyped, this helps prevents a lot of issue, but again, you still need to test against the implementation too.


Not testing is not an option. 

You can always choosean hybrid approach. There are reasons why testing has to be done in application (e.g. using dynamic queries, inline SQL so if you need to extract them out if you use pgtap)

On the other hand, if you encapsulated most of your business logic in functions, or you want to test against the schema or roles etc, Pg tap helps reduce a lot of boilerplate.


# Why use functions instead of create type.

In postgres, you can create custom types and reuse them for modelling value types, e.g. email etc. However, it is not so easy to customize the error messages.

Using function has some advantages . One of them is it is easier to control the error message. Though it does not help in localisation. 
Another advantage is you can perform multiple validations, e.g. checking that password is not empty string, and its length is more than a given length.
Single point of failure, but also a single source of truth for repeatable business logic.

You can actually add function constraints too for domain type.

Table designs and dates
- we can have a mutable entity, they will have all timestamps updated created and deleted
- the entity is an event, we just need the created at date
- the entity status changes over time, we need valid through date

Schema in application or outside
- in, easier to maintain, develop and design
- single monolith, hard to break out, because the schema is already there
- another application cannot share the same schema
- code generation tool easier, because schema is available, but if you need to generate code for another application, you need to duplicate the schema
- schema becomes single source of truth, everything else must refer back to this
