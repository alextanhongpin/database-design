# Typical analytics mistakes

- count is wrong after filtering
- did not filter deleted_at is null
- performing inner join instead of left join (sometimes we want the full data)
- aggregating wrongly

## Reducing complex sql in applications

- create views
- create functions (or lateral joins)
- use CTE expressions


## Useful analytics
http://www.silota.com/docs/recipes/
