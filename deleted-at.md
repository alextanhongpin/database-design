# Deleted at

- aka soft delete, compared to actual deletion of the resource(hard-delete)
- its not the same as active flag. In fact, you can have both. Those woth deleted at will no longer be returned from the API, however, non active ones may still be returned in Admin API, but not shown to the public
- why keep the deleted record? As a tombstone, so that the id or entity is no longer reused. E.g. a deleted email address should not be reused, in case ownership changes. An old user anandon his email address, but a new user takes ownership of that. Ro avoid new user from accessing the old records, we can mark the email as deleted. 
