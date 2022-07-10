# Homebrew install specific version

```bash
$ brew install postgresql@9.6
```

# Check if port 5432 is being used

```bash
$ lsof -i :5432

# Kill the port.
$ kill -9 <pid>
```
