## Dealing with slowly changing dimensions


Schema used from the book `Developing Time-Oriented Database Application in SQL`, https://www2.cs.arizona.edu/~rts/tdbbook.pdf:
```
VT_Begin
BT_End
TT_Start
TT_Begin
```

From [wiki](https://en.m.wikipedia.org/wiki/Temporal_database):

- *Valid time* is the time period during which fact is true in the real world
- *Transaction time* is the time period during which a fact stored in the database was known.
- *Decision time* is the time period during which fact stored in the database was decided to be valid(note: If we have the created_at date for the row, it can be used as a the decision time. It would probably be more useful too to note who modify the data, by attaching either the id or role of the modifier, or both)

```
ValidFrom
ValidTill
Entered
Superseded
```

We will use the wiki's version, with an addition on a column `modified_by` to indicate who performed the action. To standardize the naming convention for time (since we use `created_at`, `updated_at`) the naming will now look as follow`:

```
- type (activity_type)??
- data (JSON) (x, not a good idea, since we don't know what facts are there)
- fact (string) this is a fact that is tracked, e.g. employee left company. this fact will have a validity, and confirmation through the transaction time.
- valid_from
- valid_till
- entered
- superseded
- decision_time
- modified_by (??)
- role (?? users, public, insurers, internal, external)
```

References:

- https://en.m.wikipedia.org/wiki/Temporal_database
- https://en.m.wikipedia.org/wiki/Slowly_changing_dimension
- https://www.kimballgroup.com/2013/02/design-tip-152-slowly-changing-dimension-types-0-4-5-6-7/
- https://www.red-gate.com/simple-talk/sql/database-administration/database-design-a-point-in-time-architecture/

## Implementing temporal database in MySQL

- To make table `T` temporal, create another table `T_history`

| Operation | `T` | `T_history` |
| - | - | - |
| Insert | Insert record | Insert record with valid_end_time as infinity |
| Update | Update record | - Update "latest" record valid_end_time to now <br> - Insert into T_history with valid end time as infinity |
| Delete | Delete record | Update valid_end_time with the current time for the "latest" record |
| Select | Select record | Select from desired date range | 

References:

- https://stackoverflow.com/questions/31252905/how-to-implement-temporal-data-in-mysql


## Issues

- how do we know who changed what
- how can we verify if the person that confirms it has the say?
- how can we know who performs action on that/confirm the validity, and what is the valid data

- fact is something that is true, and should be validated by a person
- an old fact can be dismissed by another person, but the record of the person will be maintained


## Other interesting discussions
- https://dba.stackexchange.com/questions/176935/how-would-i-track-all-price-changes-in-a-db-in-order-to-get-the-price-of-x-pro
- https://stackoverflow.com/questions/39060709/changes-of-product-price-in-database-design
- https://martinfowler.com/eaaDev/timeNarrative.html
- https://blog.cloudera.com/blog/2017/05/bi-temporal-data-modeling-with-envelope/
- https://www.datasciencecentral.com/profiles/blogs/temporal-databases-why-you-should-care-and-how-to-get-started-2
- good illustration on temporal https://nftb.saturdaymp.com/temporal-database-design/

## Event Sourcing vs Temporal

Event sourcing deals with objects, temporal databases deals with event record.

- https://softwareengineering.stackexchange.com/questions/349098/event-sourcing-vs-sql-server-temporal-tables)

## Validity

When we store data in the database, we need to consider the validity of the data for each specific columns. These will often resort to large redos in the database schema if not planned well. For starters, let's assume that we have the following type of data:

- static: These are facts that are valid forever. Take date of birth for example, this fact will never change over time unless recorded wrongly. Other type of data includes blood type, gender (note that gender could change, unless if we are storing this as a fact)
- dynamic, uncertain: The value could change over time, say address or occupation etc. But we do not know when. The dates can only be recorded once we know when the value change. E.g. marriage, job title, 
- dynamic, range: These data has a specific range of validity, e.g. 1 year for a mobile contract etc. But they may not necessarily be honored (the contract can end early, or they can be extended)


## Modifier

What is even more interesting is that a data can be approved/modifed by different users. In this scenario, we would definitely question who has the rights/what is the final correct data that we have?

This is normally found in approval systems, or system with state machines (status changes from Pending to Approved etc).

## Conditionals/Statuses

It's common to store status in a database. 
If there's only one status - it's a fact. If each row have the same fact, it becomes redundant and can probably be excluded from the table. E.g. A software house in MY recorded the country of the user created_location as Malaysia, which is repeated for every row.

If there are two, ask yourself if it can be more. If not, consider storing it as tinyint(1) bool to reduce storage.

If there are more, create a reference table to store the statuses. 

## Postgres

There are no native implementation, but there is a `temporal_tables` extensions available for postgres.

http://clarkdave.net/2015/02/historical-records-with-postgresql-and-temporal-tables-and-sql-2011/
https://wiki.postgresql.org/images/6/64/Fosdem20150130PostgresqlTemporal.pdf


## Working with temporal data structure

Items may expire, say a product or events, or maybe user accounts that are deactivated or friends that has been removed . When performing data analytics or recommendations, care needs to be taken to prevent recommending this to users.

Another thing to consider is not when the item expires, but when it is updated with new values. Take for example product pricing. We cannot update the product directly, since other users will reference this product if they have purchased it before. This could lead to inconsistency of the data. Therefor, a better way is to deprecate the old product, and create new row. New users will only have access to the new product, while old users will maintain their old product relationship.


## Temporal

https://martinfowler.com/eaaDev/TemporalProperty.html


## how to ensure theres no overlapping time in postgres
https://dba.stackexchange.com/questions/176474/how-to-check-if-there-are-no-time-gaps-in-postgresql


## About bitemporal

https://hdombrovskaya.files.wordpress.com/2016/09/bi_temporal_pg_open_2016.pdf
https://www.marklogic.com/blog/bitemporal/#:~:text=Bitemporal%2C%20which%20was%20released%20in,database%20(the%20system%20time).
