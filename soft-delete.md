## Using postgres rule to perform soft delete

Disadvantage: needs to be performed for each table.

```sql
CREATE OR REPLACE RULE delete_venue AS
  ON DELETE TO venues DO INSTEAD
    UPDATE venues
    SET active = 'f'
    WHERE venues.venue_id = old.venue_id;
```



## Soft delete vs hard delete
- how does it work for nested entities?

hard delete
- on delete cascade will remove all child entity with the parent id
soft delete
- it will not allow delete (how to handle the error message?)
- soft delete parent will not automatically soft delete the children. Needs to be performed manually in the reverse order
- but normally soft deleting child means there’s a possibility you won’t be able to access the children anymore from the ui
- if there are unique constraint, then when handling constraint, we need to set deleted at to null
- need to add filter for deleted at is null for each queries (solution: use view)
- need to add handling for unique constraint for create

other alternatives?
- hard delete, but copy the row to another table for auditing
- this can be done through listener in the application (not safe), or database triggers


**References**:
http://abstraction.blog/2015/06/28/soft-vs-hard-delete
https://docs.sourcegraph.com/dev/postgresql


## Soft delete in real world

In real world, delete is rarely practical. Say we have a product, iphone in our list of database. Then it is discontinued and we decided not to show them anymore. But we should not delete the product, because it exists. Instead of setting it to deleted at only (which does not provide any context to the users viewing the database), it is better to add a status, e.g. `DISCONTINUED` to indicate that the product is not longer manufactured. Also, if the product has a specific lifespan, we can also provide a more precise daate range (e.g. **valid_from** and **valid_till** columns) in which the product was valid (e.g. events, product pricing, promotions, sales all share this quality). 

From the UI perspective, we have two options:

1. don't show the product anymore
2. show the product with the discontinued label


