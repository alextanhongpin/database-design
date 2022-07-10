# UUID Cursor pagination

Regardless of whether id exists or not, this will throw an error `invalid input syntax for type uuid: ""`
```sql
SELECT *
FROM your_table
  WHERE CASE WHEN created_at IS NOT NULL AND NULLIF(id, '') IS NOT NULL
      THEN (created_at, id) < (now(), '')
      ELSE true
  END
  ORDER BY created_at DESC, id DESC
LIMIT 10
```

Also the `WHERE` conditional makes the sql more complex to debug.
It is easier to set the current timestamp to now and id to max uuid `ffffffff-ffff-ffff-ffff-ffffffffffff`.
