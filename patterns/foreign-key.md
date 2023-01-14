# Foreign keys

Are foreign key absolutely necessary? When designing FK, they coupled tables that are notsupposed to be coupled together.

There are a few examples, such as notification table, invoice and payment tables, order tables that might require joins to multiple tables.

However, instead of defining foreign key that references the multitude of tables it could join to, it would be easier to just store the id as text, e.g. entity-unique-id

- https://ardalis.com/related-data-without-foreign-keys/
- https://www.alibabacloud.com/blog/an-in-depth-understanding-of-aggregation-in-domain-driven-design_598034
