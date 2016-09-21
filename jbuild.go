package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jeshuam/jbuild/args"
	jbuildCommands "github.com/jeshuam/jbuild/command"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/op/go-logging"
)

var (
	log    = logging.MustGetLogger("jbuild")
	format = logging.MustStringFormatter(
		`%{color}%{level:.1s} %{shortfunc}() >%{color:reset} %{message}`)

	validCommands = map[string]bool{
		"build": true,
		"test":  true,
		"run":   true,
		"clean": true,
	}
)

func printUsage() {
	fmt.Println("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
}

func main() {
	// Parse flags.
	flag.Parse()

	// Setup logging.
	logging.SetFormatter(format)
	if args.ShowLog {
		logging.SetLevel(logging.DEBUG, "jbuild")
	} else {
		logging.SetLevel(logging.CRITICAL, "jbuild")
	}

	// Load flags.
	log.Debug("Loading flags...")
	if err := args.Load(); err != nil {
		log.Fatal(err.Error())
	}

	log.Debug("Done loading flags!")

	// Make sure at least the command was passed.
	if len(flag.Args()) < 1 {
		log.Error("No command passed on command-line")
		printUsage()
		return
	}

	// Get the command.
	command := flag.Args()[0]
	if !validCommands[command] {
		log.Errorf("Unknown command '%s'", command)
		printUsage()
		return
	}

	// If we are cleaning, just delete the output directory.
	if command == "clean" {
		log.Infof("Cleaning output directory '%s'", args.OutputDir)
		if err := os.RemoveAll(args.OutputDir); err != nil {
			log.Fatalf("Could not clean output directory: '%s'", err)
		}

		return
	}

	// If we aren't cleaning, get more arguments.
	if len(flag.Args()) < 2 {
		log.Error("No targets specified on the command-line")
		printUsage()
		return
	}

	// Get the current processing target.
	targetArgs := flag.Args()[1:]

	// Save 2 lists: a set of targets specified, and a set of targets to process.
	var firstTargetSpecified interfaces.TargetSpec
	targetsSpecified := make(map[string]interfaces.TargetSpec)
	targetsToBuild := make(map[string]interfaces.TargetSpec)
	for _, target := range targetArgs {
		log.Infof("Loading target(s) '%s'", target)
		specs, err := config.MakeTargetSpec(target, common.CurrentDir)
		if err != nil {
			log.Fatalf("Failed to load target '%s': %s", target, err)
		}

		for _, spec := range specs {
			log.Infof("Processing target spec '%s'", spec)

			// Make sure this spec is valid.
			if command == "test" && !strings.HasSuffix(spec.Type(), "test") {
				log.Warningf("Ignoring non-test target '%s'\n", spec)
				continue
			} else if command == "run" && !strings.HasSuffix(spec.Type(), "binary") {
				log.Warningf("Ignoring non-binary target '%s'\n", spec)
				continue
			}

			log.Infof("Check '%s' for cycles", spec)
			if err := util.CheckForDependencyCycles(spec); err != nil {
				log.Fatalf("Error: %s", err)
			}

			log.Infof("Validating '%s'", spec)
			if err := spec.Target().Validate(); err != nil {
				log.Fatalf("Error: %s", err)
			}

			// Save the target.
			if len(targetsSpecified) == 0 {
				firstTargetSpecified = spec
			}

			targetsSpecified[spec.String()] = spec
			targetsToBuild[spec.String()] = spec
			for _, spec := range spec.Target().AllDependencies() {
				targetsToBuild[spec.String()] = spec
			}
		}
	}

	// Build the targets.
	log.Info("Building targets...")
	jbuildCommands.BuildTargets(targetsToBuild)

	// Further process the targets.
	if command == "run" {
		log.Infof("Running '%s'", firstTargetSpecified)
		cmd := exec.Command(firstTargetSpecified.Target().OutputFiles()[0], flag.Args()[2:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
	} else if command == "test" {
		if len(targetsSpecified) == 1 {
			log.Infof("Testing 1 target")
		} else {
			log.Infof("Testing %d targets", len(targetsSpecified))
		}

		jbuildCommands.RunTests(targetsSpecified)
	}
}
