package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

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

func findWorkspaceDir(cwd string) string {
	// Find the workspace directory.
	workspaceDir, _, err := config.FindWorkspaceFile(cwd)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// If the output directory flag was relative, make it absolute relative to
	// the workspace directory.
	if !filepath.IsAbs(common.OutputDirectory) {
		common.OutputDirectory = filepath.Join(workspaceDir, common.OutputDirectory)
	}

	return workspaceDir
}

func printUsageAndExit() {
	log.Fatalf("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
}

func main() {
	flag.Parse()

	// Setup the logger.
	logging.SetFormatter(format)
	if !*jbuildCommands.UseProgress {
		logging.SetLevel(logging.DEBUG, "jbuild")
	} else {
		logging.SetLevel(logging.CRITICAL, "jbuild")
	}

	// First, see if we are in a workspace.
	if common.WorkspaceDir == "" {
		common.WorkspaceDir = findWorkspaceDir(common.CurrentDir)
	}

	// Make sure at least the command was passed.
	if len(flag.Args()) < 1 {
		printUsageAndExit()
	}

	// Get the command.
	command := flag.Args()[0]
	if !validCommands[command] {
		fmt.Printf("Unknown command '%s'.\n", command)
		return
	}

	// If we are cleaning, just delete the output directory.
	if command == "clean" {
		// jbuildCommands.Clean(common.WorkspaceDir)
		return
	}

	// If we aren't cleaning, get more arguments.
	if len(flag.Args()) < 2 {
		fmt.Println("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
		return
	}

	// Get the current processing target.
	targetArgs := flag.Args()[1:]

	// Save 2 lists: a set of targets specified, and a set of targets to process.
	var firstTargetSpecified interfaces.TargetSpec
	targetsSpecified := make(map[string]interfaces.TargetSpec)
	targetsToBuild := make(map[string]interfaces.TargetSpec)
	for _, target := range targetArgs {
		specs, err := config.MakeTargetSpec(target, common.CurrentDir)
		if err != nil {
			fmt.Printf("Failed: %s", err)
			return
		}

		for _, spec := range specs {
			// Make sure this spec is valid.
			if command == "test" && !strings.HasSuffix(spec.Type(), "test") {
				log.Warningf("Ignoring non-test target %s\n", spec)
				continue
			} else if command == "run" && !strings.HasSuffix(spec.Type(), "binary") {
				log.Warningf("Ignoring non-binary target %s\n", spec)
				continue
			}

			if len(targetsSpecified) == 0 {
				firstTargetSpecified = spec
			}

			targetsSpecified[spec.String()] = spec
			targetsToBuild[spec.String()] = spec

			err = util.CheckForDependencyCycles(spec)
			if err != nil {
				fmt.Printf("%s", err)
				return
			}

			err = spec.Target().Validate()
			if err != nil {
				fmt.Printf("%s", err)
				return
			}

			for _, spec := range spec.Target().Dependencies() {
				targetsToBuild[spec.String()] = spec
			}
		}
	}

	// Build the targets.
	jbuildCommands.BuildTargets(targetsToBuild)

	// Further process the targets.
	if command == "run" {
		firstTargetSpecified.Target().Run(flag.Args()[2:])
	} else if command == "test" {
		jbuildCommands.RunTests(targetsSpecified)
	}
}
