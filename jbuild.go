package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/processor"
	"github.com/op/go-logging"
)

var (
	log    = logging.MustGetLogger("jbuild")
	format = logging.MustStringFormatter(
		`%{color}%{level:.1s} %{shortfunc}() >%{color:reset} %{message}`)
)

func main() {
	// Setup the logger.
	logging.SetFormatter(format)

	// Parse the command line arguments.
	targetArgs := flag.Args()
	if len(targetArgs) == 0 {
		fmt.Println("Usage: jbuild [flags] target <targets...>")
		return
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
	targetsToProcess := make([]*config.Target, 0)
	for _, targetSpec := range canonicalTargetSpecs {
		target, err := config.LoadTarget(targetSpec)
		if err != nil {
			log.Fatalf("Could not load target '%s': %v", targetSpec, err)
		}

		target.CheckForDependencyCycles()
		targetsToProcess = append(targetsToProcess, target)
		targetsToProcess = append(targetsToProcess, target.AllDependencies()...)
	}

	/// Now we have a list of targets we want to process, the next step is to
	/// actually process them! To process them, we will use a series of processors
	/// depending on the type of the target.
	newTargetsToProcess := make([]*config.Target, 0, len(targetsToProcess))
	targetChannel := make(chan processor.ProcessingResult)
	for len(targetsToProcess) > 0 {
		// Process all targets we need to; do nothing if there are no targets that
		// need processing.
		for _, target := range targetsToProcess {
			if target.ReadyToProcess() {
				log.Infof("Processing %s...", target)
				err := processor.Process(target, targetChannel)
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
			log.Infof("Finished processing %s!", result.Target)
		}
	}
}
