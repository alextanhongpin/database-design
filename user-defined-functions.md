# UDF

```sql
-- +goose Up
DROP FUNCTION IF EXISTS fn_test;
-- +goose StatementBegin
-- SQL in this section is executed when the migration is applied.
CREATE FUNCTION fn_test(a int, b int) RETURNS int 
DETERMINISTIC
NO SQL
BEGIN
	DECLARE result int default 0;
	SET result = a + b;
	RETURN result;
END;
-- +goose StatementEnd
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP FUNCTION IF EXISTS fn_test;
```
