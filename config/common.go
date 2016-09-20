package config

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/fatih/camelcase"
	"github.com/jeshuam/jbuild/config/cc"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/interfaces"
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
func LoadTargetSpecs(json map[string]interface{}, key, cwd string) ([]interfaces.Spec, error) {
	// First, load the array of strings from the JSON object.
	rawSpecs := LoadStrings(json, key)

	// Place to store the final result.
	specs := make([]interfaces.Spec, 0, len(rawSpecs))

	// Now expand each glob. Note that things might not expand if they aren't
	// actually globs; that's OK. Start by making the target spec. There is no
	// need to actually load anything; we just want to know what the absolute
	// path relative to the workspace is.
	for _, rawSpec := range rawSpecs {
		// Try to load a set of globs, but only if they are prefixed by glob:.
		if strings.HasPrefix(rawSpec, "glob:") {
			globSpecs := MakeFileSpecGlob(strings.TrimLeft(rawSpec, "glob:"), cwd)
			if len(globSpecs) > 0 {
				specs = append(specs, globSpecs...)
				continue
			}
		} else {
			fileSpec := MakeFileSpec(rawSpec, cwd)
			if fileSpec != nil {
				specs = append(specs, fileSpec)
				continue
			}
		}

		targetSpecs, targetErr := MakeTargetSpec(rawSpec, cwd)
		if len(targetSpecs) > 0 {
			for _, spec := range targetSpecs {
				specs = append(specs, spec)
			}
			continue
		}

		dirSpec := MakeDirSpec(rawSpec, cwd)
		if dirSpec != nil {
			specs = append(specs, dirSpec)
			continue
		}

		// If we got here, it wasn't a valid spec.
		return nil, targetErr
	}

	return specs, nil
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

func LoadTargetFromJson(spec interfaces.TargetSpec, target interfaces.Target, targetJson map[string]interface{}) error {
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
			targetSpecs, err := LoadTargetSpecs(targetJson, fieldName, spec.Path())
			if err != nil {
				return err
			}

			platformTargetSpecs, err := LoadTargetSpecs(platformOptionsJson, fieldName, spec.Path())
			if err != nil {
				return err
			}

			targetSpecs = append(targetSpecs, platformTargetSpecs...)
			targetField.Set(reflect.ValueOf(targetSpecs))

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
