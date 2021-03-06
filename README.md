# mg
mg is database migrations command.

support databases
- PostgreSQL
- MySQL

### install

```bash
$ go get -u github.com/nissy/mg/cmd/mg
```

Mac OS X
```bash
$ brew tap nissy/mg
$ brew install nissy/mg/mg
```

### config

- default config read is `mg.toml` in current directory
- environment variable support

```toml
[postgres-sample]
  driver = "postgres"
  dsn = "postgres://user:password@127.0.0.1:5432/dbname?sslmode=disable"
  source_dir = [
    "./testdata/postgres/migrates",
    "./testdata/postgres/seeds"
  ]

[mysql-sample]
  driver = "mysql"
  dsn = "user:password@tcp(127.0.0.1:3306)/dbname"
  source_dir = [
    "./testdata/mysql/migrates",
    "./testdata/mysql/seeds"
  ]

[environment-variable-sample]
  driver = "postgres"
  dsn = "postgres://user:${PASSWORD}@${HOSTNAME}:5432/dbname?sslmode=disable"
  source_dir = [
    "./testdata/postgres/migrates",
    "./testdata/postgres/seeds"
  ]
```

options

```toml
[option-sample] # section name
  driver = "postgres" # database driver
  dsn = "postgres://user:${PASSWORD}@${HOSTNAME}:5432/dbname?sslmode=disable" # database dsn
  source_dir = [ # database source directorys
    "./testdata/postgres/migrates",
    "./testdata/postgres/seeds"
  ]
  up_annotation = "+goose Up" # database up command annotation
  down_annotation = "+goose Down" # database down command annotation
  version_table = "migration_versions" # versions use table name
  version_start_number = 2019060819341936 # version start number
  json_format = true # output message format
```


### source sql

```sql
-- @migrate.up
CREATE TABLE users (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  name varchar(255) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  created_at datetime DEFAULT NULL,
  updated_at datetime DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

-- @migrate.down
DROP TABLE users;
```

### commands

 `mg [options] <command> [sections...]`

#### up

apply current version or later.

```bash
$ mg up postgres-sample
OK 2019060819341935 to postgres-sample is testdata/postgres/migrates/2019060819341935_users.sql
OK 2019060819341948 to postgres-sample is testdata/postgres/seeds/2019060819341948_users.sql
```

#### down

back to previous version.

```bash
$ mg down postgres-sample
OK 2019060819341948 to postgres-sample is testdata/postgres/seeds/2019060819341948_users.sql
```

#### status

display the current applied status.

```bash
$ mg status postgres-sample
Version of postgres-sample:
    unapplied:
        2019060819341811 testdata/postgres/seeds/2019060819341811_jobs.sql
    current:
        2019060819341935
    apply:
        2019060819341948 testdata/postgres/seeds/2019060819341948_users.sql
```

exit with 1 if there is version that has not been applied.

```
Error: Section is postgres-sample There are versions that do not apply.
exit status 1
```

#### force-up

apply all versions not currently applied.

```bash
$ mg force-up postgres-sample
OK 2019060819341811 to postgres-sample is testdata/postgres/seeds/2019060819341811_jobs.sql
OK 2019060819341948 to postgres-sample is testdata/postgres/seeds/2019060819341948_users.sql
```

### help
```
Usage:
    mg [options] <command> [sections...]
Options:
    -c string
        Set configuration file. (default "mg.toml")
    -n string
        Create empty source file.
    -h bool
        This help.
    -v bool
        Display the version of mg.
Commands:
    up       Apply current version or later.
    force-up Apply all versions not currently applied.
    down     Back to previous version.
    status   Display the current applied status.
```
