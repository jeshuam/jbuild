package config

import (
	"fmt"
	"io/ioutil"

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
