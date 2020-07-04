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


## Groups
We can extend the example above with the concept of groups. Instead of creating individual policy for each user, we can just create a group, and let the user (`subject`) *inherit* the group (with the policy that is attached to it).

In the book store example, we can for example have the following roles:

- `store-owner`: have full access
- `employee`: can read and update
- `guest`: read-only access

Instead of creating another table to represent the `group`, we will just reuse the `subject` table, but introduce a new column called `parent_id` that will refer to itself:

```sql
CREATE TABLE subject (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE,
	parent_id INT REFERENCES subject(id),
	CHECK (id <> parent_id) -- Cannot be a parent to itself.
);
DROP TABLE subject;
```

We will now insert the following `group` into the table.
```sql
INSERT INTO subject (name) VALUES 
('store-owner'), 
('employee'), 
('guest');
```

Then, insert the users (`subject`) that are associated to the `group`:

```sql
INSERT INTO subject (name, parent_id)
VALUES 
('alice', 1),  
('bob', 2),
('john', 3);
```

Let's perform a query on the `subject` table:
```sql
SELECT *
FROM subject;
```

Output:
```
id      name            parent_id
1	store-owner	NULL
2	employee	NULL
3	guest		NULL
4	alice	1
5	bob	2
6	john	3
```

We can safely assume that if the `parent_id` is not null, then the `subject` must belong to a `group`. To see which user (`subject`) belongs to which group, we can use this query:
```sql
WITH groups AS (
	SELECT
		child.id AS id,
		child.name AS subject,
		parent.name AS group
	FROM subject parent
	JOIN subject child ON (child.parent_id = parent.id)
)
SELECT * 
FROM groups;
```

Output:
```sql
id.     subject. group. 
4	alice	 store-owner
5	bob	 employee
6	john	guest
```

Let's insert the policy now, which will associate the `subject` (user or group) with the `action` and `resource` they can act upon:
```sql
INSERT INTO policy (subject_id, object_id, action_id)
VALUES 
(4, 1, 1), -- Owner (alice) can read.
(4, 1, 2), -- Owner (alice) can write.
(4, 1, 3), -- Owner (alice) can update.
(4, 1, 4), -- Owner (alice) can delete.
(5, 1, 1), -- Employee (bob) can read.
(5, 1, 3), -- Employee (bob) can update.
(6, 1, 1); -- Guest (john) can read.
```

Querying the policy:

```sql
SELECT 
	subject.name AS subject,
	object.name AS object,
	action.name AS action
FROM policy
JOIN subject ON (policy.subject_id = subject.parent_id)
JOIN object ON (policy.object_id = object.id)
JOIN action ON (policy.action_id = action.id)

UNION

SELECT 
	subject.name AS subject,
	object.name AS object,
	action.name AS action
FROM policy
JOIN subject ON (policy.subject_id = subject.id)
JOIN object ON (policy.object_id = object.id)
JOIN action ON (policy.action_id = action.id)

order by subject, object, action;
```

Output:

```sql
subject object action
alice	book	delete
alice	book	read
alice	book	update
alice	book	write
bob	book	read
bob	book	update
john	book	read
```
