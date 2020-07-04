## Attribute-Based Access Control

Implementing basic attribute-based Access Control (based on go's Caspbin, modelling `object`, `subject`, `action`).

**Action** is what a `subject` performs on an `object`. For example, if we want to setup basic permissions for a CRUD api to restrict access to the resource `books`, then we will have the following:

- `books:create`
- `books:update`
- `books:read`
- `books:delete`

We will setup the `action` table as shown below:
```sql
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
```

**Object**, or **resource** is the entity in which we can apply actions on. To keep things simple, we will store only singular naming of the entity in the database. You can choose to use singular naming, but the point is to be consistent!
```sql
CREATE TABLE IF NOT EXISTS object (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);

DROP TABLE object;

INSERT INTO object (name)
VALUES ('book');
```

**Subject** is the user that we identify that can perform (or not) action/actions on object/objects. Note that the `subject` table is not necessarily the `user` table. In a more complex case where we want to create `groups` of permissions, subject can be referring to a particular group. But for simplicity, we just assume here that subject refers to a user. We can use `names`, `email` or any other unique identifier to identify them:
```sql
CREATE TABLE subject (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);
DROP TABLE subject;

INSERT INTO subject (name)
VALUES ('john');
```

**Policy** is the access control layer that defines who (`subject`) can do what (`action`) on a particular resource (`object`). Note again that this is just a naive implementation, without groups etc.
```
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
