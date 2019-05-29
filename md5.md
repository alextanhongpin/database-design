## Storing hash

We can store the hash of the md5 as binary. The md5 length will always be 64 bytes (32 chars). In MySQL unique index, they will only compare the first 64 characters. If we want to avoid indexing the long string (e.g. url, tokens etc) which could potentially have collision up to the first 64 characters, we can create another column and hash the target string instead and set the column to be unique.

```sql
-- Statement
CREATE TABLE IF NOT EXISTS tablename (
  hash BINARY(16) NOT NULL,
)


-- Insert
UNHEX(MD5(str))
```
