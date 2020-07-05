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

-- Giving a name to index prevent us from creating duplicate indices.

-- Translation: Subject that belongs to a group must be unique.
CREATE UNIQUE INDEX subject_name_parent_id_idx ON subject(name, parent_id) WHERE parent_id IS NOT NULL;
-- Translation: Subject must be unique.
CREATE UNIQUE INDEX subject_name_idx ON subject(name) WHERE parent_id IS NULL;

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


## Using modern postgres features to model a simple ABAC

- recursive CTE
- partial index
- variadic functions
- setof return type
- using `CREATE TYPE`

### Creating tables

```sql
-- Cleanup.
DROP TABLE action CASCADE;
DROP TABLE object CASCADE;
DROP TABLE policy CASCADE;
DROP TABLE subject CASCADE;
```

Create the `action` table:
```sql
CREATE TABLE IF NOT EXISTS action (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);
```

Create the `object` table:
```sql
CREATE TABLE IF NOT EXISTS object (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE
);
```

Create the `subject` table:
```sql
CREATE TABLE subject (
	id serial PRIMARY KEY,
	name text NOT NULL,
	parent_id INT REFERENCES subject(id),
	CHECK (id <> parent_id) -- Cannot be a parent to itself.
);

-- Giving a name to index prevent us from creating duplicate indices.
CREATE UNIQUE INDEX subject_name_parent_id_idx ON subject(name, parent_id) WHERE parent_id IS NOT NULL;
CREATE UNIQUE INDEX subject_name_idx ON subject(name) WHERE parent_id IS NULL;
```

Create the `policy` table:
```sql
CREATE TABLE IF NOT EXISTS policy (
	id serial PRIMARY KEY,
	subject_id int NOT NULL REFERENCES subject(id),
	object_id int NOT NULL REFERENCES object(id),
	action_id int NOT NULL REFERENCES action(id),
	UNIQUE (subject_id, object_id, action_id)
);
```

Query to find all groups:
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

Query to find all roles and parent groups:
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

ORDER BY subject, object, action;
```

## Functions

We want to create useful functions that we can use to model the ABAC:

- `create_object(...text)`: creates one or more resources, e.g. `book`, `car`, the domain we are modelling
- `create_subject(...text)`: creates one or more subjects. Subject can be a group, e.g. `store-owner`, `employee` or just a person `alice`, `bob` etc.
- `create_action(...text)`: creates a list of actions, e.g. `create`, `read`, `update`, `delete` to model CRUD
- `add_subjects_to_group(group_name, ...subjects)`: adds one or more subject to a group. Note that groups cannot be nested - a group cannot belong to another group. Only individual subject can belong to a group. This is to ensure we don't introduce cyclic relationship (`alice` belongs to `store-owner`, `store-owner` belongs to `alice`)
- `remove_subjects_from_group(group_name, ...subjects)`: removes one or more subjecs from a group (if it exists). 
- `create_policy(subject, object, action)`: create a policy for a given subject (or group), e.g. `create_policy('store-owner', 'book', 'create')`
- `check_policy(subject, object, action)`: checks if a policy existed for a given subject (or group), e.g. `check_policy('store-owner', 'book', 'create')` will return `true` if it exists
- `check_permissions(subject_name text, object_name text)`: checks the given subject's permission on a given object. This will be recursive - if the `subject` belongs to a `group`, it will return the roles as well

```sql
CREATE OR REPLACE FUNCTION create_object(VARIADIC names text[]) RETURNS SETOF object
AS $$
	INSERT INTO object (name)
	SELECT unnest(names::text[])
	ON CONFLICT (name) DO NOTHING
	RETURNING *;
$$ LANGUAGE SQL;


CREATE OR REPLACE FUNCTION create_subject(VARIADIC names text[]) RETURNS SETOF subject AS $$
	INSERT INTO subject (name)
	SELECT unnest(names::text[])
	ON CONFLICT (name) WHERE parent_id IS NULL DO NOTHING
	RETURNING *;
$$ LANGUAGE SQL;


CREATE OR REPLACE FUNCTION create_action(VARIADIC names text[]) RETURNS SETOF action AS $$
	INSERT INTO action (name)
	SELECT unnest(names::text[])
	ON CONFLICT (name) DO NOTHING
	RETURNING *;
$$ LANGUAGE SQL;


CREATE OR REPLACE FUNCTION add_subjects_to_group(group_name text, VARIADIC subject_names text[]) RETURNS SETOF subject AS $$
	WITH groups AS (
		SELECT id 
		FROM subject 
		WHERE name = group_name
		AND parent_id IS NULL -- Avoid having nested groups.
	)
	INSERT INTO subject (name, parent_id)
	SELECT UNNEST(subject_names), (SELECT id FROM groups) AS parent_id
	ON CONFLICT (name, parent_id) WHERE parent_id IS NOT NULL DO NOTHING
	RETURNING *;
$$ LANGUAGE sql;


CREATE OR REPLACE FUNCTION remove_subjects_from_group(group_name text, VARIADIC subject_names text[]) RETURNS SETOF subject AS $$
	WITH current_group AS (
		SELECT id
		FROM subject
		WHERE name = group_name
	)
	DELETE FROM subject
	WHERE name = ANY(subject_names::text[])
	AND parent_id = (SELECT id FROM current_group)
	RETURNING *
$$ LANGUAGE sql;

CREATE OR REPLACE FUNCTION create_policy(subject_name text, object_name text, action_name text) RETURNS SETOF policy AS $$
	DECLARE 
		sid int;
		oid int;
		aid int;
	BEGIN
		SELECT id INTO sid FROM subject WHERE name = subject_name;
		SELECT id INTO oid FROM object WHERE name = object_name;
		SELECT id INTO aid FROM action WHERE name = action_name;
		RETURN query
		WITH p AS (
			INSERT INTO policy (subject_id, object_id, action_id)
			VALUES (sid, oid, aid)
			ON CONFLICT (subject_id, object_id, action_id) DO NOTHING
			RETURNING *
		) 
		TABLE p;
	END;
$$ LANGUAGE plpgsql; 


CREATE OR REPLACE FUNCTION check_policy(subject_name text, object_name text, action_name text) RETURNS boolean AS $$
	WITH RECURSIVE subjects(id, name, parent_id) AS (
		SELECT id, name, parent_id
		FROM subject
		WHERE name = subject_name
		
		UNION
		
		SELECT s.id, s.name, s.parent_id 
		FROM subjects, subject s
		WHERE s.id = subjects.parent_id
	)
	SELECT EXISTS (
		SELECT 1 
		FROM policy
		WHERE subject_id IN (SELECT id FROM subjects)
			AND action_id IN (SELECT id FROM action WHERE name = action_name)
			AND object_id IN (SELECT id FROM object WHERE name = object_name)
	)
$$ LANGUAGE sql; 


DROP TYPE permission CASCADE;
CREATE TYPE permission AS (subject text, object text, action text, group_name text);

CREATE OR REPLACE FUNCTION check_permissions(subject_name text, object_name text) RETURNS SETOF permission AS $$
	WITH RECURSIVE SUBJECTS(id, name, parent_id) AS (
		SELECT id, name, parent_id
		FROM subject
		WHERE name = subject_name
		
		UNION
		
		SELECT s.id, s.name, s.parent_id 
		FROM subjects, subject s
		WHERE s.id = subjects.parent_id
	)
	SELECT subject_name, object.name, action.name, NULLIF(subject.name, subject_name)
	FROM policy
	JOIN action ON (action.id = policy.action_id)
	JOIN object ON (object.id = policy.object_id)
	JOIN subject ON (subject.id = policy.subject_id)
	WHERE subject_id IN (SELECT id FROM subjects)
		AND object_id IN (SELECT id FROM object WHERE name = object_name)
$$ LANGUAGE sql; 

-- Show all created functions.
SELECT
	routine_name,
    routine_definition
FROM
    information_schema.routines 
WHERE
    specific_schema LIKE 'public';
    

select * from create_subject('alice', 'bob', 'john', 'store-owner', 'employee');
select * from create_action('create', 'read', 'update', 'delete');
select * from create_object('book');

select * from add_subjects_to_group('store-owner', 'alice', 'bob');
select * from add_subjects_to_group('employee', 'bob', 'john');
select * from remove_subjects_from_group('store-owner', 'bob');

select * from create_policy('store-owner', 'book', 'create');
select * from create_policy('store-owner', 'book', 'read');
select * from create_policy('store-owner', 'book', 'update');
select * from create_policy('store-owner', 'book', 'delete');

select * from create_policy('employee', 'book', 'update');
select * from create_policy('employee', 'book', 'read');

-- Let's check the employee's permission.
select * from check_policy('employee', 'book', 'create');
select * from check_policy('employee', 'book', 'read');
select * from check_policy('employee', 'book', 'update');
select * from check_policy('employee', 'book', 'delete');

-- Let's check John's permission, who is an employee.
select * from check_policy('john', 'book', 'read');

select * from check_permissions('store-owner', 'book');
select exists (select 1 from check_permissions('store-owner', 'book'));
select * from check_permissions('alice', 'book');
select * from check_permissions('john', 'book');
select * from check_permissions('unknown', 'book');
select exists (select 1 from check_permissions('unknown', 'book'));
```


Output for `select * from check_permissions('store-owner', 'book')`:
```
subject.        object. action.  group_name.
store-owner	book	delete	 NULL
store-owner	book	update	 NULL
store-owner	book	read	 NULL
store-owner	book	create	 NULL
```


Output for `select * from check_permissions('john', 'book')`:
```
subject. object. action.  group_name.
john	 book	 update	  employee
john	 book	 read	  employee
```

### Best practices

We need to deal with the following constraints
- most of the logic involves creating unique entities
- the operation should be idempotent - running the same create will not create duplicate records, it will just not throw errors too
- we can only create, not update. This makes it easier to prevent changing the user's role unknowingly. 
- a user can belong to several groups, not just one
- any entity that has nested entity cannot be deleted, unless those entities are deleted first (no cascade delete)

To avoid creating too many policies, we can come up with a simple rule:
- create only policy for owners (users with roles). e.g. bookstore owner
- assume guest mode (readonly) for all users, so we don’t need to create those rule. Create only mutation roles (update, delete, create) for specific users. If the user does not have create access (like most web applications) we can skip it too. 

