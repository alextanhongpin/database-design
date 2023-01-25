# Foreign keys

Are foreign key absolutely necessary? When designing FK, they coupled tables that are notsupposed to be coupled together.

There are a few examples, such as notification table, invoice and payment tables, order tables that might require joins to multiple tables.

However, instead of defining foreign key that references the multitude of tables it could join to, it would be easier to just store the id as text, e.g. entity-unique-id


## Non-foreign keys


If foreign key is not necessary, then combine both the entity name as well as the id when referencing them.

For example, when referencing an order item with id `xyz` in an `invoice` table, then the value would be `order-item-xyz`. The reason of prefixing the id with the entity name is that different entity may have the same id, so just referencing the id `xyz` might be insufficient.

The other alternative is to store the type and id in separate columns (similar to rails polymorphism [^1]). However, when sending the id to external party (e.g. stripe), we will still need to prefix the id for uniqueness.

- https://ardalis.com/related-data-without-foreign-keys/
- https://www.alibabacloud.com/blog/an-in-depth-understanding-of-aggregation-in-domain-driven-design_598034

[^1]: https://guides.rubyonrails.org/association_basics.html
