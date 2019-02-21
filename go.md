# Setting connection limits for mysql library

```go
db.SetMaxOpenConns(5)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```


# Using JSON

```go
	var b []byte
	stmt := `
		SELECT JSON_OBJECT(
			"id", HEX(id),
			"email", email,
			"email_verified", IF(email_verified = 1, true, false) IS true
		) FROM employee
	`
	err = db.QueryRow(stmt).Scan(&b)
	if err != nil {
		log.Fatal(err)
	}
	var m model.Employee
	if err := json.Unmarshal(b, &m); err != nil {
		log.Fatal(err)
	}
	log.Println(m)
```
