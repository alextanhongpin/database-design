# Creating references

Sometimes we want to limit the selection of data for certain fields. A basic example is gender, where it can have only 'M' or 'F'. But what if the data is something that changes over time, say category or subcategory of an item, or country/currency list?

There are few ways to enforce the constraint:
- enum
- set
- check constraint 
- lookup table

## Example with Check Constraint

```sql
CREATE TABLE user (
  name varchar(255),
  gender char,
  CONSTRAINT valid gender
     CHECK (gender IN 'M', 'F')
)
```
## Example with Lookup Table

```sql
CREATE TABLE user (
  name varchar(255),
  gender char,
  FOREIGN KEY (gender) REFERENCES gender(type)
)

CREATE TABLE gender(
  type char PRIMARY KEY
)
```

A typical mistake is to create a reference table with an auto-increment primary key, and linking that id to the table. It makes querying more troublesome, as one as to select the type back from the reference table. 

References:
- https://www.sitepoint.com/community/t/using-enum-vs-check-constraint-vs-lookup-tables/6704/6
- https://dev.mysql.com/doc/refman/8.0/en/constraint-enum.html
