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

	// Filename should return the full name of the file.
	Filename() string

	// FsWorkspacePath should return the path to the root of the workspace.
	FsWorkspacePath() string

	// FsOutputPath should return the path to where the output file is kept.
	FsOutputPath() string
	FsOutputDir() string

	// FsPath should return the fully OS path to the file.
	FsPath() string

	IsGenerated() bool
}

type DirSpec interface {
	Spec

	// FsWorkspacePath should return the path to the root of the workspace.
	FsWorkspacePath() string

	// FsOutputPath should return the path to where the output file is kept.
	FsOutputPath() string

	// FsPath should return the fully OS path to the file.
	FsPath() string
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

	// Dependencies should return a list of the direct dependencies of this target
	// spec. This is to determine which targets must be processed before. If all
	// is set, return all direct and indirect dependencies (i.e. recursive).
	Dependencies(all bool) []TargetSpec

	ReadyToProcess() bool
}
