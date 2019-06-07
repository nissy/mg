package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/nissy/mg"
)

var (
	cfgFile = flag.String("c", "mg.toml", "set configuration file.")
	up      = flag.String("up", "", "up migrations.")
	down    = flag.String("down", "", "down migrations.")
	isHelp  = flag.Bool("h", false, "this help")
)

func main() {
	if err := run(); err != nil {
		if _, perr := fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error()); perr != nil {
			panic(err)
		}
	}
}

func run() (err error) {
	flag.Parse()

	if *isHelp {
		if _, err := fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0]); err != nil {
			panic(err)
		}
		flag.PrintDefaults()
		return nil
	}

	if len(*cfgFile) > 0 {
		m, err := mg.ReadConfig(*cfgFile)
		if err != nil {
			return err
		}

		switch {
		case len(*up) > 0:
			if vv, ok := m[*up]; ok {
				if err := vv.Exec(*up, mg.DoUp); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Selection is %s does not exist.", *up)
			}
		case len(*down) > 0:
			if vv, ok := m[*down]; ok {
				if err := vv.Exec(*down, mg.DoDown); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Selection is %s does not exist.", *down)
			}
		default:
			return errors.New("Please specify an option.")
		}
	}

	return nil
}
