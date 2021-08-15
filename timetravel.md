# Timetravel in postgres

Prior to Postgres 12, there was an extension to implement time travel. However, it was removed. You can probably emulate them now with just tstzrange and some trigger hacks.

https://www.postgresql.org/docs/11/contrib-spi.html
