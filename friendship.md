# Create friendship database schema
```sql
-- Pseudo-code, not actual SQL!
create table friendship (
	user uuid
	friend uuid
	status enum (pending, accepted/approved, rejected, block)
	primary key (user, friend)
	foreign key (user) references user(id),
	foreign key (friend) references user(id)
)
```
https://www.codedodle.com/2014/12/social-network-friends-database.html
