## Cursor pagination

- the cursor field should be unique and sortable. we need to sort the items by the cursor first, before selecting them, hence it might not work with sorting.
- a non-unique field will screw the pagination (considered a column name, and there are multiple names, we cannot use the operator `>= or <=`, but `> or <` instead, and we need another unique field (sortable one) to distinguish them both.
- the column selected as a cursor must not be null
- sorting seems to be a little difficult (and not necessarily performance) when not using the unique cursor key

References:
- https://jsonapi.org/profiles/ethanresnick/cursor-pagination/
- https://hackernoon.com/guys-were-doing-pagination-wrong-f6c18a91b232



## Cursor or keyset pagination 

- preferred over offset limit
- works under certain condition (best with a single column that is unique, unchanged)
- how to deal with multiple columns? create a new column that is unique and computed
- might not work well if you need to sort by values that changes dynamically, e.g. product stocks count, product rating, because when the value changes, the pagination may break (duplicate items etc)
- might not work if you need to jump between pages
