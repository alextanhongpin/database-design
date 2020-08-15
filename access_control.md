## Database Administration


postgres administration for microservice
- creating roles
- managing multiple users and microservices
- creating rules
- restricting actions through permissions
- creating views?
- delete permission
- readonly permission for non-replicas
https://ncona.com/2020/01/postgresql-user-management/
https://cloud.ibm.com/docs/databases-for-postgresql?topic=databases-for-postgresql-user-management
https://aws.amazon.com/blogs/database/managing-postgresql-users-and-roles/#:~:text=PostgreSQL%20lets%20you%20grant%20permissions,appropriate%20role%20to%20each%20user.

- alternative is to create rules for user using trigger
- we can set config in postgres in a transaction, and only activate the trigger if certain rule is fulfilled

## Root user
Create the database as the root user. Then grant the user the database rights.

```
GRANT ALL PRIVILEGES ON database_name.* TO user@'%';
```
