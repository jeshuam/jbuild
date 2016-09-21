package util

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/deckarep/golang-set"
	"github.com/jeshuam/jbuild/config/interfaces"
)

var (
	// A cache from file spec strings --> targets. This will allow the same target
	// to be loaded multiple times with no performance degradations.
	TargetCache = make(map[string]interfaces.Target, 0)

	// A cache of file spec --> file spec object.
	SpecCache = make(map[string]interfaces.Spec, 0)
)

func GetAllDependencies(specs []interfaces.Spec) []interfaces.TargetSpec {
	deps := make([]interfaces.TargetSpec, 0, len(specs))
	for _, spec := range specs {
		switch spec.(type) {
		case interfaces.TargetSpec:
			deps = append(deps, spec.(interfaces.TargetSpec))
			deps = append(deps, spec.(interfaces.TargetSpec).Target().AllDependencies()...)
		}
	}

	return deps
}

func GetDependencies(specs []interfaces.Spec) []interfaces.TargetSpec {
	deps := make([]interfaces.TargetSpec, 0, len(specs))
	for _, spec := range specs {
		switch spec.(type) {
		case interfaces.TargetSpec:
			deps = append(deps, spec.(interfaces.TargetSpec))
		}
	}

	return deps
}

func EnsureDependenciesAreOfType(specs []interfaces.Spec, types mapset.Set) error {
	// for _, spec := range specs {
	// 	if !types.Contains(spec.Type()) {
	// 		return errors.New(fmt.Sprintf("Invalid type %s: options are %s", spec.Type(), types))
	// 	}
	// }

	return nil
}

func checkForDependencyCyclesRecurse(
	spec interfaces.Spec, visited []string, seq int) error {
	target := TargetCache[spec.String()]

	// If this node has already been visited in the current recursive stack, then
	// there must be a cycle in the graph.
	for i := 0; i < seq; i++ {
		if visited[i] == spec.String() {
			cycle := strings.Join(visited[i:seq], " --> ")
			return errors.New(fmt.Sprintf("Cycle detected: %s --> %s", cycle, spec))
		}
	}

	// Add this item to sequence number `seq`. If the array is already big enough,
	// then just replace what is in there. Otherwise, add a new element.
	if len(visited) > seq {
		visited[seq] = spec.String()
	} else {
		visited = append(visited, spec.String())
	}

	// Look through each child of the current target and recurse to it, first
	// adding this node to the visited list.
	for _, dep := range target.Dependencies() {
		err := checkForDependencyCyclesRecurse(dep, visited, seq+1)
		if err != nil {
			return err
		}
	}

	return nil
}

func CheckForDependencyCycles(spec interfaces.Spec) error {
	visited := make([]string, 0, 1)
	visited = append(visited, spec.String())
	target := TargetCache[spec.String()]
	for _, dep := range target.Dependencies() {
		err := checkForDependencyCyclesRecurse(dep, visited, 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadyToProcess(spec interfaces.TargetSpec) bool {
	for _, dep := range spec.Target().AllDependencies() {
		if !dep.Target().Processed() {
			return false
		}
	}

	return true
}

func OSPathToWSPath(path string) string {
	return "//" + strings.Trim(
		strings.Replace(path, string(os.PathSeparator), "/", -1), "/")
}
