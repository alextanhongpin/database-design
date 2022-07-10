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


## Designing Friendship column

To simplify query last time, I added a trigger to compute the hash of the sorted ids when defining a user/friend relationship. 

This is one interesting problem that I faced when designing a friendship table - I need to create two rows with both the user id (user_id, friend_id) pair. However, querying becomes complex, as now I querying for the pair requires a union (and indices on both side). One way to solve it is to create another column that is the hash of both ids, sorted. The idea is to create a trigger that will sort both ids, hash them as md5, and store it in another column.
```sql
select (select array(select unnest (ARRAY[user_id, friend_id]) as x ORDER BY x)  as j) from relationship;
select md5(array_to_string(array_agg(id), '')) 
from (
	select * 
	from (values ('6769d922-ac68-11ea-8c70-9b8806d7aa41'), ('6769d922-ac68-11ea-8c70-9b8806d7aa41')) 
	as f(id) 
	order by f
) tmp;
```

Alternative way (simpler) that I came up with after diving deeper:

```sql
select 
	MD5(row(
		case 
			when user_id < friend_id 
			then (user_id, friend_id)
			else (friend_id, user_id)
		end
	)::text)
from (
	values 
	('7d7849d0-b94f-11ea-92be-43016fd48059', '8175c79c-b94f-11ea-92be-ab6d21fe7fb3'),
	('8175c79c-b94f-11ea-92be-ab6d21fe7fb3', '7d7849d0-b94f-11ea-92be-43016fd48059')
) as f(user_id, friend_id);
```

I added a trigger to update the column when both the user ids are inserted:
```sql
-- +migrate Up

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION hash_columns()
RETURNS TRIGGER AS $$
DECLARE
	hash text;
BEGIN
	-- NEW.hash = MD5(ROW(NEW.user_id, NEW.friend_id)::text);
	-- Take ids of both columns, sort it alphabetically, aggregate them into an
	-- array, and join them as a string before md5-ing them.
	SELECT MD5(array_to_string(array_agg(id), '')) INTO hash
	FROM (
		SELECT *
		FROM (VALUES (NEW.user_id), (NEW.friend_id))
		AS t(id)
		ORDER BY t
	) tmp;
	NEW.hash = hash;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TABLE IF NOT EXISTS relationship (
	id uuid DEFAULT uuid_generate_v1mc() PRIMARY KEY,
	user_id uuid NOT NULL REFERENCES "user"(id),
	friend_id uuid NOT NULL REFERENCES "user"(id),
	hash TEXT NOT NULL,
	status int NOT NULL DEFAULT 0,
	created_at timestamp with time zone NOT NULL DEFAULT now(),
	updated_at timestamp with time zone NOT NULL DEFAULT now(),
	deleted_at timestamp with time zone,
	CHECK(friend_id <> user_id),
	UNIQUE (user_id, friend_id)
);

CREATE INDEX idx_hash on relationship(hash);

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON relationship
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

-- A trigger to generate unique id of both user_id and friend_id for sorting purposes.
CREATE TRIGGER hash_id
BEFORE INSERT OR UPDATE ON relationship
FOR EACH ROW
EXECUTE PROCEDURE hash_columns();


-- +migrate Down
DROP trigger IF EXISTS set_timestamp
ON relationship CASCADE;

DROP TABLE IF EXISTS relationship;
```

The result is complicated, and can be simplified in postgres 12 with the always generated column:
`
```sql
create table friends (
	user_id uuid not null default uuid_generate_v1mc(),
	friend_id uuid not null default uuid_generate_v1mc(),
	hash text generated always as (MD5(
		case 
			when user_id < friend_id 
			then user_id::text || friend_id::text
			else friend_id::text || user_id::text
		end
	)) STORED
);

insert into friends (user_id, friend_id) values 
	('7d7849d0-b94f-11ea-92be-43016fd48059', '8175c79c-b94f-11ea-92be-ab6d21fe7fb3'),
	('8175c79c-b94f-11ea-92be-ab6d21fe7fb3', '7d7849d0-b94f-11ea-92be-43016fd48059');
```

Output:
```
user_id | friend_id | hash
8175c79c-b94f-11ea-92be-ab6d21fe7fb3 | 7d7849d0-b94f-11ea-92be-43016fd48059 | 94b0c513159a95bcb34bad9b72196204
7d7849d0-b94f-11ea-92be-43016fd48059 | 8175c79c-b94f-11ea-92be-ab6d21fe7fb3 | 94b0c513159a95bcb34bad9b72196204
```
