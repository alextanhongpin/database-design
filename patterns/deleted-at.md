# Deleted at

- aka soft delete, compared to actual deletion of the resource(hard-delete)
- its not the same as active flag. In fact, you can have both. Those woth deleted at will no longer be returned from the API, however, non active ones may still be returned in Admin API, but not shown to the public
- why keep the deleted record? As a tombstone, so that the id or entity is no longer reused. E.g. a deleted email address should not be reused, in case ownership changes. An old user anandon his email address, but a new user takes ownership of that. Ro avoid new user from accessing the old records, we can mark the email as deleted. 


## Avoid reusing values

Sometimes, we want to avoid the reusal of certain unique name like user slug. Lets say you have a unique product skug name that you share the link to your buyer. However, you later rename the product name to another name. What happens to the old slug?

- Ideally you want to avoid broken links, so the old slug should point back to the previous slug before rename. This requires you to keep the history of the friendly id url and somehow do a lookup to see if it is valid. Ideally when you have such history table, all lookups will then be based on the history table instaed of the table where the current slug exists.
- Can someone reuse the old slug? preferably not. Only the same user should be able to reuse the same old slug, otherwise user can impersonate another user or their products which can be misleading. Imaginine clicking a link to your last purchased product to find out its a different product than you have purchased before
- special handling on the client side is required to redirect user or replace the url slug to the correct one. For example, the orginal slug is foo. The new slug is bar. When i visit foo, I can still load foo product, but the url will change to the new one which is bar.
- Do not share the param with another identifier. When fetching by slug, dont allow the same param to be used to fetch by id. So GET /users/:slugOrId is bad. Imagine you habe a user with id 1, and the same user has name 34. I stead of fetching the user by id 1, you might accidentally fetch user by id 34. This is an exagerration, because you should not use numeric id nor numeric name, but it serves to demonstrate the scenario.
