# Should email be stored as lowercase?

Don't bother digging the rabbit hole - just lowercase the email when you store it. You can also use citext to store the original case, which allows case insensitive matching.

RFC standard says otherwise, but google/lastpass doesn't care what casing you use to login the account. It's also better for user registration on mobile since they can accidentally uppercase the first character, and become unable to login later.

- https://stackoverflow.com/questions/21887735/should-emails-in-mongodb-be-stored-as-lowercase
- https://ux.stackexchange.com/questions/16848/if-your-login-is-an-email-should-you-canonicalize-lower-case-it
- https://community.auth0.com/t/creating-a-user-converts-email-to-lowercase/6678
