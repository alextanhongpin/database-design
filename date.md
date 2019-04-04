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
