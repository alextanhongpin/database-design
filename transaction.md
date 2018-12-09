# Sample transaction with Node.js

```js
async function main() {
  const conn = await db.getConnection()
  try {
    const stmt = `
      INSERT INTO ()...
    `
    await conn.query('START TRANSACTION')
    await conn.execute(stmt, [])
    // If all is successful until this point, commit the 
    // transaction.
    await conn.query('COMMIT')
  } catch (error) {
    // Perform rollback when an error occurred.
    await conn.query('ROLLBACK')
  } finally {
    // Release the connection at the end to save resources.
    await conn.release()
  }
} 
```
