# Comment

Comments can be added for tables, columns as well as functions. Sometimes it is useful to add comments to allow others to understand the design better (single source of truth). Some people use comments too to store error messages (when creating a custom postgres type for example). 


## mysql: Adding comment after table has been created

```mysql
alter table product_country modify column `country` varchar(255) NOT NULL DEFAULT '' comment 'hello world';
```


# Best practices on Commenting in SQL

Use the following format when adding comments in SQL to explain the parts. It is easier to place all the comments at the top, then inlining them, and then adding the reference (`#A`, `#B`) to the line.


```sql
-- Retrieve recent accounts and rank them
-- #A: Accounts created in the last month, including today
-- #B: Give each account a rank, sorted by created at.
WITH recent_accounts AS (
  SELECT *
  FROM accounts
  WHERE created_at < now() - interval '1 month' -- #A
)
SELECT *, RANK() OVER (ORDER BY created_at DESC) -- #B
FROM recent_accounts
```

TL;DR;
- if steps are procedural, avoid inline comments
- logging sql queries makes comment visible
