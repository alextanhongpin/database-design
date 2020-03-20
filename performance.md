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


## Find out cache ratio, and index utilization for Postgres


### To find the cache ratio of your database
```sql
SELECT 
  sum(heap_blks_read) as heap_read,
  sum(heap_blks_hit)  as heap_hit,
  sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as ratio
FROM 
  pg_statio_user_tables;
```

### Understanding index usage

To generate a list of your tables in your database with the largest ones first and the percentage of time which they use an index you can run:
```sql
SELECT 
  relname, 
  100 * idx_scan / (seq_scan + idx_scan) percent_of_times_index_used, 
  n_live_tup rows_in_table
FROM 
  pg_stat_user_tables
WHERE 
    seq_scan + idx_scan > 0 
ORDER BY 
  n_live_tup DESC;
```
### Index Cache Hit Rate
```sql
SELECT 
  sum(idx_blks_read) as idx_read,
  sum(idx_blks_hit)  as idx_hit,
  (sum(idx_blks_hit) - sum(idx_blks_read)) / sum(idx_blks_hit) as ratio
FROM 
  pg_statio_user_indexes;
```
  
References:
- https://www.citusdata.com/blog/2016/10/12/count-performance/
- http://www.craigkerstiens.com/2012/10/01/understanding-postgres-performance/
 
## Checking Postgres performance

```
BEGIN;
    EXPLAIN ANALYZE sql_statement;
ROLLBACK;
```
