## Temporal capabilities

Exclude if there exists rows with duplicate id and name.

```sql
create table tests (
	id int,
	name text,
	exclude using gist (id with =, name with =)
);
insert into tests (id, name) values(2,'jane');
```
