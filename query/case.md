## Using case to handle range

```postgres
postgres db
with users_with_level as (
	select *, 
		case 
			when points < 2 then 'Freshie'
			when points < 10 then 'Rookie'
			when points < 40 then 'Wonderkid'
			when points < 125 then 'Prodigy'
			when points < 250 then 'Genius'
			when points < 750 then 'Master'
			when points < 1500 then 'Grand Master'
			when points < 3000 then 'Wizard'
			when points < 6000 then 'God of Wisdom'
			when points >= 6000 then 'Unicorn'
		end as level_name
	from user_profiles
),
levels as (
	select * 
	from (values 
	('Freshie', 1, 0), 
	('Rookie', 2, 2),
	('Wonderkid', 3, 10),
	('Prodigy', 4, 40),
	('Genius', 5, 125),
	('Master', 6, 250),
	('Grand Master', 7, 750),
	('Wizard', 8, 1500),
	('God of Wisdom', 9, 3000),
	('Unicorn', 10, 6000))
	levels (name, level, min_points)
)
select 
	count(*) as total_users,
	levels.name as level_name, 
	(array_agg(levels.level))[1] as level_rank,
	(array_agg(levels.min_points))[1] as level_min_point,
	avg(points) as level_avg_point,
	max(points) as level_max_point,
	min(points) as level_min_point
from users_with_level
join levels on (levels.name = users_with_level.level_name)
group by levels.name
order by level_rank desc;
```
