package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	Verbose    = false
	DryRun     = true
	SystemRoot = "/"
)

func drylogf(format string, args ...interface{}) {
	if DryRun {
		log.Println("[DRY]", fmt.Sprintf(format, args...))
	}
}

func vlogf(format string, args ...interface{}) {
	if Verbose {
		log.Print("# ", fmt.Sprintf(format, args...))
	}
}

var defaultStages = []string{
	"get-address",
}

func inRoot(path ...string) string {
	return filepath.Join(SystemRoot, filepath.Join(path...))
}

func main() {
	status := Main(os.Args[1:])
	os.Exit(status)
}

func Main(args []string) int {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	chroot := flag.String("C", "", "Chroot to directory (optional)")
	flag.BoolVar(&DryRun, "D", DryRun, "Dry run")
	flag.BoolVar(&Verbose, "v", Verbose, "Verbose logging")
	flag.StringVar(&SystemRoot, "r", SystemRoot, "Absolute path to root dir")
	flag.Parse()

	if DryRun {
		setDryRun()
	}

	if err := setChroot(*chroot); err != nil {
		log.Printf("Error setting chroot: %v", err)
		return 1
	}

	args = flag.Args()
	if n := len(args); n == 0 || (n == 1 && args[0] == "default") {
		args = defaultStages
	}

	// Parse stages for execution
	stages, err := parseStages(args)
	if err == flag.ErrHelp {
		return 2
	} else if err != nil {
		log.Printf("Unable to parse comand line: %v", err)
		return 1
	}

	for nth, stage := range stages {
		name := stage.Name()
		log.Print("# Beginning stage ", name)
		prefix := fmt.Sprintf("[%d/%d] %s: ", nth+1, len(stages), name)
		log.SetPrefix(prefix)

		// TODO: If an error is returned, prompt user if they would like to continue,
		// assuming they've a) suspended the installer corrected whatever went wrong and b)
		// want to proceed with the installation.
		//
		// Maybe provide an interrupt() function for any stage to call?
		err := stage.Run()
		if err != nil {
			log.Printf("Fatal error in stage %q: %v", stage.Name(), err)
			return 1
		}

		log.SetPrefix("")
	}

	return 0
}

type Stage interface {
	Name() string
	Configure(*flag.FlagSet)
	Validate() error
	Run() error
}

type newStageFunc func() (Stage, error)

var stageConstructors = map[string]newStageFunc{
	"get-address": newGetAddressStage,
}

func parseStages(args []string) (stages []Stage, err error) {
	for len(args) > 0 {
		name := args[0]
		stage, err := newStage(name)
		if err != nil {
			return nil, fmt.Errorf("unable to create stage %q: %v", name, err)
		}

		flags := flag.NewFlagSet(name, flag.ContinueOnError)
		stage.Configure(flags)

		switch err := flags.Parse(args[1:]); err {
		case nil:
		case flag.ErrHelp:
			return nil, err
		default:
			return nil, fmt.Errorf("error parsing %q arguments: %v", name, err)
		}

		if err := stage.Validate(); err != nil {
			return nil, fmt.Errorf("unable to configure stage %q: %v", name, err)
		}

		stages = append(stages, stage)
		args = flags.Args()
	}
	return stages, nil
}

func newStage(name string) (Stage, error) {
	ctor := stageConstructors[name]
	if ctor == nil {
		return nil, errors.New("unrecognized stage name")
	}
	return ctor()
}

// enableService creates a service link, if one does not already exist, for the default runlevel.
// It checks first to ensure that serviceName's directory exists.
func enableService(serviceName string) error {
	serviceDir := inRoot("/etc/sv", serviceName)
	stat, err := osLstat(serviceDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("cannot enable service %q: %v", serviceName, err)
	} else if !stat.IsDir() {
		return fmt.Errorf("cannot enable service %q: %q is not a directory", serviceName, serviceDir)
	}
	return symlink(serviceDir, inRoot("etc/runit/runsvdir/default", serviceName), false)
}
