package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	jbuildCommands "github.com/jeshuam/jbuild/command"
	"github.com/jeshuam/jbuild/common"
	// "github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/config2"
	"github.com/jeshuam/jbuild/config2/interfaces"
	"github.com/jeshuam/jbuild/config2/util"
	"github.com/jeshuam/jbuild/progress"
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
	workspaceDir, _, err := config2.FindWorkspaceFile(cwd)
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
	// if !*jbuildCommands.UseProgress {
	// 	logging.SetLevel(logging.DEBUG, "jbuild")
	// } else {
	// 	logging.SetLevel(logging.CRITICAL, "jbuild")
	// }

	// First, see if we are in a workspace.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get cwd: %v", err)
	}

	if common.WorkspaceDir == "" {
		common.WorkspaceDir = findWorkspaceDir(cwd)
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
	targetsSpecified := make(map[string]interfaces.TargetSpec)
	targetsToBuild := make(map[string]interfaces.TargetSpec)

	// Use config v2.
	fmt.Printf("Using ConfigV2\n")
	for _, target := range targetArgs {
		spec, err := config2.MakeTargetSpec(target, common.CurrentDir)
		if err != nil {
			fmt.Printf("Failed: %s", err)
			return
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

	// Display each of the targets in the target cache (it's a good list!).
	for targetSpec := range util.TargetCache {
		fmt.Printf("Loaded %s\n", targetSpec)
	}

	for spec := range targetsToBuild {
		if util.ReadyToProcess(targetsToBuild[spec]) {
			fmt.Printf("Ready: %s\n", spec)
		}
	}

	// Build the targets.
	taskQueue := make(chan common.CmdSpec)
	for i := 0; i < *jbuildCommands.Threads; i++ {
		go func() {
			for {
				task := <-taskQueue
				common.RunCommand(task.Cmd, task.Result, task.Complete)
			}
		}()
	}

	if *jbuildCommands.UseProgress {
		if *jbuildCommands.SimpleProgress {
			// For simple progress bars, manually set the maximum number of ops.
			totalOps := 0
			for specName := range targetsToBuild {
				totalOps += targetsToBuild[specName].Target().TotalOps()
			}

			progress.SetTotalOps(totalOps)
			progress.Start()
		} else {
			fmt.Printf("\n\n")
			progress.StartComplex()
		}
	} else {
		progress.Disable()
	}

	var (
		results = make(chan config2.ProcessingResult)

		targetsStarted = make(map[string]bool, 0)
		targetsBuilt   = make(map[string]bool, 0)
	)

	for len(targetsBuilt) < len(targetsToBuild) {
		for specName, _ := range targetsToBuild {
			spec := targetsToBuild[specName]
			_, targetStarted := targetsStarted[specName]
			if !targetStarted && util.ReadyToProcess(spec) {
				log.Infof("Processing %s...", specName)
				progressBar := progress.AddBar(spec.Target().TotalOps(), specName)

				go func() {
					err := spec.Target().Process(progressBar, taskQueue)
					if err != nil {
						log.Fatalf("Error while processing %s: %v", specName, err)
					}

					results <- config2.ProcessingResult{spec, err}
				}()

				targetsStarted[specName] = true
			}
		}

		// Get results from running targets.
		result := <-results
		if result.Err != nil {
			log.Fatal(result.Err)
		} else {
			targetsBuilt[result.Spec.String()] = true
			log.Infof("Finished processing %s!", result.Spec)
		}
	}

	// jbuildCommands.BuildTargets(targetsToBuild)

	return

	// Convert the targets into their canonical format, i.e. the long format.
	// canonicalTargetSpecs := make([]*config.TargetSpec, 0, len(targetArgs))
	// for _, target := range targetArgs {
	// 	canonicalTargets, err := config.CanonicalTargetSpec(common.WorkspaceDir, cwd, target)
	// 	if err != nil {
	// 		log.Fatalf("Invalid target name '%s': %v", target, err)
	// 	}

	// 	canonicalTargetSpecs = append(canonicalTargetSpecs, canonicalTargets...)
	// }

	// // If more than one spec was specified and we are running, then error.
	// if len(canonicalTargetSpecs) > 1 && command == "run" {
	// 	log.Fatalf("Multiple targets specified with run command.")
	// }

	// /// Now that we have a list of target specs, we can go and load the targets.
	// /// This involves going to each target file
	// targetsSpecified := config.TargetSet{}
	// targetsToProcess := config.TargetSet{}
	// for _, targetSpec := range canonicalTargetSpecs {
	// 	// Load the target at the given specification.
	// 	target, err := config.LoadTarget(targetSpec)
	// 	if err != nil {
	// 		log.Fatalf("Could not load target '%s': %v", targetSpec, err)
	// 	}

	// 	// Make sure this target has no cycles.
	// 	target.CheckForDependencyCycles()

	// 	// If the target type doesn't match the command, then discard it.
	// 	if !target.IsBinary() && command == "run" {
	// 		log.Warningf("Ignoring target %s (type %s)", target, target.Type)
	// 	} else if !target.IsTest() && command == "test" {
	// 		log.Warningf("Ignoring target %s (type %s)", target, target.Type)
	// 	} else {
	// 		// Process this target, and all of it's dependencies.
	// 		targetsSpecified.Add(target)
	// 		targetsToProcess.Add(target)
	// 		for _, dep := range target.AllDependencies() {
	// 			targetsToProcess.Add(dep)
	// 		}
	// 	}
	// }

	// // If there are no targets to process, print a message and exit.
	// if len(targetsToProcess) == 0 {
	// 	log.Fatalf("No targets to process for command %s", command)
	// }
}
