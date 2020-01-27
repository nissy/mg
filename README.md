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

migrate to the latest version.

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

display the status of migrate.

```bash
$ mg status postgres-sample
Version of postgres-sample:
    current:
        2019060819341935
    unapplied:
        2019060819341948 testdata/postgres/seeds/2019060819341948_users.sql
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
    up      Migrate to the latest version.
    down    Back to previous version.
    status  Display the status of migrate.
```
