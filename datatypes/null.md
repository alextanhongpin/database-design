

Database Null vs Not Null

Opinions may differ, but avoid using null at all for your database columns. 

1. There are scenarios where null will be useful, but having null usually complicates application especially for strongly typed language like golang. It leads to nil pointer possibility and application crashes.
2. Some argue null is useful to indicate a column is not set. When you come across that situation, ask yourself first, should the row be created if the column value does not exists? Most of the time, null columns are due to poorly normalised table, and hence having columns that are nullable unless certain criteria is set.
3. text. Is null the same as empty string? Definitely not, but even if you set it as nullable, you still have to check if the data is not null and is not empty string in the application. If you want to set nullable text but disallow empty string, add a check length > 0. 
4. unique null text. One useful scenerio for nullable text column is unique column? For user registration, name is optional in the beginning. User is created without name and is only prompted after they sign in. In this case, without the null constraint, the name will default to empty string. Two rows with empty string will conflict on the uniqueness constraint.
5. integer. At times, we do not want our quantity to default to 0. It could be entirely misleading for the application business use case to have default set to 0, and hence null is preferable. But again, think if that column itself is denormalized, and could be placed in another table where the value is required on creation.
6. bool. I have yet to see any use case where we want to have a null, true or false condition. It is misleading, and the extra check seems unnecessary.
7. date. Probably the most common nullable field is the deleted at date. This is commonly used to indicate soft deletion (or tombstoning).
8. Setting uniqueness partial index on where null is recommended.

https://www.red-gate.com/hub/product-learning/sql-prompt/problems-with-adding-not-null-columns-or-making-nullable-columns-not-null-ei028
https://stackoverflow.com/questions/21777697/why-should-i-avoid-null-values-in-a-sql-database/21777962
https://dba.stackexchange.com/questions/5222/why-shouldnt-we-allow-nulls
https://www.bennadel.com/blog/85-why-null-values-should-not-be-used-in-a-database-unless-required.htm
https://www.doorda.com/solutions/the-null-debate-should-you-use-null-values-in-your-sql-database/
https://www.mssqltips.com/sqlservertip/6303/dealing-with-a-no-null-requirement-for-data-modeling-in-sql-server/
https://www.sqlservercentral.com/articles/database-design-follies-null-vs-not-null




