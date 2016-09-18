package config2

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/client9/xson/hjson"
	"github.com/jeshuam/jbuild/common"
)

// Error to return with BUILD file related errors.
type buildFileError struct {
	File, Msg string
}

func (this *buildFileError) Error() string {
	return fmt.Sprintf("Could not load BUILD file %s: %v", this.File, this.Msg)
}

func makeBuildFileError(file, msg string) *buildFileError {
	err := new(buildFileError)
	err.File = file
	err.Msg = msg
	return err
}

func FindWorkspaceFile(cwd string) (string, string, error) {
	workspaceFile := "WORKSPACE"

	path_parts := strings.Split(cwd, pathSeparator)
	for i := len(path_parts); i > 0; i-- {
		workspaceDir := strings.Join(path_parts[:i], pathSeparator)
		workspaceFilePath := workspaceDir + pathSeparator + workspaceFile

		// Check this path for a WORKSPACE file. If it exists, return.
		if common.FileExists(workspaceFilePath) {
			return workspaceDir, workspaceFile, nil
		}
	}

	// Could not find file; this is fatal.
	msg := fmt.Sprintf("Could not find %s file starting at %s", workspaceFile, cwd)
	return "", "", errors.New(msg)
}

// Load a BUILD file at the given path. Will return a generic map representing
// the contents of the BUILD file, where the keys are the target names.
func LoadBuildFile(path string) (map[string]interface{}, error) {
	if !common.FileExists(path) {
		return nil, makeBuildFileError(path, "file not found")
	}

	// Load the BUILD file.
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, makeBuildFileError(path, err.Error())
	}

	targetsJson := make(map[string]interface{})
	err = hjson.Unmarshal(content, &targetsJson)
	if err != nil {
		return nil, makeBuildFileError(path, err.Error())
	}

	return targetsJson, nil
}
