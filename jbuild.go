package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
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

func main() {
	flag.Parse()

	// Setup the logger.
	logging.SetFormatter(format)

	// Save a nice printing color.
	cPrint := color.New(color.FgHiBlue, color.Bold).PrintfFunc()
	if runtime.GOOS == "windows" {
		cPrint = color.New(color.FgHiCyan, color.Bold).PrintfFunc()
	}

	// Make sure at least the command was passed.
	if len(flag.Args()) < 1 {
		fmt.Println("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
		return
	}

	// Get the command.
	command := flag.Args()[0]

	// If we are cleaning, just delete the output directory.
	if command == "clean" {
		cPrint("$ rm -rf %s", common.OutputDirectory)
		err := os.RemoveAll(common.OutputDirectory)
		if err != nil {
			fmt.Printf("error: %s\n", err)
		}

		return
	}

	// If we aren't cleaning, get more arguments.
	if len(flag.Args()) < 2 {
		fmt.Println("Usage: jbuild [flags] build|test|run|clean [target [targets...]]")
		return
	}

	// Get the current processing target.
	targetArgs := flag.Args()[1:]
	runFlags := flag.Args()[2:]

	// Validate the command arg.
	if !validCommands[command] {
		fmt.Printf("Unknown command '%s'.\n", command)
		return
	}

	// If we are running, there should only be a single target.
	if command == "run" {
		if strings.HasSuffix(targetArgs[0], ":all") {
			fmt.Printf("Invalid specified :all for command run.")
			return
		}

		targetArgs = []string{targetArgs[0]}
	}

	/// First, find the root of the workspace
	// Get the current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get cwd: %v", err)
	}

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

	/// Convert the targets into their canonical format, i.e. the long format.
	canonicalTargetSpecs := make([]*config.TargetSpec, len(targetArgs))
	for i, target := range targetArgs {
		canonicalTarget, err := config.CanonicalTargetSpec(workspaceDir, cwd, target)
		if err != nil {
			log.Fatalf("Invalid target name '%s': %v", target, err)
		}

		canonicalTargetSpecs[i] = canonicalTarget
	}

	/// Load any special, meta targets.
	expandedTargetSpecs := make([]*config.TargetSpec, 0, len(canonicalTargetSpecs))
	for _, targetSpec := range canonicalTargetSpecs {
		if targetSpec.Name == "all" {
			expandedTargets, err := config.ListTargetNames(targetSpec, command)
			if err != nil {
				log.Fatalf("Could not expand target '%s': %v", targetSpec, err)
			}

			for _, expandedSpec := range expandedTargets {
				expandedTargetSpecs = append(expandedTargetSpecs, expandedSpec)
			}

			cPrint("Expanding %s to %d targets.\n", targetSpec, len(expandedTargets))
		} else if targetSpec.Name == "..." {
			expandedTargets, err := config.ListTargetNamesRecursive(targetSpec, command)
			if err != nil {
				log.Fatalf("Could not expand target '%s': %v", targetSpec, err)
			}

			for _, expandedSpec := range expandedTargets {
				expandedTargetSpecs = append(expandedTargetSpecs, expandedSpec)
			}
		} else {
			expandedTargetSpecs = append(expandedTargetSpecs, targetSpec)
		}
	}

	/// Now that we have a list of target specs, we can go and load the targets.
	/// This involves going to each target file
	var firstTargetSpecified *config.Target = nil
	targetsSpecified := config.TargetSet{}
	targetsToProcess := config.TargetSet{}
	for _, targetSpec := range expandedTargetSpecs {
		target, err := config.LoadTarget(targetSpec)
		if err != nil {
			log.Fatalf("Could not load target '%s': %v", targetSpec, err)
		}

		// If the target is not runnable, but we were told to run, then fail.
		if !target.IsBinary() && command == "run" {
			fmt.Printf("Cannot run target of type %s (%s)\n", target.Type, target)
			return
		}

		if !target.IsTest() && command == "test" {
			fmt.Printf("Cannot test target of type %s (%s)\n", target.Type, target)
			return
		}

		if len(targetsSpecified) == 0 {
			firstTargetSpecified = target
		}

		target.CheckForDependencyCycles()
		targetsToProcess.Add(target)
		targetsSpecified.Add(target)
		for _, dep := range target.AllDependencies() {
			targetsToProcess.Add(dep)
		}
	}

	// First, build all targets.
	jbuildCommands.BuildTargets(targetsToProcess)

	// If we were running, there should only be one argument. Just run it.
	if command == "run" {
		jbuildCommands.Run(firstTargetSpecified, runFlags)
	} else if command == "test" {
		jbuildCommands.RunTests(targetsSpecified)
	}
}
