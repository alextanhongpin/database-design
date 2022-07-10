## Dump

```bash
# Dump all databases.
$ mysqldump -h ${DB_HOST} -u ${DB_USER} -p --all-databases > backup.sql

# Dump only one database.
$ mysqldump -h ${DB_HOST} -u ${DB_USER} -p --databases ${DB_NAME} > backup.sql

# Perform it in a single transaction.
$ mysqldump --all-databases --single-transaction -u root -h ${DB_HOST} -p > ./db/backup/all_databases.sql

# Dump a single database.
$ mysqldump -u username -ppassword database_name  > the_whole_database_dump.sql

# Dump a single table from a database.
$ mysqldump -u username -ppassword database_name table_name > single_table_dump.sql

# Dump a single table with conditions.
$ mysqldump -u username -ppassword database_name table_name --where="date_created='2013-06-25'" > few_rows_dump.sql

# Dump a max of 700 rows from the database (Good for testing partial data from production).
$ mysqldump -u root -p -h hostname --single-transaction --opt --where="1 limit 700" dbname < dump.sql
```

Postgres:

```bash
$ pg_dump -h ${DB_HOST} -U ${DB_USER} -W -F t ${DB_NAME} > ./db/backup/all_databases.tar
```

## Restore

```bash
# Must be a root user. And the database name must match.
$ mysql -h ${DB_HOST} -u root -p ${DB_NAME} < backup.sql
```

Postgres:

```bash
$ pg_restore -h${DB_HOST} -d ${DB_NAME} ./db/backup/all_databases.tar -c -U ${DB_USER}
```
