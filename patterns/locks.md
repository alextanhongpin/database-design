# Locking in Postgres

- There are two kinds of locking, optimistic and pessimistic
- For pessimistic locks, there are two further locks, exclusive and shared

## Optimistic locking

a.k.a optimistic concurrency locking by using versioning.

```sql
UPDATE accounts SET balance = balance - 1
WHERE user_id = 1 AND version = 1;
```

An implementation with trigger, similar to Hibernate's Oplock [2].
```sql
CREATE TABLE tab (
	id integer primary key,
	somefield text not null,
	version int not null default 0
);

CREATE OR REPLACE FUNCTION update_row_version() RETURNS trigger AS $$
BEGIN
	IF TG_OP = 'UPDATE' AND NEW.version = OLD.version AND NEW.* IS DISTINCT FROM OLD.* THEN
		NEW.version = NEW.version + 1;
	END IF;
	RETURN NEW;
END;
$$ language plpgsql;

COMMENT ON FUNCTION update_row_version() IS 'Increments the record version if a row changed by an update and it''s version was not incremented by the user';


CREATE TRIGGER update_row_version
BEFORE UPDATE ON tab
FOR EACH ROW
EXECUTE PROCEDURE update_row_version();
```

## Skip Locked

- useful for running background jobs
- skip those rows that has been locked

## Select for update

- row locking for a specific row to prevent other resources from modifying them
- however, this can lead to deadlocks [1]
- to avoid deadlocks, use serializable isolation

## Serializable

- Use serializable to avoid modifying data that has changed
- Example, original post description is `hello` user A update post description to `hello1` at 10.00am. Another user B updates post description to `hello2` at 10.01am and save the post, which will be blocked by user A. User A saves at 10.02 am. User B saves is committed, but user B does not know that the description has change to `hello1`
- In this scenario, we want the user to be aware of the updates before they make a new change - in others words, each change must be sequential
- we can solve it with optimistic concurrency locking, with versioning, but it could lead to deadlocks
- alternative to `select for update`
```sql
begin isolation level serializable;
```

## Advisory locks

- Allows you to implement application-level concurrency patterns.
- Can be session-level, or transaction-level advisory locks
- Session-level locks needs to be explicitly released
- Transaction-level locks are bound to transaction, and are released when the transaction ends
- Pros, no table locking
- See pitfall with go programming language [7]

Usecases
- for same applications (multiple nodes) that are connected to the same database, we can use advisory locks as distributed locks
- to coordinate database migration when there are multiple application
- for atomicity of external api calls, e.g. call payment api once for a specific id

## Skip Lock 

Can be used to implement job queues in postgres, see [8]:

```sql
BEGIN;
DELETE FROM your_table WHERE id = (
	SELECT * 
	FROM your_table
	ORDER BY id
	FOR UPDATE SKIP LOCKED
	LIMIT 1
);

-- do your work

COMMIT;
```

## Using Postgres to replace redis

See [9]
- database queue with skip lock
- application locks with advisory lock

# References
1. [Postgresql: Explicit Locking](https://www.postgresql.org/docs/current/explicit-locking.html)
2. [Hibernate Oplocks](https://wiki.postgresql.org/wiki/Hibernate_oplocks)
3. [2ndquadrant: Postgresql anti-pattern: Read-modify-write-cycle](https://www.2ndquadrant.com/en/blog/postgresql-anti-patterns-read-modify-write-cycles/)
4. [StackOverflow: Optimistic concurrency control across tables in postgres](https://stackoverflow.com/questions/37801598/optimistic-concurrency-control-across-tables-in-postgres)
5. [Optimistic-pessimistic locking sql](https://learning-notes.mistermicheels.com/data/sql/optimistic-pessimistic-locking-sql/)
6. https://particular.net/blog/optimizations-to-scatter-gather-sagas
7. https://engineering.qubecinema.com/2019/08/26/unlocking-advisory-locks.html
8. https://www.2ndquadrant.com/en/blog/what-is-select-skip-locked-for-in-postgresql-9-5/
9. https://spin.atomicobject.com/2021/02/04/redis-postgresql/
