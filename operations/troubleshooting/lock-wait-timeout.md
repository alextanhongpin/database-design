# Lock Wait Timeout


## Access mysql shell

```bash
$ alias dc=docker-compose
$ dc exec mysql mysql
```

How to simulate lock wait timeout in mysql?

```sql
show databases;
use test;

create table users (
    id bigint auto_increment,
    name varchar(80) not null,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp on update current_timestamp,
    primary key (id)
);

-- Insert a bunch of users
insert into users(name) values ('alice'), ('bob'), ('charles');
```

## Check existing session variables

[innodb_lock_wait_timeout](https://dev.mysql.com/doc/refman/5.7/en/innodb-parameters.html#sysvar_innodb_lock_wait_timeout) default is 50s:

```mysql
mysql> select @@session.innodb_lock_wait_timeout;
+-----------------------------+
| @@session.innodb_lock_wait_timeout|
+-----------------------------+
|                          50 |
+-----------------------------+
1 row in set (0.00 sec)
```

Setting the value to 5s:

```mysql
set @@session.innodb_lock_wait_timeout = 5;
```

Example error when exceeded:

```mysql
ERROR 1205 (HY000): Lock wait timeout exceeded; try restarting transaction
```


[max_execution_time](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_max_execution_time) default is 0. The value is `milliseconds`, not `seconds` like `innodb_lock_wait_timeout`.
```mysql

mysql> select @@session.max_execution_time;
+------------------------------+
| @@session.max_execution_time |
+------------------------------+
|                            0 |
+------------------------------+
1 row in set (0.00 sec)

```


Example error when exceeded:

```sql
ERROR 3024 (HY000): Query execution was interrupted, maximum statement execution time exceeded
```

We can replace `session` with `global` to see the global variables.


## Other timeouts

There's also [wait_timeout](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_wait_timeout) as well as `interactive_timeout`, but it doesn't seem useful.

## Show locked tables

```mysql

show open tables where in_use>0;
```

## Check pending InnoDB transactions

```sql
SELECT * FROM `information_schema`.`innodb_trx` ORDER BY `trx_started` \G;
```

## Check lock dependency - what blocks what


mysql> SELECT * FROM `information_schema`.`innodb_locks` \G;
*************************** 1. row ***************************
    lock_id: 8001:40:3:23
lock_trx_id: 8001
  lock_mode: X
  lock_type: RECORD
 lock_table: `test`.`users`
 lock_index: PRIMARY
 lock_space: 40
  lock_page: 3
   lock_rec: 23
  lock_data: 1
*************************** 2. row ***************************
    lock_id: 8002:40:3:23
lock_trx_id: 8002
  lock_mode: X
  lock_type: RECORD
 lock_table: `test`.`users`
 lock_index: PRIMARY
 lock_space: 40
  lock_page: 3
   lock_rec: 23
  lock_data: 1
2 rows in set, 1 warning (0.00 sec)

ERROR:
