## Select as time zone

```
SELECT '2020-04-28 09:27:58.597317'::timestamptz AT TIME ZONE 'Asia/Singapore';
SELECT '2020-04-28 09:27:58.597317'::timestamptz AT TIME ZONE 'sgt';
```


## Updating from timestamp with out time zone to timestamp with time zone for postgres

Safe to update column. Timestamp information will be added.

```postgres
create table a(t1 timestamp without time zone, t2 timestamptz);
insert into a (t1) values ('2020-04-28 09:27:58.597317');
update a set t2 = t1;
select * from a;
alter table a alter column t1 type timestamptz;
select * from a;


select t1 at time zone 'asia/singapore' at time zone 'utc' t1,
t2 at time zone 'sgt' at time zone 'utc' t1_sgt,
t2 at time zone 'sgt' t1_sgt2,
t2 at time zone 'utc' at time zone 'asia/singapore' t2,
t1 as ori_t1,
t2 as ori_t2
from a;

drop table a;
```

## Setting
```
select current_setting('timezone');
set session time zone 'Asia/Singapore';
select name, created_at at time zone 'utc' at time zone 'sgt' from users order by created_at desc limit 10;
select name, created_at at time zone 'UTC' at time zone 'SGT' from users order by created_at desc limit 10;
```

## Notes

When querying `between` or date range, always remember to query at the correct timezone
