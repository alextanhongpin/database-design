# WAL

Attempt to use WAL for snapshot testing, but seems like it is not possible:

```sql
show wal_level;
show max_replication_slots;
show max_wal_senders;

SELECT pg_create_logical_replication_slot('replication_slot', 'test_decoding');
SELECT pg_create_logical_replication_slot('replication_slot', 'pgoutput');

SELECT slot_name, plugin, slot_type, database, active, restart_lsn, confirmed_flush_lsn FROM pg_replication_slots;
CREATE PUBLICATION pub FOR ALL TABLES;

create table users (name text);
drop table users;

SELECT * FROM pg_publication_tables WHERE pubname='pub';
select * from pg_replication_slots;

insert into users (name) values ('john');
update users set name = 'jane';
delete from users;
-- ERROR:  cannot delete from table "users" because it does not have a replica identity and publishes deletes
-- HINT:  To enable deleting from the table, set REPLICA IDENTITY using ALTER TABLE.

alter table users replica identity full;

-- for test_decoding
SELECT * FROM pg_logical_slot_get_changes('replication_slot', NULL, NULL);

-- for pgoutput
-- https://www.postgresql.org/docs/current/protocol-logicalrep-message-formats.html
-- https://github.com/kyleconroy/pgoutput/tree/master
SELECT *, data::text FROM pg_logical_slot_peek_binary_changes('replication_slot', null, null, 'proto_version', '1', 'publication_names', 'pub');


SELECT pg_drop_replication_slot('replication_slot');
```
