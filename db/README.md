# `db` package

This package houses the types and functionality for internal Antidote data management.

In db/models, there are models for two main types of things in Antidote:
- "Live" state (prepended by `live*`), such as livelessons and livesesssions. These track runtime state of Antidote, such as launched lessons, session awareness, etc.
- Curriculum resource definitions

There are also functions enforced by an interface `DataManager` which do the expected CRUD operations for all of these models with the underlying datastore.

This interface also enforces functions for importing curriculum resource types into memory, where appropriate. These do not interact with the underlying datastore, but higher-order code may make use of these functions as a precursor to actually inserting those

# Postgres Stuff

Running a test instance in docker:

```
docker run --rm --name pg-docker \
    -e POSTGRES_PASSWORD=docker \
    -d -p 5432:5432  \
    -v $HOME/docker/volumes/postgres:/var/lib/postgresql/data \
    postgres
```

Connect to postgres via bash:

```
docker exec -it pg-docker psql -h localhost -U postgres -d antidote
```

Create database if you haven't already - this is not up to Antidote to do - it assumes the database
already exists.

```
CREATE DATABASE antidote;
```

Show databases:

\l


Show tables:

\dt

Show schema for a specific table:

\d+ lessons



# TODO

- Testing
- Using as few tags as possible - the ORM does a lot of heavy lifting
- jsonschema - validate based on this
- Configuration for the utility should likely be in a file.

# Questions

- How does jsonschema handle arrays? What about the postgres ORM?