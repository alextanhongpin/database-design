## Creating slug with postgres function

```sql
CREATE EXTENSION IF NOT EXISTS "unaccent";

CREATE OR REPLACE FUNCTION slugify("value" TEXT)
RETURNS TEXT AS $$
  -- removes accents (diacritic signs) from a given string --
  WITH "unaccented" AS (
    SELECT unaccent("value") AS "value"
  ),
  -- lowercases the string
  "lowercase" AS (
    SELECT lower("value") AS "value"
    FROM "unaccented"
  ),
  -- remove single and double quotes
  "removed_quotes" AS (
    SELECT regexp_replace("value", '[''"]+', '', 'gi') AS "value"
    FROM "lowercase"
  ),
  -- replaces anything that's not a letter, number, hyphen('-'), or underscore('_') with a hyphen('-')
  "hyphenated" AS (
    SELECT regexp_replace("value", '[^a-z0-9\\-_]+', '-', 'gi') AS "value"
    FROM "removed_quotes"
  ),
  -- trims hyphens('-') if they exist on the head or tail of the string
  "trimmed" AS (
    SELECT regexp_replace(regexp_replace("value", '\-+$', ''), '^\-', '') AS "value"
    FROM "hyphenated"
  )
  SELECT "value" FROM "trimmed";
$$ LANGUAGE SQL STRICT IMMUTABLE;
```

Run:
```sql
-- Having to escape the quote here to demo...
select slugify('The World''s "Best" Cafés!');
```

Trigger on `create` (see notes):

```sql
CREATE FUNCTION public.set_slug_from_title() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.slug := slugify(NEW.title);
  RETURN NEW;
END
$$;
```

Attach trigger:

```sql
CREATE TRIGGER "t_news_insert" BEFORE INSERT ON "news" FOR EACH ROW WHEN (NEW.title IS NOT NULL AND NEW.slug IS NULL)
EXECUTE PROCEDURE set_slug_from_title();
```

Best practices:
- create slug only during create, and if it is not provided. We can use trigger to create a default slug, or `COALESCE(NULLIF($1, ''), slugify($1))`
- don't update slug when you update the name - it will cause urls to be broken, unless you have a mechanism to prevent that. Take a look at rail's history slug implementation. Note that this can be tricky too in a lot of situation, because once a slug becomes history, we need to ensure it will no longer be used by other entity. Else it could cause infinite redirect.
- the function does not handle duplicate slugs, we can append a number behind (make sure it's not the count of the number of similar slugs!, else when one of them is removed, it will cause problem. e.g. hallo1, hallo3, hallo4 will always cause duplicate, since the count will always be three).

References:
- https://www.kdobson.net/2019/ultimate-postgresql-slug-function/
