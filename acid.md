## ACID Property

- __Atomicity__: Transaction must be treated as an atomic unit, that is, either all of its operations are executed or none. There must be no state in a database where a transaction is left partially completed. 
- __Consistency__: The database must remain in a consistent state after any transaction. No transactions should have any adverse effect on the data residing in the database.
- __Isolation__: In a database system where more than one transactions are being executed simultaneously and in parallel, the property of isolation states that all the transactions will be carried out and executed as if it is the only transaction in the system. No transaction will affect the existence of any other transaction.
- __Durability__: The database should be durable enough to hold all its latest updates even if the system fails or restarts. If a transaction updates a chunk of data in a database and commits but the system fails before the data could be written on to the disk, then that data will be updated once the system springs back into action.


## Database transaction isolation levels

A transaction isolation level is defined by the following phenomena:

Dirty read: A dirty read is the situation where a transaction reads a data that has not yet been committed. For example, transaction 1 updates a row and leaves is uncommitted. Meanwhile, transactions 2 reads the updated row. If transaction 1 rolls back the change, transaction 2 will have read data that is considered to never to have existed.
Non-repeatable read. Non repeatable read occurs when a transaction reads same row twice, and get different value each time. For example, transaction T1 reads data. Due to concurrency, transaction T2 updates the same data and commit. Now if T1 rereads the same data, it will retrieve a different value.
Phantom read: Phantom read occurs when two same queries are executed, but the rows retrieved by the two are different. For example, transaction T1 retrieves a set of rows that satisfy some search criteria. Now transaction T2 generates some new rows that match the search criteria for transaction T1. If transaction T1 re-executes the statement that reads the row, it gets a different set of rows this time.

SQL standard defines four isolation levels:
Read uncommitted: The lowest isolation level. In this level, one transaction may read not yet committed changes made by other transaction, thereby allowing dirty reads. In this level, transactions are not isolated from each other.
Read committed: This isolation levels guarantees that any data read is committed at the moment it is read. Thus it does not allow dirty read. The transaction holds a read or write lock on the current row, and thus prevent other transactions from reading, updating, or deleting them.
Repeatable read: This is the most restrictive isolation level. The transaction holds read locks on all rows it reference and writes locks on all rows it inserts, updates or deletes. Since other transaction cannot read, update, or delete these rows, consequently it avoids non-repeatable read.
Serializable: this is the highest isolation level. A serializable execution is guaranteed to be serializable. Serializable execution is defined to be an execution of operations in which concurrently executing transactions appear to be serially executing.

| Isolation level | Dirty reads | non-repeatable reads | Phantoms |
| - | - | - | - |
| Read uncommitted | May occur | May occur | May occur |
| Read committed | Don’t occur | May occur | May occur |
| Repeatable read | Don’t occur | Don’t occur | May occur |
| Serializable | Don’t occur | Don’t occur | Don’t occur |
