## Find all items that overlaps with a point with radius of 5km
```
-- Required by earthdistance.
CREATE EXTENSION cube;
CREATE EXTENSION earthdistance;

SELECT ll_to_earth(51.5032,-0.1349);
SELECT point(51.5032,-0.1349);
-- 5,000 distance in metres.
SELECT earth_box(ll_to_earth(51.5032,-0.1349), 5000)
@> ll_to_earth(50.0, -0.13);
```
