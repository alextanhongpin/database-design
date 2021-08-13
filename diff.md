Diffing changes

Is it possible to design the entity where only the diff changes are applied?

While it's possible, it's not worth the effort. In order to get a diff, you first need to fetch the original data source if it exists.

Therefore it would have been simpler if you fetch, make the changes and replace the row straight, aka put operation.
