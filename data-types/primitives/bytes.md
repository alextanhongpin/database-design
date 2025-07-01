# Bytes requirement

## In MySQL

A column uses one length byte if values require no more than 255 bytes, two length bytes if values may require more than 255 bytes.

## UUID Storage

How many bytes does uuid have?
Uuid are 128 bit value, which means they take 16 bytes of ram to hold a value. A text representation will take 32 bytes (two bytes per character), plus 4 hyphens, plus the two brackets.


## Postgres datatype

https://www.postgresql.org/docs/10/datatype-numeric.html
