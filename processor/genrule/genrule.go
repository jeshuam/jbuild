package genrule

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
)

type GenruleProcessor struct {
}

func copyGenrule(target *config.Target) error {
	// Copy the input file to the output file.
	outputDir := filepath.Join(common.OutputDirectory, "gen", target.Spec.PathSystem())
	inputDir := filepath.Join(target.Spec.Workspace, target.Spec.PathSystem())
	outputFile := filepath.Join(outputDir, target.Options.Out)
	var inputFile string
	if len(target.Options.In) > 0 {
		inputFile = filepath.Join(inputDir, target.Options.In)
	} else if len(target.PlatformOptions.In) > 0 {
		inputFile = filepath.Join(inputDir, target.PlatformOptions.In)
	} else {
		return errors.New(fmt.Sprintf("Missing input file for genrule %s", target.Spec))
	}

	// Make the gen directory if possible.
	err := os.MkdirAll(outputDir, 0755)
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

func (p GenruleProcessor) Process(target *config.Target, taskQueue chan common.CmdSpec) error {
	defer target.ProgressBar.Finish()

	if strings.HasSuffix(target.Type, "copy") {
		return copyGenrule(target)
	}

	return nil
}
