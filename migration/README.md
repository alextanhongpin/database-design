# Migration



## What

Migrations are operations that makes changes to the database. This can be creating/deleting tables/columns.

## Why

Migrations are not as easy as it seems. When done incorrectly, it can 

- cause unexpected downtime in production

- in shared-staging environment (aka single staging environment where multiple devs push their non-production ready code for testing), migration can easily go out-of-order, depending on who runs the migrations first

- in local development, if you work on different branches that requires different migration, it can make your local db schema out of sync (one migration file is present in one branch, but not in another)



## How

There are actually two ways to perform migration, the more common approach is to generate versioned migration files and commit it along the version control.



The other approach is used by Github, and involves _diffing_ the changes between current and target schema, and applying the different to the tables. In short, instead of writing multiple versioned migrations files that will be applied to the database, only the final schema is checked-in the source control.



This has the following implications:

- there's no need to keep a lot of migration files, we only store the final schema

- there's no worry of out-of-order migration, this operation only attempts to _sync_ the schema to the target schema. This could still result in data loss when working in different branches.

- There's no need to write custom migration (e.g. `alter table add column ...`). The tool should be smart enough to generate the statements required to perform the migration.

- Schema will not go out of sync (for example, if there's an attempt to run the migration outside of the application, this will be sync back to follow the application schema).
