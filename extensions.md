# Useful extensions

Case-insensitive text, `citext`:

```sql
create extension citext;

create table if not exists users (
	email_ci citext unique,
	email text unique
);

insert into users (email_ci, email) 
values ('Y@mail.com', 'X@mail.com');

select * from users where email_ci = 'x@mail.com';
```
