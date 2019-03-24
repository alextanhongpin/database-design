Simple regular expression to sanitize sql string. Example in golang:
```go
valid := regexp.MustCompile("^[A-Za-z0-9_]+$")
if !valid.MatchString(ordCol) {
    // invalid column name, do not proceed in order to prevent SQL injection
}
```
