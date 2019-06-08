# mg
mg is database migrations command.

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