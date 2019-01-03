## Dump

```bash
# Dump all databases.
$ mysqldump -h ${DB_HOST} -u ${DB_USER} -p --all-databases > backup.sql

# Dump only one database.
$ mysqldump -h ${DB_HOST} -u ${DB_USER} -p --databases ${DB_NAME} > backup.sql
```

## Restore

```bash
# Must be a root user. And the database name must match.
$ mysql -h ${DB_HOST} -u root -p ${DB_NAME} < backup.sql
```
