# Dealing with counters

There are several ways to achieve this
- triggers on update/delete
- with CTE to insert
- perform at application level (insert, then update)
- materialized views
- computed values (in postgres, there’s only generated values now)

## Implementing Rail's counter cache using trigger

```sql
DROP TABLE fake_question CASCADE;
CREATE TABLE IF NOT EXISTS fake_question (
	id uuid DEFAULT uuid_generate_v4() PRIMARY KEY,
	comments_count int not null default 0,
	body text not null
);

DROP TABLE fake_comment;
CREATE TABLE IF NOT EXISTS fake_comment (
	id uuid DEFAULT uuid_generate_v4() PRIMARY KEY,
	question_id uuid NOT NULL REFERENCES fake_question(id),
	body text NOT NULL
);

CREATE FUNCTION increment_counter(table_name text, column_name text, id uuid, step integer)
	RETURNS VOID AS $$
		BEGIN
			EXECUTE format('UPDATE %I SET %I = %I + $1 WHERE id = $2', table_name, column_name, column_name)
			USING step, id;
		END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION counter_cache()
RETURNS trigger AS $$
	DECLARE
		table_name text := TG_ARGV[0];
		counter_name text := TG_ARGV[1];
		fk_name text := TG_ARGV[2];
		fk_changed boolean := false;
		fk_value uuid;
		record record;
	BEGIN
		IF TG_OP = 'UPDATE' THEN
			record := NEW;
			EXECUTE format('SELECT ($1).%I != ($2).%I', fk_name, fk_name)
			INTO fk_changed
			USING OLD, NEW;
		END IF;
		
		IF TG_OP = 'DELETE' OR fk_changed THEN
			record := OLD;
			EXECUTE format('SELECT ($1).%I', fk_name)
			INTO fk_value USING record;
			PERFORM increment_counter(table_name, counter_name, fk_value, -1);
		END IF;
		
		IF TG_OP = 'INSERT' OR fk_changed THEN
			record := NEW;
			EXECUTE format('SELECT ($1).%I', fk_name)
			INTO fk_value USING record;
			PERFORM increment_counter(table_name, counter_name, fk_value, +1);
		END IF;
		
		RETURN record;
	END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_questions_comments_count
AFTER INSERT OR UPDATE OR DELETE ON fake_comment
FOR EACH ROW EXECUTE PROCEDURE counter_cache('fake_question', 'comments_count', 'question_id');
```

The implementation is based on the reference below, except the usage of `format` for create the statement, and using `uuid` as the foreign key type.
References:
- http://shuber.io/porting-activerecord-counter-cache-behavior-to-postgres/


## Using CTE to perform double insert

Creating the tables:
```sql
create table fake_question (
	id serial primary key,
	name text not null,
	comments_count int not null default 0
);
drop table fake_question;

create table fake_comment (
	id serial primary key,
	name text not null,
	question_id int not null references fake_question(id)
);
drop table fake_comment;
```

Insert the first question:

```sql
insert into fake_question (name) values ('hello');
```

Insert the comment, and incrementing the question's comments count:
```sql
with inserted_comment as (
	insert into fake_comment (name, question_id) values ('my comment', 1)
	returning *
)
update fake_question set comments_count = comments_count + 1
where id = (select id from inserted_comment)
returning *;

select * from fake_question;
```

Deleting the comment, and decrementing the question's comments count:
```sql
with deleted_comment as (
	delete from fake_comment where id = 1
	returning *
)
update fake_question set comments_count = comments_count - 1
where id = (select id from deleted_comment)
returning *;
```

What if we want the `RETURNING *` to return `comment` instead of `question`?
```sql
-- Insert comment.
with updated_question as (
	update fake_question set comments_count = comments_count + 1
	where id = 1
	returning id
)
insert into fake_comment (name, question_id) values ('my comment', (select id from updated_question))
returning *;

-- Delete comment.
with updated_question as (
	update fake_question set comments_count = comments_count - 1
	where id = (select question_id from fake_comment where id = 3 limit 1)
	returning *
)
delete from fake_comment where (id, question_id) = (select 3, question_id from updated_question)
returning *;
```
