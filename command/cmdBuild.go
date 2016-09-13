package command

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/processor"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

var (
	useProgress    = flag.Bool("progress_bars", true, "Whether or not to use progress bars.")
	simpleProgress = flag.Bool("use_simple_progress", true, "Use the simple progress system rather than multiple bars.")

	threads = flag.Int("threads", runtime.NumCPU()+1, "Number of processing threads to use.")
)

func init() {
	if !*useProgress {
		logging.SetLevel(logging.DEBUG, "jbuild")
	} else {
		logging.SetLevel(logging.CRITICAL, "jbuild")
	}
}

func setupProgressBars(targetsToBuild config.TargetSet) {
	if *useProgress {
		if *simpleProgress {
			// For simple progress bars, manually set the maximum number of ops.
			totalOps := 0
			for target := range targetsToBuild {
				totalOps += target.TotalOps()
			}

			progress.SetTotalOps(totalOps)
			progress.Start()
		} else {
			fmt.Printf("\n\n")
			progress.StartComplex()
		}
	}
}

func buildTargets(targetsToBuild config.TargetSet, taskQueue chan common.CmdSpec) {
	var (
		results = make(chan processor.ProcessingResult)

		targetsStarted = config.TargetSet{}
		targetsBuilt   = config.TargetSet{}
	)

	for len(targetsBuilt) < len(targetsToBuild) {
		for target, _ := range targetsToBuild {
			if !targetsStarted.Contains(target) && target.ReadyToProcess() {
				log.Infof("Processing %s...", target)
				err := processor.Process(target, results, taskQueue)
				if err != nil {
					log.Fatalf("Error while processing %s: %v", target, err)
				}

				targetsStarted.Add(target)
			}
		}

		// Get results from running targets.
		result := <-results
		if result.Err != nil {
			log.Fatal(result.Err)
		} else {
			targetsBuilt.Add(result.Target)
			log.Infof("Finished processing %s!", result.Target)
		}
	}
}

func BuildTargets(targetsToBuild config.TargetSet) {
	// Make a task queue, which runs commands that are passed to it.
	taskQueue := make(chan common.CmdSpec)
	for i := 0; i < *threads; i++ {
		go func() {
			for {
				task := <-taskQueue
				common.RunCommand(task.Cmd, task.Result, task.Complete)
			}
		}()
	}

	// Setup the progress bar display.
	setupProgressBars(targetsToBuild)

	// Keep looping until all targets are built. By the time this is called, any
	// cycles should have been found already.
	buildTargets(targetsToBuild, taskQueue)
	progress.Finish()
}
