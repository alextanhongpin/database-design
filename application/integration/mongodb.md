# MongoDB

- How to ensure referential integrity is respected in NoSQL?
  - MongoDB now supports [transaction](https://docs.mongodb.com/manual/core/transactions/)
- you can use functions in mongodb
- when dealing with associations, how to deal with deletion of parent? Say we have users, and users have many posts. Each post will have a `user_id` embedded in it, but if the user is deleted, the post remains.
  - We need to delete all associations that are associated with the user before deleting the user. Else, we have to come up with a complex script to figure which documents are left hanging without users.
  - Alternatively, we can just soft delete the user

## Patterns
- https://www.mongodb.com/blog/post/building-with-patterns-a-summary
- https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-1
