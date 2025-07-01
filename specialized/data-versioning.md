Often, business logic changes at the code level, but those information are not recorded in the db.

For example, you may introduce a new login option, such as fb and goodle, when you previously only had email. However, when doing analytics later, there is no way to segment users before he feature is introduced and after.

One possible way is to version the users, as well as record the date in a separate table.
