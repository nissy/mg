# mg
mg is database migrations command.

### config

mg.toml
```toml
[production]
  driver = "postgres"
  dsn = "postgres://user:password@hostname:5432/mg?sslmode=disable"
  source_dir = ["./db/migrate", "./db/seed"]
  version_table = "mg_versions"

[development]
  driver = "postgres"
  dsn = "postgres://user:password@hostname:5432/mg?sslmode=disable"
  source_dir = ["./db/migrate", "./db/seed", "./db/test"]
  version_table = "mg_versions"
```

### command

up
```bash
$ mg up development
```

down
```bash
$ mg down development
```

### help
```
Usage:
    mg [options] <command> [sections...]
Options:
    -c string
        set configuration file. (default "mg.toml")
    -h bool
        this help.
    -v bool
        show version and exit.
Commands:
    up      Migrate the DB to the most recent version available
    down    Roll back the version by 1
```
