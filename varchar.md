## Country

Al Jumahiriyah al Arabiyah al Libiyah ash Shabiyah al Ishtirakiyah al Uzma also known as Libya is the world's longest country name at 74 characters with spaces and 63 characters without.

```sql
country varchar(80) NOT NULL DEFAULT ''
```


## URL 

https://stackoverflow.com/questions/219569/best-database-field-type-for-a-url

```sql
url varchar(2083) NOT NULL DEFAULT '';
```

Checking can be done at the application side. If the length exceeded that of 2083, just warn the client or suggest them to use a url shortener.
