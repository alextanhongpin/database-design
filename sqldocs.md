# SQLDocs

Unfortunately, there is nothing similar to OpenAPI docs for now for the SQL ecosystem.

One of the most important layer in most services, the `repository` layer remains undocumented. Tools like ORM etc hides the SQL statements, and it is hard to extract that from your application.

Sometimes it is useful to have the generated SQL, so that you can debug and run it on a client (e.g. for fixing data or executing the logic outside of application).

You can still document it locally, by keeping a copy of the SQLs (or maybe snapshotting them when running unit tests) locally.

You can just add the entry `sqldocs/` in `.git/exclude/hook` to exclude the folder `sqldocs`.

The `sqldocs` will then contain READMEs of the domain for each sql.

````markdown
<!--readme.md-->
# Auth Domain

## Create new users

Creates a new user with email and password. 
```sql
insert into users (email, password) values
('john.doe@mail.com', crypt('new password', gen_salt('md5')) 
returning *
```

## Ban users
...
````
