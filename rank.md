```sql
SELECT
  ROW_NUMBER() OVER (ORDER BY total_review_count_rank + recent_review_count_rank + score, total_review_count_rank, score, recent_review_count_rank) AS rank, 
  ({{endDate}}::date - interval '30 days')::date AS inclusive_from_date,
  {{endDate}}::date AS exclusive_to_date,
  id,
  name, 
  total_review_count_rank, 
  recent_review_count_rank, 
  score as rating_rank,
  total_review_count_rank + recent_review_count_rank + score AS total_rank,
  COALESCE(total_review_count, 0) AS current_total_reviews,
  COALESCE(correct_recent_review_count, 0) AS current_recent_reviews,
  COALESCE(average_rating, 0) AS current_average_rating
FROM (
  SELECT 
    *,
    coalesce(recent_review_count, 0) as correct_recent_review_count,
    DENSE_RANK() OVER (order by coalesce(total_review_count, 0) desc) as total_review_count_rank,
    DENSE_RANK() OVER (order by coalesce(recent_review_count, 0) desc) as recent_review_count_rank,
    DENSE_RANK() OVER (order by coalesce(
                       ((positive + 1.9208) / (positive + negative) - 1.96 * SQRT((positive * negative) / (positive + negative) + 0.9604) /
                      (positive + negative)) / (1 + 3.8416 / (positive + negative)), 0) desc) AS score
  FROM (
    SELECT *
    FROM product_items
    WHERE deleted_at IS null
	) pi LEFT JOIN LATERAL (
  SELECT
    COALESCE(count(*), 0) AS total_review_count,
    COALESCE(count(*) FILTER (WHERE pir.created_at >= {{endDate}}::date - interval '30 days'), 0) AS recent_review_count,
    COALESCE(avg(rating), 0) AS average_rating,
    sum(
      case
        when rating <= 1 then 0
        when rating <= 2 then 0.25
        when rating <= 3 then 0.5
        when rating <= 4 then 0.75
        when rating <= 5 then 1
        else 1
      end
    ) AS positive,
    sum(
      case
        when rating <= 1 then 1
        when rating <= 2 then 0.75
        when rating <= 3 then 0.5
        when rating <= 4 then 0.25
        when rating <= 5 then 0
        else 0
      end
    ) AS negative
  FROM
    product_item_reviews pir
  WHERE pir.created_at < {{endDate}}::date AND
  pi.id = pir.item_id AND
  pir.deleted_at IS NULL
GROUP BY
  pir.item_id
) tmp ON true
  WHERE id IN (
    SELECT product_items.id FROM product_items 
    JOIN product_item_categories ON (product_items.category_id = product_item_categories.id)
    WHERE {{category}}
  )
) tmp
WHERE deleted_at IS NULL
ORDER BY total_rank, total_review_count_rank, score, recent_review_count_rank
```
