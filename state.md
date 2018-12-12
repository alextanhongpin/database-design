## For nodejs mysql library
```
// use query vs execute
When update, result.changedRows
When delete, result.affectedRows
When insert, result.insertId

When duplicate field:
expect(error.code).toBe('ER_DUP_ENTRY')
```
