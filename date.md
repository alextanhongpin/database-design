## Query rows from the last nth minute

```sql
CREATE TABLE test (date datetime);
INSERT INTO test (date) VALUES (CURRENT_TIMESTAMP());

-- Query item from the last minute.
SELECT * FROM test WHERE date > date_sub(now(), interval 1 minute);
```

## Query expired item after 1 month

```sql
SELECT * from TABLE WHERE registered < DATE_SUB(CURRENT_DATE(), INTERVAL 30 day)
```

## Get rows that is today

```sql
SELECT users.id, DATE_FORMAT(users.signup_date, '%Y-%m-%d') 
FROM users 
WHERE DATE(signup_date) = CURDATE()
```

## Format to dd-mm-yyyy

```sql
SELECT date_format(NOW(), '%d-%m-%Y') as ddmmyyyy;
```


## How to store opening hours
```sql
shop_id binary(16)
-- weekday tinyint(1) -- SELECT DAYOFWEEK('2007-02-03'), values from 1 to 7. But how about starting and ending in different days?
start_day tinyint(1) 
end_day tinyint(1)
-- opening_hour TIME -- format 'HH:MM:SS' NOTE: There's no type time in MySQL
opens TIME
-- closing_hour TIME
closes TIME
timezone GMT +2
closed_dates JSON
```

- How to deal with timezone? Store the Timezone info for people viewing in different countries. Since the opening/closing time will differ greatly. Else, need to convert the timezone to UTC.
- How to deal with opening time and closing time for different days? Store the start day and end day, e.g. start day is Sunday, end day is Mondy, opening hours is 10.00 a.m., closing hours is 2.00 a.m.
- to query all the opening time, just select all from the same shop id
- how to handle exceptional cases (closed on 1 day)? store the additional closed dates as a json array, then iterate and compare the date when it is closed.
- what if the store is opened for 24/7? start_day, end_day is the same, opening hour and closing hour is the same
- what if it has two opening hours on the same day? create the same entry for the same weekday, with different time
- how to set if the store is closed on a particular date? set the closed dates
- what if the store is not open on a day? don't create the entry

https://stackoverflow.com/questions/19545597/way-to-store-various-shop-opening-times-in-a-database
http://www.remy-mellet.com/blog/288-storing-opening-and-closing-times-in-database-for-stores/
https://stackoverflow.com/questions/4464898/best-way-to-store-working-hours-and-query-it-efficiently

## Calculate Age

```
SELECT TIMESTAMPDIFF(YEAR, '1970-02-01', CURDATE()) AS age
```

## Group by age bucket

```sql
SELECT
    SUM(IF(age < 20,1,0)) as 'Under 20',
    SUM(IF(age BETWEEN 20 and 29,1,0)) as '20 - 29',
    SUM(IF(age BETWEEN 30 and 39,1,0)) as '30 - 39',
    SUM(IF(age BETWEEN 40 and 49,1,0)) as '40 - 49',
...etc.
FROM inquiries;
```


## Golang test time
Go time.Time has nanosecond resolution, MySQL datetime has second resolution (use datetime(6) for microseconds). Go has a timezone, MySQL doesn't.
```
  Expected: '2019-04-04 11:45:41.170518 +0800 +08 m=+0.223437462'
  Actual:   '2019-04-04 03:45:41 +0000 UTC'
```

To make the test work, round the seconds and convert to UTC:
```
time.Now().Round(time.Second).UTC())
```

## SQL Date

Hoping to get all records within a month...
```sql
BETWEEN 2019-01-01 AND 2019-01-31
```

But this will only take records less than `2019-01-31 00:00:00`. The below query is correct:
```sql
BETWEEN 2019-01-01 AND 2019-02-01
```

## SQL Timezone

Common mistake is to query the start to end of the date in UTC, which is different when comparing against local timezone.
```sql
SELECT * FROM mysql.time_zone;
SELECT * FROM mysql.time_zone_name;
select current_timestamp;
-- 2019-04-09 02:08:18

select CONVERT_TZ(current_timestamp, 'GMT', 'Singapore');
-- 2019-04-09 10:08:24


-- To query the difference
SELECT
id, created_at, 
DATE(created_at) AS date_utc, 
DATE(convert_tz(created_at, 'GMT', 'Singapore')) AS date_local
FROM employee_activity 
WHERE DATE(convert_tz(created_at, 'GMT', 'Singapore')) != DATE(created_at);
```

To find the records on a specific date (local timezone!):

```sql
-- This query is incorrect, because it will select those dates that are based on UTC.
SELECT
id, created_at, 
DATE(created_at) AS date_utc, 
DATE(convert_tz(created_at, 'GMT', 'Singapore')) AS date_local
FROM employee_activity 
WHERE DATE(created_at) = '2019-03-04';

-- This query is correct, because the dates are first converted into local timezone before queried.
SELECT
id, created_at, 
DATE(created_at) AS date_utc, 
DATE(convert_tz(created_at, 'GMT', 'Singapore')) AS date_local
FROM employee_activity 
WHERE DATE(CONVERT_TZ(created_at, 'GMT', 'Singapore')) = '2019-03-04';
```


## Difference in days, hours ...
```
mysql> SELECT TIMESTAMPDIFF(MONTH,'2003-02-01','2003-05-01');
        -> 3
mysql> SELECT TIMESTAMPDIFF(YEAR,'2002-05-01','2001-01-01');
        -> -1
mysql> SELECT TIMESTAMPDIFF(MINUTE,'2003-02-01','2003-05-01 12:05:55');
        -> 128885
```

For difference in days:

```sql
datediff(current_timestamp, created_at)
```

To check how many days have elapsed (10 is the number of days elapsed):

```sql
datediff(current_timestamp, created_at) > 10
```


## Default/Null Date

There are some disadvantages of using NULL date, eg. it cannot be indexed (you might need it later), and marshalling them can be a pain when using a strongly typed language (null types needs to be type asserted). 

There are some cases where the NULL values can be [optimized](https://dev.mysql.com/doc/refman/8.0/en/is-null-optimization.html).

For dates, it's best to use a default date range rather than `null`, with the only exception being the `deleted_at` date (since it is easier to check if `deleted_at IS NULL` rather than `deleted_at = 9999-12-31').

TL;DR;

- valid_from: `1000-01-01`
- valid_till: `9999-12-31`

## DATE vs DATETIME

For validity period that has a period ranging within days/weeks/months/year, using `DATE` will be sufficient. 

For actions (approvals, update, creation, logging, audit), use `DATETIME` for better accuracy.


## Date Elapsed

```mysql
-- Get the last day of the month.
select last_day(current_date) as last_day;

-- Get the first day of the month.
select DATE_ADD(
	DATE_ADD(LAST_DAY(current_date),INTERVAL 1 DAY),
	INTERVAL - 1 MONTH) AS first_day;

-- Get the max date (if registered on the same month) or the start of the month
select GREATEST('2019-03-12', DATE_ADD(
	DATE_ADD(LAST_DAY(current_date),INTERVAL 1 DAY),
	INTERVAL - 1 MONTH));
	
-- Find the difference in date between the current date and the last day.

-- Get the max date (if registered on the same month) or the start of the month.
-- The order matters - the end of the month must be first.
select datediff(
	last_day(current_date), 
	-- Compare the subscription date vs the start of the month, the greater one takes priority
	GREATEST('2019-04-12', DATE_ADD(
	DATE_ADD(LAST_DAY(current_date),INTERVAL 1 DAY),
	INTERVAL - 1 MONTH))
);

-- Latest date - now.
select datediff('2019-05-31', current_date);
```

## Timezone difference

Rather than setting the timezone information for the user, set the timezone information on the products/events instead. So if the user purchases the product with the said timezone, it is much easier to process the difference in the timezone. With that said, this means that for each product, there is a need to create different product with different timezone, and there's a logic required to show the different products by different countries too, possibly by the user location or ip information.

Why does this matter? Because if there's a promotion in Malaysia (GMT+8), then if the sale is supposed to end at 12:00 am GMT+8, if the server time is set to UTC instead, the closing time would have been different (it would end earlier) and this could cause a lot of miscommunication.

That said, it is best to store the dates as UTC in the database. But the timezone information should be stored somewhere else too so that the dates can be computed correctly.


## Date Range


When calculating the date difference Use `curdate()/current_date()`, not `now()` since `now()` includes the time.

```sql
# Get policy expiring in 7 days, UTC time
policy.end_date = DATE_ADD(CURDATE(), INTERVAL 7 DAY);

# Get policy expiring in 30 days, UTC time
policy.end_date = DATE_ADD(CURDATE(), INTERVAL 30 DAY);
```

