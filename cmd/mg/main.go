package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/nissy/mg"
)

var (
	filename = flag.String("c", "mg.toml", "set configuration file.")
	up       = flag.String("up", "", "up migrations.")
	down     = flag.String("down", "", "down migrations.")
	number   = flag.Int("n", 0, "number of migrations to execute.")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	flag.Parse()

	if len(*filename) > 0 {
		m, err := mg.ReadConfig(*filename)
		if err != nil {
			return err
		}

		switch {
		case len(*up) > 0:
			if vv, ok := m[*up]; ok {
				if err := vv.Up(*up, *number); err != nil {
					return err
				}
			} else {
				return errors.New(fmt.Sprintf("%s is notfound.", *up))
			}
		case len(*down) > 0:
			if vv, ok := m[*down]; ok {
				if err := vv.Down(*down, *number); err != nil {
					return err
				}
			} else {
				return errors.New(fmt.Sprintf("%s", *down))
			}
		}
	}

	return nil
}
