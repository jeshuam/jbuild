package interfaces

type Spec interface {
	// Dir should return the directory that this Spec references, relative to the
	// root of the workspace.
	Dir() string

	// Path should return the fully-qualified OS path to the directory this Spec
	// references.
	Path() string

	// String should return a string representation of this spec. This value must
	// be unique within each workspace.
	String() string

	// Return the type of the spec or contained target. Should be unique.
	Type() string
}

type FileSpec interface {
	Spec

	// File should return the filename that this FileSpec references.
	File() string

	// FilePath should return the fully-qualified OS path to the file this Spec
	// references.
	FilePath() string

	// OutputPath should return the fully-qualified OS path to the output
	// directory for this target.
	OutputPath() string

	// WorkspacePath should return the path relative to the root of the workspace.
	WorkspacePath() string
}

type DirSpec interface {
	Spec

	// WorkspacePath should return the path relative to the root of the workspace.
	WorkspacePath() string
}

type TargetSpec interface {
	Spec

	// Name should return the name of this TargetSpec.
	Name() string

	// Target should return a reference to the target this spec refers to.
	Target() Target

	// OutputPath should return the fully-qualified OS path to the output
	// directory for this target.
	OutputPath() string
}
