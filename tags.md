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

Todo: we probably don't need the junction table. Just the tags table, and the entity with the tags column will do.
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



## Using tsvector to store tags



```sql
DROP TABLE IF EXISTS photos;
CREATE TABLE IF NOT EXISTS photos (
	id int GENERATED ALWAYS AS IDENTITY,

	tags tsvector,

	PRIMARY KEY (id)
);

INSERT INTO photos (tags) VALUES
('#nofilter #amazing #cool'::tsvector),
('#nofilter #notlikethis'::tsvector),
('#swimming #diving'::tsvector),
('#swim #dive'::tsvector),
(array_to_tsvector('{#swim, #swimming, #living}'::text[])),
(NULL);

-- Add index to improve performance.
-- There's GIN and GIST index
-- https://www.compose.com/articles/indexing-for-full-text-search-in-postgresql/#:~:text=PostgreSQL%20provides%20two%20index%20types,document%20collection%20will%20be%20situational.
-- https://stackoverflow.com/questions/28975517/difference-between-gist-and-gin-index
-- TL;DR: Use gist for faster update and smaller size.
-- Also see the usage of generated columns here.
-- https://www.postgresql.org/docs/current/textsearch-tables.html

-- CREATE INDEX weighted_tsv_idx ON photos USING GIST (tags);
CREATE INDEX photos_tags_idx ON photos USING GIN (tags);


VACUUM ANALYZE photos;
REINDEX TABLE photos;



-- This does not do any preprocesing and assumes the vectors are normalized.
SELECT 'I like swimming'::tsvector;
SELECT ' I  like  swimming'::tsvector;
SELECT '#nofilter #instaworthy'::tsvector;

-- If whitespace between words needs to be preserved, wrap them in double single quotes.
SELECT 'I like ''to swim'''::tsvector;

-- This handles normalization, probably not what we want to use for tags.
SELECT to_tsvector('english', 'I like swimming');





-- View all.
SELECT *
FROM photos;


-- To query, we use tsquery.
-- This does not normalize the vector.
SELECT 'swimming:*'::tsquery; 				-- swimming:*

-- This normalizes the vector.
SELECT to_tsquery('english', 'swimming:*'); -- swim:*


-- Filter with specific tags.
SELECT *
FROM photos
WHERE tags @@ '#nofilter'::tsquery;

-- This is valid too. The text is automatically cast to tsquery, not to_tsquery.
SELECT *
FROM photos
WHERE tags @@ '#nofilter';

-- Find by prefix.
SELECT *
FROM photos
WHERE tags @@ '#:*';


-- Disable seqscan because there's not much rows.
SET enable_seqscan = OFF;
EXPLAIN ANALYZE
SELECT *
FROM photos
WHERE tags @@ '#swim:*';


-- This does not work, they are array of array of tags.
SELECT array_agg(DISTINCT tags)
FROM photos;


SELECT to_tsvector('english', string_agg(tags::text, ' '))
FROM photos;

EXPLAIN ANALYZE
SELECT string_agg(tags::text, ' ')::tsvector
FROM photos;

-- Using custom aggregate.
CREATE AGGREGATE tsvector_agg(tsvector) (
   STYPE = pg_catalog.tsvector,
   SFUNC = pg_catalog.tsvector_concat,
   INITCOND = ''
);

SELECT tsvector_agg(tags)
FROM photos;

SELECT string_agg(tags::text, ' ')::tsvector
FROM photos;

-- Get all unique occurances.
SELECT distinct unnest(tsvector_to_array(tags)) tag
FROM photos order by tag;

SELECT to_tsvector(string_agg(tags::text, ' '))
FROM photos;

-- https://www.postgresql.org/docs/current/functions-textsearch.html

-- Find count of all tags.
SELECT
	unnest(tsvector_to_array(tags)) tag,
	count(*)
FROM photos
group by tag;


-- Insert 1,000,000 data. Will take ~2 minutes.
insert into photos (tags)
select
    ('#' || left(md5(i::text), 4) ||
    ' #' || left(md5(random()::text), 4)||
    ' #' || left(md5(random()::text), 4)
    )::tsvector
from generate_series(1, 1000000) s(i);
```
