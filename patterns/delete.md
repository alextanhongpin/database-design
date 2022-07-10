# How should delete stament looks like im application?


1. with conditional, e.g when it matches the where. might accidentally delete multiple row
2. by id primary key. so application need to first find that entity before issuing delete. ensure deletion of one row
3. set deleted at as soft delete
4. use trigger instead or rule to override and set deleted at when deleteis issued. do t need to worry about cascade deletion. but need to be careful if the child can still be queried elsewhere
5. when deleted, insert as lon in another table
