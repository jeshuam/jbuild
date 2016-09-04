package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
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

	threads      = flag.Int("threads", runtime.NumCPU()+1, "Number of processing threads to use.")
	useProgress  = flag.Bool("progress_bars", true, "Whether or not to use progress bars.")
	forceRunTest = flag.Bool("force_run_tests", false, "Whether or not we should for tests to run.")

	validCommands = map[string]bool{
		"build": true,
		"test":  true,
		"run":   true,
	}
)

func main() {
	flag.Parse()

	// Setup the logger.
	logging.SetFormatter(format)
	if *useProgress {
		logging.SetLevel(logging.CRITICAL, "jbuild")
		progress.Start()
	} else {
		logging.SetLevel(logging.DEBUG, "jbuild")
		progress.Disable()
	}

	// Parse the command line arguments.
	if len(flag.Args()) < 2 {
		fmt.Println("Usage: jbuild [flags] build|test|run target <targets...>")
		return
	}

	// Get the current processing target.
	command := flag.Args()[0]
	targetArgs := flag.Args()[1:]
	runFlags := flag.Args()[2:]

	// Validate the command arg.
	if !validCommands[command] {
		fmt.Printf("Unknown command '%s'.\n", command)
		return
	}

	// If we are running, there should only be a single target.
	if command == "run" {
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

	/// Now that we have a list of target specs, we can go and load the targets.
	/// This involves going to each target file
	var firstTargetSpecified *config.Target = nil
	targetsSpecified := make(map[*config.Target]bool, 0)
	targetsToProcess := make(map[*config.Target]bool, 0)
	for _, targetSpec := range canonicalTargetSpecs {
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
		targetsToProcess[target] = true
		targetsSpecified[target] = true
		for _, dep := range target.AllDependencies() {
			targetsToProcess[dep] = true
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

	/// Now we have a list of targets we want to process, the next step is to
	/// actually process them! To process them, we will use a series of processors
	/// depending on the type of the target.
	newTargetsToProcess := make(map[*config.Target]bool, 0)
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
				newTargetsToProcess[target] = true
			}
		}

		targetsToProcess = newTargetsToProcess
		newTargetsToProcess = map[*config.Target]bool{}

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
		cPrint := color.New(color.FgHiBlue, color.Bold).PrintfFunc()
		if runtime.GOOS == "windows" {
			cPrint = color.New(color.FgHiCyan, color.Bold).PrintfFunc()
		}
		cPrint("\n$ %s\n", strings.Join(cmd.Args, " "))
		cmd.Run()
	}

	if command == "test" {
		log.Info("Testing...")

		// Get some colored prints.
		gPrint := color.New(color.FgHiGreen, color.Bold).SprintfFunc()
		rPrint := color.New(color.FgHiRed, color.Bold).SprintfFunc()

		// Get the output.
		fmt.Printf("\n")

		// Function to display the test result.
		getTestResultFunction := func(target *config.Target, cached bool) func(error, time.Duration) {
			return func(err error, duration time.Duration) {
				// Save the test results.
				var passed bool
				if err != nil {
					barrier := "================================================================="
					msg := fmt.Sprintf("FAILED in %s", duration)
					if cached {
						msg = msg + " (cached)"
					}
					fmt.Printf("\t%s: %s\n", rPrint(msg), target)
					fmt.Printf("\n%s\n%s%s\n", barrier, err, barrier)
					passed = false
				} else {
					msg := fmt.Sprintf("PASSED in %s", duration)
					if cached {
						msg = msg + " (cached)"
					}
					fmt.Printf("\t%s: %s\n", gPrint(msg), target)
					passed = true
				}

				// Save the test results.
				common.SaveTestResult(target.Output[0], passed, fmt.Sprintf("%s", err), duration)
			}
		}

		// Function to run a single test.
		runTest := func(target *config.Target, results chan error) {
			// First, look for a cached result file.
			result := common.LoadTestResult(target.Output[0])
			if result != nil {
				if result.Passed {
					getTestResultFunction(target, true)(nil, result.Duration)
				} else {
					getTestResultFunction(target, true)(errors.New(result.Result), result.Duration)
				}

				results <- nil
				return
			}

			// If no cached result was found, then run the command again.
			cmd := exec.Command(target.Output[0], "--gtest_color=yes")
			common.RunCommand(cmd, results, getTestResultFunction(target, false))
		}

		// Run the commands.
		results := make(chan error)
		for target, _ := range targetsSpecified {
			go runTest(target, results)
		}

		// Collect all results.
		for range targetsSpecified {
			<-results
		}
	}
}
