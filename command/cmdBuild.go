package command

import (
	"fmt"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/jeshuam/jbuild/progress"
)

func setupProgressBars(targetsToBuild map[string]interfaces.TargetSpec) {
	if !args.ShowLog {
		if args.UseSimpleProgress {
			// For simple progress bars, manually set the maximum number of ops.
			totalOps := 0
			for targetSpec := range targetsToBuild {
				totalOps += targetsToBuild[targetSpec].Target().TotalOps()
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
}

func buildTargets(targetsToBuild map[string]interfaces.TargetSpec, taskQueue chan common.CmdSpec) {
	var (
		results = make(chan config.ProcessingResult)

		targetsStarted = make(map[string]bool, len(targetsToBuild))
		targetsBuilt   = make(map[string]bool, len(targetsToBuild))
	)

	for len(targetsBuilt) < len(targetsToBuild) {
		for specName, _ := range targetsToBuild {
			spec := targetsToBuild[specName]
			_, targetStarted := targetsStarted[specName]
			if !targetStarted && util.ReadyToProcess(spec) {
				log.Infof("Processing %s...", specName)
				var progressBar *progress.ProgressBar
				if spec.Target().TotalOps() > 0 {
					progressBar = progress.AddBar(spec.Target().TotalOps(), specName)
				}

				go func() {
					err := spec.Target().Process(progressBar, taskQueue)
					if err != nil {
						log.Fatalf("Error while processing %s: %v", specName, err)
					}

					results <- config.ProcessingResult{spec, err}
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
}

func BuildTargets(targetsToBuild map[string]interfaces.TargetSpec) {
	// Make a task queue, which runs commands that are passed to it.
	taskQueue := make(chan common.CmdSpec)
	for i := 0; i < args.Threads; i++ {
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

	if !args.ShowLog {
		progress.Finish()
	}
}
