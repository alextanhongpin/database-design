## UUID v4

There are no official support for uuid v4 in Mysql, and the `uuid` function is only for uuid v1.

However, you can create a custom function to do so [^1]:

```sql
DELIMITER --
CREATE FUNCTION uuid_v4() RETURNS CHAR(36)

BEGIN
    RETURN LOWER(CONCAT(
            HEX(RANDOM_BYTES(4)),
            '-', HEX(RANDOM_BYTES(2)),
            '-4', SUBSTR(HEX(RANDOM_BYTES(2)), 2, 3),
            '-', HEX(FLOOR(ASCII(RANDOM_BYTES(1)) / 64) + 8), SUBSTR(HEX(RANDOM_BYTES(2)), 2, 3),
            '-', hex(RANDOM_BYTES(6))
        ));
END;
--
DELIMITER ;
```

Simply running this will produce an error:
```
ERROR 1418 (HY000): This function has none of DETERMINISTIC, NO SQL, or READS SQL DATA in its declaration and binary logging is enabled (you *might* want to use the less safe log_bin_trust_function_creators variable)
```

The stackoverflow [^2] discusses this issue in depth.

```sql
SET GLOBAL log_bin_trust_function_creators = 1;
```

You can now generate uuid v4:

```sql
SELECT uuid_v4();
SELECT uuid_to_bin(uuid_v4());
SELECT bin_to_uuid(uuid_to_bin(uuid_v4()));

SELECT uuid_to_bin('47cd9133-c46f-4306-af5c-c16cbc4bc047');
SELECT bin_to_uuid(uuid_to_bin('47cd9133-c46f-4306-af5c-c16cbc4bc047'));
```

Do note that the `uuid_v4()` cannot be used as default function when creating table:

```sql
CREATE TABLE users (
	uuid binary default (uuid_to_bin(uuid_v4())),
	name varchar(255) not null
);

ERROR 3770 (HY000): Default value expression of column 'uuid' contains a disallowed function: `uuid_v4`.
```

With `uuid` v1 however, it works:
```sql
CREATE TABLE users (
	uuid binary(16) default (uuid_to_bin(uuid())),
	name varchar(255) not null
);
```

[^1]: https://emmer.dev/blog/generating-v4-uuids-in-mysql/
[^2]: https://stackoverflow.com/questions/26015160/deterministic-no-sql-or-reads-sql-data-in-its-declaration-and-binary-logging-i
