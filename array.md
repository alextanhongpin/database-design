# Postgres Array

```sql
CREATE TABLE posts (
	title text NOT NULL PRIMARY KEY,
	tags text[]
);

INSERT INTO posts (title, tags) 
VALUES ('Hello world', '{postgres, triggers, go}');

SELECT * 
FROM posts 
WHERE tags @> '{"go"}';
```
