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
