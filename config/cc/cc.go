package cc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

var (
	log   = logging.MustGetLogger("jbuild")
	locks = make(map[string]*sync.Mutex)
)

// Compile the source files within the given target.
func compileFiles(args *args.Args, target *Target, progressBar *progress.ProgressBar, taskQueue chan common.CmdSpec, force bool) ([]string, int, error) {
	objs := make([]string, 0, len(target.srcs()))
	results := make(chan error, len(target.srcs()))
	nCompiled := 0

	for _, srcFile := range target.srcs() {
		// Display the source file we are building.
		progressBar.SetSuffix(srcFile.String())

		// Work out the full path to the source file. This will need to be provided
		// to the compiler.
		srcPath := srcFile.FilePath()
		objPath := filepath.Join(srcFile.OutputPath(), srcFile.File()) + ".o"
		objs = append(objs, objPath)

		// Make the directory of the obj if needed.
		err := os.MkdirAll(filepath.Dir(objPath), 0755)
		if err != nil {
			return nil, 0, err
		}

		// If the object is newer than the source file, don't compile it again.
		if !force {
			srcStat, _ := os.Stat(srcPath)
			objStat, _ := os.Stat(objPath)
			srcChanged := true
			depsChanged := false
			if objStat != nil {
				depsChanged = target.depsChangedSince(objStat)
				srcChanged = !objStat.ModTime().After(srcStat.ModTime())
			}

			// Recompile this file if the deps or src has changed.
			if !depsChanged && !srcChanged {
				progressBar.Increment()
				continue
			}
		}

		// Get the lock for this src file.
		lock, ok := locks[srcPath]
		if !ok {
			lock = new(sync.Mutex)
			locks[srcPath] = lock
		}

		// Build the compilation command.
		log.Debugf("... compile %s", srcFile)
		cmd := compileCommand(args, target, srcPath, objPath)

		// Run the command.
		nCompiled++
		taskQueue <- common.CmdSpec{cmd, lock, results, func(string, bool, time.Duration) {
			progressBar.Increment()
		}}
	}

	// Check results.
	for i := 0; i < nCompiled; i++ {
		err := <-results
		if err != nil {
			return nil, 0, err
		}
	}

	return objs, nCompiled, nil
}

func linkObjects(args *args.Args, target *Target, progressBar *progress.ProgressBar, taskQueue chan common.CmdSpec, objects []string, nCompiled int) (string, error) {
	// Throw and error if there are no source files and this isn't a library.
	if target.IsBinary() && (len(target.srcs()) == 0 && len(target.Deps) == 0) {
		return "", errors.New(fmt.Sprintf("No source files/deps found for binary %s", target))
	}

	// Work out the output filepath.
	outputPath := target.OutputPath()
	if nCompiled == 0 && common.FileExists(outputPath) && !target.depsUpdated() {
		progressBar.Increment()
		return outputPath, nil
	}

	// Make the error channel.
	result := make(chan error)

	// Get the lock for this src file.
	lock, ok := locks[outputPath]
	if !ok {
		lock = new(sync.Mutex)
		locks[outputPath] = lock
	}

	// Now, we need to build up the command to run.
	log.Debugf("... link %s", outputPath)
	cmd := linkCommand(args, target, objects, outputPath)

	// Run the command.
	taskQueue <- common.CmdSpec{cmd, lock, result, func(string, bool, time.Duration) {
		progressBar.Increment()
	}}

	err := <-result
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func copyData(target *Target, progressBar *progress.ProgressBar) error {
	for _, dataSpec := range target.data() {
		inputFile := dataSpec.FilePath()
		outputFile := filepath.Join(dataSpec.OutputPath(), dataSpec.File())
		dataStat, _ := os.Stat(inputFile)
		dataOutStat, _ := os.Stat(outputFile)
		if dataOutStat != nil && dataStat.ModTime().After(dataOutStat.ModTime()) {
			os.Remove(outputFile)
		}

		// If the output file doesn't exist, then copy it.
		if !common.FileExists(outputFile) {
			err := os.Link(inputFile, outputFile)
			if err != nil {
				return err
			}
		}

		progressBar.Increment()
	}

	return nil
}
