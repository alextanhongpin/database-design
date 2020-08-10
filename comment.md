# Comment

Comments can be added for tables, columns as well as functions. Sometimes it is useful to add comments to allow others to understand the design better (single source of truth). Some people use comments too to store error messages (when creating a custom postgres type for example). 


## mysql: Adding comment after table has been created

```mysql
alter table product_country modify column `country` varchar(255) NOT NULL DEFAULT '' comment 'hello world';
```
