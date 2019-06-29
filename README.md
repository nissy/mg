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
- `${PASSWORD}` and `${HOSTNAME}` are set from environment variables

```toml
[production]
  driver = "postgres"
  dsn = "postgres://user:${PASSWORD}@${HOSTNAME}:5432/mg?sslmode=disable"
  source_dir = ["./db/migrate", "./db/seed"]
  version_table = "mg_versions"

[development]
  driver = "postgres"
  dsn = "postgres://user:password@hostname:5432/mg?sslmode=disable"
  source_dir = ["./db/migrate", "./db/seed", "./db/test"]
  version_table = "mg_versions"
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

roll back one version.

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
    down    Roll back one version.
    status  Display the status of migrate.
```
