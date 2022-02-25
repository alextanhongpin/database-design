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


## postgres dump and restore

```sql
select tablename from pg_tables where tableowner ='admin'
and tablename ilike '%somename' or tablename ilike '%anothername';

# restoring in parallel jobs while disabling triggers brings the restore time from 5 minutes to <5 seconds
restore:
  	export PGPASSWORD=yourpassword
	# Use your superuser credentials to restore the database (required for disabling triggers).
	pg_restore --dbname your-db --data-only --disable-triggers --superuser ${user} --jobs 4 --exit-on-error --host=localhost --port=5432 --username=${user} --no-password ./your-data-dump.dump



dump:
	pg_dump --data-only --format=custom --table "*your_table" --table "*another_table" dbname-development > dump.sql
```
