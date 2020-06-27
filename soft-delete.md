## Using postgres rule to perform soft delete

Disadvantage: needs to be performed for each table.

```sql
CREATE OR REPLACE RULE delete_venue AS
  ON DELETE TO venues DO INSTEAD
    UPDATE venues
    SET active = 'f'
    WHERE venues.venue_id = old.venue_id;
```



## soft delete vs hard delete
- how does it work for nested entities?

hard delete
- on delete cascade will remove all child entity with the parent id
soft delete
- it will not allow delete (how to handle the error message?)
- soft delete parent will not automatically soft delete the children. Needs to be performed manually in the reverse order
- but normally soft deleting child means there’s a possibility you won’t be able to access the children anymore from the ui
- if there are unique constraint, then when handling constraint, we need to set deleted at to null
- need to add filter for deleted at is null for each queries
- need to add handling for unique constraint for create

other alternatives?
- hard delete, but copy the row to another table for auditing
- this can be done through listener in the application (not safe), or database triggers


**References**:
http://abstraction.blog/2015/06/28/soft-vs-hard-delete
https://docs.sourcegraph.com/dev/postgresql
