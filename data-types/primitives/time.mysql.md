
# MySQL

## Timezone

```sql
select @@time_zone;
select now();
select convert_tz(now(), @@time_zone, 'Asia/Jakarta');
select convert_tz('2023-11-22', 'Asia/Jakarta', @@time_zone);
```
