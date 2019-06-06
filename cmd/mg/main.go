package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/nissy/mg"
)

var (
	cfgFile = flag.String("c", "mg.toml", "set configuration file.")
	up      = flag.String("up", "", "up migrations.")
	down    = flag.String("down", "", "down migrations.")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	flag.Parse()

	if len(*cfgFile) > 0 {
		m, err := mg.ReadConfig(*cfgFile)
		if err != nil {
			return err
		}

		switch {
		case len(*up) > 0:
			if vv, ok := m[*up]; ok {
				if err := vv.Up(*up); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Error: Selection is %s does not exist.", *up)
			}
		case len(*down) > 0:
			if vv, ok := m[*down]; ok {
				if err := vv.Down(*down); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Error: Selection is %s does not exist.", *down)
			}
		}
	}

	return nil
}
