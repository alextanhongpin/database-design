## Check the number of full scans

The output from EXPLAIN shows ALL in the type column when MySQL uses a full table scan to resolve a query.

```mysql
SHOW GLOBAL STATUS WHERE Variable_name like 'select%';
```

Output:
```
Select_full_join	10733
Select_full_range_join	27
Select_range	189884
Select_range_check	0
Select_scan	2038944
```

References:
- https://dev.mysql.com/doc/refman/8.0/en/table-scan-avoidance.html
