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

