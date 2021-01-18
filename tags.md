## Modelling Tags with Postgres Array

The performance is faster thanusing join tables, http://www.databasesoup.com/2015/01/tag-all-things.html.
```sql
CREATE TABLE IF NOT EXISTS tags (
	tags text[] not null default '{}'
);

-- Index for performance.
CREATE INDEX ON tags USING gin(tags);

INSERT INTO tags (tags) VALUES (ARRAY['hello', 'world']);

SELECT * FROM tags WHERE tags @> ARRAY['hello'];
SELECT * FROM tags WHERE tags @> ARRAY['haha'];
```

Pros:
- save storage, no additional table required
- no table joins, fast query
- fast lookup with index (some benchmark shows that the performance is faster than using table joins)
- fast save

Cons:
- same value can appear in the array
- values may be duplicated (Job, Jobs, job)
- while it is fast to search tags for a document, searching for most popular tag can be hard (need to aggregate all tags values first, then count corresponding documents with tags)
- update/delete is not reflected in the array
  - we can store the ids of the tags instead of string, but this requires another lookup)
  - if tags are arbitrary readonly data, then we don't need to enforce such constraints)
- saving an empty array can clear all tags

Using table joins has its pros and cons too

Pros:
- easier for analytics (finding most popular tags, and getting tags for a particular document is easier)
- documents count can be stored alongside the tags table, to reduce query
- updates/deletes in the tags will be reflected (though by right they should be not deleted)

Cons:
- joins can be slow
- saving tags can be painful, mostly requires deleting all old tags, and recreating the associations, unless we do upsert only. If we want to save tags that consist of some added and deleted tags, this may be complicated to perform)


## Tagging v2

A solution that stores the array of tags (denormalization) as well as creating a junction table with triggers. FIX, only delete if counter reaches zero.

```sql
create table if not exists pg_temp.posts (
	id int generated always as identity,
	body text not null,
	tags text[] not null default '{}'::text[],
	primary key (id)
);
create table if not exists pg_temp.tags (
	id int generated always as identity,
	name citext not null unique,
	created_at timestamptz not null default current_timestamp,
	counter int not null default 1,
	primary key (id)
);
create table if not exists pg_temp.post_tag (
	id int generated always as identity,
	post_id int not null,
	tag_id int not null,
	primary key (id),
	foreign key(post_id) references pg_temp.posts,
	foreign key(tag_id) references pg_temp.tags
);

create or replace function pg_temp.trigger_tag() returns trigger as $$
	declare
		removing_ids int[];
		adding_ids int[];
		junction_table text = TG_ARGV[0];
		entity_column text = TG_ARGV[1];
	begin
		with combined as (
			select distinct unnest(
				COALESCE(OLD.tags, '{}'::text[]) || 
				COALESCE(NEW.tags, '{}'::text[])
			) as name
		),
		removed as (
			select name 
			from combined
			where not ARRAY[name] <@ COALESCE(NEW.tags, '{}'::text[])
		),
		added as (
			select name 
			from combined 
			where not array[name] <@ COALESCE(OLD.tags, '{}'::text[])
		),
		ids_to_remove as (
			update pg_temp.tags 
			set counter = counter - 1
			where name in (select name from removed)
			returning id
		),
		ids_to_add  as(
			insert into pg_temp.tags(name) 
				select name 
				from added 
			on conflict(name) 
			do update set counter = pg_temp.tags.counter + 1 
			returning *
		)
		select 
			array(select id from ids_to_add), 
			array(select id from ids_to_remove where counter = 0) 
		into adding_ids, removing_ids;
		
		RAISE NOTICE 'got adding % and removing %', adding_ids, removing_ids;

		-- Clear the junction table.
		execute format('
			delete from %I 
			where %I = $1 and tag_id = ANY($2::int[])', junction_table, entity_column) 
			using NEW.id, removing_ids;
			
		execute format('
			insert into %I (%I, tag_id) 
				select $1, tmp.id 
				from (select unnest($2::int[]) as id) tmp 
			returning id', junction_table, entity_column) using NEW.id, adding_ids;
		RETURN NEW;
	end;
$$ language plpgsql;

-- TODO: Handle delete
drop trigger update_tags on pg_temp.posts;
create trigger update_tags 
after insert or update 
on pg_temp.posts 
for each row
execute procedure pg_temp.trigger_tag('post_tag', 'post_id');

insert into pg_temp.posts (body, tags) values ('hello', '{hello, world, this}'::text[]);
update pg_temp.posts set tags = '{hello, alice}'::text[];
update pg_temp.posts set tags = '{john, doe}'::text[];
```

