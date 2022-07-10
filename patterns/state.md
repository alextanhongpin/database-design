## For nodejs mysql library
```
// use query vs execute
When update, result.changedRows
When delete, result.affectedRows
When insert, result.insertId

When duplicate field:
expect(error.code).toBe('ER_DUP_ENTRY')
```


## Single source of truth

Should the database hold only the final state of the data? or the sequence of steps to produce the final state?

Most of the time, keeping the final state of the date is all we need, but often, we want to keep track of the sequence of the states. Take for example a claim approval database schema. It becomes cumbersome to know what happens in between (like who approved the claim, because multiple people can approve them) or why is it rejected (need to add a reject reason). And what about repeating processes? Like it has been submitted, but then rejected a few times, but the database only stores the final state of the data. This is not particularly useful. If we want to keep the history logs, we can do a hybrid datastore. We keep the immutable state in the database, but keep the events/interactions in another table. This way, we can always find out how the data is being constructed. The main table storing the final state should only keep the state of the current event (submitted/pending) etc, and it can be inferred from the activity/event/feed table.
