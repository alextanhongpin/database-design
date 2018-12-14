# Sample transaction with Node.js

```js
async function main() {
  const db = await mysql.createPool({
    database: config.database,
    host: config.host,
    password: config.password,
    user: config.user
  })

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
# Sample result from nodejs

```js
// This will be returned when running execute.
ResultSetHeader {
  fieldCount: 0,
  affectedRows: 1,
  insertId: 0,
  info: 'Rows matched: 1  Changed: 1  Warnings: 0',
  serverStatus: 2,
  warningStatus: 0,
  changedRows: 1 
}

// To check if the row is updated.
const isUpdated = !!result.changedRows

// To get the last id created (int, auto-incremented primary key)
const id = result.insertId
```
