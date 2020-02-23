## Status Patterns

Dealing with database design, we are often confronted with two kinds of design decision, design to make it work now, or design to cater for future things that might not even be required.

If things are certain, the solution would have been simple. That's why we design for the future

Design choices
1) multiple columns with Boolean flag
2) single status column (denormalized, enum or reference table)
3) bitwise column
4) history column for states the item is in (if status can only be in one direction for example)

Posts table statuses
- published
- draft
- internal
- freezed
- deleted
Why is it good to have a single status column ?
- straightforward
- new statuses can be added without adding new columns
- able to deal with changing requirements
- update happens at a single column (we don't have to toggle multiple columns when they can only be in one state, thus avoiding errors)

Why is it bad
- can only have one status at a time (note we can make it a conditional status, eg. published and freezed)
- more work to be done to filter items from dB (note we can use partial filters)
- querying two or more different statuses complicates queries (select * where status in (published, deleted, draft)
- cannot track changes in status, can only be in a single state at a time, though the state can be conditional)

Bitwise column
- can be indexed
- can hold max 32 statuses
- querying is harder on the client side, need to build the bit


## Types of Status

We should also consider the type of status a row can be in the database

- one status at a time
- multiple statuses at a time
- toggle type (0 or 1)
- forward state, can only move in one direction (checkout, purchased, paid, delivered)
- history (we need to know the previous status that the row was in for auditing, undo etc, and probably who make those changes, if there can be more than one party that is required for approval other than the owner of the entity)
- workflow kind 

## Wordpress Status


Posts in WordPress can have one of 8 statuses:

- publish: viewable by everyone
- future: scheduled to be published in a future date
- draft: incomplete post viewable by anyone with proper user role
- pending: awaiting a user with `publish_post` capability to publish
- private: viewable only to WordPress users at Administrator level
- trash: posts in trash are assigned the trash value
- auto-draft: revisions that WordPress saves automatically while you are editing
- inherit: used with a child post to determine the actual status from the parent post


https://wordpress.org/support/article/post-status/
