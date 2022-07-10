# Checking if port 5432 is used, and killing the process that is running
```bash
$ lsof -i :5432
$ kill -9 <pid>
```

# Install using homebrew

Use docker if possible:

```go
# Install older version
$ brew install postgresql@9.6
```
