# Storing unique counts

How can we use bloom filter to store unique counts?

In the database, we can store the bytes of the bloom filter **per user**.

Before incrementing the count, e.g. page views, we can first check if the user has viewed the page or not. Only increment if the user hasn't.

The issue with this approach is we dont exactly know when the user has viewed the post.

But we can always combine both approach, storing exact counts as well as using bloom filter.

## Atomicity

Is the operation atomic? It can be.

1. start transaction
2. select the user bloom filter
3. create if not exists
4. check if entry exists in bloom filter
5. if exists, return
6. otherwise, update the bloom filter and increment the count
7. commit

## Bottleneck count

The bottleneck will be on the count table, since we have one row only.

Instead, we can create multiple rows, e.g 10 rows and do a `select for update ... skip locked`.

The total count is then the sum of all individual counts. When do we decide on when to add new row? We can check the rate of page views. 

