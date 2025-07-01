## Postgres BETWEEN SYMMETRIC


BETWEEN SYMMETRIC is the same as BETWEEN except there is no requirement that the argument to the left of AND be less than or equal to the argument on the right. If it is not, those two arguments are automatically swapped, so that a nonempty range is always implied.

```
> select * from generate_series(1,10) as numbers(a)
    where numbers.a between symmetric 6 and 3;
```

Also, read more here
https://wiki.postgresql.org/wiki/Don't_Do_This
