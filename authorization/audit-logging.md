## Audit-Logging

https://www.codeproject.com/Articles/105768/Audit-Trail-Tracing-Data-Changes-in-Database

## CQRS/Event Sourcing vs Audit Logging vs Temporal Database


## Audit postgres options

- https://wiki.postgresql.org/wiki/Audit_trigger_91plus
    - the implementation uses hstore, if you want to use jsonb, check out https://github.com/razorlabs/pg-json-audit-trigger
- logical decoding

https://severalnines.com/database-blog/postgresql-audit-logging-best-practices
https://www.cybertec-postgresql.com/en/row-change-auditing-options-for-postgresql/
https://github.com/pgMemento/pgMemento

## PGMemento

```sql
-- Running this twice will produce the message 'pgMemento is already intialized for public schema.';
SELECT pgmemento.init('public');

-- Add individual tables.
SELECT pgmemento.create_table_audit('table-name');
```


## Audit Logging vs Event sourcing


Which is better? 
- audit Logging normally creates a copy of the old row and inserts it into another table
- event sourcing are more specific - it records the changes before and after, and only save those changes
- the single source of truth for audit logging table is the logs table, similar to event sourcing. Because all the changes are recorded there.
