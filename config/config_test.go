package config

import (
	"strings"
	"testing"
)

func TestWorkspaceFile(t *testing.T) {
	// Keep track of the old internal functions.
	_oldFileExists := fileExists
	_oldPathSeparator := pathSeparator
	_oldWorkspaceFilename := *workspaceFilename

	// Mock function which returns true for all paths with a specific prefix.
	var mockFileExists = func(filepath string) bool {
		return strings.HasPrefix(filepath, "/test/exists")
	}

	// Replace the internals.
	pathSeparator = "/"
	fileExists = mockFileExists

	/// Run tests.
	path, filename, err := WorkspaceFile("/test/exists")
	if err != nil || path != "/test/exists" || filename != "WORKSPACE" {
		t.Fatalf("Did not find file which existed in the first path searched")
	}

	path, filename, err = WorkspaceFile("/test/exists/some/other/path")
	if err != nil || path != "/test/exists/some/other/path" || filename != "WORKSPACE" {
		t.Fatalf("Did not find file which existed in subsequent paths searched")
	}

	path, filename, err = WorkspaceFile("/test/not_exists")
	if err == nil {
		t.Fatalf("File reported found when it should not have been")
	}

	path, filename, err = WorkspaceFile("")
	if err == nil {
		t.Fatalf("File reported found when no argument passed")
	}

	// Make sure the workspace filename flag is used correctly.
	*workspaceFilename = "CUSTOM"
	path, filename, err = WorkspaceFile("/test/exists")
	if filename != "CUSTOM" {
		t.Fatalf("Did not use the workspace_filename flag correctly")
	}

	// Restore the old internal functions.
	*workspaceFilename = _oldWorkspaceFilename
	pathSeparator = _oldPathSeparator
	fileExists = _oldFileExists
}
