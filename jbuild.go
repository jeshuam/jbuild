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

	threads     = flag.Int("threads", runtime.NumCPU()+1, "Number of processing threads to use.")
	useProgress = flag.Bool("progress_bars", true, "Whether or not to use progress bars.")

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
		logging.SetLevel(logging.INFO, "jbuild")
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
	targetsSpecified := make([]*config.Target, 0)
	targetsToProcess := make([]*config.Target, 0)
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

		target.CheckForDependencyCycles()
		targetsToProcess = append(targetsToProcess, target)
		targetsSpecified = append(targetsSpecified, target)
		targetsToProcess = append(targetsToProcess, target.AllDependencies()...)
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

	// Log a starting message.
	cPrint := color.New(color.FgHiBlue, color.Bold).PrintfFunc()
	cPrint("\n$ jbuild %s %s\n\n\n", command, strings.Join(targetArgs, " "))

	/// Now we have a list of targets we want to process, the next step is to
	/// actually process them! To process them, we will use a series of processors
	/// depending on the type of the target.
	newTargetsToProcess := make([]*config.Target, 0, len(targetsToProcess))
	targetChannel := make(chan processor.ProcessingResult)
	nCompletedTargets := 0
	nTargetsToProcess := len(targetsToProcess)
	for nCompletedTargets < nTargetsToProcess {
		// Process all targets we need to; do nothing if there are no targets that
		// need processing.
		for _, target := range targetsToProcess {
			if target.ReadyToProcess() {
				log.Infof("Processing %s...", target)
				err := processor.Process(target, targetChannel, taskQueue)
				if err != nil {
					log.Fatalf("Error while processing %s: %v", target, err)
				}
			} else {
				newTargetsToProcess = append(newTargetsToProcess, target)
			}
		}

		targetsToProcess = newTargetsToProcess
		newTargetsToProcess = []*config.Target{}

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
		cmd := exec.Command(targetsSpecified[0].Output[0], runFlags...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		log.Infof("Running %s...", cmd.Args)
		cmd.Run()
	}

	if command == "test" {
		log.Info("Testing...")
		result := make(chan error)
		for _, target := range targetsSpecified {
			go func() {
				cmd := exec.Command(target.Output[0])
				common.RunCommand(cmd, result, func() {})
			}()
		}

		// Get the output.
		for i := 0; i < len(targetsSpecified); i++ {
			err := <-result
			if err != nil {
				log.Errorf("FAILED: %s", err)
			}
		}
	}
}
