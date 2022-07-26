# Using deferred constraint

When running tests, we usually start a transaction and rollback to avoid mutation the database. However, when working with deferred constraint, if we did not commit the transaction, we won't be able to see the changes.

> You can issue `SET CONSTRAINTS ALL IMMEDIATE` right before the end of the transaction, which will lead to the immediate execution of the deferred constraints.
