# pg_dumper

A small "HTTP-wrapper" around `pg_dump`.

It can be used to trigger dumping and restoring of a Postgres database.

This can be useful to reset or prepare a database during automated tests that don't have control over, for example, 
the transactions (e.g. during [Black-box testing](https://en.wikipedia.org/wiki/Black-box_testing)).

An automated test can then simply control the database state by means of an HTTP "signal".

## Usage example

_Using the example [docker-compose.yaml](docker-compose.yaml). It also contains configuration hints._

In a terminal window: `docker-compose up`

_Hint: use `docker-compose down --volumes` to clean up any data that might exist for the example postgres database._

Then, in a different terminal window:

### List all known dumps (should be empty)

`curl -v "localhost:9999/list"`

### Dump the current database

`curl -v "localhost:9999/dump?name=some-state"`

### List again (should contain the dump that was just created)

`curl -v "localhost:9999/list"`

### Restore an earlier made dump

`curl -v "localhost:9999/restore?name=some-state"`
