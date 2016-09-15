package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

	threads        = flag.Int("threads", runtime.NumCPU()+1, "Number of processing threads to use.")
	testThreads    = flag.Int("test_threads", runtime.NumCPU()/2, "Number of threads to use when running tests.")
	useProgress    = flag.Bool("progress_bars", true, "Whether or not to use progress bars.")
	simpleProgress = flag.Bool("use_simple_progress", true, "Use the simple progress system rather than multiple bars.")
	forceRunTests  = flag.Bool("force_run_tests", false, "Whether or not we should for tests to run.")
	testRuns       = flag.Uint("test_runs", 1, "Number of times to run each test.")
	testOutput     = flag.String("test_output", "errors", "The verbosity of test output to show. Can be all|errors|none.")

	validCommands = map[string]bool{
		"build": true,
		"test":  true,
		"run":   true,
		"clean": true,
	}
)

func main() {
	flag.Parse()

	// If testRuns is provided, then force tests to run.
	if *testRuns > 1 {
		*forceRunTests = true
	}

	// Setup the logger.
	logging.SetFormatter(format)
	if *useProgress {
		logging.SetLevel(logging.CRITICAL, "jbuild")
		if *simpleProgress {
			progress.Start()
		} else {
			fmt.Printf("\n\n")
			progress.StartComplex()
		}
	} else {
		logging.SetLevel(logging.DEBUG, "jbuild")
		progress.Disable()
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

	// If we are cleaning, just delete the output directory.
	if command == "clean" {
		cPrint("$ rm -rf %s", filepath.Join(workspaceDir, common.OutputDirectory))
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
		targetArgs = []string{targetArgs[0]}
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

	// If we are using simple progress bars, then pre-set the total number of ops.
	if *simpleProgress {
		totalOps := 0
		for target, _ := range targetsToProcess {
			totalOps += target.TotalOps()
		}

		progress.SetTotalOps(totalOps)
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
		cmd.Dir = firstTargetSpecified.Spec.OutputPath()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
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

		// Function to run a single test.
		testResults := map[string][]common.TestResult{}
		testReusltsMutex := sync.Mutex{}
		concurrentTestSemaphore := make(chan bool, *testThreads)
		runTest := func(target *config.Target, results chan error) {
			// First, look for a cached result file.
			if !*forceRunTests {
				result := common.LoadTestResult(target.Output[0])
				if result != nil {
					testReusltsMutex.Lock()
					testResults[target.Spec.String()] = append(testResults[target.Spec.String()], *result)
					testReusltsMutex.Unlock()
					results <- nil
					return
				}
			}

			// If no cached result was found, then run the command again.
			cmd := exec.Command(target.Output[0], "--gtest_color=yes")
			concurrentTestSemaphore <- true
			common.RunCommand(cmd, results, func(output string, success bool, d time.Duration) {
				testReusltsMutex.Lock()
				testResults[target.Spec.String()] = append(
					testResults[target.Spec.String()],
					common.TestResult{success, output, d, false})
				common.SaveTestResult(target.Output[0], success, output, d)
				testReusltsMutex.Unlock()
				<-concurrentTestSemaphore
			})
		}

		// Run the commands.
		results := make(chan error)
		timesToRunEachTest := *testRuns
		var i uint
		for target, _ := range targetsSpecified {
			for i = 0; i < timesToRunEachTest; i++ {
				go runTest(target, results)
			}
		}

		// Wait for all targets to be processed.
		totalTestRuns := uint(len(targetsSpecified)) * timesToRunEachTest
		for i = 0; i < totalTestRuns; i++ {
			<-results
		}

		// Display the results.
		for target, results := range testResults {
			var (
				nPasses, nFails int
				totalDuration   time.Duration
				cached          bool
			)

			// Aggregate the results.
			for _, result := range results {
				totalDuration += result.Duration
				if result.Cached {
					cached = true
				}

				if result.Passed {
					nPasses++
				} else {
					nFails++
				}
			}

			// Find the average duration.
			averageDuration := totalDuration / time.Duration(len(results))

			// Display the results.
			if nFails > 0 {
				msg := fmt.Sprintf("FAILED in %s", averageDuration)
				if len(results) > 1 {
					msg += " (mean)"
				}

				if cached {
					msg += " (cached)"
				}

				if nFails > 1 {
					msg += fmt.Sprintf(", %d/%d runs failed", nFails, len(results))
				}

				fmt.Printf("\t%s: %s\n", rPrint(msg), target)

				// If we failed and only did a single run, the display the result. We
				// don't want to display the results if there were multiple runs.
				if len(results) == 1 && *testOutput != "none" {
					barrier := "================================================================="
					fmt.Printf("\n%s\n%s%s\n", barrier, results[0].Result, barrier)
				}
			} else {
				msg := fmt.Sprintf("PASSED in %s", averageDuration)
				if len(results) > 1 {
					msg += " (mean)"
				}

				if cached {
					msg += " (cached)"
				}

				fmt.Printf("\t%s: %s\n", gPrint(msg), target)
				if len(results) == 1 && *testOutput == "all" {
					barrier := "================================================================="
					fmt.Printf("\n%s\n%s%s\n", barrier, results[0].Result, barrier)
				}
			}
		}
	}
}
