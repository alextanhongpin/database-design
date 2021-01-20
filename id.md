# MySQL

Store the uuids as order binary ids to save space and improve performance, as well as sortability.

# General

- use uuid for generated rows
- use int id for reference table (countries, currency etc), but mask the id when returning to client (e.g. using hashid)
