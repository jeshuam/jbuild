package config

import (
	"testing"
)

func TestMakeFileSpecForFullySpecifiedTarget(t *testing.T) {
	fileExists = func(filepath string) bool {
		return false
	}

	spec, err := MakeFileSpec("//path/to/target:name")
	if err != nil {
		t.Fa
	}

	if spec.Path != "path/to/target" {
		t.Fail()
	}

	if spec.Name != "name" {
		t.Fail()
	}

	if spec.String() != "//path/to/target:name" {
		t.Fail()
	}
}
