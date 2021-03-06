# Locking in Postgres

- There are two kinds of locking, optimistic and pessimistic
- For pessimistic locks, there are two further locks, exclusive and shared


## Skip Locked

- useful for running background jobs
- skip those rows that has been locked

## Select for update

- row locking for a specific row to prevent other resources from modifying them


## Advisory locks

- Allows you to implement application-level concurrency patterns.
- Can be session-level, or transaction-level advisory locks
- Session-level locks needs to be explicitly released
- Transaction-level locks are bound to transaction, and are released when the transaction ends


Usecases
- for same applications (multiple nodes) that are connected to the same database, we can use advisory locks as distributed locks
- to coordinate database migration when there are multiple application
- for atomicity of external api calls, e.g. call payment api once for a specific id

