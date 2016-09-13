package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	jbuildCommands "github.com/jeshuam/jbuild/command"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
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
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get cwd: %v", err)
	}

	workspaceDir := findWorkspaceDir(cwd)

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
		jbuildCommands.Clean(workspaceDir)
		return
	}

	// If we aren't cleaning, get more arguments.
	if len(flag.Args()) < 2 {
		fmt.Println("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
		return
	}

	// Get the current processing target.
	targetArgs := flag.Args()[1:]

	// Convert the targets into their canonical format, i.e. the long format.
	canonicalTargetSpecs := make([]*config.TargetSpec, 0, len(targetArgs))
	for _, target := range targetArgs {
		canonicalTargets, err := config.CanonicalTargetSpec(workspaceDir, cwd, target)
		if err != nil {
			log.Fatalf("Invalid target name '%s': %v", target, err)
		}

		canonicalTargetSpecs = append(canonicalTargetSpecs, canonicalTargets...)
	}

	// If more than one spec was specified and we are running, then error.
	if len(canonicalTargetSpecs) > 1 && command == "run" {
		log.Fatalf("Multiple targets specified with run command.")
	}

	/// Now that we have a list of target specs, we can go and load the targets.
	/// This involves going to each target file
	targetsSpecified := config.TargetSet{}
	targetsToProcess := config.TargetSet{}
	for _, targetSpec := range canonicalTargetSpecs {
		// Load the target at the given specification.
		target, err := config.LoadTarget(targetSpec)
		if err != nil {
			log.Fatalf("Could not load target '%s': %v", targetSpec, err)
		}

		// Make sure this target has no cycles.
		target.CheckForDependencyCycles()

		// If the target type doesn't match the command, then discard it.
		if !target.IsBinary() && command == "run" {
			log.Warningf("Ignoring target %s (type %s)", target, target.Type)
		} else if !target.IsTest() && command == "test" {
			log.Warningf("Ignoring target %s (type %s)", target, target.Type)
		} else {
			// Process this target, and all of it's dependencies.
			targetsSpecified.Add(target)
			targetsToProcess.Add(target)
			for _, dep := range target.AllDependencies() {
				targetsToProcess.Add(dep)
			}
		}
	}

	// If there are no targets to process, print a message and exit.
	if len(targetsToProcess) == 0 {
		log.Fatalf("No targets to process for command %s", command)
	}

	log.Infof("Processing %d targets: %s\n", len(targetsToProcess), targetsToProcess)

	// First, build all targets.
	jbuildCommands.BuildTargets(targetsToProcess)

	// If we were running, there should only be one argument. Just run it.
	if command == "run" {
		runTarget, _ := config.LoadTarget(canonicalTargetSpecs[0])
		jbuildCommands.Run(runTarget, flag.Args()[2:])
	} else if command == "test" {
		jbuildCommands.RunTests(targetsSpecified)
	}
}
