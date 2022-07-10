## Using Check Constraint to guard against polymorphic association

```diff sql
create table test_subscription (
+ 	-- NOTE: Don't need to double-check, since we already check it at the bottom
-	type text check (type in ('question', 'answer', 'comment')),
+	type text not null,
	answer_id uuid references answer(id),
	question_id uuid references question(id),
	comment_id uuid references comment(id),
	check ((type = 'question' and question_id is not null) or (type = 'answer' and answer_id is not null) or (type = 'comment' and comment_id is not null))
);

-- Modifying check constraint.
alter table test_subscription 
drop constraint test_subscription_check, 
add constraint test_subscription_check 
check ((type = 'question' and question_id is not null) or (type = 'answer' and answer_id is not null) or (type = 'comment' and comment_id is not null));
```

Improvement (removed unnecessay type column), adding partial index:
```sql
create table subscription_type (
	id serial primary key,
	category text not null,
	action text not null,
	unique (category, action)
);


create table subscription (
	subscription_type_id int not null references subscription_type(id),
	answer_id uuid references answer(id),
	question_id uuid references question(id),
	comment_id uuid references comment(id),
	check (
		(answer_id is not null)::int + 
		(question_id is not null)::int + 
		(comment_id is not null)::int
		= 1
	)
);

create unique index on subscription (answer_id, subscription_type_id) where answer_id is not null;
create unique index on subscription (question_id, subscription_type_id) where question_id is not null;
create unique index on subscription (comment_id, subscription_type_id) where comment_id is not null;
```


References:
https://hashrocket.com/blog/posts/modeling-polymorphic-associations-in-a-relational-database

## Class Table Inheritance

```sql
CREATE TABLE room (
    room_id serial primary key,
    room_type VARCHAR not null,

    CHECK CONSTRAINT room_type in ("standard_room","family_room"),
    UNIQUE (room_id, room_type)
);

CREATE_TABLE standard_room (
    room_id integer primary key,
    room_type VARCHAR not null default "standard_room",

    FOREIGN KEY (room_id, room_type) REFERENCES room (room_id, room_type),
    CHECK CONSTRAINT room_type  = "standard_room"
);
CREATE_TABLE family_room (
    room_id integer primary key,
    room_type VARCHAR not null default "family_room",

    FOREIGN KEY (room_id, room_type) REFERENCES room (room_id, room_type),
    CHECK CONSTRAINT room_type  = "family_room"
);
```

## Other approaches for performing polymorphism in database
- the rails way (creating a `polymorphic_type` and `polymorphic_id` column. Don't do this. Even though it is simple, you really lose the referential integrity. Deleting the associations will also leave the association "hanging")
- multiple database
- table inheritance
- using union to provide interface for polymorphic types!
- multiple foreign keys, with a constraint to check the type

https://hashrocket.com/blog/posts/modeling-polymorphic-associations-in-a-relational-database
http://duhallowgreygeek.com/polymorphic-association-bad-sql-smell/
https://www.vertabelo.com/blog/inheritance-in-a-relational-database/
https://stackoverflow.com/questions/5466163/same-data-from-different-entities-in-database-best-practice-phone-numbers-ex/5471265#5471265


## Achieving true polymorphic?

This is inspired from graphql [global object identification](https://graphql.org/learn/global-object-identification/) implementation, where all entities inherit a single node and each entity has a unique id.


```sql
create table if not exists pg_temp.node (
	id uuid not null default gen_random_uuid(),
	type text not null check (type in ('feed', 'post', 'comment', 'human')),
	primary key (id, type),
	unique (id)
);

create or replace function pg_temp.gen_node_id(_type text) returns uuid as $$
	declare
		_id uuid;
	begin
		insert into pg_temp.node (type) values (_type)
		returning id into _id;
		return _id;
	end;
$$ language plpgsql;


create table if not exists pg_temp.human (
	id uuid not null default pg_temp.gen_node_id('human'),
	type text not null default 'human' check (type = 'human'),
	name text not null,
	email text not null,
	unique (email),
	primary key(id), -- We don't need to define the 'type' here as primary key, because the constraint will have already been applied in the node table.
	foreign key(id, type) references node(id, type)
);

insert into pg_temp.human(name, email) values ('john', 'john.doe@mail.com');


create table if not exists pg_temp.feed (
	id uuid not null default pg_temp.gen_node_id('feed'),
	type text not null default 'feed' check (type = 'feed'),
	user_id uuid not null,
	body text not null,
	primary key(id),
	foreign key(id, type) references pg_temp.node(id, type),
	foreign key(user_id) references pg_temp.human(id)
);

create table if not exists pg_temp.post (
	id uuid not null default pg_temp.gen_node_id('post'),
	type text not null default 'post' check (type = 'post'),
	user_id uuid not null,
	body text not null,
	primary key(id),
	foreign key(id, type) references pg_temp.node(id, type),
	foreign key(user_id) references pg_temp.human(id)
);

create table if not exists pg_temp.comment (
	id uuid not null default pg_temp.gen_node_id('comment'),
	type text not null default 'comment' check (type = 'comment'),
	user_id uuid not null,
	body text not null,
	commentable_id uuid not null,
	commentable_type text not null check (commentable_type in ('post', 'feed')),
	primary key(id),
	foreign key(id, type) references node(id, type),
	foreign key(commentable_id, commentable_type) references node(id, type),
	foreign key(user_id) references pg_temp.human(id)
);


insert into pg_temp.feed(user_id, body) values 
((select id from pg_temp.human), 'this is a new feed');

insert into pg_temp.post(user_id, body) values 
((select id from pg_temp.human), 'this is a new post');

insert into pg_temp.comment (commentable_id, commentable_type, user_id, body) values
((select id from pg_temp.post), 'post', (select id from pg_temp.human), 'this is a comment on post');

insert into pg_temp.comment (commentable_id, commentable_type, user_id, body) values
((select id from pg_temp.feed), 'feed', (select id from pg_temp.human), 'this is a comment on feed');
```

Polymorphic table comment:
```sql
select * from pg_temp.comment;
```


| id | type | user_id | body | commentable_id | commentable_type |
| -- | ---- | --------| ---- | -------------- | ---------------- |
| 31e12f91-ea42-492f-988e-3eb17bb1b9dc	| comment	| 396b8bae-fff9-44cc-b28e-49a22625a671	| this is a comment on post	| 09b8ccb0-8dd9-4a8e-b0d6-a28b296ac7a4	| post| 
| 6704d0cb-4d83-41c0-b3eb-1d32bc05452f| 	comment| 	396b8bae-fff9-44cc-b28e-49a22625a671| 	this is a comment on feed| 	b64b571b-02ae-4c55-b786-cc6764b40eda| 	feed| 
