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


# TODO

- Testing
- Using as few tags as possible - the ORM does a lot of heavy lifting
- jsonschema - validate based on this
- Configuration for the utility should likely be in a file.

# Questions

- How does jsonschema handle arrays? What about the postgres ORM?