# Designing Category-Subcategory relationship

- one category can have multiple subcategory (one-to-many relationship)
- one subcategory can only have one category (one-to-one relationship)

There's two way to design this relationship. We can place all in one table:

```
| category_id | id (PK) | label    |
| -           | 1       | Weather  |
| 1           | 2       | Rainy    |
| 1           | 3       | Sunny    |
| -           | 4       | Food     |
| 4           | 5       | Chinese  |
| 5           | 6       | Japanese |
```

- if the `category_id` is not defined, it means the `id` is the `category` id
- if the `category_id` is defined, the `id` refers to the `subcategory` id

Use-cases:
- get all category: query the id without category_id
- get all subcategory: query the id with category_id
- get by category: query those with the category_id that matches the id (category id)

We can index both fields, and to avoid null, set those without category id to `-1`.
