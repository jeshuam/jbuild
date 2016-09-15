package genrule

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/mattn/go-zglob"
)

type GenruleProcessor struct {
}

func inputFile(target *config.Target) (string, error) {
	var inputFile string
	if len(target.Options.In) > 0 {
		inputFile = target.Options.In
	} else if len(target.PlatformOptions.In) > 0 {
		inputFile = target.PlatformOptions.In
	} else {
		return "", errors.New(fmt.Sprintf("Missing input file for genrule %s", target.Spec))
	}

	return filepath.Join(target.Spec.WorkspacePath(), inputFile), nil
}

func outputDir(target *config.Target) string {
	return filepath.Join(common.OutputDirectory, "gen", target.Spec.PathSystem())
}

func copyGenrule(target *config.Target) error {
	// Copy the input file to the output file.
	outputFile := filepath.Join(outputDir(target), target.Options.Out)
	inputFile, err := inputFile(target)
	if err != nil {
		return err
	}

	// Make the gen directory if possible.
	err = os.MkdirAll(outputDir(target), 0755)
	if err != nil {
		return err
	}

	// Check if we have to do anything.
	inStat, _ := os.Stat(inputFile)
	outStat, _ := os.Stat(outputFile)
	if outStat != nil && outStat.ModTime().After(inStat.ModTime()) {
		return nil
	}

	if common.FileExists(outputFile) {
		os.Remove(outputFile)
	}

	err = os.Link(inputFile, outputFile)
	if err != nil {
		return err
	}

	return nil
}

func scriptGenrule(target *config.Target, taskQueue chan common.CmdSpec) error {
	// Get the script filename.
	scriptFile, err := inputFile(target)
	if err != nil {
		return err
	}

	// Make a temporary directory in which to run the script.
	tempDir, err := ioutil.TempDir("", "genrule_"+target.Spec.Name)
	defer os.RemoveAll(tempDir)
	if err != nil {
		return err
	}

	// Run the script in this directory.
	result := make(chan error)
	cmd := exec.Command(scriptFile)
	cmd.Dir = tempDir
	taskQueue <- common.CmdSpec{cmd, result, func(string, bool, time.Duration) {}}

	// Wait for the result.
	err = <-result
	if err != nil {
		return err
	}

	// Copy each of the files created in the temp directory to the generated out
	// directory.
	files, err := zglob.Glob(filepath.Join(tempDir, "**", "*"))
	if err != nil {
		return err
	}

	for _, file := range files {
		relPath, err := filepath.Rel(tempDir, filepath.Dir(file))
		if err != nil {
			return err
		}

		// Make the directory for this output file.
		err = os.MkdirAll(filepath.Join(outputDir(target), relPath), 0755)
		if err != nil {
			return err
		}

		// Only copy the file if the generated file was created before the script
		// was updated.
		outputFile := filepath.Join(outputDir(target), relPath, filepath.Base(file))
		inStat, _ := os.Stat(scriptFile)
		outStat, _ := os.Stat(outputFile)
		if outStat != nil && outStat.ModTime().After(inStat.ModTime()) {
			return nil
		}

		if common.FileExists(outputFile) {
			os.Remove(outputFile)
		}

		// Copy the file.
		err = os.Link(file, outputFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p GenruleProcessor) Process(target *config.Target, taskQueue chan common.CmdSpec) error {
	defer target.ProgressBar.Finish()

	if strings.HasSuffix(target.Type, "copy") {
		return copyGenrule(target)
	} else if strings.HasSuffix(target.Type, "script") {
		return scriptGenrule(target, taskQueue)
	} else {
		return errors.New("Invalid genrule type " + target.Type)
	}

	return nil
}
