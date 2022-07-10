## Creating multiple database in docker

```yaml
version: "3.7"
services:
  db:
    container_name: "app_db"
    image: "postgres:11.5-alpine"
    volumes:
      - ./pg-init-scripts:/docker-entrypoint-initdb.d
    environment:
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_MULTIPLE_DATABASES=app,app_test
```

```bash
  
#!/bin/bash

set -e
set -u

function create_user_and_database() {
	local database=$1
	echo "  Creating user and database '$database'"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
	    CREATE USER $database;
	    CREATE DATABASE $database;
	    GRANT ALL PRIVILEGES ON DATABASE $database TO $database;
EOSQL
}

if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
	echo "Multiple database creation requested: $POSTGRES_MULTIPLE_DATABASES"
	for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
		create_user_and_database $db
	done
	echo "Multiple databases created"
fi
```


## Easily setup docker-compose

In your `.zshrc`, include this alias:

```sh
alias init-db='svn export https://github.com/alextanhongpin/docker-samples/trunk/postgres/docker-compose.yml'
```

Now you can just run `init-db` anytime to setup a local `docker-compose.yml`. `svn` will just download one file from the given url.
