## Setting character set (deprecated as of MySQL 8, see below)

```sql
CREATE TABLE IF NOT EXISTS tableName (
   ...
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;
```


## To check if `utf8mb4` is set correctly.
```sql
mysql> SHOW VARIABLES WHERE Variable_name LIKE 'character\_set\_%' OR Variable_name LIKE 'collation%';
+--------------------------+--------------------+
| Variable_name            | Value              |
+--------------------------+--------------------+
| character_set_client     | utf8mb4            |
| character_set_connection | utf8mb4            |
| character_set_database   | utf8mb4            |
| character_set_filesystem | binary             |
| character_set_results    | utf8mb4            |
| character_set_server     | utf8mb4            |
| character_set_system     | utf8               |
| collation_connection     | utf8mb4_unicode_ci |
| collation_database       | utf8mb4_unicode_ci |
| collation_server         | utf8mb4_unicode_ci |
+--------------------------+--------------------+
10 rows in set (0.00 sec)
```

## Connection string

Example in golang:
```go
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci", opt.User, opt.Password, opt.Host, opt.Database)
```

## MySQL 8

> From MySQL 8.0, utf8mb4 is the default character set, and the default collation for utf8mb4 is utf8mb4_0900_ai_ci. MySQL 8.0 is also coming with a whole new set of Unicode collations for the utf8mb4 character set. [^1]


[^1]: https://dev.mysql.com/blog-archive/mysql-8-0-collations-migrating-from-older-collations/#:~:text=From%20MySQL%208.0%2C%20utf8mb4%20is,of%20the%20complete%20Unicode%209.0.
