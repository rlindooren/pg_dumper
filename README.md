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

### List all dumps

_Response should be empty initially._

`curl "localhost:9999/list"`

### Dump the current database

`curl "localhost:9999/dump?name=some-state"`

_The response should look something like this:_

```
SUCCESS
Succesfully executed pg_dump with arguments: [--clean --format=plain -f /dumps/some-state.dump.sql]):
```

### List again

`curl "localhost:9999/list"`

_The response should now contain the dump that was just created, in the previous step. E.g.:_

```
some-state (/dumps/some-state.dump.sql @ 2022-10-04 12:33:27.316599119 +0000 UTC)
```

### Restore an earlier made dump

`curl "localhost:9999/restore?name=some-state"`

_The response should look something like this:_

```
SUCCESS
Succesfully executed psql with arguments: [-f /dumps/some-state.dump.sql]):
```
