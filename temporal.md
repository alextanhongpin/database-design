Dealing with slowly changing dimensions


https://en.m.wikipedia.org/wiki/Temporal_database
https://en.m.wikipedia.org/wiki/Slowly_changing_dimension
https://www.kimballgroup.com/2013/02/design-tip-152-slowly-changing-dimension-types-0-4-5-6-7/
https://www.red-gate.com/simple-talk/sql/database-administration/database-design-a-point-in-time-architecture/

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
