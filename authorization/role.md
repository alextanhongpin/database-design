## Roles and Scopes

Users can have different scopes for accessing a resource. They are named `action:resource`. The `resource` naming should be plural. Some examples may be:

- create:books
- update:books
- delete:books
- read:books


But these are typical CRUD queries. If there are specific ones, we can also create scopes for them:

- search:books
- publish:books 

Note that `publish:book`, while it is mostly just updating the status of the resource, it is marked as a different scope, because it is a different usecase.

References:
- https://github.com/alextanhongpin/evolutionary-architecture/blob/master/role_and_scopes.md

## Limiting scopes

For most scenarios, a user can only modify the resources created by them (unless they are admins). Reading a resource depends on the user access rights. Some resources can be read by all users, while others are only readable by owner/ops etc. 

If the resource is public, note that the access rights does not apply anymore. Let's say we have a public profile page for User A. By default, User A blocks User B, so User B can't see the User A's profile. But if this is made public, then anyone can just access the public page. One way is to limit the resources shown in public. And once the user is authorized, show them more details.


## Public vs Private

Public apis should return as little information as possible. Distinguish the public and private resources by limiting what to be returned.

Note that for `golang`, we can shadow the structs to ensure less information is returned. This would however, require you to create duplicate redundant structs.

## Context

For accessing the user's own data, use the suffix `me` (just to standardize). The `ctx` may contain the additional user's scope and/or role that is in the authorization token, to distinguish it from the arguments for the query.

```
Repository
- read(query)
- readMe(ctx, query), whereby ctx = {id: "1", scope: "", isAuthorized: true, role: ""}
```
The sql query might look as follow:

```sql
-- Public
SELECT name FROM user

-- Private
SELECT name, email FROM user WHERE id = ?
```

## Nested permissions

There are some scenarios where the access rights can be nested. Take for example an `book`, `organization`, `role` and `user` resource. The user is logged in and wants to perform modification on some resources that belongs to the organization, but only possible if the user has the `admin` role. This makes the query slightly more complicated, since we need to query the user, check if the user belongs to the organization, check if the book belongs to the organization (you cannot modify another book from another organization), and check if user has the correct role.


```sql
SELECT title, author, id
FROM book b
INNER JOIN organization o ON (b.orgId = o.id)
INNER JOIN user u ON (b.owner = u.id)
WHERE b.id = ?
```

Note we are using `INNER JOIN` instead of `LEFT JOIN`, so it should not return a data at all if it doesn't exist. 

Some thoughts here:
- how would the query looks like for NOSQL?
- can we cache the access? (what if it has been modified/role removed?)


## Row Level Security (RLS)

There are no RLS available for MySQL. To design an ACL for database, we can consider using some principles from the unix level permissions.
