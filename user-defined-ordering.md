User defined ordering

https://stackoverflow.com/questions/38923376/return-a-new-string-that-sorts-between-two-given-strings/38927158#38927158

https://stackoverflow.com/questions/40718900/jiras-lexorank-algorithm-for-new-stories

How figma does it

https://www.figma.com/blog/realtime-editing-of-ordered-sequences/#fractional-indexing/

https://steve.dignam.xyz/2020/03/31/practical-ordering/

https://begriffs.com/posts/2018-03-20-user-defined-order.html

How Pinterest does it

https://medium.com/pinterest-engineering/how-we-built-rearranging-pins-b11052e95c8b


## Sorting with predefined list
```sql
with t(id, name) as (values
	(1, 'hello'), 
	(2, 'world')
)
select * 
from t
order by array_positions('{2,1,3}'::int[], id);

select array_positions('{1,2,3}'::int[], 1); -- {1}
```

With ordinality
```sql
with t(id, name) as (values
	(1, 'hello'), 
	(2, 'world')
)
select * 
from t 
join (select * from unnest('{2,1}'::int[]) with ordinality t(id, idx)) o on (o.id = t.id) 
order by idx;
```
