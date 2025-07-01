 # Using views vs composing in application
 
 it is possible to rely on plpgsql to return json directly, rather than querying in the app and composing the value. 
 
 - reduce serializing/deserializing in app
 - does not require code changes (except adding migration files to update the views)


cons
- access control might complicate stuff
- generated columns, especially those that requires business logic might not be possible in plpgsql
