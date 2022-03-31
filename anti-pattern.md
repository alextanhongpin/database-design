# Data vs Query

I have made this mistake a lot of times, and I still do it.

> Never mix data with query


The data stored in postgres should be as raw as possible.

- We should not mix derived data into the table that stores data
- sometimes we need derived data to be sored, because it makes query possible
- e.g. is author and books, we want to store the total books written by annauthor, because there is a requirement to rank authors by how many books to write, which involves pagination
- any logic that requires fetching subset of data with some filter logic requires computing the whole table
- we can use a lot of approach such as subquery, lateral join, views for real time data or materialized view for eventually consistent data
- we can also create another table, e.g author meta or author books count
- using materialised view makes it easy if accuracy is not important, as you xan refresh them periodically through trigger
 
