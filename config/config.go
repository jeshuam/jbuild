package config

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/client9/xson/hjson"
	"github.com/op/go-logging"
)

var (
	log               = logging.MustGetLogger("jbuild")
	workspaceFilename = flag.String("workspace_filename", "WORKSPACE", "Name of the WORKSPACE file at the root of the tree.")
	buildFilename     = flag.String("build_filename", "BUILD", "Name of the BUILD file in each directory.")

	// Save any loaded BUILD file content to a cache for easy recall.
	buildFileCache = make(map[string]map[string]interface{})
)

// Make this module variables to allow for simple testing.
var pathSeparator = string(os.PathSeparator)
var fileExists = func(filepath string) bool {
	_, err := os.Stat(filepath)
	if err == nil {
		return true
	}

	return false
}

// Load the the workspace file based on the current directory. This will scan
// upwards until it finds a WORKSPACE file.
func FindWorkspaceFile(cwd string) (string, string, error) {
	workspaceFile := *workspaceFilename

	path_parts := strings.Split(cwd, pathSeparator)
	for i := len(path_parts); i > 0; i-- {
		workspaceDir := strings.Join(path_parts[:i], pathSeparator)
		workspaceFilePath := workspaceDir + pathSeparator + workspaceFile

		// Check this path for a WORKSPACE file. If it exists, return.
		if fileExists(workspaceFilePath) {
			return workspaceDir, workspaceFile, nil
		}
	}

	// Could not find file; this is fatal.
	msg := fmt.Sprintf("Could not find %s file starting at %s", workspaceFile, cwd)
	return "", "", errors.New(msg)
}

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
	// If the BUILD file doesn't exist, then stop.
	targetsJSON, inCache := buildFileCache[path]
	if !inCache {
		if !fileExists(path) {
			return nil, makeBuildFileError(path, "file not found")
		}

		// Load the BUILD file.
		log.Debugf("Loading build file %s", path)
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, makeBuildFileError(path, err.Error())
		}

		targetsJSON = make(map[string]interface{})
		err = hjson.Unmarshal(content, &targetsJSON)
		if err != nil {
			return nil, makeBuildFileError(path, err.Error())
		}

		buildFileCache[path] = targetsJSON
	}

	return targetsJSON, nil
}
