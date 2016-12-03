package jbuild

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	jbuildCommands "github.com/jeshuam/jbuild/command"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

var (
	validCommands = map[string]bool{
		"build": true,
		"test":  true,
		"run":   true,
		"clean": true,
	}

	format = logging.MustStringFormatter(
		`%{color}%{level:.1s} %{shortfunc}() >%{color:reset} %{message}`)
)

func printUsage() {
	fmt.Println("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
}

func JBuildRun(args args.Args, cmdArgs []string) error {
	log := logging.MustGetLogger("jbuild")

	// Disable logging if necessary.
	logging.SetFormatter(format)
	if args.ShowLog {
		logging.SetLevel(logging.DEBUG, "jbuild")
		progress.Disable()
	} else {
		logging.SetLevel(logging.CRITICAL, "jbuild")
	}

	// Make sure at least the command was passed.
	if len(cmdArgs) < 1 {
		printUsage()
		return errors.New("No command passed on command-line")
	}

	// Get the command.
	command := cmdArgs[0]
	if !validCommands[command] {
		printUsage()
		return errors.New(fmt.Sprintf("Unknown command '%s'", command))
	}

	// If we are cleaning, just delete the output directory.
	if command == "clean" {
		log.Infof("Cleaning output directory '%s'", args.OutputDir)
		if err := os.RemoveAll(args.OutputDir); err != nil {
			return errors.New(
				fmt.Sprintf("Could not clean output directory: '%s'", err))
		}

		// Maybe clean external repos.
		if args.CleanExternalRepos {
			log.Infof("Cleaning external repos...")
			if err := os.RemoveAll(args.ExternalRepoDir); err != nil {
				return err
			}
		}

		return nil
	}

	// If we aren't cleaning, get more arguments.
	if len(cmdArgs) < 2 {
		printUsage()
		return errors.New("No targets specified on the command-line")
	}

	// Get the current processing target.
	targetArgs := cmdArgs[1:]
	if command == "run" {
		targetArgs = targetArgs[:1]
	}

	// Save 2 lists: a set of targets specified, and a set of targets to process.
	var firstTargetSpecified interfaces.TargetSpec
	targetsSpecified := make(map[string]interfaces.TargetSpec)
	targetsToBuild := make(map[string]interfaces.TargetSpec)
	for _, target := range targetArgs {
		log.Infof("Loading target(s) '%s'", target)
		relStart, _ := filepath.Rel(args.WorkspaceDir, args.CurrentDir)
		specs, err := config.MakeTargetSpec(&args, target, relStart, args.WorkspaceDir)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to load target '%s': %s", target, err))
		}

		for _, spec := range specs {
			// log.Infof("Processing target spec '%s'", spec)

			// Make sure this spec is valid.
			if command == "test" && !strings.HasSuffix(spec.Type(), "test") {
				log.Warningf("Ignoring non-test target '%s'\n", spec)
				continue
			} else if command == "run" && !strings.HasSuffix(spec.Type(), "binary") {
				log.Warningf("Ignoring non-binary target '%s'\n", spec)
				continue
			}

			// log.Infof("Check '%s' for cycles", spec)
			if err := util.CheckForDependencyCycles(spec); err != nil {
				return err
			}

			// log.Infof("Validating '%s'", spec)
			if err := spec.Target().Validate(); err != nil {
				return err
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
	err := jbuildCommands.BuildTargets(&args, targetsToBuild)
	if err != nil {
		return err
	}

	// Further process the targets.
	if command == "run" {
		log.Infof("Running '%s'", firstTargetSpecified)
		cmd := exec.Command(firstTargetSpecified.Target().OutputFiles()[0], cmdArgs[2:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if args.ShowCommands {
			log.Infof("$ %s", cmd.Args)
		}
		cmd.Run()
	} else if command == "test" {
		if len(targetsSpecified) == 1 {
			log.Infof("Testing 1 target")
		} else {
			log.Infof("Testing %d targets", len(targetsSpecified))
		}

		jbuildCommands.RunTests(&args, targetsSpecified)
	}

	return nil
}
