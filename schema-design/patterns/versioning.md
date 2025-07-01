# How do we create Revisions in database?

Use stored procedure
- https://dev.to/livioribeiro/use-your-database-part-3---creating-a-revision-system-20j7

This is similar to temporal database design, or slowly changing dimensions. Another option is to use CDC (change data capture) to capture the changes in the database rows and stream them somewhere else to be stored).
