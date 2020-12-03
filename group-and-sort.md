## Grouping by fields, and taking the first row ordered by a column.
```sql
-- Get the most recent fx with CTE.
WITH t AS (
	SELECT
		source_currency_id, 
		target_currency_id, 
		max(effective_date) AS effective_date
	FROM fx_rate
	GROUP BY (source_currency_id, target_currency_id)
)
SELECT fx_rate.*
FROM fx_rate 
JOIN t ON (fx_rate.source_currency_id = t.source_currency_id AND fx_rate.target_currency_id = t.target_currency_id AND fx_rate.effective_date = t.effective_date);



-- Get the most recent fx with row_number.
WITH summary AS (
	SELECT
		*,
		ROW_NUMBER() OVER (PARTITION BY source_currency_id, target_currency_id ORDER BY effective_date DESC) as rank
	FROM fx_rate
)
SELECT *
FROM summary WHERE rank = 1; 

SELECT * 
FROM fx_rate 
WHERE source_currency_id = currency_code_to_id('SGD')
AND target_currency_id = currency_code_to_id('MYR')
ORDER BY effective_date DESC 
LIMIT 1;
```
