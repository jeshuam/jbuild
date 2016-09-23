package command

import (
	"errors"
	"fmt"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

type processingResult struct {
	Spec interfaces.TargetSpec
	Err  error
}

func setupProgressBars(args *args.Args, targetsToBuild map[string]interfaces.TargetSpec) {
	if !args.ShowLog {
		if args.UseSimpleProgress {
			// For simple progress bars, manually set the maximum number of ops.
			totalOps := 0
			for _, targetSpec := range targetsToBuild {
				totalOps += targetSpec.Target().TotalOps()
			}

			progress.SetTotalOps(totalOps)
			progress.Start()
		} else {
			fmt.Printf("\n\n")
			progress.StartComplex()
		}
	}
}

func buildTargets(args *args.Args, targetsToBuild map[string]interfaces.TargetSpec, taskQueue chan common.CmdSpec) error {
	var (
		log     = logging.MustGetLogger("jbuild")
		results = make(chan processingResult)

		targetsStarted = make(map[string]bool, len(targetsToBuild))
		targetsBuilt   = make(map[string]bool, len(targetsToBuild))
	)

	for len(targetsBuilt) < len(targetsToBuild) {
		startedThisRound := 0
		log.Info("Starting new round of targets...")
		for _, spec := range targetsToBuild {
			_, targetStarted := targetsStarted[spec.String()]
			if targetStarted {
				log.Infof("Skipping %s, already started.", spec)
			}

			// if !util.ReadyToProcess(spec) {
			// 	log.Infof("Skipping %s, not ready to process.", spec)
			// }
			if !targetStarted && util.ReadyToProcess(spec) {
				log.Infof("Processing %s...", spec)

				// Start processing this target.
				go func(spec interfaces.TargetSpec) {
					// Setup the progress bar if necessary.
					var progressBar *progress.ProgressBar
					if spec.Target().TotalOps() > 0 {
						progressBar = progress.AddBar(spec.Target().TotalOps(), spec.String())
					}

					err := spec.Target().Process(args, progressBar, taskQueue)
					fmt.Printf("GOT HERE FOR %s\n", spec)
					results <- processingResult{spec, err}
				}(spec)

				targetsStarted[spec.String()] = true
				startedThisRound++
			}
		}

		if startedThisRound == 0 {
			return errors.New("No target started this round!")
		}

		for i := 0; i < startedThisRound; i++ {
			// Get results from running targets.
			result := <-results
			if result.Err != nil {
				return result.Err
			} else {
				targetsBuilt[result.Spec.String()] = true
				log.Infof("Finished processing %s!", result.Spec)
			}
		}
	}

	return nil
}

func BuildTargets(args *args.Args, targetsToBuild map[string]interfaces.TargetSpec) error {
	// Make a task queue, which runs commands that are passed to it.
	taskQueue := make(chan common.CmdSpec)
	for i := 0; i < args.Threads; i++ {
		go func() {
			for {
				task := <-taskQueue
				common.RunCommand(args, task.Cmd, task.Result, task.Complete)
			}
		}()
	}

	// Setup the progress bar display.
	setupProgressBars(args, targetsToBuild)

	// Keep looping until all targets are built. By the time this is called, any
	// cycles should have been found already.
	err := buildTargets(args, targetsToBuild, taskQueue)
	if err != nil {
		return err
	}

	if !args.ShowLog {
		progress.Finish()
	}

	return nil
}
