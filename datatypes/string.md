# Strings operations

```sql
select text from reviews where text similar to '%.{10,}%';
select text from reviews where text similar to '%[a-zA-Z0-9]{10,}%';
select text, regexp_matches(text, '^\w{10,}$') from reviews;
```
