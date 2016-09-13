package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	jbuildCommands "github.com/jeshuam/jbuild/command"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/processor"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

var (
	log    = logging.MustGetLogger("jbuild")
	format = logging.MustStringFormatter(
		`%{color}%{level:.1s} %{shortfunc}() >%{color:reset} %{message}`)

	threads        = flag.Int("threads", runtime.NumCPU()+1, "Number of processing threads to use.")
	useProgress    = flag.Bool("progress_bars", true, "Whether or not to use progress bars.")
	simpleProgress = flag.Bool("use_simple_progress", true, "Use the simple progress system rather than multiple bars.")

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
	if !*useProgress {
		logging.SetLevel(logging.DEBUG, "jbuild")
	} else {
		logging.SetLevel(logging.CRITICAL, "jbuild")
	}

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

	/// Make some goroutines which can be used to run commands.
	taskQueue := make(chan common.CmdSpec)
	for i := 0; i < *threads; i++ {
		go func() {
			for {
				task := <-taskQueue
				common.RunCommand(task.Cmd, task.Result, task.Complete)
			}
		}()
	}

	// Enable or disable logging.
	if *useProgress {
		if *simpleProgress {
			progress.Start()
		} else {
			fmt.Printf("\n\n")
			progress.StartComplex()
		}
	}

	// If we are using simple progress bars, then pre-set the total number of ops.
	if *simpleProgress {
		totalOps := 0
		for target := range targetsToProcess {
			totalOps += target.TotalOps()
		}

		progress.SetTotalOps(totalOps)
	}

	/// Now we have a list of targets we want to process, the next step is to
	/// actually process them! To process them, we will use a series of processors
	/// depending on the type of the target.
	newTargetsToProcess := config.TargetSet{}
	targetChannel := make(chan processor.ProcessingResult)
	nCompletedTargets := 0
	nTargetsToProcess := len(targetsToProcess)
	for nCompletedTargets < nTargetsToProcess {
		// Process all targets we need to; do nothing if there are no targets that
		// need processing.
		for target, _ := range targetsToProcess {
			if target.ReadyToProcess() {
				log.Infof("Processing %s...", target)
				err := processor.Process(target, targetChannel, taskQueue)
				if err != nil {
					log.Fatalf("Error while processing %s: %v", target, err)
				}
			} else {
				newTargetsToProcess.Add(target)
			}
		}

		targetsToProcess = newTargetsToProcess
		newTargetsToProcess = config.TargetSet{}

		// Wait for some process to respond.
		result := <-targetChannel
		if result.Err != nil {
			log.Fatal(result.Err)
		} else {
			nCompletedTargets++
			log.Infof("Finished processing %s!", result.Target)
		}
	}

	// Finish the progress bars.
	progress.Finish()

	// If we were running, there should only be one argument. Just run it.
	if command == "run" {
		cmd := exec.Command(firstTargetSpecified.Output[0], runFlags...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cPrint("\n$ %s\n", strings.Join(cmd.Args, " "))
		cmd.Run()
	}

	if command == "test" {
		jbuildCommands.RunTests(targetsSpecified)
	}
}
