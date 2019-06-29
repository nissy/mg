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
$ brew install nissy/mg/mg
```

### config

- default config read is `mg.toml` in current directory
- environment variable support

```toml
[postgres-sample]
  driver = "postgres"
  dsn = "postgres://user:password@127.0.0.1:5432/dbname?sslmode=disable"
  source_dir = ["./data/migrates", "./data/seeds"]
  version_table = "migration_versions"

[mysql-sample]
  driver = "mysql"
  dsn = "user:password@tcp(127.0.0.1:3306)/dbname"
  source_dir = ["./data/migrates", "./data/seeds"]
  version_table = "migration_versions"

[environment-variable-sample]
  driver = "postgres"
  dsn = "postgres://user:${PASSWORD}@${HOSTNAME}:5432/dbname?sslmode=disable"
  source_dir = ["./data/migrates", "./data/seeds"]
  version_table = "migration_versions"
```

### commands

 `mg [options] <command> [sections...]`

#### up

migrate to the latest version.

```bash
$ mg up development
OK migrates/2019060819341935_users.sql to development
OK seeds/2019060819341948_users.sql to development
```

#### down

back to previous version.

```bash
$ mg down development
OK seeds/2019060819341948_users.sql to development
```

#### status

display the status of migrate.

```bash
$ mg status development
Version of development:
    current:
        2019060819341935
    unapplied:
        2019060819341948 seeds/2019060819341948_users.sql
```

### help
```
Usage:
    mg [options] <command> [sections...]
Options:
    -c string
        Set configuration file. (default "mg.toml")
    -h bool
        This help.
    -v bool
        Display the version of mg.
Commands:
    up      Migrate to the latest version.
    down    Back to previous version.
    status  Display the status of migrate.
```
