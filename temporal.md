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
