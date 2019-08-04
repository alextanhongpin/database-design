## Calculating running total

Running total is the summation of sequence of numbers which is updated each time a number is added to the sequence, by adding the value of the new number to the previous running total. A.k.a cumulative function.


To compute the total number of users joined in a day:

```sql
select date, count(user_id)
from users_joined
group by date
order by date
```

To compute the running total:

```sql
select 	date, 
		count(user_id) as count
		sum(count(user_id)) over (order by date) as running_total
from users_joined
group by date
order by date
```

## Calculating running/moving average in SQL

A.k.a moving average or rolling average. Computes the running average over a selection of rows for the past number of time periods. Moving average is used to smooth out the highs and lows of the data set and get a feel for the trends in the data.

To compute a moving average that averages the previous three periods:

```sql
select quarter
	revenue,
	avg(revenue) over (order by quarter rows between 3 preceding and current row)
from amazon_revenue
```

More on postgres window functions:
http://www.postgresqltutorial.com/postgresql-window-function/


## The following is the results of the queries performed with the dataset from `data/order.sql` with postgres.

```sql
SELECT 	EXTRACT(MONTH FROM ord_date) AS month, 
		COUNT(*) 
FROM 	t_order
GROUP BY EXTRACT(MONTH FROM ord_date) 
ORDER BY EXTRACT(MONTH FROM ord_date);
```

Output: 
```
 month | count
-------+-------
     2 |     1
     3 |     1
     4 |     2
     5 |     3
     6 |     4
     7 |    11
     8 |     4
     9 |     6
    10 |     2
(9 rows)
```

```sql
SELECT 	EXTRACT(MONTH FROM ord_date) AS month, 
		COUNT(*),
		SUM(COUNT(*)) OVER (ORDER BY EXTRACT(MONTH FROM ord_date))
FROM 	t_order
GROUP BY EXTRACT(MONTH FROM ord_date) 
ORDER BY EXTRACT(MONTH FROM ord_date);
```

Output:
```
 month | count | sum
-------+-------+-----
     2 |     1 |   1
     3 |     1 |   2
     4 |     2 |   4
     5 |     3 |   7
     6 |     4 |  11
     7 |    11 |  22
     8 |     4 |  26
     9 |     6 |  32
    10 |     2 |  34
(9 rows)
```
