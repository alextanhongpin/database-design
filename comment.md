## Adding comment after table has been created

```mysql
alter table product_country modify column `country` varchar(255) NOT NULL DEFAULT '' comment 'hello world';
```
