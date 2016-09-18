package interfaces

type Spec interface {
	// Path returns the fully-qualified, OS-compliant path to the target or file
	// in the spec. If the spec is a file, then it will include the filename,
	// otherwise it will only include the directory.
	Path() string
	OutputPath() string

	// Return the directory of the spec.
	Dir() string
	Name() string

	// IsTarget returns true iff this spec refers to a target.
	IsTarget() bool

	Target() Target

	// Display the spec as a string. This should use / as the path separator by
	// default.
	String() string

	Init() error

	Validate() error

	Type() string
}
