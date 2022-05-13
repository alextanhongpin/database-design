Postgres introduce generated columns.

There are many usecases of generated column. 

e.g. we want to create a unique index based on several columns. We can instead md5 all those columns and add a unique index to one single column to save storage. Of course it does not help with indexing soeed. E.g. is address table.
