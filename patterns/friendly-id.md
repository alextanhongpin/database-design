# Friendly Ids



## What

Using names to represent a resource instead of numeric ids.



Made popular in Rails.

- https://norman.github.io/friendly_id/FriendlyId.html

- https://norman.github.io/friendly_id/FriendlyId/History.html



One of the perils of using integer id is when creating an API that requires id to be returned. We want to avoid returning integer id to avoid endpoint enumeration.



Google has an article about url identifiers too: https://cloud.google.com/blog/products/api-management/api-design-choosing-between-names-and-identifiers-in-urls



Using alternatives to mask ids only adds more complexity to the system, as the system now has to encode/decode the ids for every single api calls. Also, analytics becomes a pain point because the clients are forced to use the masked identifier, and other non-tech users have to depend on the engineers that works on creating the implementation to decode back those identifiers.



If possible, avoid using numeric id (int/bigint) alltogether.



## Why

Mainly because numeric id is more (?) performant when it comes to sorting as compared to when using random uuids/long text.

You can still using two columns, the primary key being the uuid, and the sort key using the serial int column that will always be unique, as compared to using timestamp.



## Other alternatives

Hash ID



## Other gotchas

If you actually use friendly ids, ensure that they are not *purely* numeric strings.



There was a case where the user name is 123 and due to how friendly id operates, it searches for the actual user with primary key 123 instead of slug 123.




