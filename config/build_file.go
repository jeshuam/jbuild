package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/client9/xson/hjson"
	"github.com/jeshuam/jbuild/common"
)

// LoadBuildFile loads the BUILD specification file located at `path` and
// returns a generic key-value mapping as the result. The BUILD file is
// actually JSON, but we use hjson to make the config easier to write.
func LoadBuildFile(path string) (map[string]interface{}, error) {
	if !common.FileExists(path) {
		return nil, errors.New(fmt.Sprintf("BUILD file not found '%s'", path))
	}

	// Load the BUILD file.
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Could not read BUILD file '%s': %s", path, err))
	}

	targetsJson := make(map[string]interface{})
	err = hjson.Unmarshal(content, &targetsJson)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Could not load BUILD file '%s': %s", path, err))
	}

	return targetsJson, nil
}
