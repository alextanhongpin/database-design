# Setting connection limits for mysql library

```
db.SetMaxOpenConns(5)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```
