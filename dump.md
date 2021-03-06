## Partial Dump

When running a local development mysql, we sometimes need data from the staging db. We can perform a partial dump using the following:

```bash
mysqldump -u root -p -h hostname \
--single-transation \ # Execute the dump as a single transaction, preventing table locks.
--opt \ # Allows the use of -where flag.
--where="1 limit 700" \ # Dump a partial set from every table in dbname consisting of max of 700 rows for each table.
dbname < dump.sql
```

Other query:
```bash
--where="id>2500000" # Take only those with id greater than 2500000
```

To restore the data:

```bash
mysql -u root -p dbname < dump.sql
```
