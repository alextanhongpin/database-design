## User defined ordering

User-defined ordering refers to custom ordering that is determined by user, rather than sorting alphabetically or numerically.

There are many usecases for UDF, such as
- ordering layers in Figma
- ordering todo lists
- custom ordering by variations, e.g. showing sizes in `S, M, L` instead of alphabetically `L, M, S`


## References
- https://news.ycombinator.com/item?id=16635440
- https://stackoverflow.com/questions/38923376/return-a-new-string-that-sorts-between-two-given-strings/38927158#38927158
- https://stackoverflow.com/questions/40718900/jiras-lexorank-algorithm-for-new-stories
- [How figma does it through fractional indexing](https://www.figma.com/blog/realtime-editing-of-ordered-sequences/#fractional-indexing/)
- https://steve.dignam.xyz/2020/03/31/practical-ordering/
- https://begriffs.com/posts/2018-03-20-user-defined-order.html
- [How Pinterest does it](https://medium.com/pinterest-engineering/how-we-built-rearranging-pins-b11052e95c8b)
- [Another implementation of fractional indexing](https://observablehq.com/@dgreensp/implementing-fractional-indexing)
- [More fractional indexing...](https://observablehq.com/@dgreensp/implementing-fractional-indexing)
- [User specified ordering](https://observablehq.com/@dgreensp/implementing-fractional-indexing)



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


## Sorting by variations

In e-commerce application, usually there are requirements to sort product variations by name. However, sorting alphabetically won't work for most cases, since for example sizes will be shown as `L, M, S` instead of `S, M, L`.


```sql
with variations (id) as (values
	('blue'),
	('green'),
	('red'),
	('35'),
	('34'),
	('XXS'),
	('XS'),
	('S'),
	('M'),
	('L'),
	('XL'),
	('XXL'),
	('3XL') -- Just to illustrate that the pattern can be random.
),
randomized as (
	select * from variations order by random()
)
select
	id
from randomized
order by array_position(array['XXS', 'XS', 'S', 'M', 'L', 'XL', 'XXL', '3XL'], id), id
```
