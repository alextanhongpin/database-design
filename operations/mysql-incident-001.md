# Incident

Useful scripts when facing incident, and how to debug issues in mysql.

## Schema

Sample test table:

```sql
create table if not exists users (
	id int auto_increment,
	name varchar(80) not null,
	email varchar(80) not null,
	primary key (id),
	unique(email)
)
```


## Show table size

```sql
SELECT
  TABLE_NAME AS `Table`,
  ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024) AS `Size (MB)`
FROM
  information_schema.TABLES
WHERE
  TABLE_SCHEMA = <dbname>
ORDER BY
  (DATA_LENGTH + INDEX_LENGTH)
DESC;
```

## Show Indexes


```
show indexes from <table> \G;
```

## Show table status

This show the table size (index + data). However, this does not show individual indexes.

```
show table status from <dbname>
```

The following columns are in bytes. To get MB, divide by (1024 * 1024).
- Avg_row_length
- Data_length
- Max_data_length
- Index_length
- Data_free


## Show indexes size

```sql
SELECT database_name, table_name, index_name,
ROUND(stat_value * @@innodb_page_size / 1024 / 1024, 2) size_in_mb
FROM mysql.innodb_index_stats
WHERE stat_name = 'size' AND index_name != 'PRIMARY'
ORDER BY size_in_mb DESC;
```


Output:


```sql
+---------------+------------+------------+------------+
| database_name | table_name | index_name | size_in_mb |
+---------------+------------+------------+------------+
| test          | users      | email      |       0.13 |
+---------------+------------+------------+------------+
1 row in set (0.06 sec)
```

## Does index persists after truncating table in Mysql?

https://dev.mysql.com/doc/refman/5.7/en/truncate-table.html


Seems like the index does get removed for email, the size reduced from 0.13mb to 0.02mb.

```
mysql> truncate table users;
Query OK, 0 rows affected (0.04 sec)

mysql> SELECT database_name, table_name, index_name,
    -> ROUND(stat_value * @@innodb_page_size / 1024 / 1024, 2) size_in_mb
    -> FROM mysql.innodb_index_stats
    -> WHERE stat_name = 'size' AND index_name != 'PRIMARY'
    -> ORDER BY size_in_mb DESC;
+---------------+------------+------------+------------+
| database_name | table_name | index_name | size_in_mb |
+---------------+------------+------------+------------+
| test          | users      | email      |       0.02 |
+---------------+------------+------------+------------+
1 row in set (0.00 sec)
```


## Kill Process

```sql
show processlist
show full processlist
```


We can emulate it by opening two sessions. In the first session, we simulate a long command by sleeping:

```sql
mysql> select sleep(30);
```

In the second session, we can see the list of running processes:
```
mysql> show processlist \G;
*************************** 1. row ***************************
     Id: 24
   User: root
   Host: localhost
     db: NULL
Command: Query
   Time: 0
  State: starting
   Info: show processlist
*************************** 2. row ***************************
     Id: 27
   User: root
   Host: localhost
     db: test
Command: Query
   Time: 1
  State: User sleep
   Info: select sleep(30)
2 rows in set (0.00 sec)

ERROR:
No query specified
```

Above, we can see a process with PID 27 that is sleeping. The `Time` shows the number of seconds that has elapsed. We can kill the PID in session 2 to stop the query execution:


```
mysql> kill 27;
Query OK, 0 rows affected (0.00 sec)
```

In session 1, after killing the query, it should terminate immediately.
We need to handle this error in the application layer.

```sql
mysql> select sleep(30);
ERROR 2006 (HY000): MySQL server has gone away
No connection. Trying to reconnect...
Connection id:    27
Current database: test

ERROR 2013 (HY000): Lost connection to MySQL server during query
mysql>
```

## Setting hint max execution time

https://dev.mysql.com/doc/refman/8.0/en/optimizer-hints.html

```
mysql> select /*+ MAX_EXECUTION_TIME(2000) */ sleep(10);
+-----------+
| sleep(10) |
+-----------+
|         1 |
+-----------+
1 row in set (2.00 sec)

```


## Mysql add column for large table


Tested with tables with 2 million rows:

```
mysql> alter table accounts add column is_married bool null;
Query OK, 0 rows affected (15.23 sec)
Records: 0  Duplicates: 0  Warnings: 0

mysql> alter table accounts add column is_active bool null;
Query OK, 0 rows affected (9.93 sec)
Records: 0  Duplicates: 0  Warnings: 0

mysql> alter table accounts add column is_verified bool not null default false;
Query OK, 0 rows affected (9.56 sec)
Records: 0  Duplicates: 0  Warnings: 0
```
