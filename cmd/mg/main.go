package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/nissy/mg"
)

var (
	cfgFile   = flag.String("c", "mg.toml", "")
	isHelp    = flag.Bool("h", false, "")
	isVersion = flag.Bool("v", false, "")
	version   = "dev"
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
		_, err := fmt.Fprint(os.Stderr, help)
		return err
	}
	if *isVersion {
		fmt.Printf("Version is %s\n", version)
		return nil
	}

	if len(*cfgFile) > 0 {
		m, err := mg.ReadConfig(*cfgFile)
		if err != nil {
			return err
		}

		if args := flag.Args(); len(args) >= 2 {
			for i := 1; i < len(args); i++ {
				switch args[0] {
				case "up":
					if vv, ok := m[args[i]]; ok {
						if err := vv.Do(mg.UpDo); err != nil {
							return err
						}
					} else {
						return fmt.Errorf("Selection is %s does not exist.", args[i])
					}
				case "down":
					if vv, ok := m[args[i]]; ok {
						if err := vv.Do(mg.DownDo); err != nil {
							return err
						}
					} else {
						return fmt.Errorf("Selection is %s does not exist.", args[i])
					}
				case "status":
					if vv, ok := m[args[i]]; ok {
						if err := vv.Do(mg.StatusDo); err != nil {
							return err
						}
					} else {
						return fmt.Errorf("Selection is %s does not exist.", args[i])
					}
				default:
					return fmt.Errorf("Command is %s does not exist.", args[0])
				}
			}
		} else {
			return errors.New("Command is incorrect.")
		}
	}

	return nil
}

var help = `Usage:
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
`
