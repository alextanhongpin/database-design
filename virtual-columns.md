# Virtual columns

When to store values in column? Database columns should not contain computed values - there are exceptions: 
- caching association counts can improve query performance, especially if the count does not need to be accurate
- e.g. users has posts, reviews, and comments. Rather than performing the query everytime to get those counts, we store them in the table
- computed columns can be sorted - we can now sort the rows by the reviews count, comments count, ranking etc
- we can use a prefix to distinguish the normal columns from virtual columns, e.g. `cached_comments_counts`, `cached_posts_count`

Store in column pros
- Can sort
- Values are easy to query

Store in columns cons
- Need to update
- Values might not be accurate over time
- Additional column for every other field

Store in redis pros
- Fast
- Can be decentralised
- Store only what you need, reduce storage on database
- Precomputed value
