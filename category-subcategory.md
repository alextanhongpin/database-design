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


## Using recursive pattern

```mysql
CREATE TABLE IF NOT EXISTS category (
	category varchar(255),
	parent_category varchar(255) NOT NULL DEFAULT '',
	description varchar(255),
	PRIMARY KEY (category, parent_category),
	FOREIGN KEY (parent_category) REFERENCES category(category)
);
INSERT INTO category (category, description) VALUES ('', 'NONE');
INSERT INTO category (category, description) VALUES ('food', 'food category');
INSERT INTO category (category, parent_category, description) VALUES ('food.dairy', 'food', 'dairy products category');
INSERT INTO category (category, parent_category, description) VALUES ('food.drinks', 'food', 'beverages products category');
SELECT parent_category, JSON_ARRAYAGG(category) AS subcategories FROM category GROUP BY parent_category;
```
