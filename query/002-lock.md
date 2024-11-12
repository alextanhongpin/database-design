# Shared vs exclusive lock

Use shared lock for reads, use exclusive lock for write.

The equivalent to golang's `sync.RWMutex`

```
mu.RLock = shared lock
mu.Lock = exclusive lock
```
