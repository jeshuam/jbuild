package config2

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/fatih/camelcase"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config2/cc"
	"github.com/jeshuam/jbuild/config2/filegroup"
	"github.com/jeshuam/jbuild/config2/interfaces"
	"github.com/mattn/go-zglob"
)

var (
	Glob = zglob.Glob
)

type ProcessingResult struct {
	Spec interfaces.Spec
	Err  error
}

////////////////////////////////////////////////////////////////////////////////
//                          Target Utility Functions                          //
////////////////////////////////////////////////////////////////////////////////

// Load a list of FileSpecs from a JSON map. The values are all globs by
// default.
func LoadTargetSpecs(json map[string]interface{}, key, base string) ([]interfaces.Spec, error) {
	// First, load the array of strings from the JSON object.
	globs := LoadStrings(json, key)

	// Place to store the final result.
	targetSpecs := make([]interfaces.Spec, 0, len(globs))

	// Now expand each glob. Note that things might not expand if they aren't
	// actually globs; that's OK. Start by making the target spec. There is no
	// need to actually load anything; we just want to know what the absolute
	// path relative to the workspace is.
	for _, glob := range globs {
		globSpec := glob
		if !strings.HasPrefix(glob, "//") {
			globSpec = "//" + strings.Replace(filepath.Join(base, glob), pathSeparator, "/", -1)
		}

		// Try to load the globs. If that doesn't work, then it must just be a
		// normal target.
		globFiles, err := Glob(filepath.Join(common.WorkspaceDir, base, glob))
		if err != nil {
			targetSpecs = append(targetSpecs, MakeTargetSpec(globSpec))
		} else {
			// Add all of the glob files found.
			for _, globFile := range globFiles {
				globFileRel, err := filepath.Rel(common.WorkspaceDir, globFile)
				if err != nil {
					return nil, err
				}

				globFile = strings.Replace(globFileRel, pathSeparator, "/", -1)
				targetSpecs = append(targetSpecs, MakeTargetSpec("//"+globFile))
			}
		}
	}

	// Initialize all of the target specs.
	for _, targetSpec := range targetSpecs {
		err := targetSpec.Init()
		if err != nil {
			return nil, err
		}
	}

	return targetSpecs, nil
}

// Load a list of strings from the given JSON map.
func LoadStrings(json map[string]interface{}, key string) []string {
	strings := make([]string, 0)
	stringArray, ok := json[key]
	if ok {
		for _, item := range stringArray.([]interface{}) {
			strings = append(strings, item.(string))
		}
	}

	return strings
}

func LoadTargetFromJson(spec interfaces.Spec, target interfaces.Target, targetJson map[string]interface{}) error {
	var targetType reflect.Type
	var targetValue reflect.Value
	switch target.(type) {
	case *cc.Target:
		targetType = reflect.TypeOf(*target.(*cc.Target))
		targetValue = reflect.ValueOf(target.(*cc.Target))
	case *filegroup.Target:
		targetType = reflect.TypeOf(*target.(*filegroup.Target))
		targetValue = reflect.ValueOf(target.(*filegroup.Target))
	default:
		return errors.New(fmt.Sprintf("Cannot load unknown target type."))
	}

	// Load platform specific options.
	platformOptionsJsonInterface, ok := targetJson[runtime.GOOS]
	platformOptionsJson := make(map[string]interface{})
	if ok {
		platformOptionsJson = platformOptionsJsonInterface.(map[string]interface{})
	}

	// Populate the target. Do this by going through each attribute in the struct
	for i := 0; i < targetType.NumField(); i++ {
		fieldName := strings.ToLower(strings.Join(camelcase.Split(targetType.Field(i).Name), "_"))
		fieldType := targetType.Field(i).Type
		targetField := targetValue.Elem().Field(i)

		// If the field starts with an _, ignore it.
		if strings.HasPrefix(fieldName, "_") {
			continue
		}

		// Ignore the output files.
		if fieldName == "output" {
			continue
		}

		// If the field name is "spec", this is a special case. We should store the
		// target's spec here.
		// TODO(jeshua): find a better way to exposing the spec to the target.
		if fieldName == "spec" {
			targetField.Set(reflect.ValueOf(spec))
			continue
		}

		switch fieldType {
		case reflect.TypeOf([]interfaces.Spec{}):
			targetSpecs, err := LoadTargetSpecs(targetJson, fieldName, spec.Dir())
			if err != nil {
				return err
			}

			platformTargetSpecs, err := LoadTargetSpecs(platformOptionsJson, fieldName, spec.Dir())
			if err != nil {
				return err
			}

			allTargetSpecs := append(targetSpecs, platformTargetSpecs...)
			targetField.Set(reflect.ValueOf(allTargetSpecs))

		case reflect.TypeOf([]string{}):
			options := LoadStrings(targetJson, fieldName)
			options = append(options, LoadStrings(platformOptionsJson, fieldName)...)
			targetField.Set(reflect.ValueOf(options))

		case reflect.TypeOf(cc.Binary):
			switch spec.Type() {
			case "c++/binary":
				targetField.Set(reflect.ValueOf(cc.Binary))
			case "c++/library":
				targetField.Set(reflect.ValueOf(cc.Library))
			case "c++/test":
				targetField.Set(reflect.ValueOf(cc.Test))
			default:
				return errors.New(fmt.Sprintf("Invalid C++ target type %s", spec.Type()))
			}

		default:
			return errors.New(fmt.Sprintf("Unknown field type %s", fieldType))
		}
	}

	return nil
}
