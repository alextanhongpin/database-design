## Data Types

This folder contains information on how to utilize datatypes in Postgres (sorry MySQL users), and demonstrates practical applications.

If there are other patterns that could be useful, please submit a PR to be added here.

## Using the right Data Type

Using the right data type for the right problem will lead to really elegant solution. On the other hand, wrong usage of data type tend to lead to (permanent) tech debts.

If you have not read about the existing data types in Postgres, do so first:

[PostgreSQL: Documentation: 14: Chapter 8. Data Types](https://www.postgresql.org/docs/current/datatype.html)



## Design Thinking: Campaign Table

#### Context

We want to design a table that stores a campaign data. The campaign contains basic information - name, image and link. However, the campaign only runs for a specific duration. The campaign can for example, start a week later, and only last for a few days. Multiple campaigns can be scheduled and run concurrently. How would you design the table?



#### Concept

The main concept for the problem above is _active period_. We need a way to indicate that the campaign is active for a specific period of time.

#### Possible solution

1. Store a flag `active` to indicate if the campaign is active. However, we need to manually toggle the flag when the campaign ends, which is not desirable.

2. Store a `start_date` and `end_date` and check if the current date is between this two date. Much better, we can now schedule the campaign in advance.

3. Use postgres's `tstzrange`. Similar to above, but makes checking much easier. 



## Design Thinking: Campaign Table V2

#### Context

There is a change in requirement - now we want to schedule the campaign, but on every first week of the month of June and July. How would you adapt your previous design to accomodate this?



#### Concept

The concept is now `multiple active periods`. Instead of having just one active period, it can have multiple. 



#### Possible solution

1. Manually toggling is still possible, though not convenient (nor precise)

2. We can create another table e.g. `campaign_periods` to store the `start_date` and `end_date`. However, we might have to deal with situation where there are entries with overlapping dates. The query become slightly more complicated too, when we have to filter for the valid period.

3. Use `tstzmultirange` in Postgres 14. Sure, this data type is only available for Postgres 14 and above. However, it solves the issue of overlapping dates, and does not require a new table to be created. It can just be a column, and the query is simply
   
   ```sql
   select '{[2022-06-01, 2022-06-7), [2022-07-01, 2022-07-7)}'::tstzmultirange @> now();
   ```
   
   
   
