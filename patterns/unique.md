MySQL has a limit of 3072 characters when setting unique index. [^1].

What this means is, if you have two rows, each with 3073 character and this two row are unique, they will be detected as non unique because MySQL only compares the first 3072 characters.

[^1]: https://dev.mysql.com/doc/refman/8.0/en/innodb-limits.html#:~:text=The%20index%20key%20prefix%20length,REDUNDANT%20or%20COMPACT%20row%20format.
