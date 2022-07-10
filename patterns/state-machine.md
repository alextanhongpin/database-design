## handling state machine in database


- we can always check if the next state is correct in the update statement
- alternatively, we can store all states in the database by category, and the valid from/to states, and always perform the additional query in the database (to avoid hardcoding). That way, we can always change the state in the database with redeploying the application)
- some tips is not to store states as string (even though at first it seems more convenient and avoids a query to the table)
- can the items be in just one status at a time (apply unique constraints)

References:
- https://tanzu.vmware.com/content/blog/maintainable-state-machines-part-2-don-t-store-state-names-in-the-database
- https://kevin.burke.dev/kevin/state-machines/
- https://kevin.burke.dev/kevin/faster-correct-database-queries/
- https://felixge.de/2017/07/27/implementing-state-machines-in-postgresql.html
- https://www.exceptionnotfound.net/designing-a-workflow-engine-database-part-1-introduction-and-purpose/


## The most basic state machine

If you have created at, updated at and deleted at column, you already have a state machine that describes the creation of the entity. Though in this case, updated at column seems to be a little generic.

Say if we have a post table, and want to have a post status to indicate the published date, unpublish data etc, we can just create a datetime column to store the state. We can have one additional status column to indicate the current status.

```
| id | body | status    | published_at | unpublished_at | 
| 1. | hi   | published | 2020-01-01   | null           |
```

Say if we are going to add a new feature to moderate the posts before it went live, we can just add another column:

```
| id | body | status    | published_at | unpublished_at | moderated_at | moderator_id |
| 1. | hi   | moderated | 2020-01-01   | null           | 2020-01-02   | 2            |
```
The example above is based on the level 1 status pattern (read the book [The Data Model Resource Book](https://www.wiley.com/en-us/The+Data+Model+Resource+Book%3A+Volume+3%3A+Universal+Patterns+for+Data+Modeling-p-9780470178454) for all patterns)
