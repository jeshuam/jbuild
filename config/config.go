package config

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/etgryphon/stringUp"
	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/config/cc"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/mattn/go-zglob"
)

var (
	Glob = zglob.Glob
)

////////////////////////////////////////////////////////////////////////////////
//                          Target Utility Functions                          //
////////////////////////////////////////////////////////////////////////////////

// Load a list of FileSpecs from a JSON map. The values are all globs by
// default.
func loadSpecs(args *args.Args, json map[string]interface{}, key, cwd, buildBase string) ([]interfaces.Spec, error) {
	// First, load the array of strings from the JSON object.
	rawSpecs := loadStrings(json, key)

	// Place to store the final result.
	specs := make([]interfaces.Spec, 0, len(rawSpecs))

	// Now try to load each spec. To do this, attempt to load each different spec
	// type in turn. If none of them work, then this spec must be invalid.
	for _, rawSpec := range rawSpecs {
		// Try to load a set of globs, but only if they are prefixed by glob:.
		if strings.HasPrefix(rawSpec, "glob:") {
			globSpecs := MakeFileSpecGlob(args, strings.TrimPrefix(rawSpec, "glob:"), cwd, buildBase)
			if len(globSpecs) > 0 {
				specs = append(specs, globSpecs...)
				continue
			}
		} else {
			fileSpec := MakeFileSpec(args, rawSpec, cwd, buildBase)
			if fileSpec != nil {
				specs = append(specs, fileSpec)
				continue
			}
		}

		dirSpec := MakeDirSpec(args, rawSpec, cwd, buildBase)
		if dirSpec != nil {
			specs = append(specs, dirSpec)
			continue
		}

		targetSpecs, err := MakeTargetSpec(args, rawSpec, cwd, buildBase)
		if err != nil {
			return nil, err
		}

		if len(targetSpecs) > 0 {
			for _, spec := range targetSpecs {
				specs = append(specs, spec)
			}

			continue
		}

		// If we got here, it wasn't a valid spec.
		return nil, errors.New(fmt.Sprintf("Could not identify type of spec '%s'", rawSpec))
	}

	return specs, nil
}

// Load a list of TargetSpecs from a JSON map.
func loadTargetSpecs(args *args.Args, json map[string]interface{}, key, cwd, buildBase string) ([]interfaces.TargetSpec, error) {
	rawSpecs := loadStrings(json, key)
	targetSpecs := make([]interfaces.TargetSpec, 0, len(rawSpecs))
	for _, rawSpec := range rawSpecs {
		targetSpec, err := MakeTargetSpec(args, rawSpec, cwd, buildBase)
		if err != nil {
			return nil, err
		}

		if len(targetSpec) == 0 {
			return nil, errors.New(fmt.Sprintf("Could not make TargetSpec '%s'", rawSpec))
		}

		targetSpecs = append(targetSpecs, targetSpec...)
	}

	return targetSpecs, nil
}

// Load a list of DirSpecs from a JSON map.
func loadDirSpecs(args *args.Args, json map[string]interface{}, key, cwd, buildBase string) ([]interfaces.DirSpec, error) {
	rawSpecs := loadStrings(json, key)
	dirSpecs := make([]interfaces.DirSpec, 0, len(rawSpecs))
	for _, rawSpec := range rawSpecs {
		dirSpec := MakeDirSpec(args, rawSpec, cwd, buildBase)
		if dirSpec == nil {
			return nil, errors.New(fmt.Sprintf("Could not make DirSpec '%s'", rawSpec))
		}

		dirSpecs = append(dirSpecs, dirSpec)
	}

	return dirSpecs, nil
}

// loadStrings loads a list of strings from the given key.
func loadStrings(json map[string]interface{}, key string) []string {
	strings := make([]string, 0)
	stringArray, ok := json[key]
	if ok {
		for _, item := range stringArray.([]interface{}) {
			strings = append(strings, item.(string))
		}
	}

	return strings
}

func loadJson(
	args *args.Args,
	targetType reflect.Type,
	targetValue reflect.Value,
	json map[string]interface{},
	key string,
	spec interfaces.TargetSpec,
	buildBase string,
	errorOnUnknownField bool) error {

	// If this is a platform specific options key, ignore it.
	if key == "linux" || key == "windows" || key == "darwin" {
		return nil
	}

	// Try to find a field of this name.
	fieldName := stringUp.CamelCase(strings.Title(key))
	fieldValue := targetValue.Elem().FieldByName(fieldName)

	// If this field doesn't exist, throw an error.
	if !fieldValue.IsValid() {
		if errorOnUnknownField {
			return errors.New(fmt.Sprintf("Unknown field '%s' in '%s'", key, spec))
		} else {
			return nil
		}
	}

	// This field is valid, let's load it.
	fieldType, _ := targetType.FieldByName(fieldName)
	allowedTypes := make(map[string]bool)
	allowedTypesRaw := strings.Split(fieldType.Tag.Get("types"), ",")
	for _, allowedType := range allowedTypesRaw {
		allowedTypes[allowedType] = true
	}

	switch fieldType.Type {
	case reflect.TypeOf([]interfaces.Spec{}):
		specs, err := loadSpecs(args, json, key, spec.Dir(), buildBase)
		if err != nil {
			return err
		}

		// Make sure all specs are valid types.
		for _, newSpec := range specs {
			if !allowedTypes[newSpec.Type()] {
				return errors.New(
					fmt.Sprintf("Invalid spec type '%s' (%s) in field '%s' of '%s', allowed = %s",
						newSpec.Type(), newSpec, key, spec, allowedTypesRaw))
			}
		}

		currentVal := fieldValue.Interface().([]interfaces.Spec)
		currentVal = append(currentVal, specs...)
		fieldValue.Set(reflect.ValueOf(currentVal))

	case reflect.TypeOf([]interfaces.DirSpec{}):
		dirSpecs, err := loadDirSpecs(args, json, key, spec.Dir(), buildBase)
		if err != nil {
			return err
		}

		currentVal := fieldValue.Interface().([]interfaces.DirSpec)
		currentVal = append(currentVal, dirSpecs...)
		fieldValue.Set(reflect.ValueOf(currentVal))

	case reflect.TypeOf([]interfaces.TargetSpec{}):
		targetSpecs, err := loadTargetSpecs(args, json, key, spec.Dir(), buildBase)
		if err != nil {
			return err
		}

		// Make sure all targetSpecs are valid types.
		for _, targetSpec := range targetSpecs {
			if !allowedTypes[targetSpec.Type()] {
				return errors.New(
					fmt.Sprintf("Invalid spec type '%s' (%s) in field '%s' of '%s', allowed = %s",
						targetSpec.Type(), targetSpec, key, spec, allowedTypesRaw))
			}
		}

		currentVal := fieldValue.Interface().([]interfaces.TargetSpec)
		currentVal = append(currentVal, targetSpecs...)
		fieldValue.Set(reflect.ValueOf(currentVal))

	case reflect.TypeOf([]string{}):
		currentVal := fieldValue.Interface().([]string)
		currentVal = append(currentVal, loadStrings(json, key)...)

		// Save the values.
		fieldValue.Set(reflect.ValueOf(currentVal))

	case reflect.TypeOf("string"):
		fieldValue.Set(reflect.ValueOf(json[key].(string)))

	case reflect.TypeOf(cc.Binary):
		switch spec.Type() {
		case "c++/binary":
			fieldValue.Set(reflect.ValueOf(cc.Binary))
		case "c++/library":
			fieldValue.Set(reflect.ValueOf(cc.Library))
		case "c++/test":
			fieldValue.Set(reflect.ValueOf(cc.Test))
		default:
			return errors.New(fmt.Sprintf("Invalid C++ target type '%s' for '%s'", spec.Type(), spec))
		}

	default:
		return errors.New(fmt.Sprintf("Unknown field type '%s' in '%s'", fieldType.Type, spec))
	}

	return nil
}

func LoadTargetFromJson(args *args.Args, spec interfaces.TargetSpec, target interfaces.Target, targetJson map[string]interface{}, buildBase string) error {
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
	platformOptionsJson := make(map[string]interface{})
	platformOptionsJsonInterface, ok := targetJson[runtime.GOOS]
	if ok {
		platformOptionsJson = platformOptionsJsonInterface.(map[string]interface{})
	}

	// If the target has a Spec field, then populate it.
	specFieldValue := targetValue.Elem().FieldByName("Spec")
	if specFieldValue.IsValid() {
		specFieldValue.Set(reflect.ValueOf(spec))
	}

	// If the target has a Args field, the populate it.
	argsFieldValue := targetValue.Elem().FieldByName("Args")
	if argsFieldValue.IsValid() {
		argsFieldValue.Set(reflect.ValueOf(args))
	}

	// Iterate through each field in the JSON. THis will allow us to log messages
	// when unknown arguments have been provided.
	for jsonKey := range targetJson {
		err := loadJson(args, targetType, targetValue, targetJson, jsonKey, spec, buildBase, true)
		if err != nil {
			return err
		}
	}

	// Load target specific JSON.
	for jsonKey := range platformOptionsJson {
		err := loadJson(args, targetType, targetValue, platformOptionsJson, jsonKey, spec, buildBase, true)
		if err != nil {
			return err
		}
	}

	for jsonKey := range args.WorkspaceOptions {
		err := loadJson(args, targetType, targetValue, args.WorkspaceOptions, jsonKey, spec, buildBase, false)
		if err != nil {
			return err
		}
	}

	for jsonKey := range args.ConfigurationOptions {
		err := loadJson(args, targetType, targetValue, args.ConfigurationOptions, jsonKey, spec, buildBase, false)
		if err != nil {
			return err
		}
	}

	log.Infof("Loaded target %s", spec)

	return nil
}
