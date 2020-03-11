package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/nissy/mg"
)

var (
	cfgFile   = flag.String("c", "mg.toml", "")
	newSource = flag.String("n", "", "")
	isHelp    = flag.Bool("h", false, "")
	isVersion = flag.Bool("v", false, "")
	version   = "dev"

	sourceTemplate = fmt.Sprintf("-- %s\n\n\n-- %s\n\n", mg.DefaultUpAnnotation, mg.DefaultDownAnnotation)
)

func init() {
	flag.Usage = func() {
		if _, err := fmt.Fprint(os.Stderr, help); err != nil {
			panic(err)
		}
	}
}

func main() {
	if err := run(); err != nil {
		if _, perr := fmt.Fprintln(os.Stderr, err.Error()); perr != nil {
			panic(err)
		}
		os.Exit(1)
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
	if len(*newSource) > 0 {
		filename := fmt.Sprintf("%s_%s.sql", time.Now().Format("20060102150405"), *newSource)
		if err := ioutil.WriteFile(filename, []byte(sourceTemplate), 0664); err != nil {
			return err
		}
		fmt.Println(filename)
		return nil
	}

	if len(*cfgFile) > 0 {
		ms, err := mg.OpenCfg(*cfgFile)
		if err != nil {
			return err
		}
		if args := flag.Args(); len(args) >= 2 {
			var werr error
			for i := 1; i < len(args); i++ {
				m, ok := ms[args[i]]
				if !ok {
					return fmt.Errorf("Error: Section is %s %s", args[i], "does not exist.")
				}
				switch args[0] {
				case "up":
					if err := m.Do(mg.UpDo); err != nil {
						return err
					}
				case "down":
					if err := m.Do(mg.DownDo); err != nil {
						return err
					}
				case "status":
					if err := m.Do(mg.StatusDo); err != nil {
						if werr == nil {
							werr = err
							continue
						}
						werr = fmt.Errorf("%w\n%s", werr, err.Error())
					}
				default:
					return fmt.Errorf("Command is %s does not exist.", args[0])
				}
			}
			if werr != nil {
				return werr
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
`
