# AWS Setup for Postgres

In the security group for type, we need to add this to allow localhost access:

| Type |	Protocol |	Port | Range |	Source |	Description |
| - | - | - | - | - | - |
| PostgreSQL |	TCP |	5432 |	0.0.0.0/0|	RDS | Access |
				
        
## Useful commands

To connect to psql:
```bash
$ psql -h HOSTNAME -p PORT -U USERNAME -W PASSWORD

\q exit 
\c connect to db
\dt list all tables
\du list all roles
\list list databases
```

## Creating new user role
```
CREATE ROLE your_username WITH LOGIN PASSWORD ‘your_password’
Alter role your_username CREATEDB;
```
