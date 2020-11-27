# Postgres

- `user` is a reserved table name, use `"user"` instead
- column names should be lowercase separated by underscore, e.g. `party_relationship`
- if you intend to use camelcase for table name or/and columns, wrap them in double quotes. While this is uncommon, but it simplifies serializing the data back to entity model for some language (probably dynamic language) and most of the client javascript naming convention is also camelcase

```sql
CREATE TABLE "public"."Post" (
	id int GENERATED ALWAYS AS IDENTITY,
	title text NOT NULL,
	"createdAt" timestamptz NOT NULL DEFAULT now(),
	content text,
	published boolean NOT NULL DEFAULT false,
	"authorId" int NOT NULL,
	FOREIGN KEY ("authorId") REFERENCES "public"."User"(id)

)
```
