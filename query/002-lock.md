# Shared vs exclusive lock

Use shared lock for reads, use exclusive lock for write.

The equivalent to golang's `sync.RWMutex`

```
mu.RLock = shared lock
mu.Lock = exclusive lock
```


What are the usecase for select for share? use it to ensure no changes in the data read.

This could be useful for snapshotting data, e.g,

1. do a select for share on product
2. insert into order the product details and description
