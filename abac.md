## Attribute-Based Access Control

Implementing basic attribute-based Access Control (based on go's Caspbin, modelling `object`, `subject`, `action`):

```sql
-- object, subject, action
CREATE TABLE IF NOT EXISTS action (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);
DROP TABLE action;


INSERT INTO action (name) VALUES
('read'), 
('write'),
('update'),
('delete');


CREATE TABLE IF NOT EXISTS object (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);

DROP TABLE object;

INSERT INTO object (name)
VALUES ('book');


CREATE TABLE subject (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);
DROP TABLE subject;

INSERT INTO subject (name)
VALUES ('john');

CREATE TABLE IF NOT EXISTS policy (
	id serial PRIMARY KEY,
	subject_id int NOT NULL REFERENCES subject(id),
	object_id int NOT NULL REFERENCES object(id),
	action_id int NOT NULL REFERENCES action(id),
	UNIQUE (subject_id, object_id, action_id)
);
DROP TABLE policy;

INSERT INTO policy (subject_id, object_id, action_id)
VALUES 
(1, 1, 1),
(1, 1, 2),
(1, 1, 3),
(1, 1, 4);
```

To query the result:
```sql
SELECT 
subject.name AS subject,
object.name AS object,
action.name AS action
FROM policy
JOIN subject ON (policy.subject_id = subject.id)
JOIN object ON (policy.object_id = object.id)
JOIN action ON (policy.action_id = action.id);
```

Output:
```
subject object  action
john	book	read
john	book	write
john	book	update
john	book	delete
```
